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

var log = logging.GetLogger()

func GetClients(cliFlags cli.Flags, conf *config.Config) (map[string]*docker.Client, *adguardhome.Client, *pihole.Client, *npm.Client, error) {
	var adguardHomeClient *adguardhome.Client
	var piholeClient *pihole.Client
	var npmClient *npm.Client

	if !cliFlags.DryRun {
		if !conf.PiholeDisabled {
			piholeClient = pihole.NewClient(conf.PiholeHost)
			err := piholeClient.Login(conf.PiholePassword)
			if err != nil {
				log.Error("Failed to login to Pi-Hole", "error", err)
				return nil, nil, nil, nil, err
			}
		}

		if !conf.AdguardHomeDisabled {
			adguardHomeClient = adguardhome.NewClient(conf.AdguardHomeHost, conf.AdguardHomeUsername, conf.AdguardHomePassword)
		}

		npmClient = npm.NewClient(conf.NpmHost, conf.NpmUsername, conf.NpmPassword)
		err := npmClient.Login()
		if err != nil {
			log.Error("Failed to login to Nginx Proxy Manager", "error", err)
			return nil, nil, nil, nil, err
		}
	}

	dockerClients := make(map[string]*docker.Client)
	if len(conf.DockerHosts) == 0 {
		conf.DockerHosts = append(conf.DockerHosts, conf.DockerHost)
	}
	for _, host := range conf.DockerHosts {
		dockerClient, err := docker.NewClient(host)
		if err != nil {
			log.Error("Failed to create docker client", "host", host, "error", err)
			continue
		}
		dockerClients[dockerClient.Host] = dockerClient
	}

	return dockerClients, adguardHomeClient, piholeClient, npmClient, nil
}
