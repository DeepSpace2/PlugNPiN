//go:build unit

package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetConfig_EnvVars(t *testing.T) {
	oldPath := dockerSecretRootPath
	defer func() { dockerSecretRootPath = oldPath }()
	dockerSecretRootPath = t.TempDir()

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
				AdguardHomeDisabled: true,
				NpmHost:             "npm.example.com",
				NpmPassword:         "password",
				NpmUsername:         "user",
				PiholeDisabled:      false,
				PiholeHost:          "pihole.example.com",
				PiholePassword:      "pihole_pass",
				DockerHost:          "unix:///var/run/docker.sock",
				RunInterval:         5 * time.Minute,
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
				AdguardHomeDisabled: true,
				NpmHost:             "npm.example.com",
				NpmPassword:         "password",
				NpmUsername:         "user",
				PiholeDisabled:      false,
				PiholeHost:          "pihole.example.com",
				PiholePassword:      "pihole_pass",
				RunInterval:         1 * time.Hour, // Default value
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
		{
			name: "No need to set Pi-Hole env vars if Pi-Hole is disabled",
			envVars: map[string]string{
				"NGINX_PROXY_MANAGER_HOST":     "npm.example.com",
				"NGINX_PROXY_MANAGER_PASSWORD": "password",
				"NGINX_PROXY_MANAGER_USERNAME": "user",
				"PIHOLE_DISABLED":              "true",
				"DOCKER_HOST":                  "unix:///var/run/docker.sock",
				"RUN_INTERVAL":                 "5m",
			},
			expectedConfig: &Config{
				AdguardHomeDisabled: true,
				NpmHost:             "npm.example.com",
				NpmPassword:         "password",
				NpmUsername:         "user",
				PiholeDisabled:      true,
				DockerHost:          "unix:///var/run/docker.sock",
				RunInterval:         5 * time.Minute,
			},
			expectErr: false,
		},
		{
			name: "Need to set Pi-Hole env vars if Pi-Hole is enabled",
			envVars: map[string]string{
				"NGINX_PROXY_MANAGER_HOST":     "npm.example.com",
				"NGINX_PROXY_MANAGER_PASSWORD": "password",
				"NGINX_PROXY_MANAGER_USERNAME": "user",
				"DOCKER_HOST":                  "unix:///var/run/docker.sock",
				"RUN_INTERVAL":                 "5m",
			},
			expectedConfig: nil,
			expectErr:      true,
		},
		{
			name: "No need to set AdguardHome env vars if AdguardHome is disabled",
			envVars: map[string]string{
				"ADGUARD_HOME_DISABLED":        "true",
				"NGINX_PROXY_MANAGER_HOST":     "npm.example.com",
				"NGINX_PROXY_MANAGER_PASSWORD": "password",
				"NGINX_PROXY_MANAGER_USERNAME": "user",
				"PIHOLE_DISABLED":              "true",
				"DOCKER_HOST":                  "unix:///var/run/docker.sock",
				"RUN_INTERVAL":                 "5m",
			},
			expectedConfig: &Config{
				AdguardHomeDisabled: true,
				NpmHost:             "npm.example.com",
				NpmPassword:         "password",
				NpmUsername:         "user",
				PiholeDisabled:      true,
				DockerHost:          "unix:///var/run/docker.sock",
				RunInterval:         5 * time.Minute,
			},
			expectErr: false,
		},
		{
			name: "Need to set AdguardHome env vars if AdguardHome is enabled",
			envVars: map[string]string{
				"ADGUARD_HOME_DISABLED":        "false",
				"PIHOLE_DISABLED":              "true",
				"NGINX_PROXY_MANAGER_HOST":     "npm.example.com",
				"NGINX_PROXY_MANAGER_PASSWORD": "password",
				"NGINX_PROXY_MANAGER_USERNAME": "user",
				"DOCKER_HOST":                  "unix:///var/run/docker.sock",
				"RUN_INTERVAL":                 "5m",
			},
			expectedConfig: nil,
			expectErr:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			unsetAllConfigEnvVars()
			for key, value := range tc.envVars {
				t.Setenv(key, value)
			}

			config, err := Get()

			if tc.expectErr {
				assert.Error(t, err)
				assert.Nil(t, config)
			} else {
				assert.Equal(t, tc.expectedConfig, config)
			}
		})
	}
}

