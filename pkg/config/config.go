package config

import (
	"errors"
	"log"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	NpmHost        string `env:"NGINX_PROXY_MANAGER_HOST,notEmpty"`
	NpmPassword    string `env:"NGINX_PROXY_MANAGER_PASSWORD,notEmpty"`
	NpmUsername    string `env:"NGINX_PROXY_MANAGER_USERNAME,notEmpty"`
	PiholeHost     string `env:"PIHOLE_HOST,notEmpty"`
	PiholePassword string `env:"PIHOLE_PASSWORD,notEmpty"`

	DockerHost  string        `env:"DOCKER_HOST"`
	RunInterval time.Duration `env:"RUN_INTERVAL" envDefault:"1h"`
}

func GetEnvVars() (*Config, error) {
	_ = godotenv.Load()
	var config Config
	if err := env.ParseWithOptions(&config, env.Options{
		OnSet: func(tag string, value any, isDefault bool) {
			if isDefault {
				log.Printf(`env: environment variable "%v" is not set, using default value "%v"`, tag, value)
			}
		},
	}); err != nil {
		return nil, err
	}

	if !validateRunInterval(config.RunInterval) {
		return nil, errors.New(`env: environment variable "RUN_INTERVAL" must be >= 0`)
	}

	return &config, nil
}

func validateRunInterval(runInterval time.Duration) bool {
	return runInterval >= 0
}
