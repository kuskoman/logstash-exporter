package config

import (
	"os"
	"testing"
)

func TestGetEnvWithDefault(t *testing.T) {
	t.Run("should return value for set environment variable", func(t *testing.T) {
		key := "TEST3"
		expected := "value"
		if err := os.Setenv(key, expected); err != nil {
			t.Fatalf("failed to set environment variable: %v", err)
		}
		actual := getEnvWithDefault(key, "default")
		if actual != expected {
			t.Errorf("expected %s but got %s", expected, actual)
		}
	})

	t.Run("should return default value for unset environment variable", func(t *testing.T) {
		expected := "default"
		actual := getEnvWithDefault("TEST_UNSET3", expected)
		if actual != expected {
			t.Errorf("expected %s but got %s", expected, actual)
		}
	})
}
