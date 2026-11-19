//go:build unit

package docker

import (
	"fmt"
	"strings"
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
		name                                   string
		container                              container.Summary
		expectedIP                             string
		expectedURLs                           []string
		expectedPort                           int
		expectedErr                            error
		expectedAdguardHomeOptionsTargetDomain string
		expectedNpmOptionsBlockExploits        bool
		expectedNpmOptionsCachingEnabled       bool
		expectedNpmOptionsScheme               string
		expectedNpmOptionsWebsocketsSupport    bool
		expectedPiholeOptionsTargetDomain      string
		expectedCreateOnHealthy                bool
	}{
		{
			name: "Happy path",
			container: container.Summary{
				Labels: map[string]string{
					IpLabel:  "192.168.1.10:8080",
					UrlLabel: "my-service.example.com",
				},
			},
			expectedIP:                          "192.168.1.10",
			expectedURLs:                        []string{"my-service.example.com"},
			expectedPort:                        8080,
			expectedErr:                         nil,
			expectedNpmOptionsBlockExploits:     true,
			expectedNpmOptionsCachingEnabled:    false,
			expectedNpmOptionsScheme:            "http",
			expectedNpmOptionsWebsocketsSupport: false,
		},
		{
			name: "Multiple URLs",
			container: container.Summary{
				Labels: map[string]string{
					IpLabel:  "192.168.1.10:8080",
					UrlLabel: "my-service.example.com,my-service.local",
				},
			},
			expectedIP:                          "192.168.1.10",
			expectedURLs:                        []string{"my-service.example.com", "my-service.local"},
			expectedPort:                        8080,
			expectedErr:                         nil,
			expectedNpmOptionsBlockExploits:     true,
			expectedNpmOptionsCachingEnabled:    false,
			expectedNpmOptionsScheme:            "http",
			expectedNpmOptionsWebsocketsSupport: false,
		},
		{
			name: "Malformed IP label - missing port",
			container: container.Summary{
				Labels: map[string]string{
					IpLabel:  "192.168.1.10",
					UrlLabel: "my-service.example.com",
				},
			},
			expectedIP:   "",
			expectedURLs: nil,
			expectedPort: 0,
			expectedErr:  &errors.MalformedIPLabelError{Msg: fmt.Sprintf("missing ':' in value of '%v' label", IpLabel)},
		},
		{
			name: "Malformed IP label - non-integer port",
			container: container.Summary{
				Labels: map[string]string{
					IpLabel:  "192.168.1.10:http",
					UrlLabel: "my-service.example.com",
				},
			},
			expectedIP:   "",
			expectedURLs: nil,
			expectedPort: 0,
			expectedErr:  &errors.MalformedIPLabelError{Msg: fmt.Sprintf("value after ':' in value of '%v' label must be an integer, got 'http'", IpLabel)},
		},
		{
			name: "NPM options",
			container: container.Summary{
				Labels: map[string]string{
					IpLabel:                          "192.168.1.10:8080",
					UrlLabel:                         "my-service.example.com",
					npmOptionsBlockExploitsLabel:     "",
					npmOptionsCachingEnabledLabel:    "true",
					npmOptionsSchemeLabel:            "https",
					npmOptionsWebsocketsSupportLabel: "",
				},
			},
			expectedIP:                          "192.168.1.10",
			expectedURLs:                        []string{"my-service.example.com"},
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
					IpLabel:                          "192.168.1.10:8080",
					UrlLabel:                         "my-service.example.com",
					npmOptionsBlockExploitsLabel:     "true",
					npmOptionsCachingEnabledLabel:    "1",
					npmOptionsWebsocketsSupportLabel: "T",
				},
			},
			expectedIP:                          "192.168.1.10",
			expectedURLs:                        []string{"my-service.example.com"},
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
					IpLabel:                          "192.168.1.10:8080",
					UrlLabel:                         "my-service.example.com",
					npmOptionsBlockExploitsLabel:     "false",
					npmOptionsCachingEnabledLabel:    "0",
					npmOptionsWebsocketsSupportLabel: "F",
				},
			},
			expectedIP:                          "192.168.1.10",
			expectedURLs:                        []string{"my-service.example.com"},
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
					IpLabel:                          "192.168.1.10:8080",
					UrlLabel:                         "my-service.example.com",
					npmOptionsBlockExploitsLabel:     "yes",
					npmOptionsCachingEnabledLabel:    "no",
					npmOptionsWebsocketsSupportLabel: "2",
				},
			},
			expectedIP:                          "192.168.1.10",
			expectedURLs:                        []string{"my-service.example.com"},
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
					IpLabel:               "192.168.1.10:8080",
					UrlLabel:              "my-service.example.com",
					npmOptionsSchemeLabel: "invalid",
				},
			},
			expectedIP:   "",
			expectedURLs: nil,
			expectedPort: 0,
			expectedErr:  &errors.InvalidSchemeError{Msg: fmt.Sprintf("value of '%v' label must be one of 'http', 'https', got 'invalid'", npmOptionsSchemeLabel)},
		},
		{
			name: "NPM options - case-insensitive scheme",
			container: container.Summary{
				Labels: map[string]string{
					IpLabel:               "192.168.1.10:8080",
					UrlLabel:              "my-service.example.com",
					npmOptionsSchemeLabel: "HTTPS",
				},
			},
			expectedIP:                          "192.168.1.10",
			expectedURLs:                        []string{"my-service.example.com"},
			expectedPort:                        8080,
			expectedErr:                         nil,
			expectedNpmOptionsBlockExploits:     true,
			expectedNpmOptionsCachingEnabled:    false,
			expectedNpmOptionsScheme:            "https",
			expectedNpmOptionsWebsocketsSupport: false,
		},
		{
			name: "Pi-Hole options - no target domain",
			container: container.Summary{
				Labels: map[string]string{
					IpLabel:  "192.168.1.10:8080",
					UrlLabel: "my-service.example.com",
				},
			},
			expectedIP:                        "192.168.1.10",
			expectedURLs:                      []string{"my-service.example.com"},
			expectedPort:                      8080,
			expectedErr:                       nil,
			expectedNpmOptionsScheme:          "http",
			expectedNpmOptionsBlockExploits:   true,
			expectedPiholeOptionsTargetDomain: "",
		},
		{
			name: "Pi-Hole options - target domain",
			container: container.Summary{
				Labels: map[string]string{
					IpLabel:                        "192.168.1.10:8080",
					UrlLabel:                       "my-service.example.com",
					piholeOptionsTargetDomainLabel: "custom.domain",
				},
			},
			expectedIP:                        "192.168.1.10",
			expectedURLs:                      []string{"my-service.example.com"},
			expectedPort:                      8080,
			expectedErr:                       nil,
			expectedNpmOptionsScheme:          "http",
			expectedNpmOptionsBlockExploits:   true,
			expectedPiholeOptionsTargetDomain: "custom.domain",
		},
		{
			name: "AdguardHome options - target domain",
			container: container.Summary{
				Labels: map[string]string{
					IpLabel:                             "192.168.1.10:8080",
					UrlLabel:                            "my-service.example.com",
					adguardHomeOptionsTargetDomainLabel: "custom.domain.adguard",
				},
			},
			expectedIP:                             "192.168.1.10",
			expectedURLs:                           []string{"my-service.example.com"},
			expectedPort:                           8080,
			expectedErr:                            nil,
			expectedNpmOptionsScheme:               "http",
			expectedNpmOptionsBlockExploits:        true,
			expectedAdguardHomeOptionsTargetDomain: "custom.domain.adguard",
		},
		{
			name: "General options - CreateOnHealthy true",
			container: container.Summary{
				Labels: map[string]string{
					IpLabel:                            "192.168.1.10:8080",
					UrlLabel:                           "my-service.example.com",
					GeneralOptionsCreateOnHealthyLabel: "true",
				},
			},
			expectedIP:                      "192.168.1.10",
			expectedURLs:                    []string{"my-service.example.com"},
			expectedPort:                    8080,
			expectedErr:                     nil,
			expectedNpmOptionsScheme:        "http",
			expectedNpmOptionsBlockExploits: true,
			expectedCreateOnHealthy:         true,
		},
		{
			name: "General options - CreateOnHealthy false",
			container: container.Summary{
				Labels: map[string]string{
					IpLabel:                            "192.168.1.10:8080",
					UrlLabel:                           "my-service.example.com",
					GeneralOptionsCreateOnHealthyLabel: "false",
				},
			},
			expectedIP:                      "192.168.1.10",
			expectedURLs:                    []string{"my-service.example.com"},
			expectedPort:                    8080,
			expectedErr:                     nil,
			expectedNpmOptionsScheme:        "http",
			expectedNpmOptionsBlockExploits: true,
			expectedCreateOnHealthy:         false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ip, urls, port, opts, err := GetValuesFromLabels(tc.container.Labels)

			assert.Equal(t, tc.expectedIP, ip)
			assert.Equal(t, tc.expectedURLs, urls)
			assert.Equal(t, tc.expectedPort, port)
			assert.Equal(t, tc.expectedErr, err)
			if err == nil {
				assert.NotNil(t, opts)
				assert.NotNil(t, opts.NPM)
				assert.NotNil(t, opts.Pihole)
				assert.NotNil(t, opts.AdguardHome)
				assert.NotNil(t, opts.GeneralOptions)
				assert.Equal(t, tc.expectedNpmOptionsBlockExploits, opts.NPM.BlockExploits)
				assert.Equal(t, tc.expectedNpmOptionsCachingEnabled, opts.NPM.CachingEnabled)
				assert.Equal(t, tc.expectedNpmOptionsScheme, opts.NPM.ForwardScheme)
				assert.Equal(t, tc.expectedNpmOptionsWebsocketsSupport, opts.NPM.AllowWebsocketUpgrade)
				assert.Equal(t, tc.expectedPiholeOptionsTargetDomain, opts.Pihole.TargetDomain)
				assert.Equal(t, tc.expectedAdguardHomeOptionsTargetDomain, opts.AdguardHome.TargetDomain)
				assert.Equal(t, tc.expectedCreateOnHealthy, opts.GeneralOptions.CreateOnHealthy)
			} else {
				assert.Nil(t, opts)
			}
		})
	}
}

