package docker

import dockerSdk "github.com/docker/go-sdk/client"

const (
	start = "start"
	die   = "die"
)

type Client struct {
	*dockerSdk.Client
	DisplayHost string
	Host        string
}

type EventType string

type ContainerEventEnum struct {
	Start EventType
	Die   EventType
}

var ContainerEvent = ContainerEventEnum{
	Start: start,
	Die:   die,
}

func (et EventType) String() string {
	return string(et)
}

func (ce ContainerEventEnum) ParseString(s string) (EventType, bool) {
	event, ok := map[string]EventType{
		start: ce.Start,
		die:   ce.Die,
	}[s]
	return event, ok
}
