package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/deepspace2/plugnpin/pkg/cli"
	"github.com/deepspace2/plugnpin/pkg/clients"
	"github.com/deepspace2/plugnpin/pkg/config"
	"github.com/deepspace2/plugnpin/pkg/logging"
	"github.com/deepspace2/plugnpin/pkg/processor"
)

var log = logging.GetLogger()

func main() {
	cliFlags := cli.ParseFlags()

	conf, err := config.GetEnvVars()
	if err != nil {
		log.Error("Failed to parse environment variables", "error", err)
		os.Exit(1)
	}

	if conf.Debug {
		logging.SetLevel(logging.DEBUG)
	} else {
		logging.SetLevel(logging.INFO)
	}

	if conf.RunInterval > 0 {
		log.Info(fmt.Sprintf("Will run every %v", conf.RunInterval))
	}

	dockerClients, adguardHomeClient, piholeClient, npmClient, err := clients.GetClients(cliFlags, conf)
	if err != nil {
		os.Exit(1)
	}

	proc := processor.New(dockerClients, adguardHomeClient, piholeClient, npmClient, cliFlags.DryRun)

	if conf.RunInterval == 0 {
		log.Info("RUN_INTERVAL is 0, will run once")
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

func shutdown(cancelCtx context.CancelFunc, wg *sync.WaitGroup) {
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGTERM)

	<-shutdownChan

	log.Info("Shutdown signal received, exiting gracefully.")
	cancelCtx()
	wg.Wait()
	log.Info("Shutdown complete.")
}
