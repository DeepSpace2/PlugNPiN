package docker

import dockerSdk "github.com/docker/go-sdk/client"

const (
	CONTAINER_HEALTHY_STATUS = "healthy"
)

type Client struct {
	*dockerSdk.Client
	DisplayHost string
	Host        string
}
