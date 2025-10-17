package docker

import (
	"context"
	"fmt"
	"log"
	"slices"
	"strconv"
	"strings"

	"github.com/deepspace2/plugnpin/pkg/clients/npm"
	"github.com/deepspace2/plugnpin/pkg/clients/pihole"
	"github.com/deepspace2/plugnpin/pkg/errors"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	dockerSdk "github.com/docker/go-sdk/client"
)

const (
	ipLabel  = "plugNPiN.ip"
	urlLabel = "plugNPiN.url"

	npmOptionsBlockExploitsLabel     = "plugNPiN.npmOptions.blockExploits"
	npmOptionsCachingEnabledLabel    = "plugNPiN.npmOptions.cachingEnabled"
	npmOptionsCertificateNameLabel   = "plugNPiN.npmOptions.certificateName"
	npmOptionsHTTP2SupportLabel      = "plugNPiN.npmOptions.http2Support"
	npmOptionsHstsEnabledLabel       = "plugNPiN.npmOptions.hstsEnabled"
	npmOptionsHstsSubdomainsLabel    = "plugNPiN.npmOptions.hstsSubdomains"
	npmOptionsSchemeLabel            = "plugNPiN.npmOptions.scheme"
	npmOptionsSslForcedLabel         = "plugNPiN.npmOptions.forceSsl"
	npmOptionsWebsocketsSupportLabel = "plugNPiN.npmOptions.websocketsSupport"
	piholeOptionsTargetDomainLabel   = "plugNPiN.piholeOptions.targetDomain"
)

var labels []string = []string{ipLabel, urlLabel}

type Client struct {
	*dockerSdk.Client
}

func NewClient() (*Client, error) {
	client, err := dockerSdk.New(context.Background())
	return &Client{client}, err
}

func (d *Client) GetRelevantContainers() ([]container.Summary, error) {
	f := filters.NewArgs()
	for _, label := range labels {
		f.Add("label", label)
	}

	log.Printf("Getting containers with labels: %v", strings.Join(labels, ", "))

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

func GetValuesFromLabels(labels map[string]string) (ip, url string, port int, npmProxyHostOptions *npm.NpmProxyHostOptions, piholeOptions *pihole.PiHoleOptions, err error) {
	ip, ok := labels[ipLabel]
	if !ok {
		return "", "", 0, nil, nil, &errors.NonExistingLabelsError{Msg: fmt.Sprintf("missing %s label", ipLabel)}
	}
	url, ok = labels[urlLabel]
	if !ok {
		return "", "", 0, nil, nil, &errors.NonExistingLabelsError{Msg: fmt.Sprintf("missing %s label", urlLabel)}
	}

	splitIPAndPort := strings.Split(ip, ":")
	if len(splitIPAndPort) == 1 {
		return "", "", 0, nil, nil, &errors.MalformedIPLabelError{Msg: fmt.Sprintf("missing ':' in value of '%v' label", ipLabel)}
	}
	ip = splitIPAndPort[0]
	port, err = strconv.Atoi(splitIPAndPort[1])
	if err != nil {
		return "", "", 0, nil, nil, &errors.MalformedIPLabelError{
			Msg: fmt.Sprintf("value after ':' in value of '%v' label must be an integer, got '%v'", ipLabel, splitIPAndPort[1]),
		}
	}

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
		return "", "", 0, nil, nil, &errors.InvalidSchemeError{
			Msg: fmt.Sprintf("value of '%v' label must be one of 'http', 'https', got '%v'", npmOptionsSchemeLabel, npmOptionsScheme),
		}
	}

	npmOptionsCertificateName := labels[npmOptionsCertificateNameLabel]
	npmOptionsHTTP2Support, _ := strconv.ParseBool(labels[npmOptionsHTTP2SupportLabel])
	npmOptionsHstsEnabled, _ := strconv.ParseBool(labels[npmOptionsHstsEnabledLabel])
	npmOptionsHstsSubdomains, _ := strconv.ParseBool(labels[npmOptionsHstsSubdomainsLabel])
	npmOptionsSslForced, _ := strconv.ParseBool(labels[npmOptionsSslForcedLabel])

	npmProxyHostOptions = &npm.NpmProxyHostOptions{
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

	piholeOptions = &pihole.PiHoleOptions{
		TargetDomain: piholeOptionsTargetDomain,
	}

	return ip, url, port, npmProxyHostOptions, piholeOptions, nil
}
