package config

import (
	"os"
	"testing"
	"time"
)

func TestGetHttpTimeout(t *testing.T) {
	t.Run("DefaultTimeout", func(t *testing.T) {
		t.Parallel()
		os.Unsetenv(httpTimeoutEnvVar)
		timeout, err := GetHttpTimeout()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if timeout != defaultHttpTimeout {
			t.Errorf("Expected default timeout of %v, got %v", defaultHttpTimeout, timeout)
		}
	})

	t.Run("CustomTimeout", func(t *testing.T) {
		t.Parallel()
		expectedTimeout := "5s"
		os.Setenv(httpTimeoutEnvVar, expectedTimeout)
		defer os.Unsetenv(httpTimeoutEnvVar)
		timeout, err := GetHttpTimeout()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		parsedTimeout, _ := time.ParseDuration(expectedTimeout)
		if timeout != parsedTimeout {
			t.Errorf("Expected timeout of %v, got %v", parsedTimeout, timeout)
		}
	})

	t.Run("InvalidTimeout", func(t *testing.T) {
		t.Parallel()
		os.Setenv(httpTimeoutEnvVar, "invalid")
		defer os.Unsetenv(httpTimeoutEnvVar)
		_, err := GetHttpTimeout()
		if err == nil {
			t.Error("Expected an error for invalid timeout, but got nil")
		}
	})
}
