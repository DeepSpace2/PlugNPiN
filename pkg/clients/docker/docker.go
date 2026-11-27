package docker

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	dockerSdk "github.com/docker/go-sdk/client"

	"github.com/deepspace2/plugnpin/pkg/clients/adguardhome"
	"github.com/deepspace2/plugnpin/pkg/clients/npm"
	"github.com/deepspace2/plugnpin/pkg/clients/pihole"
	"github.com/deepspace2/plugnpin/pkg/errors"
	"github.com/deepspace2/plugnpin/pkg/logging"
)

type ClientOptions struct {
	AdguardHome    *adguardhome.AdguardHomeOptions
	GeneralOptions GeneralOptions
	NPM            *npm.NpmProxyHostOptions
	Pihole         *pihole.PiHoleOptions
}

type GeneralOptions struct {
	CreateOnHealthy bool
}

var log = logging.GetLogger("docker")

func NewClient(host string) (*Client, error) {
	client, err := dockerSdk.New(context.Background(), dockerSdk.WithDockerHost(host))
	var displayHost string
	if host == "" {
		displayHost = "local"
	} else {
		displayHost = host
	}
	return &Client{Client: client, Host: host, DisplayHost: displayHost}, err
}

// GetRelevantContainers returns the running containers that carry both required
// labels. The filtering happens here instead of in the Docker API because the
// API's label filter is case-sensitive, while PlugNPiN matches label keys
// case-insensitively.
func (d *Client) GetRelevantContainers() ([]container.Summary, error) {
	log.Info(fmt.Sprintf("Scanning running containers for the %v and %v labels (case-insensitive)", IpLabel, UrlLabel), "host", d.DisplayHost)

	containers, err := d.ContainerList(context.Background(), container.ListOptions{})
	if err != nil {
		return nil, err
	}

	relevantContainers := make([]container.Summary, 0, len(containers))
	for _, c := range containers {
		if hasRequiredLabels(c.Labels) {
			relevantContainers = append(relevantContainers, c)
		}
	}

	return relevantContainers, nil
}

func GetParsedContainerName(container container.Summary) string {
	return strings.Trim(container.Names[0], "/")
}

func GetValuesFromLabels(labels map[string]string) (ip string, urls []string, port int, opts *ClientOptions, err error) {
	values := labelValues(labels)

	ip, ok := values.value(IpLabel)
	if !ok {
		return "", nil, 0, nil, &errors.NonExistingLabelsError{Msg: fmt.Sprintf("missing %s label", IpLabel)}
	}
	urlsString, ok := values.value(UrlLabel)
	if !ok {
		return "", nil, 0, nil, &errors.NonExistingLabelsError{Msg: fmt.Sprintf("missing %s label", UrlLabel)}
	}

	urls = strings.Split(urlsString, ",")

	splitIPAndPort := strings.Split(ip, ":")
	if len(splitIPAndPort) == 1 {
		return "", nil, 0, nil, &errors.MalformedIPLabelError{Msg: fmt.Sprintf("missing ':' in value of '%v' label", IpLabel)}
	}
	ip = splitIPAndPort[0]
	port, err = strconv.Atoi(splitIPAndPort[1])
	if err != nil {
		return "", nil, 0, nil, &errors.MalformedIPLabelError{
			Msg: fmt.Sprintf("value after ':' in value of '%v' label must be an integer, got '%v'", IpLabel, splitIPAndPort[1]),
		}
	}

	opts = &ClientOptions{}

	opts.GeneralOptions = GeneralOptions{
		CreateOnHealthy: values.boolOrDefault(GeneralOptionsCreateOnHealthyLabel, false),
	}

	npmOptionsScheme := values.valueOrDefault(npmOptionsSchemeLabel, "http")
	npmOptionsScheme = strings.ToLower(npmOptionsScheme)
	if !slices.Contains([]string{"http", "https"}, npmOptionsScheme) {
		return "", nil, 0, nil, &errors.InvalidSchemeError{
			Msg: fmt.Sprintf("value of '%v' label must be one of 'http', 'https', got '%v'", npmOptionsSchemeLabel, npmOptionsScheme),
		}
	}

	opts.NPM = &npm.NpmProxyHostOptions{
		AccessListName:        values.valueOrDefault(npmOptionsAccessListNameLabel, ""),
		AdvancedConfig:        values.valueOrDefault(npmOptionsAdvancedConfigLabel, ""),
		AllowWebsocketUpgrade: values.boolOrDefault(npmOptionsWebsocketsSupportLabel, false),
		BlockExploits:         values.boolOrDefault(npmOptionsBlockExploitsLabel, true),
		CachingEnabled:        values.boolOrDefault(npmOptionsCachingEnabledLabel, false),
		CertificateName:       values.valueOrDefault(npmOptionsCertificateNameLabel, ""),
		ForwardScheme:         npmOptionsScheme,
		HTTP2Support:          values.boolOrDefault(npmOptionsHTTP2SupportLabel, false),
		HstsEnabled:           values.boolOrDefault(npmOptionsHstsEnabledLabel, false),
		HstsSubdomains:        values.boolOrDefault(npmOptionsHstsSubdomainsLabel, false),
		SslForced:             values.boolOrDefault(npmOptionsSslForcedLabel, false),
	}

	opts.Pihole = &pihole.PiHoleOptions{
		TargetDomain: values.valueOrDefault(piholeOptionsTargetDomainLabel, ""),
	}

	opts.AdguardHome = &adguardhome.AdguardHomeOptions{
		TargetDomain: values.valueOrDefault(adguardHomeOptionsTargetDomainLabel, ""),
	}

	return ip, urls, port, opts, nil
}

func (d *Client) InspectContainer(ctx context.Context, containerId string) (container.InspectResponse, error) {
	// If the incoming context doesn't already have a deadline,
	// enforce a 5-second safety bound for this specific Docker call.
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
	}
	return d.ContainerInspect(ctx, containerId)
}

func (d *Client) HasHealthcheck(containerInspectResponse container.InspectResponse) bool {
	return containerInspectResponse.Config != nil &&
		containerInspectResponse.Config.Healthcheck != nil &&
		len(containerInspectResponse.Config.Healthcheck.Test) > 0
}

func (d *Client) IsHealthy(containerInspectResponse container.InspectResponse) bool {
	return containerInspectResponse.State != nil &&
		containerInspectResponse.State.Health != nil &&
		containerInspectResponse.State.Health.Status == CONTAINER_HEALTHY_STATUS
}

func (d *Client) GetShortContainerId(containerId string) string {
	if len(containerId) < 12 {
		return containerId
	}
	return containerId[:12]
}
