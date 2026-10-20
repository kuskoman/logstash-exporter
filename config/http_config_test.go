package config

import (
	"testing"
	"time"
)

func TestGetHttpTimeout(t *testing.T) {
	t.Run("DefaultTimeout", func(t *testing.T) {
		t.Setenv(httpTimeoutEnvVar, "")
		timeout, err := GetHttpTimeout()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if timeout != defaultHttpTimeout {
			t.Errorf("Expected default timeout of %v, got %v", defaultHttpTimeout, timeout)
		}
	})

	t.Run("CustomTimeout", func(t *testing.T) {
		expectedTimeout := "5s"
		t.Setenv(httpTimeoutEnvVar, expectedTimeout)
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
		t.Setenv(httpTimeoutEnvVar, "invalid")
		_, err := GetHttpTimeout()
		if err == nil {
			t.Error("Expected an error for invalid timeout, but got nil")
		}
	})
}

func TestGetHttpInsecure(t *testing.T) {
	t.Run("DefaultInsecure", func(t *testing.T) {
		t.Setenv(httpInsecureEnvVar, "")
		insecure := GetHttpInsecure()
		if insecure != false {
			t.Errorf("Expected default insecure of %v, got %v", false, insecure)
		}
	})

	t.Run("CustomInsecure", func(t *testing.T) {
		expectedInsecure := true
		t.Setenv(httpInsecureEnvVar, "true")
		insecure := GetHttpInsecure()
		if insecure != expectedInsecure {
			t.Errorf("Expected insecure of %v, got %v", expectedInsecure, insecure)
		}
	})

	t.Run("InvalidInsecure", func(t *testing.T) {
		expectedInsecure := false
		t.Setenv(httpInsecureEnvVar, "invalid")
		insecure := GetHttpInsecure()
		if insecure != expectedInsecure {
			t.Errorf("Expected insecure of %v, got %v", expectedInsecure, insecure)
		}
	})
}
