//go:build e2e

package e2e_tests

import (
	"context"
	"os"

	imageApi "github.com/docker/docker/api/types/image"
	dockerCliClient "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/moby/term"
)

func pullImage(ctx context.Context, dockerCli *dockerCliClient.Client, img string) error {
	reader, err := dockerCli.ImagePull(ctx, img, imageApi.PullOptions{})
	if err != nil {
		return err
	}

	defer reader.Close()
	termFd, isTerm := term.GetFdInfo(os.Stderr)
	jsonmessage.DisplayJSONMessagesStream(reader, os.Stderr, termFd, isTerm, nil)
	return nil
}

