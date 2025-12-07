package docker

import (
	"context"

	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	dockerClient "github.com/docker/docker/client"
)

func Listen(ctx context.Context, handler func(events.Message)) error {
	c, err := dockerClient.NewClientWithOpts(dockerClient.WithHostFromEnv())
	if err != nil {
		return err
	}
	defer c.Close()

	f := filters.NewArgs()
	f.Add("type", "container")
	f.Add("event", ContainerEvent.Start.String())
	f.Add("event", ContainerEvent.Stop.String())
	f.Add("event", ContainerEvent.Kill.String())

	log.Info("Listening for Docker events...")

	messages, errs := c.Events(ctx, events.ListOptions{
		Filters: f,
	})

	for {
		select {
		case <-ctx.Done():
			log.Info("Stopping stream of Docker events")
			return ctx.Err()
		case event := <-messages:
			handler(event)
		case err := <-errs:
			if err != nil {
				log.Error("Failed to receive event", "error", err)
			}
			return err
		}
	}
}
