package processor

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/deepspace2/plugnpin/pkg/clients/docker"
	"github.com/deepspace2/plugnpin/pkg/clients/npm"
	"github.com/deepspace2/plugnpin/pkg/clients/pihole"
	"github.com/deepspace2/plugnpin/pkg/errors"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
)

type Processor struct {
	dockerClient *docker.Client
	piholeClient *pihole.Client
	npmClient    *npm.Client
	dryRun       bool
}

func New(dockerClient *docker.Client, piholeClient *pihole.Client, npmClient *npm.Client, dryRun bool) *Processor {
	return &Processor{
		dockerClient: dockerClient,
		piholeClient: piholeClient,
		npmClient:    npmClient,
		dryRun:       dryRun,
	}
}

func (p *Processor) RunScheduled(ctx context.Context, interval time.Duration) {
	p.RunOnce()

	if interval == 0 {
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("Will run again in %v. Press Ctrl+C to exit.", interval)

	for {
		select {
		case <-ticker.C:
			p.RunOnce()
			log.Printf("Will run again in %v. Press Ctrl+C to exit.", interval)
		case <-ctx.Done():
			return
		}
	}
}

func (p *Processor) ListenForEvents(ctx context.Context) {
	err := docker.Listen(ctx, func(event events.Message) {
		p.handleDockerEvent(event)
	})

	if err != nil && err != context.Canceled {
		log.Printf("ERROR: Docker event listener stopped: %v", err)
	}
}

func (p *Processor) RunOnce() {
	containers, err := p.dockerClient.GetRelevantContainers()
	if err != nil {
		log.Printf("ERROR getting containers: %v", err)
		return
	}

	log.Printf("Found %v containers", len(containers))

	for _, container := range containers {
		p.preprocessContainer(container)
	}
	log.Println("Done")
}

func (p *Processor) preprocessContainer(container container.Summary) {
	parsedContainerName := docker.GetParsedContainerName(container)

	ip, url, port, err := docker.GetValuesFromLabels(container.Labels)
	if err != nil {
		switch err.(type) {
		case *errors.NonExistingLabelsError:
			log.Printf("Skipping container '%v': %v", parsedContainerName, err)
		case *errors.MalformedIPLabelError:
			log.Printf("ERROR handling container '%v': %v", parsedContainerName, err)
		}
		return
	}
	p.processContainer(parsedContainerName, "start", ip, url, port)
}

func (p *Processor) handleDockerEvent(event events.Message) {
	containerName, ok := event.Actor.Attributes["name"]
	if !ok {
		log.Printf("Skipping event for container with no name: %v", event.Actor.ID)
		return
	}

	ip, url, port, err := docker.GetValuesFromLabels(event.Actor.Attributes)
	if err != nil {
		switch err.(type) {
		case *errors.NonExistingLabelsError:
			// This is not an error, it just means the container is not relevant for us
			return
		case *errors.MalformedIPLabelError:
			log.Printf("ERROR handling event for container '%v': %v", containerName, err)
		}
		return
	}
	p.processContainer(containerName, string(event.Action), ip, url, port)
}

func (p *Processor) processContainer(name, action, ip, url string, port int) {
	msg := fmt.Sprintf("Handling container '%v': ip=%v, port=%v, host=%v", name, ip, port, url)

	if p.dryRun {
		msg += ". In dry run mode, not doing anything."
		log.Println(msg)
		return
	}

	log.Println(msg)

	switch action {
	case "start":
		log.Printf("Adding entry to Pi-Hole for container '%v'", name)
		err := p.piholeClient.AddDNSHostEntry(url, ip)
		if err != nil {
			log.Printf("ERROR failed to add entry to Pi-Hole: %v", err)
		}

		log.Printf("Adding entry to Nginx Proxy Manager for container '%v'", name)
		err = p.npmClient.AddProxyHost(npm.ProxyHost{
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
	case "stop", "kill":
		log.Printf("Deleting entry from Pi-Hole for container '%v'", name)
		err := p.piholeClient.DeleteDNSHostEntry(url, ip)
		if err != nil {
			log.Printf("ERROR failed to delete entry from Pi-Hole: %v", err)
		}

		log.Printf("Deleting entry from Nginx Proxy Manager for container '%v'", name)
		err = p.npmClient.DeleteProxyHost(url)
		if err != nil {
			log.Printf("ERROR failed to delete entry from Nginx Proxy Manager: %v", err)
		}
	}
}
