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

	ip, url, port, npmProxyHostOptions, piholeOptions, err := docker.GetValuesFromLabels(container.Labels)
	if err != nil {
		switch err.(type) {
		case *errors.NonExistingLabelsError:
			log.Printf("Skipping container '%v': %v", parsedContainerName, err)
		case *errors.MalformedIPLabelError, *errors.InvalidSchemeError:
			log.Printf("ERROR handling container '%v': %v", parsedContainerName, err)
		}
		return
	}
	p.processContainer(docker.ContainerEvent.Start, parsedContainerName, ip, url, port, *piholeOptions, *npmProxyHostOptions)
}

func (p *Processor) handleDockerEvent(event events.Message) {
	containerName, ok := event.Actor.Attributes["name"]
	if !ok {
		log.Printf("Skipping event for container with no name: %v", event.Actor.ID)
		return
	}

	ip, url, port, npmProxyHostOptions, piholeOptions, err := docker.GetValuesFromLabels(event.Actor.Attributes)
	if err != nil {
		switch err.(type) {
		case *errors.NonExistingLabelsError:
			// This is not an error, it just means the container is not relevant for us
			return
		case *errors.MalformedIPLabelError, *errors.InvalidSchemeError:
			log.Printf("ERROR handling event for container '%v': %v", containerName, err)
		}
		return
	}
	containerEvent, _ := docker.ContainerEvent.ParseString(string(event.Action))
	p.processContainer(containerEvent, containerName, ip, url, port, *piholeOptions, *npmProxyHostOptions)
}

func (p *Processor) handlePiHole(containerEvent docker.EventType, containerName, url, ip string, piholeOptions pihole.PiHoleOptions) {
	if p.piholeClient != nil {
		switch containerEvent {
		case docker.ContainerEvent.Start:
			if piholeOptions.TargetDomain == "" {
				log.Printf("Adding a local DNS record to Pi-Hole for container '%v'", containerName)
				err := p.piholeClient.AddDnsRecord(url, p.npmClient.GetIP())
				if err != nil {
					log.Printf("ERROR failed to add a local DNS record to Pi-Hole: %v", err)
				}
			} else {
				log.Printf("Adding a local CNAME record to Pi-Hole for container '%v'", containerName)
				err := p.piholeClient.AddCNameRecord(url, piholeOptions.TargetDomain)
				if err != nil {
					log.Printf("ERROR failed to add a local CNAME record to Pi-Hole: %v", err)
				}
			}
		case docker.ContainerEvent.Stop, docker.ContainerEvent.Kill:
			if piholeOptions.TargetDomain == "" {
				log.Printf("Deleting local DNS record from Pi-Hole for container '%v'", containerName)
				err := p.piholeClient.DeleteDnsRecord(url, ip)
				if err != nil {
					log.Printf("ERROR failed to delete local DNS record from Pi-Hole: %v", err)
				}
			} else {
				log.Printf("Deleting local CNAME record from Pi-Hole for container '%v'", containerName)
				err := p.piholeClient.DeleteCNameRecord(url, piholeOptions.TargetDomain)
				if err != nil {
					log.Printf("ERROR failed to delete local CNAME record from Pi-Hole: %v", err)
				}
			}
		}
	}
}

func (p *Processor) handleNpm(containerEvent docker.EventType, containerName, url, ip string, port int, npmProxyHostOptions npm.NpmProxyHostOptions) {
	switch containerEvent {
	case docker.ContainerEvent.Start:
		npmProxyHost := npm.ProxyHost{
			AllowWebsocketUpgrade: npmProxyHostOptions.AllowWebsocketUpgrade,
			BlockExploits:         npmProxyHostOptions.BlockExploits,
			CachingEnabled:        npmProxyHostOptions.CachingEnabled,
			ForwardScheme:         npmProxyHostOptions.ForwardScheme,
			HTTP2Support:          npmProxyHostOptions.HTTP2Support,
			HstsEnabled:           npmProxyHostOptions.HstsEnabled,
			HstsSubdomains:        npmProxyHostOptions.HstsSubdomains,
			SslForced:             npmProxyHostOptions.SslForced,

			DomainNames: []string{url},
			ForwardHost: ip,
			ForwardPort: port,
			Locations:   []npm.Location{},
			Meta:        npm.Meta{},
		}

		if npmProxyHostOptions.CertificateName != "" {
			npmCertificateID := p.npmClient.GetCertificateIDByName(npmProxyHostOptions.CertificateName)
			if npmCertificateID != nil {
				npmProxyHost.CertificateID = *npmCertificateID
			}
		}

		log.Printf("Adding entry to Nginx Proxy Manager for container '%v'", containerName)

		err := p.npmClient.AddProxyHost(npmProxyHost)
		if err != nil {
			log.Printf("ERROR failed to add entry to Nginx Proxy Manager: %v", err)
		}
	case docker.ContainerEvent.Stop, docker.ContainerEvent.Kill:
		log.Printf("Deleting entry from Nginx Proxy Manager for container '%v'", containerName)
		err := p.npmClient.DeleteProxyHost(url)
		if err != nil {
			log.Printf("ERROR failed to delete entry from Nginx Proxy Manager: %v", err)
		}
	}
}

func (p *Processor) processContainer(containerEvent docker.EventType, containerName, ip, url string, port int, piholeOptions pihole.PiHoleOptions, npmProxyHostOptions npm.NpmProxyHostOptions) {
	msg := fmt.Sprintf("Handling container '%v': ip=%v, port=%v, host=%v", containerName, ip, port, url)

	if p.dryRun {
		msg += ". In dry run mode, not doing anything."
		log.Println(msg)
		return
	}

	log.Println(msg)

	p.handlePiHole(containerEvent, containerName, url, ip, piholeOptions)
	p.handleNpm(containerEvent, containerName, url, ip, port, npmProxyHostOptions)
}
