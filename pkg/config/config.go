package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"

	"github.com/deepspace2/plugnpin/pkg/logging"
)

var (
	log                  = logging.GetLogger()
	dockerSecretRootPath = filepath.Join(string(os.PathSeparator), "run", "secrets")
)

type Config struct {
	Debug bool `env:"DEBUG" envDefault:"false"`

	AdguardHomeDisabled bool   `env:"ADGUARD_HOME_DISABLED" envDefault:"true"`
	AdguardHomeHost     string `env:"ADGUARD_HOME_HOST" secret:"true"`
	AdguardHomePassword string `env:"ADGUARD_HOME_PASSWORD" secret:"true"`
	AdguardHomeUsername string `env:"ADGUARD_HOME_USERNAME" secret:"true"`

	NpmHost     string `env:"NGINX_PROXY_MANAGER_HOST" secret:"true"`
	NpmPassword string `env:"NGINX_PROXY_MANAGER_PASSWORD" secret:"true"`
	NpmUsername string `env:"NGINX_PROXY_MANAGER_USERNAME" secret:"true"`

	PiholeDisabled bool   `env:"PIHOLE_DISABLED" envDefault:"false"`
	PiholeHost     string `env:"PIHOLE_HOST" secret:"true"`
	PiholePassword string `env:"PIHOLE_PASSWORD" secret:"true"`

	DockerHost  string        `env:"DOCKER_HOST"`
	DockerHosts []string      `env:"DOCKER_HOSTS"`
	RunInterval time.Duration `env:"RUN_INTERVAL" envDefault:"1h"`
}

func getValueFromSecret(secretFile string) (string, error) {
	path := filepath.Join(dockerSecretRootPath, secretFile)
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", fmt.Errorf("failed to read secret file %v: %w", path, err)
	}

	return strings.TrimSpace(string(content)), nil
}

func Get() (*Config, error) {
	var config Config

	// Reflection-based secret loading
	val := reflect.ValueOf(&config).Elem()
	typ := val.Type()

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.Tag.Get("secret") == "true" && val.Field(i).String() == "" {
			envName := field.Tag.Get("env")
			secretVal, err := getValueFromSecret(envName)
			if err != nil {
				return nil, err
			}
			val.Field(i).SetString(secretVal)
		}
	}

	_ = godotenv.Load()
	err := env.ParseWithOptions(&config, env.Options{
		OnSet: func(tag string, value any, isDefault bool) {
			if isDefault {
				log.Info(fmt.Sprintf(`env: environment variable '%v' is not set, using default value '%v'`, tag, value))
			}
		},
	})
	if err != nil {
		return nil, err
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &config, nil
}

func (c *Config) Validate() error {
	if c.RunInterval < 0 {
		return errors.New(`env: environment variable 'RUN_INTERVAL' must be >= 0`)
	}

	if c.NpmHost == "" {
		return errors.New(`env: NGINX_PROXY_MANAGER_HOST is required but not set via env var or secret`)
	}
	if c.NpmUsername == "" {
		return errors.New(`env: NGINX_PROXY_MANAGER_USERNAME is required but not set via env var or secret`)
	}
	if c.NpmPassword == "" {
		return errors.New(`env: NGINX_PROXY_MANAGER_PASSWORD is required but not set via env var or secret`)
	}

	if !c.PiholeDisabled {
		if c.PiholeHost == "" {
			return errors.New(`env: PIHOLE_HOST is required but not set via env var or secret`)
		}
		if c.PiholePassword == "" {
			return errors.New(`env: PIHOLE_PASSWORD is required but not set via env var or secret`)
		}
	}

	if !c.AdguardHomeDisabled {
		if c.AdguardHomeHost == "" {
			return errors.New(`env: ADGUARD_HOME_HOST is required but not set via env var or secret`)
		}
		if c.AdguardHomeUsername == "" {
			return errors.New(`env: ADGUARD_HOME_USERNAME is required but not set via env var or secret`)
		}
		if c.AdguardHomePassword == "" {
			return errors.New(`env: ADGUARD_HOME_PASSWORD is required but not set via env var or secret`)
		}
	}

	return nil
}
