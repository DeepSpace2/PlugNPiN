package docker

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/deepspace2/plugnpin/pkg/errors"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	dockerSdk "github.com/docker/go-sdk/client"
)

const (
	ipLabel  = "plugNPiN.ip"
	urlLabel = "plugNPiN.url"
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

func GetValuesFromLabels(labels map[string]string) (ip, url string, port int, err error) {
	ip, ok := labels[ipLabel]
	if !ok {
		return "", "", 0, &errors.NonExistingLabelsError{Msg: fmt.Sprintf("missing %s label", ipLabel)}
	}
	url, ok = labels[urlLabel]
	if !ok {
		return "", "", 0, &errors.NonExistingLabelsError{Msg: fmt.Sprintf("missing %s label", urlLabel)}
	}

	splitIPAndPort := strings.Split(ip, ":")
	if len(splitIPAndPort) == 1 {
		return "", "", 0, &errors.MalformedIPLabelError{Msg: fmt.Sprintf("missing ':' in value of '%v' label", ipLabel)}
	}
	ip = splitIPAndPort[0]
	port, err = strconv.Atoi(splitIPAndPort[1])
	if err != nil {
		return "", "", 0, &errors.MalformedIPLabelError{
			Msg: fmt.Sprintf("value after ':' in value of '%v' label must be an integer, got '%v'", ipLabel, splitIPAndPort[1]),
		}
	}

	return ip, url, port, nil
}
