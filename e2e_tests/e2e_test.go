//go:build e2e

package e2e_tests

import (
	"context"
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/caarlos0/env/v11"
	containerApi "github.com/docker/docker/api/types/container"
	imageApi "github.com/docker/docker/api/types/image"
	dockerApi "github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	"github.com/deepspace2/plugnpin/pkg/clients/adguardhome"
	"github.com/deepspace2/plugnpin/pkg/clients/docker"
	"github.com/deepspace2/plugnpin/pkg/clients/npm"
	"github.com/deepspace2/plugnpin/pkg/clients/pihole"
	"github.com/deepspace2/plugnpin/pkg/logging"
	"github.com/deepspace2/plugnpin/pkg/processor"
)

type config struct {
	AdguardHomeTag string `env:"ADGUARD_HOME_IMAGE_TAG,required,notEmpty"`
	NpmTag         string `env:"NPM_IMAGE_TAG,required,notEmpty"`
	PiholeTag      string `env:"PIHOLE_IMAGE_TAG,required,notEmpty"`
}

const (
	adguardHomeContainerName = "plugnpin-e2e-test-adguardhome"
	adguardHomeImage         = "adguard/adguardhome"

	npmContainerName = "plugnpin-e2e-test-npm"
	npmImage         = "jc21/nginx-proxy-manager"

	piholeContainerName = "plugnpin-e2e-test-pihole"
	piholeImage         = "pihole/pihole"

	testContainerImage = "busybox"
	testContainerName  = "plugnpin-e2e-test-testcontainer"
)

var logger = logging.GetLogger()

func getEnvVars() (*config, error) {
	err := godotenv.Load(".env.test")
	if err != nil {
		return nil, err
	}
	var config config
	if err := env.Parse(&config); err != nil {
		return nil, err
	}
	return &config, nil
}

