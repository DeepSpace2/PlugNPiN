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
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cliFlags := cli.ParseFlags()

	config, err := config.Get()
	if err != nil {
		log.Error("Failed to parse environment variables", "error", err)
		os.Exit(1)
	}

	if config.Debug {
		logging.SetLevel(logging.DEBUG)
	} else {
		logging.SetLevel(logging.INFO)
	}

	if config.RunInterval > 0 {
		log.Info(fmt.Sprintf("Will run every %v", config.RunInterval))
	}

	dockerClients, adguardHomeClient, piholeClient, npmClient, err := clients.GetClients(cliFlags, config)
	if err != nil {
		os.Exit(1)
	}

	proc := processor.New(dockerClients, adguardHomeClient, piholeClient, npmClient, cliFlags.DryRun)

	if config.RunInterval == 0 {
		log.Info("RUN_INTERVAL is 0, will run once")
		proc.RunOnce(ctx)
		return
	}

	var wg sync.WaitGroup

	wg.Go(func() {
		proc.ListenForEvents(ctx)
	})

	wg.Go(func() {
		proc.RunScheduled(ctx, config.RunInterval)
	})

	<-ctx.Done()
	log.Info("Shutdown signal received, exiting gracefully.")
	wg.Wait()
	log.Info("Shutdown complete.")
}
