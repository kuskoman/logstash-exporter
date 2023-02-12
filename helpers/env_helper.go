package helpers

import (
	"errors"
	"log"
	"os"

	"github.com/joho/godotenv"
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}
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
