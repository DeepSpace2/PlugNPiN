package config

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

func GetEnvVars() (map[string]string, error) {
	_ = godotenv.Load()
	envVars := map[string]string{}
	for _, mandatoryEnvVar := range []string{
		"PIHOLE_HOST",
		"PIHOLE_PASSWORD",
		"NGINX_PROXY_MANAGER_HOST",
		"NGINX_PROXY_MANAGER_USERNAME",
		"NGINX_PROXY_MANAGER_PASSWORD",
	} {
		if value, exists := os.LookupEnv(mandatoryEnvVar); !exists {
			return nil, fmt.Errorf("missing env var %v", mandatoryEnvVar)
		} else {
			envVars[mandatoryEnvVar] = value
		}
	}

	for nonMandatoryEnvVar, defaultValue := range map[string]string{
		"RUN_INTERVAL": "1h",
	} {
		var valueToSet string
		if value, exists := os.LookupEnv(nonMandatoryEnvVar); !exists {
			log.Printf("Env var '%v' is not set, using default value '%v'", nonMandatoryEnvVar, defaultValue)
			valueToSet = defaultValue
		} else {
			valueToSet = value
		}
		envVars[nonMandatoryEnvVar] = valueToSet
	}

	return envVars, nil
}

func ParseRunInterval(runIntervalStr string) time.Duration {
	runInterval, err := time.ParseDuration(runIntervalStr)
	if err != nil || runInterval < 0 {
		log.Fatalf("Invalid RUN_INTERVAL: %v", err)
	}
	return runInterval
}
