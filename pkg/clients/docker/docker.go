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
	ip, ok := getLabel(labels, IpLabel)
	if !ok {
		return "", nil, 0, nil, &errors.NonExistingLabelsError{Msg: fmt.Sprintf("missing %s label", IpLabel)}
	}
	urlsString, ok := getLabel(labels, UrlLabel)
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

	generalOptionsCreateOnHealthyValue, _ := getLabel(labels, GeneralOptionsCreateOnHealthyLabel)
	generalOptionsCreateOnHealthy, _ := strconv.ParseBool(generalOptionsCreateOnHealthyValue)
	opts.GeneralOptions = GeneralOptions{CreateOnHealthy: generalOptionsCreateOnHealthy}

	npmOptionsBlockExploitsLabelValue, exists := getLabel(labels, npmOptionsBlockExploitsLabel)
	if !exists {
		npmOptionsBlockExploitsLabelValue = "true"
	}

	npmOptionsBlockExploits, _ := strconv.ParseBool(npmOptionsBlockExploitsLabelValue)
	npmOptionsWebsocketsSupportValue, _ := getLabel(labels, npmOptionsWebsocketsSupportLabel)
	npmOptionsWebsocketsSupport, _ := strconv.ParseBool(npmOptionsWebsocketsSupportValue)
	npmOptionsCachingEnabledValue, _ := getLabel(labels, npmOptionsCachingEnabledLabel)
	npmOptionsCachingEnabled, _ := strconv.ParseBool(npmOptionsCachingEnabledValue)

	npmOptionsScheme, exists := getLabel(labels, npmOptionsSchemeLabel)
	if !exists {
		npmOptionsScheme = "http"
	}
	npmOptionsScheme = strings.ToLower(npmOptionsScheme)
	if !slices.Contains([]string{"http", "https"}, npmOptionsScheme) {
		return "", nil, 0, nil, &errors.InvalidSchemeError{
			Msg: fmt.Sprintf("value of '%v' label must be one of 'http', 'https', got '%v'", npmOptionsSchemeLabel, npmOptionsScheme),
		}
	}

	npmOptionsAdvancedConfig, _ := getLabel(labels, npmOptionsAdvancedConfigLabel)
	npmOptionsCertificateName, _ := getLabel(labels, npmOptionsCertificateNameLabel)
	npmOptionsAccessListName, _ := getLabel(labels, npmOptionsAccessListNameLabel)
	npmOptionsHTTP2SupportValue, _ := getLabel(labels, npmOptionsHTTP2SupportLabel)
	npmOptionsHTTP2Support, _ := strconv.ParseBool(npmOptionsHTTP2SupportValue)
	npmOptionsHstsEnabledValue, _ := getLabel(labels, npmOptionsHstsEnabledLabel)
	npmOptionsHstsEnabled, _ := strconv.ParseBool(npmOptionsHstsEnabledValue)
	npmOptionsHstsSubdomainsValue, _ := getLabel(labels, npmOptionsHstsSubdomainsLabel)
	npmOptionsHstsSubdomains, _ := strconv.ParseBool(npmOptionsHstsSubdomainsValue)
	npmOptionsSslForcedValue, _ := getLabel(labels, npmOptionsSslForcedLabel)
	npmOptionsSslForced, _ := strconv.ParseBool(npmOptionsSslForcedValue)

	opts.NPM = &npm.NpmProxyHostOptions{
		AccessListName:        npmOptionsAccessListName,
		AdvancedConfig:        npmOptionsAdvancedConfig,
		AllowWebsocketUpgrade: npmOptionsWebsocketsSupport,
		BlockExploits:         npmOptionsBlockExploits,
		CachingEnabled:        npmOptionsCachingEnabled,
		CertificateName:       npmOptionsCertificateName,
		ForwardScheme:         npmOptionsScheme,
		HTTP2Support:          npmOptionsHTTP2Support,
		HstsEnabled:           npmOptionsHstsEnabled,
		HstsSubdomains:        npmOptionsHstsSubdomains,
		SslForced:             npmOptionsSslForced,
	}

	piholeOptionsTargetDomain, _ := getLabel(labels, piholeOptionsTargetDomainLabel)

	opts.Pihole = &pihole.PiHoleOptions{
		TargetDomain: piholeOptionsTargetDomain,
	}

	adguardHomeOptionsTargetDomain, _ := getLabel(labels, adguardHomeOptionsTargetDomainLabel)

	opts.AdguardHome = &adguardhome.AdguardHomeOptions{
		TargetDomain: adguardHomeOptionsTargetDomain,
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