func pullRequiredImages(t *testing.T, ctx context.Context, dockerApi *dockerApi.Client, containers []Container) {
	g, gCtx := errgroup.WithContext(ctx)

	for i := range containers {
		g.Go(func() error {
			err := pullImage(gCtx, dockerApi, containers[i].image)
			if err != nil {
				return fmt.Errorf("failed to pull image %s: %w", containers[i].image, err)
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		cleanup(t, ctx, dockerApi, containers, nil)
		t.Fatal(err)
	}
}

func startRequiredContainers(t *testing.T, ctx context.Context, dockerCli *dockerApi.Client, containers []Container) {
	g, gCtx := errgroup.WithContext(ctx)

	for i := range containers {
		g.Go(func() error {
			cfg := &containerApi.Config{
				Cmd:          containers[i].cmd,
				Env:          containers[i].env,
				Image:        containers[i].image,
				Labels:       containers[i].labels,
				ExposedPorts: make(nat.PortSet),
			}

			// Automatically populate ExposedPorts from the PortBindings in HostConfig
			if containers[i].hostConfig != nil {
				for port := range containers[i].hostConfig.PortBindings {
					cfg.ExposedPorts[port] = struct{}{}
				}
			}

			response, err := dockerCli.ContainerCreate(
				gCtx,
				cfg,
				containers[i].hostConfig,
				nil,
				nil,
				containers[i].name,
			)
			if err != nil {
				logger.Error("error creating container", "name", containers[i].name, "error", err)
				return fmt.Errorf("failed to create container %s: %v", containers[i].name, err)
			}
			containers[i].id = response.ID
			logger.Info("container started", "name", containers[i].name, "id", containers[i].id)
			err = dockerCli.ContainerStart(ctx, containers[i].id, containerApi.StartOptions{})
			if err != nil {
				return fmt.Errorf("failed to start container %s: %v", containers[i].name, err)
			}

			if containers[i].exposedPort != "" {
				containerInfo, err := dockerCli.ContainerInspect(ctx, containers[i].id)
				if err != nil {
					return fmt.Errorf("failed to inspect container %s: %v", containers[i].name, err)
				}
				bindings := containerInfo.NetworkSettings.Ports[containers[i].exposedPort]
				if len(bindings) > 0 {
					hostPort := bindings[0].HostPort
					containers[i].url = fmt.Sprintf("http://127.0.0.1:%s", hostPort)
					logger.Info("Discovered URL for container", "name", containers[i].name, "url", containers[i].url)
				}
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		cleanup(t, ctx, dockerCli, containers, nil)
		t.Fatal(err)
	}
}

func setClients(t *testing.T, containers []Container) (*docker.Client, *pihole.Client, *npm.Client, *adguardhome.Client) {
	dockerClient, err := docker.NewClient("")
	if err != nil {
		t.Fatalf("Failed to create docker client: %v", err)
	}

	var piholeURL, npmURL, adguardHomeURL string
	for _, c := range containers {
		switch c.name {
		case piholeContainerName:
			piholeURL = c.url
		case npmContainerName:
			npmURL = c.url
		case adguardHomeContainerName:
			adguardHomeURL = c.url
		}
	}

	piholeClient := pihole.NewClient(piholeURL)
	logger.Info("Waiting for Pi-hole to be ready...")
	piholeLoginTimeout := time.After(60 * time.Second)
	piholeLoginTicker := time.NewTicker(3 * time.Second)
	defer piholeLoginTicker.Stop()
PiholeLoginLoop:
	for {
		select {
		case <-piholeLoginTimeout:
			t.Fatalf("Timed out waiting for Pi-hole to be ready at %s", piholeURL)
		case <-piholeLoginTicker.C:
			err = piholeClient.Login("password")
			if err == nil {
				logger.Info("Successfully logged into Pi-hole")
				break PiholeLoginLoop
			}
			logger.Error("Pi-hole not ready, retrying...", "error", err)
		}
	}

	npmClient := npm.NewClient(npmURL, "a@a.com", "aaaaaaaa")
	logger.Info("Waiting for Nginx Proxy Manager to be ready...")
	npmLoginTimeout := time.After(60 * time.Second)
	npmLoginTicker := time.NewTicker(3 * time.Second)
	defer npmLoginTicker.Stop()
NPMLoginLoop:
	for {
		select {
		case <-npmLoginTimeout:
			t.Fatalf("Timed out waiting for Nginx Proxy Manager to be ready at %s", npmURL)
		case <-npmLoginTicker.C:
			err = npmClient.Login()
			if err == nil {
				logger.Info("Successfully logged into Nginx Proxy Manager")
				break NPMLoginLoop
			}
			logger.Error("NPM not ready, retrying...", "error", err)
		}
	}

	adguardHomeClient := adguardhome.NewClient(adguardHomeURL, "", "")

	return dockerClient, piholeClient, npmClient, adguardHomeClient
}

func setup(t *testing.T, ctx context.Context, dockerCli *dockerApi.Client, containers []Container) (*docker.Client, *pihole.Client, *npm.Client, *adguardhome.Client) {
	pullRequiredImages(t, ctx, dockerCli, containers)
	startRequiredContainers(t, ctx, dockerCli, containers)
	dockerClient, piholeClient, npmClient, adguardHomeClient := setClients(t, containers)
	return dockerClient, piholeClient, npmClient, adguardHomeClient
}

func cleanup(t *testing.T, ctx context.Context, dockerCli *dockerApi.Client, containers []Container, npmClient *npm.Client) {
	logger.Info("In cleanup")

	if npmClient != nil {
		npmProxyHosts, err := npmClient.GetProxyHosts()
		if err == nil {
			npmClient.DeleteProxyHosts(slices.Collect(maps.Keys(npmProxyHosts)))
		}
	}
	imagesToRemove := make(map[string]struct{})
	var wg sync.WaitGroup
	for i := range containers {
		imagesToRemove[containers[i].image] = struct{}{}
		wg.Go(func() {
			if containers[i].id == "" {
				return
			}
			err := dockerCli.ContainerRemove(ctx, containers[i].id, containerApi.RemoveOptions{
				Force: true,
			})
			if err != nil {
				logger.Error("Could not remove container", "name", containers[i].name, "error", err)
			}
		})
	}
	wg.Wait()

	for image := range imagesToRemove {
		_, err := dockerCli.ImageRemove(ctx, image, imageApi.RemoveOptions{
			Force: true,
		})
		if err != nil {
			logger.Error("Could not remove image", "image", image, "error", err)
		}
	}
}

func TestE2E(t *testing.T) {
	logger.Info("In TestE2E")

	conf, err := getEnvVars()
	if err != nil {
		t.Fatalf("Failed to load e2e test env vars: %v", err)
	}

	ctx := context.Background()
	dockerCli, err := dockerApi.NewClientWithOpts(dockerApi.FromEnv)
	if err != nil {
		t.Fatalf("Failed to create docker api client: %v", err)
	}

	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	containers := []Container{
		{
			image: fmt.Sprintf("%s:%s", npmImage, conf.NpmTag),
			name:  npmContainerName,
			hostConfig: &containerApi.HostConfig{
				Binds: []string{
					fmt.Sprintf("%s/npm-data/data:/data", workingDir),
					fmt.Sprintf("%s/npm-data/letsencrypt:/etc/letsencrypt", workingDir),
				},
				PortBindings: nat.PortMap{
					nat.Port("81/tcp"): []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: ""}},
				},
			},
			exposedPort: nat.Port("81/tcp"),
		},
		{
			env:   []string{`FTLCONF_webserver_api_password=password`},
			image: fmt.Sprintf("%s:%s", piholeImage, conf.PiholeTag),
			name:  piholeContainerName,
			hostConfig: &containerApi.HostConfig{
				PortBindings: nat.PortMap{
					nat.Port("80/tcp"): []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: ""}},
				},
			},
			exposedPort: nat.Port("80/tcp"),
		},
		{
			image: fmt.Sprintf("%s:%s", adguardHomeImage, conf.AdguardHomeTag),
			name:  adguardHomeContainerName,
			hostConfig: &containerApi.HostConfig{
				PortBindings: nat.PortMap{
					nat.Port("3000/tcp"): []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: ""}},
				},
				Binds: []string{
					fmt.Sprintf("%s/adguardhome-data/conf:/opt/adguardhome/conf", workingDir),
				},
			},
			exposedPort: nat.Port("3000/tcp"),
		},
		{
			image: testContainerImage,
			cmd:   []string{"tail", "-f", "/dev/null"},
			labels: map[string]string{
				docker.IpLabel:  "1.1.1.1:8080",
				docker.UrlLabel: "busybox1.home",
			},
			name:       testContainerName + "-1",
			hostConfig: &containerApi.HostConfig{},
		},
		{
			image: testContainerImage,
			cmd:   []string{"tail", "-f", "/dev/null"},
			labels: map[string]string{
				docker.IpLabel:  "2.2.2.2:8080",
				docker.UrlLabel: "busybox2.home,busybox2.local",
			},
			name:       testContainerName + "-2",
			hostConfig: &containerApi.HostConfig{},
		},
	}

	dockerClient, piholeClient, npmClient, adguardHomeClient := setup(t, ctx, dockerCli, containers)

	t.Cleanup(func() {
		cleanup(t, ctx, dockerCli, containers, npmClient)
	})

	time.Sleep(2 * time.Second)

	proc := processor.New(
		map[string]*docker.Client{dockerClient.Host: dockerClient},
		adguardHomeClient,
		piholeClient,
		npmClient,
		false,
	)

	proc.RunOnce()

	piholeDnsRecords, err := piholeClient.GetDnsRecords()
	if err != nil {
		t.Fatalf("Failed to get pihole DNS records: %v", err)
	}

	npmProxyHosts, err := npmClient.GetProxyHosts()
	if err != nil {
		t.Fatalf("Failed to get NPM proxy hosts: %v", err)
	}

	adguardDnsRewrites, err := adguardHomeClient.GetDnsRewrites()
	if err != nil {
		t.Fatalf("Failed to get AdGuard Home DNS rewrites: %v", err)
	}

	for _, container := range containers {
		urlsString, dockerUrlLabelExists := container.labels[docker.UrlLabel]
		if dockerUrlLabelExists {
			urls := strings.Split(urlsString, ",")
			for _, url := range urls {
				piholeDnsRecordIP, exists := piholeDnsRecords[pihole.DomainName(url)]

				// Assert that the "add" functionality worked
				require.True(t, exists, "A pihole DNS record should exist for the url %s", url)
				require.Equal(t, pihole.IP(npmClient.GetIP()), piholeDnsRecordIP, "The pihole DNS record should point to the NPM container's IP")
				require.Contains(t, npmProxyHosts, url, "The NPM proxy hosts should contain the url %s", url)

				adguardHomeDnsRewriteIP, exists := adguardDnsRewrites[adguardhome.DomainName(url)]
				require.True(t, exists, "An AdGuard Home DNS rewrite should exist for the url %s", url)
				require.Equal(t, adguardhome.IP(npmClient.GetIP()), adguardHomeDnsRewriteIP, "The AdGuard Home DNS rewrite should point to the NPM container's IP")
			}

			// Deleting to assert delete functionality
			piholeClient.DeleteDnsRecords(urls)
			npmClient.DeleteProxyHosts(urls)
			adguardHomeClient.DeleteDnsRewrites(urls, npmClient.GetIP())

			piholeDnsRecords, err := piholeClient.GetDnsRecords()
			if err != nil {
				t.Fatalf("Failed to get pihole DNS records after delete: %v", err)
			}
			npmProxyHosts, err := npmClient.GetProxyHosts()
			if err != nil {
				t.Fatalf("Failed to get NPM proxy hosts after delete: %v", err)
			}
			adguardDnsRewrites, err := adguardHomeClient.GetDnsRewrites()
			if err != nil {
				t.Fatalf("Failed to get AdGuard Home DNS rewrites after delete: %v", err)
			}

			// Assert that the "delete" functionality worked
			for _, url := range urls {
				require.NotContains(t, piholeDnsRecords, pihole.DomainName(url), "The pihole DNS record should be deleted for %s", url)
				require.NotContains(t, npmProxyHosts, url, "The NPM proxy host should be deleted for %s", url)
				require.NotContains(t, adguardDnsRewrites, url, "The AdGuard Home DNS rewrite should be deleted for %s", url)
			}
		}
	}
}