func TestGetValuesFromLabels_CaseInsensitiveKeys(t *testing.T) {
	testCases := []struct {
		name                    string
		labels                  map[string]string
		expectedIP              string
		expectedURLs            []string
		expectedPort            int
		expectedScheme          string
		expectedCreateOnHealthy bool
		expectedTargetDomain    string
	}{
		{
			name: "Lowercased keys",
			labels: map[string]string{
				strings.ToLower(IpLabel):                            "192.168.1.10:8080",
				strings.ToLower(UrlLabel):                           "my-service.example.com,my-service.local",
				strings.ToLower(npmOptionsSchemeLabel):              "HTTPS",
				strings.ToLower(GeneralOptionsCreateOnHealthyLabel): "true",
				strings.ToLower(piholeOptionsTargetDomainLabel):     "target.home",
			},
			expectedIP:              "192.168.1.10",
			expectedURLs:            []string{"my-service.example.com", "my-service.local"},
			expectedPort:            8080,
			expectedScheme:          "https",
			expectedCreateOnHealthy: true,
			expectedTargetDomain:    "target.home",
		},
		{
			name: "Arbitrary casing",
			labels: map[string]string{
				"PlugNPiN.Ip":  "10.0.0.5:3000",
				"PLUGNPIN.URL": "mixed.home",
			},
			expectedIP:     "10.0.0.5",
			expectedURLs:   []string{"mixed.home"},
			expectedPort:   3000,
			expectedScheme: "http",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ip, urls, port, opts, err := GetValuesFromLabels(tc.labels)

			assert.NoError(t, err)
			assert.Equal(t, tc.expectedIP, ip)
			assert.Equal(t, tc.expectedURLs, urls)
			assert.Equal(t, tc.expectedPort, port)
			if assert.NotNil(t, opts) {
				assert.Equal(t, tc.expectedScheme, opts.NPM.ForwardScheme)
				assert.Equal(t, tc.expectedCreateOnHealthy, opts.GeneralOptions.CreateOnHealthy)
				assert.Equal(t, tc.expectedTargetDomain, opts.Pihole.TargetDomain)
			}
		})
	}
}

