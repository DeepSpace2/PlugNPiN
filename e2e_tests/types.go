//go:build e2e

package e2e_tests

import (
	containerApi "github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
)

type Container struct {
	cmd         []string
	env         []string
	exposedPort nat.Port
	hostConfig  *containerApi.HostConfig
	id          string
	image       string
	labels      map[string]string
	name        string
	url         string
}
