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

var (
	logger                 = logging.GetLogger("e2e")
	globalImagesToRemove   = make(map[string]struct{})
	globalImagesToRemoveMu sync.Mutex
)

func TestMain(m *testing.M) {
	code := m.Run()

	ctx := context.Background()
	dockerCli, err := dockerApi.NewClientWithOpts(dockerApi.FromEnv)
	if err == nil {
		logger.Info("Performing global image cleanup")
		for image := range globalImagesToRemove {
			_, _ = dockerCli.ImageRemove(ctx, image, imageApi.RemoveOptions{
				Force: true,
			})
		}
	}

	os.Exit(code)
}

func getInfraContainers(conf *config, workingDir string) []Container {
	return []Container{
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
	}
}

func getEnvVars() (*config, error) {
	err := godotenv.Load(".env.test")
	if err != nil {
		return nil, err
	}
	var cfg config
	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func pullRequiredImages(t *testing.T, ctx context.Context, dockerApi *dockerApi.Client, containers []Container) {
	g, gCtx := errgroup.WithContext(ctx)

	for i := range containers {
		g.Go(func() error {
			err := pullImage(gCtx, dockerApi, containers[i].image)
			if err != nil {
				return fmt.Errorf("failed to pull image %s: %w", containers[i].image, err)
			}

			globalImagesToRemoveMu.Lock()
			globalImagesToRemove[containers[i].image] = struct{}{}
			globalImagesToRemoveMu.Unlock()

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		cleanup(dockerApi, containers, nil)
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
				Healthcheck:  containers[i].healthcheck,
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
			err = dockerCli.ContainerStart(gCtx, containers[i].id, containerApi.StartOptions{})
			if err != nil {
				return fmt.Errorf("failed to start container %s: %v", containers[i].name, err)
			}

			if containers[i].exposedPort != "" {
				containerInfo, err := dockerCli.ContainerInspect(gCtx, containers[i].id)
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
		cleanup(dockerCli, containers, nil)
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

	piholeClient := pihole.NewClient(piholeURL, "password")
	logger.Info("Waiting for Pi-Hole to be ready...")
	piholeLoginTimeout := time.After(60 * time.Second)
	piholeLoginTicker := time.NewTicker(3 * time.Second)
	defer piholeLoginTicker.Stop()
PiholeLoginLoop:
	for {
		select {
		case <-piholeLoginTimeout:
			t.Fatalf("Timed out waiting for Pi-Hole to be ready at %s", piholeURL)
		case <-piholeLoginTicker.C:
			err = piholeClient.Login()
			if err == nil {
				logger.Info("Successfully logged into Pi-Hole")
				break PiholeLoginLoop
			}
			logger.Error("Pi-Hole not ready, retrying...", "error", err)
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

func cleanup(dockerCli *dockerApi.Client, containers []Container, npmClient *npm.Client) {
	logger.Info("In cleanup")

	// Use background context for cleanup to ensure it completes even if test is canceled
	ctx := context.Background()

	if npmClient != nil {
		npmProxyHosts, err := npmClient.GetProxyHosts()
		if err == nil {
			_, _ = npmClient.DeleteProxyHosts(slices.Collect(maps.Keys(npmProxyHosts)))
		}
	}
	var wg sync.WaitGroup
	for i := range containers {
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

	infraContainers := getInfraContainers(conf, workingDir)
	testContainers := []Container{
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
	containers := append(infraContainers, testContainers...)

	dockerClient, piholeClient, npmClient, adguardHomeClient := setup(t, ctx, dockerCli, containers)

	t.Cleanup(func() {
		cleanup(dockerCli, containers, npmClient)
	})

	time.Sleep(2 * time.Second)

	proc := processor.New(
		map[string]*docker.Client{dockerClient.Host: dockerClient},
		adguardHomeClient,
		piholeClient,
		npmClient,
		false,
	)

	proc.RunOnce(ctx)

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
			_, err = piholeClient.DeleteDnsRecords(urls)
			require.NoError(t, err, "Failed to delete Pi-Hole DNS records")

			_, err = npmClient.DeleteProxyHosts(urls)
			require.NoError(t, err, "Failed to delete NPM proxy hosts")

			_, err = adguardHomeClient.DeleteDnsRewrites(urls, npmClient.GetIP())
			require.NoError(t, err, "Failed to delete AdGuard Home DNS rewrites")

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

func TestE2E_CreateOnHealthy(t *testing.T) {
	logger.Info("In TestE2E_CreateOnHealthy")

	conf, err := getEnvVars()
	if err != nil {
		t.Fatalf("Failed to load e2e test env vars: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dockerCli, err := dockerApi.NewClientWithOpts(dockerApi.FromEnv)
	if err != nil {
		t.Fatalf("Failed to create docker api client: %v", err)
	}

	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	infraContainers := getInfraContainers(conf, workingDir)

	dockerClient, piholeClient, npmClient, adguardHomeClient := setup(t, ctx, dockerCli, infraContainers)

	t.Cleanup(func() {
		cleanup(dockerCli, infraContainers, npmClient)
	})

	proc := processor.New(
		map[string]*docker.Client{dockerClient.Host: dockerClient},
		adguardHomeClient,
		piholeClient,
		npmClient,
		false,
	)

	var wg sync.WaitGroup
	wg.Go(func() {
		proc.ListenForEvents(ctx)
	})

	testURL := "healthy-test.home"
	testContainers := []Container{
		{
			image: testContainerImage,
			cmd:   []string{"tail", "-f", "/dev/null"},
			labels: map[string]string{
				docker.IpLabel:  "3.3.3.3:8080",
				docker.UrlLabel: testURL,
				docker.GeneralOptionsCreateOnHealthyLabel: "true",
			},
			name:       testContainerName + "-healthy-trigger",
			hostConfig: &containerApi.HostConfig{},
			healthcheck: &containerApi.HealthConfig{
				Test:     []string{"CMD-SHELL", "stat /tmp/healthy || exit 1"},
				Interval: 1 * time.Second,
				Retries:  30,
			},
		},
	}

	err = pullImage(ctx, dockerCli, testContainers[0].image)
	require.NoError(t, err, "Failed to pull test container image")
	startRequiredContainers(t, ctx, dockerCli, testContainers)

	t.Cleanup(func() {
		_ = dockerCli.ContainerRemove(context.Background(), testContainers[0].id, containerApi.RemoveOptions{Force: true})
	})

	// Assert entry DOES NOT exist while unhealthy
	logger.Info("Asserting entry does not exist while container is unhealthy")
	require.Never(t, func() bool {
		npmProxyHosts, _ := npmClient.GetProxyHosts()
		_, exists := npmProxyHosts[testURL]
		return exists
	}, 3*time.Second, 1*time.Second, "Entry should not be created for unhealthy container")

	// Trigger healthy state
	logger.Info("Triggering healthy state")
	err = markContainerHealthy(ctx, dockerCli, testContainers[0].id)
	require.NoError(t, err)

	// Wait for Docker to mark it healthy
	logger.Info("Waiting for Docker to mark container as healthy")
	require.Eventually(t, func() bool {
		inspect, inspectErr := dockerCli.ContainerInspect(ctx, testContainers[0].id)
		if inspectErr != nil {
			return false
		}
		return inspect.State.Health != nil && inspect.State.Health.Status == "healthy"
	}, 15*time.Second, 1*time.Second, "Container should become healthy")

	// Assert entry exists
	logger.Info("Asserting entry now exists after container became healthy")
	require.Eventually(t, func() bool {
		npmProxyHosts, _ := npmClient.GetProxyHosts()
		_, exists := npmProxyHosts[testURL]
		return exists
	}, 10*time.Second, 1*time.Second, "Entry should be created after container becomes healthy")

	// Assert entry is REMOVED on Die
	logger.Info("Stopping container and asserting entry removal")
	err = dockerCli.ContainerStop(ctx, testContainers[0].id, containerApi.StopOptions{})
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		npmProxyHosts, _ := npmClient.GetProxyHosts()
		_, exists := npmProxyHosts[testURL]
		return !exists
	}, 10*time.Second, 1*time.Second, "Entry should be removed after container dies")

	cancel()
	wg.Wait()
}

func TestE2E_CreateOnHealthy_NoHealthcheck(t *testing.T) {
	logger.Info("In TestE2E_CreateOnHealthy_NoHealthcheck")

	conf, err := getEnvVars()
	if err != nil {
		t.Fatalf("Failed to load e2e test env vars: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dockerCli, err := dockerApi.NewClientWithOpts(dockerApi.FromEnv)
	if err != nil {
		t.Fatalf("Failed to create docker api client: %v", err)
	}

	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	infraContainers := getInfraContainers(conf, workingDir)

	dockerClient, _, npmClient, adguardHomeClient := setup(t, ctx, dockerCli, infraContainers)

	t.Cleanup(func() {
		cleanup(dockerCli, infraContainers, npmClient)
	})

	proc := processor.New(
		map[string]*docker.Client{dockerClient.Host: dockerClient},
		adguardHomeClient,
		nil,
		npmClient,
		false,
	)

	var wg sync.WaitGroup
	wg.Go(func() {
		proc.ListenForEvents(ctx)
	})

	testURL := "no-healthcheck-test.home"
	testContainers := []Container{
		{
			image: testContainerImage,
			cmd:   []string{"tail", "-f", "/dev/null"},
			labels: map[string]string{
				docker.IpLabel:  "4.4.4.4:8080",
				docker.UrlLabel: testURL,
				docker.GeneralOptionsCreateOnHealthyLabel: "true",
			},
			name:       testContainerName + "-no-healthcheck",
			hostConfig: &containerApi.HostConfig{},
			// NO healthcheck defined
		},
	}

	err = pullImage(ctx, dockerCli, testContainers[0].image)
	require.NoError(t, err, "Failed to pull test container image")
	startRequiredContainers(t, ctx, dockerCli, testContainers)

	t.Cleanup(func() {
		_ = dockerCli.ContainerRemove(context.Background(), testContainers[0].id, containerApi.RemoveOptions{Force: true})
	})

	// Assert entry NEVER exists
	logger.Info("Asserting entry is NOT created for container without healthcheck")
	require.Never(t, func() bool {
		npmProxyHosts, _ := npmClient.GetProxyHosts()
		_, exists := npmProxyHosts[testURL]
		return exists
	}, 5*time.Second, 1*time.Second, "Entry should NOT be created when no healthcheck is defined")

	cancel()
	wg.Wait()
}
