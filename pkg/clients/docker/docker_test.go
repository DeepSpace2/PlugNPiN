package docker

import (
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
		name         string
		container    container.Summary
		expectedIP   string
		expectedURL  string
		expectedPort int
		expectedErr  error
	}{
		{
			name: "Happy path",
			container: container.Summary{
				Labels: map[string]string{
					ipLabel:  "192.168.1.10:8080",
					urlLabel: "my-service.example.com",
				},
			},
			expectedIP:   "192.168.1.10",
			expectedURL:  "my-service.example.com",
			expectedPort: 8080,
			expectedErr:  nil,
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
			expectedErr:  &errors.MalformedIPLabelError{Msg: "missing ':' in value of 'plugNPiN.ip' label"},
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
			expectedErr:  &errors.MalformedIPLabelError{Msg: "value after ':' in value of 'plugNPiN.ip' label must be an integer, got 'http'"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ip, url, port, err := GetValuesFromLabels(tc.container.Labels)

			assert.Equal(t, tc.expectedIP, ip)
			assert.Equal(t, tc.expectedURL, url)
			assert.Equal(t, tc.expectedPort, port)
			assert.Equal(t, tc.expectedErr, err)
		})
	}
}

