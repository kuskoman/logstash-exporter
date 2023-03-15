package config

import (
	"os"

	"github.com/joho/godotenv"
)

// InitializeEnv loads the environment variables from the .env file
func InitializeEnv() error {
	return godotenv.Load()
}

func getEnvWithDefault(key string, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
