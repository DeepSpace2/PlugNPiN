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
	"github.com/deepspace2/plugnpin/pkg/metrics"
)

var log = logging.GetLogger("processor")

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
	p.RunOnce(ctx)

	if interval == 0 {
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Info(fmt.Sprintf("Will run again in %v. Press Ctrl+C to exit.", interval))

	for {
		select {
		case <-ticker.C:
			p.RunOnce(ctx)
			log.Info(fmt.Sprintf("Will run again in %v. Press Ctrl+C to exit.", interval))
		case <-ctx.Done():
			return
		}
	}
}

func (p *Processor) ListenForEvents(ctx context.Context) {
	for _, client := range p.dockerClients {
		go func(client *docker.Client) {
			log.Info("Starting event listener", "host", client.DisplayHost)
			err := docker.Listen(ctx, client, func(event events.Message) {
				p.handleDockerEvent(ctx, event, client)
			})
			if err != nil && err != context.Canceled {
				log.Error("Docker event listener stopped", "host", client.DisplayHost, "error", err)
			}
		}(client)
	}
}

func (p *Processor) RunOnce(ctx context.Context) {
	for _, dockerClient := range p.dockerClients {
		scanStartTime := time.Now()
		containers, err := dockerClient.GetRelevantContainers()
		if err != nil {
			log.Error("Failed to get containers", "host", dockerClient.DisplayHost, "error", err)
			continue
		}

		log.Info(fmt.Sprintf("Found %v containers", len(containers)), "host", dockerClient.DisplayHost)
		metrics.SetDiscoveredContainers(dockerClient.DisplayHost, len(containers))

		for _, container := range containers {
			p.preprocessContainer(ctx, container, dockerClient)
		}

		scanDurationSeconds := time.Since(scanStartTime).Seconds()
		metrics.ObserveScanDuration(dockerClient.DisplayHost, scanDurationSeconds)
	}
	log.Info("Done")
}

func (p *Processor) preprocessContainer(ctx context.Context, container container.Summary, dockerClient *docker.Client) {
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
	p.processContainer(ctx, events.ActionStart, container.ID, dockerClient, parsedContainerName, ip, urls, port, opts)
}

func (p *Processor) handleDockerEvent(ctx context.Context, event events.Message, dockerClient *docker.Client) {
	containerName, ok := event.Actor.Attributes["name"]
	if !ok {
		log.Info(fmt.Sprintf("Skipping event for container with no name: %v", event.Actor.ID), "host", dockerClient.DisplayHost)
		return
	}

	ip, urls, port, opts, err := docker.GetValuesFromLabels(event.Actor.Attributes)
	if err != nil {
		switch err.(type) {
		case *errors.NonExistingLabelsError:
			// This is not an error, it just means the container is not relevant for us
			return
		case *errors.MalformedIPLabelError, *errors.InvalidSchemeError:
			log.Error("Failed to handle event for container", "host", dockerClient.DisplayHost, "container", containerName, "error", err)
		}
		return
	}
	p.processContainer(ctx, event.Action, event.Actor.ID, dockerClient, containerName, ip, urls, port, opts)
}

func (p *Processor) shouldSkip(generalOptions *docker.GeneralOptions, event events.Action) bool {
	return (event == events.ActionStart && generalOptions.CreateOnHealthy) || (event == events.ActionHealthStatusHealthy && !generalOptions.CreateOnHealthy)
}

