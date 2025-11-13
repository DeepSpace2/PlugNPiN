package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/deepspace2/plugnpin/pkg/cli"
	"github.com/deepspace2/plugnpin/pkg/clients/docker"
	"github.com/deepspace2/plugnpin/pkg/clients/npm"
	"github.com/deepspace2/plugnpin/pkg/clients/pihole"
	"github.com/deepspace2/plugnpin/pkg/config"
	"github.com/deepspace2/plugnpin/pkg/processor"
)

func shutdown(cancelCtx context.CancelFunc, wg *sync.WaitGroup) {
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGTERM)

	<-shutdownChan

	log.Println("Shutdown signal received, exiting gracefully.")
	cancelCtx()
	wg.Wait()
	log.Println("Shutdown complete.")
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

	var piholeClient *pihole.Client
	var npmClient *npm.Client

	if !cliFlags.DryRun {
		if !conf.PiholeDisabled {
			piholeClient = pihole.NewClient(conf.PiholeHost)
			err = piholeClient.Login(conf.PiholePassword)
			if err != nil {
				log.Fatalf("ERROR failed to login to Pi-Hole: %v", err)
			}
		}

		npmClient = npm.NewClient(conf.NpmHost, conf.NpmUsername, conf.NpmPassword)
		err = npmClient.Login()
		if err != nil {
			log.Fatalf("ERROR failed to login to Nginx Proxy Manager: %v", err)
		}
	}

	dockerClient, err := docker.NewClient()
	if err != nil {
		log.Fatalf("ERROR creating docker client: %v", err)
	}
	defer dockerClient.Close()

	proc := processor.New(dockerClient, piholeClient, npmClient, cliFlags.DryRun)

	if conf.RunInterval == 0 {
		log.Println("RUN_INTERVAL is 0, will run once")
		proc.RunOnce()
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()
		proc.ListenForEvents(ctx)
	}()

	go func() {
		defer wg.Done()
		proc.RunScheduled(ctx, conf.RunInterval)
	}()

	shutdown(cancel, &wg)
}