func TestGetValuesFromLabels_AllLabelsWithLowercasedKeys(t *testing.T) {
	labels := map[string]string{
		strings.ToLower(IpLabel):                             "192.168.1.10:8080",
		strings.ToLower(UrlLabel):                            "my-service.example.com",
		strings.ToLower(GeneralOptionsCreateOnHealthyLabel):  "true",
		strings.ToLower(adguardHomeOptionsTargetDomainLabel): "adguard.target",
		strings.ToLower(npmOptionsAccessListNameLabel):       "my-access-list",
		strings.ToLower(npmOptionsAdvancedConfigLabel):       "proxy_set_header X-Test 1;",
		strings.ToLower(npmOptionsBlockExploitsLabel):        "false",
		strings.ToLower(npmOptionsCachingEnabledLabel):       "true",
		strings.ToLower(npmOptionsCertificateNameLabel):      "my-certificate",
		strings.ToLower(npmOptionsHTTP2SupportLabel):         "true",
		strings.ToLower(npmOptionsHstsEnabledLabel):          "true",
		strings.ToLower(npmOptionsHstsSubdomainsLabel):       "true",
		strings.ToLower(npmOptionsSchemeLabel):               "https",
		strings.ToLower(npmOptionsSslForcedLabel):            "true",
		strings.ToLower(npmOptionsWebsocketsSupportLabel):    "true",
		strings.ToLower(piholeOptionsTargetDomainLabel):      "pihole.target",
	}

	_, _, _, opts, err := GetValuesFromLabels(labels)

	assert.NoError(t, err)
	if assert.NotNil(t, opts) {
		assert.True(t, opts.GeneralOptions.CreateOnHealthy)
		assert.Equal(t, "adguard.target", opts.AdguardHome.TargetDomain)
		assert.Equal(t, "my-access-list", opts.NPM.AccessListName)
		assert.Equal(t, "proxy_set_header X-Test 1;", opts.NPM.AdvancedConfig)
		assert.False(t, opts.NPM.BlockExploits)
		assert.True(t, opts.NPM.CachingEnabled)
		assert.Equal(t, "my-certificate", opts.NPM.CertificateName)
		assert.True(t, opts.NPM.HTTP2Support)
		assert.True(t, opts.NPM.HstsEnabled)
		assert.True(t, opts.NPM.HstsSubdomains)
		assert.Equal(t, "https", opts.NPM.ForwardScheme)
		assert.True(t, opts.NPM.SslForced)
		assert.True(t, opts.NPM.AllowWebsocketUpgrade)
		assert.Equal(t, "pihole.target", opts.Pihole.TargetDomain)
	}
}

