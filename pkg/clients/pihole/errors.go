package pihole

import (
	"errors"
)

var (
	errAuthRefreshFailed = errors.New("failed to refresh Pi-Hole authentication")
	errMissingSessionId  = errors.New("missing Pi-Hole session ID")
)
