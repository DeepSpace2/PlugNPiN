package docker

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
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

const (
	GeneralOptionsCreateOnHealthyLabel = "plugNPiN.options.createOnHealthy"
	IpLabel                            = "plugNPiN.ip"
	UrlLabel                           = "plugNPiN.url"

	adguardHomeOptionsTargetDomainLabel = "plugNPiN.adguardHomeOptions.targetDomain"
	npmOptionsAccessListNameLabel       = "plugNPiN.npmOptions.accessListName"
	npmOptionsAdvancedConfigLabel       = "plugNPiN.npmOptions.advancedConfig"
	npmOptionsBlockExploitsLabel        = "plugNPiN.npmOptions.blockExploits"
	npmOptionsCachingEnabledLabel       = "plugNPiN.npmOptions.cachingEnabled"
	npmOptionsCertificateNameLabel      = "plugNPiN.npmOptions.certificateName"
	npmOptionsHTTP2SupportLabel         = "plugNPiN.npmOptions.http2Support"
	npmOptionsHstsEnabledLabel          = "plugNPiN.npmOptions.hstsEnabled"
	npmOptionsHstsSubdomainsLabel       = "plugNPiN.npmOptions.hstsSubdomains"
	npmOptionsSchemeLabel               = "plugNPiN.npmOptions.scheme"
	npmOptionsSslForcedLabel            = "plugNPiN.npmOptions.forceSsl"
	npmOptionsWebsocketsSupportLabel    = "plugNPiN.npmOptions.websocketsSupport"
	piholeOptionsTargetDomainLabel      = "plugNPiN.piholeOptions.targetDomain"
)

var labels []string = []string{IpLabel, UrlLabel}

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

func (d *Client) GetRelevantContainers() ([]container.Summary, error) {
	f := filters.NewArgs()
	for _, label := range labels {
		f.Add("label", label)
	}

	log.Info(fmt.Sprintf("Getting containers with labels: %v", strings.Join(labels, ", ")), "host", d.DisplayHost)

	return d.ContainerList(
		context.Background(),
		container.ListOptions{
			Filters: f,
		},
	)
}

func GetParsedContainerName(container container.Summary) string {
	return strings.Trim(container.Names[0], "/")
}

func GetValuesFromLabels(labels map[string]string) (ip string, urls []string, port int, opts *ClientOptions, err error) {
	ip, ok := labels[IpLabel]
	if !ok {
		return "", nil, 0, nil, &errors.NonExistingLabelsError{Msg: fmt.Sprintf("missing %s label", IpLabel)}
	}
	urlsString, ok := labels[UrlLabel]
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

	generalOptionsCreateOnHealthy, _ := strconv.ParseBool(labels[GeneralOptionsCreateOnHealthyLabel])
	opts.GeneralOptions = GeneralOptions{CreateOnHealthy: generalOptionsCreateOnHealthy}

	npmOptionsBlockExploitsLabelValue, exists := labels[npmOptionsBlockExploitsLabel]
	if !exists {
		npmOptionsBlockExploitsLabelValue = "true"
	}

	npmOptionsBlockExploits, _ := strconv.ParseBool(npmOptionsBlockExploitsLabelValue)
	npmOptionsWebsocketsSupport, _ := strconv.ParseBool(labels[npmOptionsWebsocketsSupportLabel])
	npmOptionsCachingEnabled, _ := strconv.ParseBool(labels[npmOptionsCachingEnabledLabel])

	npmOptionsScheme, exists := labels[npmOptionsSchemeLabel]
	if !exists {
		npmOptionsScheme = "http"
	}
	npmOptionsScheme = strings.ToLower(npmOptionsScheme)
	if !slices.Contains([]string{"http", "https"}, npmOptionsScheme) {
		return "", nil, 0, nil, &errors.InvalidSchemeError{
			Msg: fmt.Sprintf("value of '%v' label must be one of 'http', 'https', got '%v'", npmOptionsSchemeLabel, npmOptionsScheme),
		}
	}

	npmOptionsAdvancedConfig := labels[npmOptionsAdvancedConfigLabel]
	npmOptionsCertificateName := labels[npmOptionsCertificateNameLabel]
	npmOptionsAccessListName := labels[npmOptionsAccessListNameLabel]
	npmOptionsHTTP2Support, _ := strconv.ParseBool(labels[npmOptionsHTTP2SupportLabel])
	npmOptionsHstsEnabled, _ := strconv.ParseBool(labels[npmOptionsHstsEnabledLabel])
	npmOptionsHstsSubdomains, _ := strconv.ParseBool(labels[npmOptionsHstsSubdomainsLabel])
	npmOptionsSslForced, _ := strconv.ParseBool(labels[npmOptionsSslForcedLabel])

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

	piholeOptionsTargetDomain := labels[piholeOptionsTargetDomainLabel]

	opts.Pihole = &pihole.PiHoleOptions{
		TargetDomain: piholeOptionsTargetDomain,
	}

	adguardHomeOptionsTargetDomain := labels[adguardHomeOptionsTargetDomainLabel]

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