func TestGetConfig_DockerSecrets(t *testing.T) {
	unsetAllConfigEnvVars()
	oldPath := dockerSecretRootPath
	defer func() { dockerSecretRootPath = oldPath }()

	tmpDir := t.TempDir()
	dockerSecretRootPath = tmpDir

	// DYNAMIC SETUP: Iterate over Config fields and create mock secrets for those tagged with secret:"true"
	typ := reflect.TypeOf(Config{})
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.Tag.Get("secret") == "true" {
			envName := field.Tag.Get("env")
			mockValue := "secret-val-for-" + envName
			err := writeFile(tmpDir, envName, mockValue)
			assert.NoError(t, err, "failed to create secret file for "+envName)
		}
	}

	t.Setenv("ADGUARD_HOME_DISABLED", "true")

	config, err := Get()
	assert.NoError(t, err)

	// DYNAMIC VERIFICATION: Ensure every secret-enabled field was loaded correctly
	cfgVal := reflect.ValueOf(config).Elem()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.Tag.Get("secret") == "true" {
			envName := field.Tag.Get("env")

			expected := "secret-val-for-" + envName
			actual := cfgVal.Field(i).String()
			assert.Equal(t, expected, actual, "Field %s was not loaded correctly from secret %s", field.Name, envName)
		}
	}
}

func TestGetConfig_SecretPrecedence(t *testing.T) {
	unsetAllConfigEnvVars()
	oldPath := dockerSecretRootPath
	defer func() { dockerSecretRootPath = oldPath }()

	tmpDir := t.TempDir()
	dockerSecretRootPath = tmpDir

	// Use reflection to find a field that supports secrets (e.g., PiholePassword)
	field, _ := reflect.TypeOf(Config{}).FieldByName("PiholePassword")
	envName := field.Tag.Get("env")

	secretValue := "secret-pw"
	err := writeFile(tmpDir, envName, secretValue)
	assert.NoError(t, err)

	// Set an environment variable for the same secret
	envValue := "env-pw"
	t.Setenv(envName, envValue)
	t.Setenv("NGINX_PROXY_MANAGER_HOST", "npm.local")
	t.Setenv("NGINX_PROXY_MANAGER_USERNAME", "npm-user")
	t.Setenv("NGINX_PROXY_MANAGER_PASSWORD", "npm-pw")
	t.Setenv("PIHOLE_HOST", "pihole.local")
	t.Setenv("ADGUARD_HOME_DISABLED", "true")

	config, err := Get()
	assert.NoError(t, err)

	// Verify precedence: env var should win
	assert.Equal(t, envValue, config.PiholePassword)
	assert.NotEqual(t, secretValue, config.PiholePassword)
}

func TestGetConfig_SecretsValidation(t *testing.T) {
	unsetAllConfigEnvVars()
	oldPath := dockerSecretRootPath
	defer func() { dockerSecretRootPath = oldPath }()

	tmpDir := t.TempDir()
	dockerSecretRootPath = tmpDir

	t.Setenv("NGINX_PROXY_MANAGER_HOST", "npm.local")
	t.Setenv("PIHOLE_HOST", "pihole.local")
	t.Setenv("ADGUARD_HOME_DISABLED", "true")

	_, err := Get()
	assert.Error(t, err, "Expected an error due to missing credentials")
}

func unsetAllConfigEnvVars() {
	typ := reflect.TypeOf(Config{})
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		envName := field.Tag.Get("env")
		if envName != "" {
			os.Unsetenv(envName)
		}
	}
}

func writeFile(dir, fileName, content string) error {
	return os.WriteFile(filepath.Join(dir, fileName), []byte(content), 0o644)
}