func TestGetValuesFromLabels_CanonicalKeyTakesPrecedence(t *testing.T) {
	labels := map[string]string{
		"plugNPiN.ip":  "1.1.1.1:1111",
		"plugnpin.ip":  "2.2.2.2:2222",
		"plugNPiN.url": "canonical.home",
		"PLUGNPIN.URL": "ignored.home",
	}

	ip, urls, port, opts, err := GetValuesFromLabels(labels)

	assert.NoError(t, err)
	assert.Equal(t, "1.1.1.1", ip)
	assert.Equal(t, []string{"canonical.home"}, urls)
	assert.Equal(t, 1111, port)
	assert.NotNil(t, opts)
}

func TestHasRequiredLabels(t *testing.T) {
	testCases := []struct {
		name     string
		labels   map[string]string
		expected bool
	}{
		{
			name:     "Canonical labels",
			labels:   map[string]string{IpLabel: "1.1.1.1:80", UrlLabel: "service.home"},
			expected: true,
		},
		{
			name:     "Differently cased labels",
			labels:   map[string]string{"PLUGNPIN.IP": "1.1.1.1:80", "plugnpin.Url": "service.home"},
			expected: true,
		},
		{
			name:     "Missing url",
			labels:   map[string]string{IpLabel: "1.1.1.1:80"},
			expected: false,
		},
		{
			name:     "Missing ip",
			labels:   map[string]string{UrlLabel: "service.home"},
			expected: false,
		},
		{
			name:     "Unrelated labels",
			labels:   map[string]string{"com.example.foo": "bar"},
			expected: false,
		},
		{
			name:     "Empty labels",
			labels:   map[string]string{},
			expected: false,
		},
		{
			name:     "Nil labels",
			labels:   nil,
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, hasRequiredLabels(tc.labels))
		})
	}
}
