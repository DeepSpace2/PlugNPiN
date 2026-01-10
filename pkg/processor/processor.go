package processor

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"

	"github.com/deepspace2/plugnpin/pkg/clients/adguardhome"
	"github.com/deepspace2/plugnpin/pkg/clients/docker"
	"github.com/deepspace2/plugnpin/pkg/clients/npm"
	"github.com/deepspace2/plugnpin/pkg/clients/pihole"
	"github.com/deepspace2/plugnpin/pkg/errors"
	"github.com/deepspace2/plugnpin/pkg/logging"
)

var log = logging.GetLogger()

type Processor struct {
	dockerClient      *docker.Client
	adguardHomeClient *adguardhome.Client
	piholeClient      *pihole.Client
	npmClient         *npm.Client
	dryRun            bool
}

func New(dockerClient *docker.Client, adguardHomeClient *adguardhome.Client, piholeClient *pihole.Client, npmClient *npm.Client, dryRun bool) *Processor {
	return &Processor{
		dockerClient:      dockerClient,
		adguardHomeClient: adguardHomeClient,
		piholeClient:      piholeClient,
		npmClient:         npmClient,
		dryRun:            dryRun,
	}
}

func (p *Processor) RunScheduled(ctx context.Context, interval time.Duration) {
	p.RunOnce()

	if interval == 0 {
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Info(fmt.Sprintf("Will run again in %v. Press Ctrl+C to exit.", interval))

	for {
		select {
		case <-ticker.C:
			p.RunOnce()
			log.Info(fmt.Sprintf("Will run again in %v. Press Ctrl+C to exit.", interval))
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
		log.Error("Docker event listener stopped", "error", err)
	}
}

func (p *Processor) RunOnce() {
	containers, err := p.dockerClient.GetRelevantContainers()
	if err != nil {
		log.Error("Failed to get containers", "error", err)
		return
	}

	log.Info(fmt.Sprintf("Found %v containers", len(containers)))

	for _, container := range containers {
		p.preprocessContainer(container)
	}
	log.Info("Done")
}

func (p *Processor) preprocessContainer(container container.Summary) {
	parsedContainerName := docker.GetParsedContainerName(container)

	ip, url, port, opts, err := docker.GetValuesFromLabels(container.Labels)
	if err != nil {
		switch err.(type) {
		case *errors.NonExistingLabelsError:
			log.Info(fmt.Sprintf("Skipping container '%v': %v", parsedContainerName, err))
		case *errors.MalformedIPLabelError, *errors.InvalidSchemeError:
			log.Error("Failed to handle container", "container", parsedContainerName, "error", err)
		}
		return
	}
	p.processContainer(docker.ContainerEvent.Start, parsedContainerName, ip, url, port, opts)
}

func (p *Processor) handleDockerEvent(event events.Message) {
	containerName, ok := event.Actor.Attributes["name"]
	if !ok {
		log.Info(fmt.Sprintf("Skipping event for container with no name: %v", event.Actor.ID))
		return
	}

	ip, url, port, opts, err := docker.GetValuesFromLabels(event.Actor.Attributes)
	if err != nil {
		switch err.(type) {
		case *errors.NonExistingLabelsError:
			// This is not an error, it just means the container is not relevant for us
			return
		case *errors.MalformedIPLabelError, *errors.InvalidSchemeError:
			log.Error("Failed to handle event for container", "container", containerName, "error", err)
		}
		return
	}
	containerEvent, _ := docker.ContainerEvent.ParseString(string(event.Action))
	p.processContainer(containerEvent, containerName, ip, url, port, opts)
}

func (p *Processor) handleAdguardHome(containerEvent docker.EventType, containerName, url, ip string, adguardHomeOptions adguardhome.AdguardHomeOptions) {
	if p.adguardHomeClient != nil {
		if adguardHomeOptions.TargetDomain != "" {
			// quick "workaround" for the fact that adguard unifies "local DNS records" and "CNAME records"
			ip = adguardHomeOptions.TargetDomain
		}

		switch containerEvent {
		case docker.ContainerEvent.Start:
			log.Info("Adding a DNS rewrite to AdGuard Home", "container", containerName, "domain", url, "answer", ip)
			err := p.adguardHomeClient.AddDnsRewrite(url, ip)
			if err != nil {
				log.Error("Failed to add a DNS rewrite to AdGuard Home", "container", containerName, "domain", url, "answer", ip, "error", err)
			}
		case docker.ContainerEvent.Die:
			log.Info("Deleting DNS rewrite from AdGuard Home", "container", containerName, "domain", url)
			err := p.adguardHomeClient.DeleteDnsRewrite(url, ip)
			if err != nil {
				log.Error("Failed to delete DNS rewrite from AdGuard Home", "container", containerName, "domain", url, "error", err)
			}
		}
	}
}

func (p *Processor) handlePiHole(containerEvent docker.EventType, containerName, url, ip string, piholeOptions pihole.PiHoleOptions) {
	if p.piholeClient != nil {
		switch containerEvent {
		case docker.ContainerEvent.Start:
			if piholeOptions.TargetDomain == "" {
				log.Info("Adding a local DNS record to Pi-Hole", "container", containerName, "url", url, "ip", ip)
				err := p.piholeClient.AddDnsRecord(url, ip)
				if err != nil {
					log.Error("Failed to add a local DNS record to Pi-Hole", "container", containerName, "url", url, "ip", ip, "error", err)
				}
			} else {
				log.Info("Adding a local CNAME record to Pi-Hole", "container", containerName, "url", url, "targetDomain", piholeOptions.TargetDomain)
				err := p.piholeClient.AddCNameRecord(url, piholeOptions.TargetDomain)
				if err != nil {
					log.Error("Failed to add a local CNAME record to Pi-Hole", "container", containerName, "url", url, "targetDomain", piholeOptions.TargetDomain, "error", err)
				}
			}
		case docker.ContainerEvent.Die:
			if piholeOptions.TargetDomain == "" {
				log.Info("Deleting local DNS record from Pi-Hole", "container", containerName, "url", url)
				err := p.piholeClient.DeleteDnsRecord(url)
				if err != nil {
					log.Error("Failed to delete local DNS record from Pi-Hole", "container", containerName, "url", url, "error", err)
				}
			} else {
				log.Info("Deleting local CNAME record from Pi-Hole", "container", containerName, "url", url, "targetDomain", piholeOptions.TargetDomain)
				err := p.piholeClient.DeleteCNameRecord(url, piholeOptions.TargetDomain)
				if err != nil {
					log.Error("Failed to delete local CNAME record from Pi-Hole", "container", containerName, "url", url, "targetDomain", piholeOptions.TargetDomain, "error", err)
				}
			}
		}
	}
}

func (p *Processor) handleNpm(containerEvent docker.EventType, containerName, url, ip string, port int, npmProxyHostOptions npm.NpmProxyHostOptions) {
	switch containerEvent {
	case docker.ContainerEvent.Start:
		npmProxyHost := npm.ProxyHost{
			AdvancedConfig:        npmProxyHostOptions.AdvancedConfig,
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

		log.Info("Adding entry to Nginx Proxy Manager", "container", containerName)

		err := p.npmClient.AddProxyHost(npmProxyHost)
		if err != nil {
			log.Error("Failed to add entry to Nginx Proxy Manager", "container", containerName, "error", err)
		}
	case docker.ContainerEvent.Die:
		log.Info("Deleting entry from Nginx Proxy Manager", "container", containerName)
		err := p.npmClient.DeleteProxyHost(url)
		if err != nil {
			log.Error("Failed to delete entry from Nginx Proxy Manager", "container", containerName, "error", err)
		}
	}
}

func (p *Processor) processContainer(containerEvent docker.EventType, containerName, ip, url string, port int, opts *docker.ClientOptions) {
	msg := "Handling container"

	if p.dryRun {
		msg += ". In dry run mode, not doing anything."
		log.Info(msg, "container", containerName, "ip", ip, "port", port, "url", url)
		return
	}

	log.Info(msg, "container", containerName, "ip", ip, "port", port, "url", url)

	if p.npmClient != nil {
		npmHost := p.npmClient.GetIP()
		if opts.AdguardHome != nil {
			p.handleAdguardHome(containerEvent, containerName, url, npmHost, *opts.AdguardHome)
		}
		if opts.Pihole != nil {
			p.handlePiHole(containerEvent, containerName, url, npmHost, *opts.Pihole)
		}
		if opts.NPM != nil {
			p.handleNpm(containerEvent, containerName, url, ip, port, *opts.NPM)
		}
	}
}