func (p *Processor) handleAdguardHome(ctx context.Context, containerEvent events.Action, urls []string, ip string, adguardHomeOptions adguardhome.AdguardHomeOptions, generalOptions *docker.GeneralOptions) {
	log := logging.FromContext(ctx)

	if p.adguardHomeClient != nil {
		if adguardHomeOptions.TargetDomain != "" {
			// quick "workaround" for the fact that adguard unifies "local DNS records" and "CNAME records"
			ip = adguardHomeOptions.TargetDomain
		}

		switch containerEvent {
		case events.ActionStart, events.ActionHealthStatusHealthy:
			log.Info("Adding a DNS rewrite to AdGuard Home", "domains", urls, "answer", ip)
			numOfAddedRewrites, err := p.adguardHomeClient.AddDnsRewrites(urls, ip)
			if err != nil {
				log.Error("Failed to add a DNS rewrite to AdGuard Home", "domains", urls, "answer", ip, "error", err)
				metrics.IncrementAdguardHomeApiRequestErrors(metrics.ADD_DNS_REWRITE)
			} else {
				metrics.IncrementAdguardHomeEntriesCreated(numOfAddedRewrites)
			}
		case events.ActionDie:
			log.Info("Deleting DNS rewrite from AdGuard Home", "domains", urls)
			numOfDeletedRewrites, err := p.adguardHomeClient.DeleteDnsRewrites(urls, ip)
			if err != nil {
				log.Error("Failed to delete DNS rewrite from AdGuard Home", "domains", urls, "error", err)
				metrics.IncrementAdguardHomeApiRequestErrors(metrics.DELETE_DNS_REWRITE)
			} else {
				metrics.IncrementAdguardHomeEntriesDeleted(numOfDeletedRewrites)
			}
		}
	}
}

func (p *Processor) handlePiHole(ctx context.Context, containerEvent events.Action, urls []string, ip string, piholeOptions pihole.PiHoleOptions, generalOptions *docker.GeneralOptions) {
	log := logging.FromContext(ctx)

	if p.piholeClient != nil {
		switch containerEvent {
		case events.ActionStart, events.ActionHealthStatusHealthy:
			var numOfAddedEntries int
			var err error

			if piholeOptions.TargetDomain == "" {
				log.Info("Adding local DNS records to Pi-Hole", "urls", urls, "ip", ip)
				numOfAddedEntries, err = p.piholeClient.AddDnsRecords(urls, ip)
				if err != nil {
					log.Error("Failed to add local DNS records to Pi-Hole", "urls", urls, "ip", ip, "error", err)
					metrics.IncrementPiHoleApiRequestErrors(metrics.ADD_DNS_RECORD)
					return
				}
			} else {
				log.Info("Adding local CNAME records to Pi-Hole", "urls", urls, "targetDomain", piholeOptions.TargetDomain)
				numOfAddedEntries, err = p.piholeClient.AddCNameRecords(urls, piholeOptions.TargetDomain)
				if err != nil {
					log.Error("Failed to add local CNAME records to Pi-Hole", "urls", urls, "targetDomain", piholeOptions.TargetDomain, "error", err)
					metrics.IncrementPiHoleApiRequestErrors(metrics.ADD_CNAME_RECORD)
					return
				}
			}
			metrics.IncrementPiHoleEntriesCreated(numOfAddedEntries)
		case events.ActionDie:
			var numOfDeletedEntries int
			var err error

			if piholeOptions.TargetDomain == "" {
				log.Info("Deleting local DNS records from Pi-Hole", "urls", urls)
				numOfDeletedEntries, err = p.piholeClient.DeleteDnsRecords(urls)
				if err != nil {
					log.Error("Failed to delete local DNS records from Pi-Hole", "urls", urls, "error", err)
					metrics.IncrementPiHoleApiRequestErrors(metrics.DELETE_DNS_RECORD)
					return
				}
			} else {
				log.Info("Deleting local CNAME records from Pi-Hole", "urls", urls, "targetDomain", piholeOptions.TargetDomain)
				numOfDeletedEntries, err = p.piholeClient.DeleteCNameRecords(urls)
				if err != nil {
					log.Error("Failed to delete local CNAME records from Pi-Hole", "urls", urls, "targetDomain", piholeOptions.TargetDomain, "error", err)
					metrics.IncrementPiHoleApiRequestErrors(metrics.DELETE_CNAME_RECORD)
					return
				}
			}
			metrics.IncrementPiHoleEntriesDeleted(numOfDeletedEntries)
		}
	}
}

