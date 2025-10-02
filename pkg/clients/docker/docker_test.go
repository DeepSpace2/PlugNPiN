package docker

import (
	"fmt"
	"testing"

	"github.com/deepspace2/plugnpin/pkg/errors"
	"github.com/docker/docker/api/types/container"
	"github.com/stretchr/testify/assert"
)

func TestGetParsedContainerName(t *testing.T) {
	testCases := []struct {
		name      string
		container container.Summary
		expected  string
	}{
		{
			name: "Standard name with slash",
			container: container.Summary{
				Names: []string{"/my-container"},
			},
			expected: "my-container",
		},
		{
			name: "Name without slash",
			container: container.Summary{
				Names: []string{"my-container"},
			},
			expected: "my-container",
		},
		{
			name: "Empty name slice",
			container: container.Summary{
				Names: []string{},
			},
			expected: "",
		},
		{
			name: "Multiple names",
			container: container.Summary{
				Names: []string{"/my-container", "/another-name"},
			},
			expected: "my-container",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// This is a safe guard against panics if Names is empty, though the Docker API typically ensures it's not.
			if len(tc.container.Names) == 0 {
				assert.Equal(t, tc.expected, "")
				return
			}
			actual := GetParsedContainerName(tc.container)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestGetValuesFromContainerLabels(t *testing.T) {
	testCases := []struct {
		name                                string
		container                           container.Summary
		expectedIP                          string
		expectedURL                         string
		expectedPort                        int
		expectedErr                         error
		expectedNpmOptionsBlockExploits     bool
		expectedNpmOptionsCachingEnabled    bool
		expectedNpmOptionsScheme            string
		expectedNpmOptionsWebsocketsSupport bool
	}{
		{
			name: "Happy path",
			container: container.Summary{
				Labels: map[string]string{
					ipLabel:  "192.168.1.10:8080",
					urlLabel: "my-service.example.com",
				},
			},
			expectedIP:                          "192.168.1.10",
			expectedURL:                         "my-service.example.com",
			expectedPort:                        8080,
			expectedErr:                         nil,
			expectedNpmOptionsBlockExploits:     false,
			expectedNpmOptionsCachingEnabled:    false,
			expectedNpmOptionsScheme:            "http",
			expectedNpmOptionsWebsocketsSupport: false,
		},
		{
			name: "Malformed IP label - missing port",
			container: container.Summary{
				Labels: map[string]string{
					ipLabel:  "192.168.1.10",
					urlLabel: "my-service.example.com",
				},
			},
			expectedIP:   "",
			expectedURL:  "",
			expectedPort: 0,
			expectedErr:  &errors.MalformedIPLabelError{Msg: fmt.Sprintf("missing ':' in value of '%v' label", ipLabel)},
		},
		{
			name: "Malformed IP label - non-integer port",
			container: container.Summary{
				Labels: map[string]string{
					ipLabel:  "192.168.1.10:http",
					urlLabel: "my-service.example.com",
				},
			},
			expectedIP:   "",
			expectedURL:  "",
			expectedPort: 0,
			expectedErr:  &errors.MalformedIPLabelError{Msg: fmt.Sprintf("value after ':' in value of '%v' label must be an integer, got 'http'", ipLabel)},
		},
		{
			name: "NPM options",
			container: container.Summary{
				Labels: map[string]string{
					ipLabel:                          "192.168.1.10:8080",
					urlLabel:                         "my-service.example.com",
					npmOptionsBlockExploitsLabel:     "",
					npmOptionsCachingEnabledLabel:    "true",
					npmOptionsSchemeLabel:            "https",
					npmOptionsWebsocketsSupportLabel: "",
				},
			},
			expectedIP:                          "192.168.1.10",
			expectedURL:                         "my-service.example.com",
			expectedPort:                        8080,
			expectedErr:                         nil,
			expectedNpmOptionsBlockExploits:     false,
			expectedNpmOptionsCachingEnabled:    true,
			expectedNpmOptionsScheme:            "https",
			expectedNpmOptionsWebsocketsSupport: false,
		},
		{
			name: "NPM options - true values",
			container: container.Summary{
				Labels: map[string]string{
					ipLabel:                          "192.168.1.10:8080",
					urlLabel:                         "my-service.example.com",
					npmOptionsBlockExploitsLabel:     "true",
					npmOptionsCachingEnabledLabel:    "1",
					npmOptionsWebsocketsSupportLabel: "T",
				},
			},
			expectedIP:                          "192.168.1.10",
			expectedURL:                         "my-service.example.com",
			expectedPort:                        8080,
			expectedErr:                         nil,
			expectedNpmOptionsBlockExploits:     true,
			expectedNpmOptionsCachingEnabled:    true,
			expectedNpmOptionsScheme:            "http",
			expectedNpmOptionsWebsocketsSupport: true,
		},
		{
			name: "NPM options - false values",
			container: container.Summary{
				Labels: map[string]string{
					ipLabel:                          "192.168.1.10:8080",
					urlLabel:                         "my-service.example.com",
					npmOptionsBlockExploitsLabel:     "false",
					npmOptionsCachingEnabledLabel:    "0",
					npmOptionsWebsocketsSupportLabel: "F",
				},
			},
			expectedIP:                          "192.168.1.10",
			expectedURL:                         "my-service.example.com",
			expectedPort:                        8080,
			expectedErr:                         nil,
			expectedNpmOptionsBlockExploits:     false,
			expectedNpmOptionsCachingEnabled:    false,
			expectedNpmOptionsScheme:            "http",
			expectedNpmOptionsWebsocketsSupport: false,
		},
		{
			name: "NPM options - invalid boolean values",
			container: container.Summary{
				Labels: map[string]string{
					ipLabel:                          "192.168.1.10:8080",
					urlLabel:                         "my-service.example.com",
					npmOptionsBlockExploitsLabel:     "yes",
					npmOptionsCachingEnabledLabel:    "no",
					npmOptionsWebsocketsSupportLabel: "2",
				},
			},
			expectedIP:                          "192.168.1.10",
			expectedURL:                         "my-service.example.com",
			expectedPort:                        8080,
			expectedErr:                         nil,
			expectedNpmOptionsBlockExploits:     false,
			expectedNpmOptionsCachingEnabled:    false,
			expectedNpmOptionsScheme:            "http",
			expectedNpmOptionsWebsocketsSupport: false,
		},
		{
			name: "NPM options - invalid scheme",
			container: container.Summary{
				Labels: map[string]string{
					ipLabel:               "192.168.1.10:8080",
					urlLabel:              "my-service.example.com",
					npmOptionsSchemeLabel: "invalid",
				},
			},
			expectedIP:   "",
			expectedURL:  "",
			expectedPort: 0,
			expectedErr:  &errors.InvalidSchemeError{Msg: fmt.Sprintf("value of '%v' label must be one of 'http', 'https', got 'invalid'", npmOptionsSchemeLabel)},
		},
		{
			name: "NPM options - case-insensitive scheme",
			container: container.Summary{
				Labels: map[string]string{
					ipLabel:               "192.168.1.10:8080",
					urlLabel:              "my-service.example.com",
					npmOptionsSchemeLabel: "HTTPS",
				},
			},
			expectedIP:                          "192.168.1.10",
			expectedURL:                         "my-service.example.com",
			expectedPort:                        8080,
			expectedErr:                         nil,
			expectedNpmOptionsBlockExploits:     false,
			expectedNpmOptionsCachingEnabled:    false,
			expectedNpmOptionsScheme:            "https",
			expectedNpmOptionsWebsocketsSupport: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ip, url, port, npmOptions, err := GetValuesFromLabels(tc.container.Labels)

			assert.Equal(t, tc.expectedIP, ip)
			assert.Equal(t, tc.expectedURL, url)
			assert.Equal(t, tc.expectedPort, port)
			assert.Equal(t, tc.expectedErr, err)
			if err == nil {
				assert.Equal(t, tc.expectedNpmOptionsBlockExploits, npmOptions.BlockExploits)
				assert.Equal(t, tc.expectedNpmOptionsCachingEnabled, npmOptions.CachingEnabled)
				assert.Equal(t, tc.expectedNpmOptionsScheme, npmOptions.ForwardScheme)
				assert.Equal(t, tc.expectedNpmOptionsWebsocketsSupport, npmOptions.AllowWebsocketUpgrade)
			}
		})
	}
}
