//go:build e2e

package e2e_tests

import (
	"context"
	"fmt"
	"os"

	containerApi "github.com/docker/docker/api/types/container"
	imageApi "github.com/docker/docker/api/types/image"
	dockerCliClient "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/moby/term"
)

func pullImage(ctx context.Context, dockerCli *dockerCliClient.Client, img string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	reader, err := dockerCli.ImagePull(ctx, img, imageApi.PullOptions{})
	if err != nil {
		return err
	}

	defer reader.Close()
	termFd, isTerm := term.GetFdInfo(os.Stderr)
	jsonmessage.DisplayJSONMessagesStream(reader, os.Stderr, termFd, isTerm, nil)
	return nil
}

func markContainerHealthy(ctx context.Context, dockerCli *dockerCliClient.Client, containerID string) error {
	execConfig := containerApi.ExecOptions{
		Cmd: []string{"touch", "/tmp/healthy"},
	}

	execCreateResp, err := dockerCli.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return fmt.Errorf("failed to create exec: %w", err)
	}

	err = dockerCli.ContainerExecStart(ctx, execCreateResp.ID, containerApi.ExecStartOptions{})
	if err != nil {
		return fmt.Errorf("failed to start exec: %w", err)
	}

	return nil
}
