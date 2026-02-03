package docker

import (
	"context"

	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
)

func Listen(ctx context.Context, dockerClient *Client, handler func(events.Message)) error {
	f := filters.NewArgs()
	f.Add("type", "container")
	f.Add("event", ContainerEvent.Start.String())
	f.Add("event", ContainerEvent.Die.String())

	log.Info("Listening for Docker events...", "host", dockerClient.DisplayHost)

	c, _ := dockerClient.Client.Client()

	messages, errs := c.Events(ctx, events.ListOptions{
		Filters: f,
	})

	for {
		select {
		case <-ctx.Done():
			log.Info("Stopping stream of Docker events", "host", dockerClient.DisplayHost)
			return ctx.Err()
		case event := <-messages:
			handler(event)
		case err := <-errs:
			if err != nil {
				log.Error("Failed to receive event", "host", dockerClient.DisplayHost, "error", err)
			}
			return err
		}
	}
}
