package clients

import (
	"github.com/deepspace2/plugnpin/pkg/cli"
	"github.com/deepspace2/plugnpin/pkg/clients/adguardhome"
	"github.com/deepspace2/plugnpin/pkg/clients/docker"
	"github.com/deepspace2/plugnpin/pkg/clients/npm"
	"github.com/deepspace2/plugnpin/pkg/clients/pihole"
	"github.com/deepspace2/plugnpin/pkg/config"
	"github.com/deepspace2/plugnpin/pkg/logging"
)

var log = logging.GetLogger("clients")

func GetClients(cliFlags cli.Flags, config *config.Config) (map[string]*docker.Client, *adguardhome.Client, *pihole.Client, *npm.Client, error) {
	var adguardHomeClient *adguardhome.Client
	var piholeClient *pihole.Client
	var npmClient *npm.Client

	if !cliFlags.DryRun {
		if !config.PiholeDisabled {
			piholeClient = pihole.NewClient(config.PiholeHost, config.PiholePassword)
			err := piholeClient.Login()
			if err != nil {
				log.Error("Failed to login to Pi-Hole", "error", err)
				return nil, nil, nil, nil, err
			}
		}

		if !config.AdguardHomeDisabled {
			adguardHomeClient = adguardhome.NewClient(config.AdguardHomeHost, config.AdguardHomeUsername, config.AdguardHomePassword)
		}

		npmClient = npm.NewClient(config.NpmHost, config.NpmUsername, config.NpmPassword)
		err := npmClient.Login()
		if err != nil {
			log.Error("Failed to login to Nginx Proxy Manager", "error", err)
			return nil, nil, nil, nil, err
		}
	}

	dockerClients := make(map[string]*docker.Client)
	if len(config.DockerHosts) == 0 {
		config.DockerHosts = append(config.DockerHosts, config.DockerHost)
	}
	for _, host := range config.DockerHosts {
		dockerClient, err := docker.NewClient(host)
		if err != nil {
			log.Error("Failed to create docker client", "host", host, "error", err)
			continue
		}
		dockerClients[dockerClient.Host] = dockerClient
	}

	return dockerClients, adguardHomeClient, piholeClient, npmClient, nil
}
