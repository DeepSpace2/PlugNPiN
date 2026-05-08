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
	dockerClients     map[string]*docker.Client
	adguardHomeClient *adguardhome.Client
	piholeClient      *pihole.Client
	npmClient         *npm.Client
	dryRun            bool
}

func New(dockerClients map[string]*docker.Client, adguardHomeClient *adguardhome.Client, piholeClient *pihole.Client, npmClient *npm.Client, dryRun bool) *Processor {
	return &Processor{
		dockerClients:     dockerClients,
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
	for _, client := range p.dockerClients {
		go func(c *docker.Client) {
			log.Info("Starting event listener", "host", c.DisplayHost)
			err := docker.Listen(ctx, c, func(event events.Message) {
				p.handleDockerEvent(event, c.DisplayHost)
			})
			if err != nil && err != context.Canceled {
				log.Error("Docker event listener stopped", "host", c.DisplayHost, "error", err)
			}
		}(client)
	}
}

func (p *Processor) RunOnce() {
	for _, dockerClient := range p.dockerClients {
		containers, err := dockerClient.GetRelevantContainers()
		if err != nil {
			log.Error("Failed to get containers", "host", dockerClient.DisplayHost, "error", err)
			continue
		}

		log.Info(fmt.Sprintf("Found %v containers", len(containers)), "host", dockerClient.DisplayHost)

		for _, container := range containers {
			p.preprocessContainer(container, dockerClient.DisplayHost)
		}
	}
	log.Info("Done")
}

func (p *Processor) preprocessContainer(container container.Summary, host string) {
	parsedContainerName := docker.GetParsedContainerName(container)

	ip, urls, port, opts, err := docker.GetValuesFromLabels(container.Labels)
	if err != nil {
		switch err.(type) {
		case *errors.NonExistingLabelsError:
			log.Info(fmt.Sprintf("Skipping container '%v': %v", parsedContainerName, err))
		case *errors.MalformedIPLabelError, *errors.InvalidSchemeError:
			log.Error("Failed to handle container", "container", parsedContainerName, "error", err)
		}
		return
	}
	p.processContainer(docker.ContainerEvent.Start, host, parsedContainerName, ip, urls, port, opts)
}

func (p *Processor) handleDockerEvent(event events.Message, host string) {
	containerName, ok := event.Actor.Attributes["name"]
	if !ok {
		log.Info(fmt.Sprintf("Skipping event for container with no name: %v", event.Actor.ID), "host", host)
		return
	}

	ip, urls, port, opts, err := docker.GetValuesFromLabels(event.Actor.Attributes)
	if err != nil {
		switch err.(type) {
		case *errors.NonExistingLabelsError:
			// This is not an error, it just means the container is not relevant for us
			return
		case *errors.MalformedIPLabelError, *errors.InvalidSchemeError:
			log.Error("Failed to handle event for container", "host", host, "container", containerName, "error", err)
		}
		return
	}
	containerEvent, _ := docker.ContainerEvent.ParseString(string(event.Action))
	p.processContainer(containerEvent, host, containerName, ip, urls, port, opts)
}

func (p *Processor) handleAdguardHome(host string, containerEvent docker.EventType, containerName string, urls []string, ip string, adguardHomeOptions adguardhome.AdguardHomeOptions) {
	if p.adguardHomeClient != nil {
		if adguardHomeOptions.TargetDomain != "" {
			// quick "workaround" for the fact that adguard unifies "local DNS records" and "CNAME records"
			ip = adguardHomeOptions.TargetDomain
		}

		switch containerEvent {
		case docker.ContainerEvent.Start:
			log.Info("Adding a DNS rewrite to AdGuard Home", "host", host, "container", containerName, "domains", urls, "answer", ip)
			err := p.adguardHomeClient.AddDnsRewrites(urls, ip)
			if err != nil {
				log.Error("Failed to add a DNS rewrite to AdGuard Home", "host", host, "container", containerName, "domains", urls, "answer", ip, "error", err)
			}
		case docker.ContainerEvent.Die:
			log.Info("Deleting DNS rewrite from AdGuard Home", "host", host, "container", containerName, "domains", urls)
			err := p.adguardHomeClient.DeleteDnsRewrites(urls, ip)
			if err != nil {
				log.Error("Failed to delete DNS rewrite from AdGuard Home", "host", host, "container", containerName, "domains", urls, "error", err)
			}
		}
	}
}

func (p *Processor) handlePiHole(host string, containerEvent docker.EventType, containerName string, urls []string, ip string, piholeOptions pihole.PiHoleOptions) {
	if p.piholeClient != nil {
		switch containerEvent {
		case docker.ContainerEvent.Start:
			if piholeOptions.TargetDomain == "" {
				log.Info("Adding local DNS records to Pi-Hole", "host", host, "container", containerName, "urls", urls, "ip", ip)
				err := p.piholeClient.AddDnsRecords(urls, ip)
				if err != nil {
					log.Error("Failed to add local DNS records to Pi-Hole", "host", host, "container", containerName, "urls", urls, "ip", ip, "error", err)
				}
			} else {
				log.Info("Adding local CNAME records to Pi-Hole", "host", host, "container", containerName, "urls", urls, "targetDomain", piholeOptions.TargetDomain)
				err := p.piholeClient.AddCNameRecords(urls, piholeOptions.TargetDomain)
				if err != nil {
					log.Error("Failed to add local CNAME records to Pi-Hole", "host", host, "container", containerName, "urls", urls, "targetDomain", piholeOptions.TargetDomain, "error", err)
				}
			}
		case docker.ContainerEvent.Die:
			if piholeOptions.TargetDomain == "" {
				log.Info("Deleting local DNS records from Pi-Hole", "host", host, "container", containerName, "urls", urls)
				err := p.piholeClient.DeleteDnsRecords(urls)
				if err != nil {
					log.Error("Failed to delete local DNS records from Pi-Hole", "host", host, "container", containerName, "urls", urls, "error", err)
				}
			} else {
				log.Info("Deleting local CNAME records from Pi-Hole", "host", host, "container", containerName, "urls", urls, "targetDomain", piholeOptions.TargetDomain)
				err := p.piholeClient.DeleteCNameRecords(urls)
				if err != nil {
					log.Error("Failed to delete local CNAME records from Pi-Hole", "host", host, "container", containerName, "urls", urls, "targetDomain", piholeOptions.TargetDomain, "error", err)
				}
			}
		}
	}
}

func (p *Processor) handleNpm(host string, containerEvent docker.EventType, containerName string, urls []string, ip string, port int, npmProxyHostOptions npm.NpmProxyHostOptions) {
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

			DomainNames: urls,
			ForwardHost: ip,
			ForwardPort: port,
			Locations:   []npm.Location{},
			Meta:        npm.Meta{},
		}

		if npmProxyHostOptions.AccessListName != "" {
			npmAccessListID, err := p.npmClient.GetAccessListIDByName(npmProxyHostOptions.AccessListName)
			if err != nil {
				log.Error("Not creating Nginx Proxy Manager entry", "host", host, "container", containerName, "error", err)
				return
			}
			npmProxyHost.AccessListID = npmAccessListID
		}

		if npmProxyHostOptions.CertificateName != "" {
			npmCertificateID, err := p.npmClient.GetCertificateIDByName(npmProxyHostOptions.CertificateName)
			if err != nil {
				log.Error("Not creating Nginx Proxy Manager entry", "host", host, "container", containerName, "error", err)
				return
			}
			npmProxyHost.CertificateID = npmCertificateID
		}

		log.Info("Adding entry to Nginx Proxy Manager", "host", host, "container", containerName)

		err := p.npmClient.AddProxyHost(npmProxyHost)
		if err != nil {
			log.Error("Failed to add entry to Nginx Proxy Manager", "host", host, "container", containerName, "error", err)
		}
	case docker.ContainerEvent.Die:
		log.Info("Deleting entry from Nginx Proxy Manager", "host", host, "container", containerName)
		err := p.npmClient.DeleteProxyHosts(urls)
		if err != nil {
			log.Error("Failed to delete entry from Nginx Proxy Manager", "host", host, "container", containerName, "error", err)
		}
	}
}

func (p *Processor) processContainer(containerEvent docker.EventType, host, containerName, ip string, urls []string, port int, opts *docker.ClientOptions) {
	msg := "Handling container"

	if p.dryRun {
		msg += ". In dry run mode, not doing anything."
		log.Info(msg, "host", host, "container", containerName, "ip", ip, "port", port, "urls", urls)
		return
	}

	log.Info(msg, "host", host, "container", containerName, "ip", ip, "port", port, "urls", urls)

	if p.npmClient != nil {
		npmHost := p.npmClient.GetIP()
		if opts.AdguardHome != nil {
			p.handleAdguardHome(host, containerEvent, containerName, urls, npmHost, *opts.AdguardHome)
		}
		if opts.Pihole != nil {
			p.handlePiHole(host, containerEvent, containerName, urls, npmHost, *opts.Pihole)
		}
		if opts.NPM != nil {
			p.handleNpm(host, containerEvent, containerName, urls, ip, port, *opts.NPM)
		}
	}
}