func (p *Processor) handleNpm(ctx context.Context, containerEvent events.Action, urls []string, ip string, port int, npmProxyHostOptions npm.NpmProxyHostOptions, generalOptions *docker.GeneralOptions) {
	log := logging.FromContext(ctx)

	switch containerEvent {
	case events.ActionStart, events.ActionHealthStatusHealthy:
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
				log.Error("Not creating Nginx Proxy Manager entry", "error", err)
				metrics.IncrementNpmApiRequestErrors(metrics.GET_ACCESS_LIST_ID)
				return
			}
			npmProxyHost.AccessListID = npmAccessListID
		}

		if npmProxyHostOptions.CertificateName != "" {
			npmCertificateID, err := p.npmClient.GetCertificateIDByName(npmProxyHostOptions.CertificateName)
			if err != nil {
				log.Error("Not creating Nginx Proxy Manager entry", "error", err)
				metrics.IncrementNpmApiRequestErrors(metrics.GET_CERTIFICATE_ID)
				return
			}
			npmProxyHost.CertificateID = npmCertificateID
		}

		log.Info("Adding entry to Nginx Proxy Manager")

		addedNpmEntry, err := p.npmClient.AddProxyHost(npmProxyHost)
		if err != nil {
			log.Error("Failed to add entry to Nginx Proxy Manager", "error", err)
			metrics.IncrementNpmApiRequestErrors(metrics.ADD_PROXY_HOST)
			return
		}
		if addedNpmEntry {
			metrics.IncrementNpmEntriesCreated()
		}
	case events.ActionDie:
		log.Info("Deleting entry from Nginx Proxy Manager")
		deletedNpmEntry, err := p.npmClient.DeleteProxyHosts(urls)
		if err != nil {
			log.Error("Failed to delete entry from Nginx Proxy Manager", "error", err)
			metrics.IncrementNpmApiRequestErrors(metrics.DELETE_PROXY_HOST)
			return
		}
		if deletedNpmEntry {
			metrics.IncrementNpmEntriesDeleted()
		}
	}
}

func (p *Processor) Shutdown() {
	if p.piholeClient != nil {
		if err := p.piholeClient.Logout(); err != nil {
			log.Warn("Failed to logout from Pi-Hole", "error", err)
		}
	}
}

func (p *Processor) processContainer(ctx context.Context, containerEvent events.Action, containerId string, dockerClient *docker.Client, containerName, ip string, urls []string, port int, opts *docker.ClientOptions) {
	log := log.With(
		"container", containerName,
		"containerId", dockerClient.GetShortContainerId(containerId),
		"event", containerEvent,
		"host", dockerClient.DisplayHost,
	)

	ctx = logging.WithLogger(ctx, log)

	if opts.GeneralOptions.CreateOnHealthy && (containerEvent == events.ActionStart || containerEvent == events.ActionHealthStatusHealthy) {
		containerInspectResponse, err := dockerClient.InspectContainer(ctx, containerId)
		if err != nil {
			log.Error("Failed to inspect container", "error", err)
			return
		}

		if !dockerClient.HasHealthcheck(containerInspectResponse) {
			log.Error("Container has 'createOnHealthy' enabled but NO healthcheck is defined. Entries will NOT be created.")
			return
		}

		// If we are in the initial sync (which uses synthetic Start events) but the
		// container is already healthy, we "upgrade" the event to Healthy.
		// This ensures shouldSkip() allows it to proceed immediately.
		if containerEvent == events.ActionStart && dockerClient.IsHealthy(containerInspectResponse) {
			containerEvent = events.ActionHealthStatusHealthy
		}
	}

	if p.shouldSkip(&opts.GeneralOptions, containerEvent) {
		if containerEvent == events.ActionStart {
			log.Info("Container is not healthy yet. Waiting for container to be healthy before creating entries.")
		}
		return
	}

	msg := "Handling container"

	if p.dryRun {
		msg += ". In dry run mode, not doing anything."
		log.Info(msg, "ip", ip, "port", port, "urls", urls)
		return
	}

	log.Info(msg, "ip", ip, "port", port, "urls", urls)

	metrics.IncrementHandledDockerEvents(dockerClient.DisplayHost, string(containerEvent))

	if p.npmClient != nil {
		npmHost := p.npmClient.GetIP()
		if opts.AdguardHome != nil {
			p.handleAdguardHome(ctx, containerEvent, urls, npmHost, *opts.AdguardHome, &opts.GeneralOptions)
		}
		if opts.Pihole != nil {
			p.handlePiHole(ctx, containerEvent, urls, npmHost, *opts.Pihole, &opts.GeneralOptions)
		}
		if opts.NPM != nil {
			p.handleNpm(ctx, containerEvent, urls, ip, port, *opts.NPM, &opts.GeneralOptions)
		}
	}
}
