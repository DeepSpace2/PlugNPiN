package docker

const (
	start = "start"
	stop  = "stop"
	kill  = "kill"
)

type EventType string

type ContainerEventEnum struct {
	Start EventType
	Stop  EventType
	Kill  EventType
}

var ContainerEvent = ContainerEventEnum{
	Start: start,
	Stop:  stop,
	Kill:  kill,
}

func (et EventType) String() string {
	return string(et)
}

func (ce ContainerEventEnum) ParseString(s string) (EventType, bool) {
	event, ok := map[string]EventType{
		"start": ce.Start,
		"stop":  ce.Stop,
		"kill":  ce.Kill,
	}[s]
	return event, ok
}
