package docker

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

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
	AdguardHome *adguardhome.AdguardHomeOptions
	Pihole      *pihole.PiHoleOptions
	NPM         *npm.NpmProxyHostOptions
}

var log = logging.GetLogger()

const (
	IpLabel  = "plugNPiN.ip"
	UrlLabel = "plugNPiN.url"

	adguardHomeOptionsTargetDomainLabel = "plugNPiN.adguardHomeOptions.targetDomain"
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

func GetValuesFromLabels(labels map[string]string) (ip, url string, port int, opts *ClientOptions, err error) {
	ip, ok := labels[IpLabel]
	if !ok {
		return "", "", 0, nil, &errors.NonExistingLabelsError{Msg: fmt.Sprintf("missing %s label", IpLabel)}
	}
	url, ok = labels[UrlLabel]
	if !ok {
		return "", "", 0, nil, &errors.NonExistingLabelsError{Msg: fmt.Sprintf("missing %s label", UrlLabel)}
	}

	splitIPAndPort := strings.Split(ip, ":")
	if len(splitIPAndPort) == 1 {
		return "", "", 0, nil, &errors.MalformedIPLabelError{Msg: fmt.Sprintf("missing ':' in value of '%v' label", IpLabel)}
	}
	ip = splitIPAndPort[0]
	port, err = strconv.Atoi(splitIPAndPort[1])
	if err != nil {
		return "", "", 0, nil, &errors.MalformedIPLabelError{
			Msg: fmt.Sprintf("value after ':' in value of '%v' label must be an integer, got '%v'", IpLabel, splitIPAndPort[1]),
		}
	}

	opts = &ClientOptions{}

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
		return "", "", 0, nil, &errors.InvalidSchemeError{
			Msg: fmt.Sprintf("value of '%v' label must be one of 'http', 'https', got '%v'", npmOptionsSchemeLabel, npmOptionsScheme),
		}
	}

	npmOptionsAdvancedConfig := labels[npmOptionsAdvancedConfigLabel]
	npmOptionsCertificateName := labels[npmOptionsCertificateNameLabel]
	npmOptionsHTTP2Support, _ := strconv.ParseBool(labels[npmOptionsHTTP2SupportLabel])
	npmOptionsHstsEnabled, _ := strconv.ParseBool(labels[npmOptionsHstsEnabledLabel])
	npmOptionsHstsSubdomains, _ := strconv.ParseBool(labels[npmOptionsHstsSubdomainsLabel])
	npmOptionsSslForced, _ := strconv.ParseBool(labels[npmOptionsSslForcedLabel])

	opts.NPM = &npm.NpmProxyHostOptions{
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

	return ip, url, port, opts, nil
}
