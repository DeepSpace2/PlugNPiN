//go:build unit

package processor

import (
	"testing"

	"github.com/deepspace2/plugnpin/pkg/clients/docker"
	"github.com/docker/docker/api/types/events"
	"github.com/stretchr/testify/assert"
)

func TestShouldSkip(t *testing.T) {
	p := &Processor{}

	testCases := []struct {
		name            string
		createOnHealthy bool
		event           events.Action
		expected        bool
	}{
		{
			name:            "CreateOnHealthy true, event Start",
			createOnHealthy: true,
			event:           events.ActionStart,
			expected:        true,
		},
		{
			name:            "CreateOnHealthy true, event Healthy",
			createOnHealthy: true,
			event:           events.ActionHealthStatusHealthy,
			expected:        false,
		},
		{
			name:            "CreateOnHealthy false, event Start",
			createOnHealthy: false,
			event:           events.ActionStart,
			expected:        false,
		},
		{
			name:            "CreateOnHealthy false, event Healthy",
			createOnHealthy: false,
			event:           events.ActionHealthStatusHealthy,
			expected:        true,
		},
		{
			name:            "CreateOnHealthy true, event Die",
			createOnHealthy: true,
			event:           events.ActionDie,
			expected:        false,
		},
		{
			name:            "CreateOnHealthy false, event Die",
			createOnHealthy: false,
			event:           events.ActionDie,
			expected:        false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opts := docker.GeneralOptions{CreateOnHealthy: tc.createOnHealthy}
			actual := p.shouldSkip(&opts, tc.event)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
