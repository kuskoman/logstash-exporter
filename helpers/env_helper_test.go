package helpers

import (
	"os"
	"testing"
)

func TestGetEnv(t *testing.T) {
	t.Run("should return value for set environment variable", func(t *testing.T) {
		expected := "value"
		key := "TEST"
		os.Setenv(key, expected)
		actual := GetEnv(key)
		if actual != expected {
			t.Errorf("expected %s but got %s", expected, actual)
		}
	})

	t.Run("should return empty string for unset environment variable", func(t *testing.T) {
		expected := ""
		actual := GetEnv("TEST_UNSET")
		if actual != expected {
			t.Errorf("expected %s but got %s", expected, actual)
		}
	})
}

func TestGetRequiredEnv(t *testing.T) {
	t.Run("should return value for set environment variable", func(t *testing.T) {
		expected := "value"
		key := "TEST2"
		os.Setenv(key, expected)
		actual, err := GetRequiredEnv(key)
		if err != nil {
			t.Errorf("expected no error but got %s", err)
		}
		if actual != expected {
			t.Errorf("expected %s but got %s", expected, actual)
		}
	})

	t.Run("should return error for unset environment variable", func(t *testing.T) {
		key := "TEST_UNSET2"
		_, err := GetRequiredEnv(key)
		if err == nil {
			t.Errorf("expected error but got none")
		}
	})
}

func TestGetEnvWithDefault(t *testing.T) {
	t.Run("should return value for set environment variable", func(t *testing.T) {
		key := "TEST3"
		expected := "value"
		os.Setenv(key, expected)
		actual := GetEnvWithDefault(key, "default")
		if actual != expected {
			t.Errorf("expected %s but got %s", expected, actual)
		}
	})

	t.Run("should return default value for unset environment variable", func(t *testing.T) {
		expected := "default"
		actual := GetEnvWithDefault("TEST_UNSET3", expected)
		if actual != expected {
			t.Errorf("expected %s but got %s", expected, actual)
		}
	})
}
