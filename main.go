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
		msg += ", adding entries to pi-hole and nginx proxy mananger."
		log.Println(msg)

		piholeClient.AddDNSHostEntry(url, ip)

		npmClient.AddProxyHost(npm.ProxyHost{
			DomainNames:   []string{url},
			ForwardScheme: "http",
			ForwardHost:   ip,
			ForwardPort:   port,
			Locations:     []npm.Location{},
			Meta:          map[string]any{},
		})
	}
}

func run(dockerClient *docker.Client, piholeClient *pihole.Client, npmClient *npm.Client, dryRun bool) {
	containers, err := dockerClient.GetRelevantContainers()
	if err != nil {
		log.Printf("ERROR getting containers: %v", err)
		return
	}

	for _, container := range containers {
		handleContainer(container, piholeClient, npmClient, dryRun)
	}
	log.Println("Done")
}

func main() {
	cliFlags := cli.ParseFlags()

	envVars, err := config.GetEnvVars()
	if err != nil {
		log.Fatal(err)
	}

	runIntervalStr := envVars["RUN_INTERVAL"]
	runInterval := config.ParseRunInterval(runIntervalStr)
	if runInterval > 0 {
		log.Printf("Will run every %v", runIntervalStr)
	}

	piholeHost := envVars["PIHOLE_HOST"]
	piholeClient := pihole.NewClient(piholeHost)
	piHolePassword := envVars["PIHOLE_PASSWORD"]
	err = piholeClient.Login(piHolePassword)
	if err != nil {
		log.Fatal(err)
	}

	npmHost := envVars["NGINX_PROXY_MANAGER_HOST"]
	npmUser := envVars["NGINX_PROXY_MANAGER_USERNAME"]
	npmPassword := envVars["NGINX_PROXY_MANAGER_PASSWORD"]
	npmClient := npm.NewClient(npmHost, npmUser, npmPassword)
	err = npmClient.Login()
	if err != nil {
		log.Fatal(err)
	}

	dockerClient, err := docker.NewClient()
	if err != nil {
		log.Fatalf("ERROR creating docker client: %v", err)
	}

	defer dockerClient.Close()

	run(dockerClient, piholeClient, npmClient, cliFlags.DryRun)

	if runInterval == 0 {
		log.Println("RUN_INTERVAL is 0, exiting")
		return
	}

	ticker := time.NewTicker(runInterval)
	defer ticker.Stop()

	log.Printf("Will run again in %v. Press Ctrl+C to exit.", runInterval)

	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-ticker.C:
			run(dockerClient, piholeClient, npmClient, cliFlags.DryRun)
			log.Printf("Will run again in %v. Press Ctrl+C to exit.", runInterval)
		case <-shutdownChan:
			log.Println("Shutdown signal received, exiting.")
			return
		}
	}
}
