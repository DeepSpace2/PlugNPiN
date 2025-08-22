package config

import (
	"os"
	"reflect"
	"testing"
	"time"
)

func TestGetEnvVars(t *testing.T) {
	// A list of all environment variables used by the Config struct
	configEnvVars := []string{
		"NGINX_PROXY_MANAGER_HOST",
		"NGINX_PROXY_MANAGER_PASSWORD",
		"NGINX_PROXY_MANAGER_USERNAME",
		"PIHOLE_HOST",
		"PIHOLE_PASSWORD",
		"DOCKER_HOST",
		"RUN_INTERVAL",
	}

	testCases := []struct {
		name           string
		envVars        map[string]string
		expectedConfig *Config
		expectErr      bool
	}{
		{
			name: "Happy Path - all vars set",
			envVars: map[string]string{
				"NGINX_PROXY_MANAGER_HOST":     "npm.example.com",
				"NGINX_PROXY_MANAGER_PASSWORD": "password",
				"NGINX_PROXY_MANAGER_USERNAME": "user",
				"PIHOLE_HOST":                  "pihole.example.com",
				"PIHOLE_PASSWORD":              "pihole_pass",
				"DOCKER_HOST":                  "unix:///var/run/docker.sock",
				"RUN_INTERVAL":                 "5m",
			},
			expectedConfig: &Config{
				NpmHost:        "npm.example.com",
				NpmPassword:    "password",
				NpmUsername:    "user",
				PiholeHost:     "pihole.example.com",
				PiholePassword: "pihole_pass",
				DockerHost:     "unix:///var/run/docker.sock",
				RunInterval:    5 * time.Minute,
			},
			expectErr: false,
		},
		{
			name: "Error - missing required env var",
			envVars: map[string]string{
				"NGINX_PROXY_MANAGER_HOST":     "npm.example.com",
				"NGINX_PROXY_MANAGER_PASSWORD": "password",
				// NGINX_PROXY_MANAGER_USERNAME is missing
				"PIHOLE_HOST":     "pihole.example.com",
				"PIHOLE_PASSWORD": "pihole_pass",
			},
			expectedConfig: nil,
			expectErr:      true,
		},
		{
			name: "Default value for RUN_INTERVAL",
			envVars: map[string]string{
				"NGINX_PROXY_MANAGER_HOST":     "npm.example.com",
				"NGINX_PROXY_MANAGER_PASSWORD": "password",
				"NGINX_PROXY_MANAGER_USERNAME": "user",
				"PIHOLE_HOST":                  "pihole.example.com",
				"PIHOLE_PASSWORD":              "pihole_pass",
			},
			expectedConfig: &Config{
				NpmHost:        "npm.example.com",
				NpmPassword:    "password",
				NpmUsername:    "user",
				PiholeHost:     "pihole.example.com",
				PiholePassword: "pihole_pass",
				RunInterval:    1 * time.Hour, // Default value
			},
			expectErr: false,
		},
		{
			name: "Invalid RUN_INTERVAL",
			envVars: map[string]string{
				"NGINX_PROXY_MANAGER_HOST":     "npm.example.com",
				"NGINX_PROXY_MANAGER_PASSWORD": "password",
				"NGINX_PROXY_MANAGER_USERNAME": "user",
				"PIHOLE_HOST":                  "pihole.example.com",
				"PIHOLE_PASSWORD":              "pihole_pass",
				"RUN_INTERVAL":                 "-5m",
			},
			expectedConfig: nil,
			expectErr:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Unset all config-related env vars to ensure a clean slate
			for _, key := range configEnvVars {
				os.Unsetenv(key)
			}

			// Set environment variables for the current test case
			// t.Setenv will handle restoring them after the test
			for key, value := range tc.envVars {
				t.Setenv(key, value)
			}

			config, err := GetEnvVars()

			if (err != nil) != tc.expectErr {
				t.Errorf("Expected error: %v, got: %v", tc.expectErr, err)
			}

			if !reflect.DeepEqual(config, tc.expectedConfig) {
				t.Errorf("Expected config: \nwant: %+v, \ngot:  %+v", tc.expectedConfig, config)
			}
		})
	}
}
