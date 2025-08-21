package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/docker/docker/api/types/container"

	"github.com/deepspace2/plugnpin/pkg/cli"
	"github.com/deepspace2/plugnpin/pkg/clients/docker"
	"github.com/deepspace2/plugnpin/pkg/clients/npm"
	"github.com/deepspace2/plugnpin/pkg/clients/pihole"
	"github.com/deepspace2/plugnpin/pkg/config"
	"github.com/deepspace2/plugnpin/pkg/errors"
)

func handleContainer(container container.Summary, piholeClient *pihole.Client, npmClient *npm.Client, dryRun bool) {
	parsedContainerName := docker.GetParsedContainerName(container)

	ip, url, port, err := docker.GetValuesFromContainerLabels(container)
	if err != nil {
		switch err.(type) {
		case *errors.NonExistingLabelsError:
			log.Printf("Skipping container '%v': %v", parsedContainerName, err)
		case *errors.MalformedIPLabelError:
			log.Printf("ERROR handling container '%v': %v", parsedContainerName, err)
		}
		return
	}

	msg := fmt.Sprintf("Found labels for container '%v': ip=%v, port=%v, host=%v", parsedContainerName, ip, port, url)

	if dryRun {
		msg += ". In dry run mode, not doing anything."
		log.Println(msg)
	} else {
		msg += ", adding entries to Pi-Hole and Nginx Proxy Manager."
		log.Println(msg)

		err := piholeClient.AddDNSHostEntry(url, ip)
		if err != nil {
			log.Printf("ERROR failed to add entry to Pi-Hole: %v", err)
		}

		err = npmClient.AddProxyHost(npm.ProxyHost{
			DomainNames:   []string{url},
			ForwardScheme: "http",
			ForwardHost:   ip,
			ForwardPort:   port,
			Locations:     []npm.Location{},
			Meta:          map[string]any{},
		})
		if err != nil {
			log.Printf("ERROR failed to add entry to Nginx Proxy Manager: %v", err)
		}
	}
}

func run(dockerClient *docker.Client, piholeClient *pihole.Client, npmClient *npm.Client, dryRun bool) {
	containers, err := dockerClient.GetRelevantContainers()
	if err != nil {
		log.Printf("ERROR getting containers: %v", err)
		return
	}

	log.Printf("Found %v containers", len(containers))

	for _, container := range containers {
		handleContainer(container, piholeClient, npmClient, dryRun)
	}
	log.Println("Done")
}

func main() {
	cliFlags := cli.ParseFlags()

	conf, err := config.GetEnvVars()
	if err != nil {
		log.Fatal(err)
	}

	if conf.RunInterval > 0 {
		log.Printf("Will run every %v", conf.RunInterval)
	}

	piholeClient := pihole.NewClient(conf.PiholeHost)
	err = piholeClient.Login(conf.PiholePassword)
	if err != nil {
		log.Fatalf("ERROR failed to login to Pi-Hole: %v", err)
	}

	npmClient := npm.NewClient(conf.NpmHost, conf.NpmUsername, conf.NpmPassword)
	err = npmClient.Login()
	if err != nil {
		log.Fatalf("ERROR failed to login to Nginx Proxy Manager: %v", err)
	}

	dockerClient, err := docker.NewClient()
	if err != nil {
		log.Fatalf("ERROR creating docker client: %v", err)
	}

	defer dockerClient.Close()

	run(dockerClient, piholeClient, npmClient, cliFlags.DryRun)

	if conf.RunInterval == 0 {
		log.Println("RUN_INTERVAL is 0, exiting")
		return
	}

	ticker := time.NewTicker(conf.RunInterval)
	defer ticker.Stop()

	log.Printf("Will run again in %v. Press Ctrl+C to exit.", conf.RunInterval)

	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-ticker.C:
			run(dockerClient, piholeClient, npmClient, cliFlags.DryRun)
			log.Printf("Will run again in %v. Press Ctrl+C to exit.", conf.RunInterval)
		case <-shutdownChan:
			log.Println("Shutdown signal received, exiting.")
			return
		}
	}
}
