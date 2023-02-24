package helpers

import (
	"errors"
	"os"

	"github.com/joho/godotenv"
)

func InitializeEnv() error {
	return godotenv.Load()
}

func GetEnv(key string) string {
	return os.Getenv(key)
}

func GetRequiredEnv(key string) (string, error) {
	value := GetEnv(key)
	if value == "" {
		return "", errors.New("required environment variable " + key + " is not set")
	}
	return value, nil
}

func GetEnvWithDefault(key string, defaultValue string) string {
	value := GetEnv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
