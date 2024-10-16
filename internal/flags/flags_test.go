package flags

import (
	"bytes"
	"os"
	"testing"

	"github.com/kuskoman/logstash-exporter/pkg/config"
)

func TestParseFlags(t *testing.T) {
	t.Run("parses flags correctly", func(t *testing.T) {
		t.Parallel()

		args := []string{
			"-version", "-help", "-watch", "-config", "/path/to/config.yml",
		}
		flagsConfig, err := ParseFlags(args)
		if err != nil {
			t.Fatalf("unexpected error parsing flags: %v", err)
		}

		if !flagsConfig.Version {
			t.Errorf("expected Version to be true, got false")
		}
		if !flagsConfig.Help {
			t.Errorf("expected Help to be true, got false")
		}
		if !flagsConfig.HotReload {
			t.Errorf("expected HotReload to be true, got false")
		}
		if flagsConfig.ConfigLocation != "/path/to/config.yml" {
			t.Errorf("expected ConfigLocation to be '/path/to/config.yml', got %s", flagsConfig.ConfigLocation)
		}
	})

	t.Run("returns error for invalid flag", func(t *testing.T) {
		args := []string{
			"-invalidFlag",
		}
		_, err := ParseFlags(args)
		if err == nil {
			t.Fatalf("expected error for invalid flag, but got none")
		}
	})
}

func TestHandleFlags(t *testing.T) {
	t.Run("prints help when Help flag is true", func(t *testing.T) {
		// Capture the output of the print functions
		output, err := captureOutput(func() {
			flagsConfig := &FlagsConfig{Help: true}
			result := HandleFlags(flagsConfig)
			if !result {
				t.Error("expected HandleFlags to return true, but got false")
			}
		})

		if err != nil {
			t.Fatalf("unexpected error capturing output: %v", err)
		}

		if !contains(output, "Usage of") {
			t.Errorf("expected 'Usage of' to be printed, but it was not found in output: %s", output)
		}
	})

	t.Run("prints version when Version flag is true", func(t *testing.T) {

		// Mock config.SemanticVersion
		config.SemanticVersion = "v1.0.0"

		output, err := captureOutput(func() {
			flagsConfig := &FlagsConfig{Version: true}
			result := HandleFlags(flagsConfig)
			if !result {
				t.Error("expected HandleFlags to return true, but got false")
			}
		})

		if err != nil {
			t.Fatalf("unexpected error capturing output: %v", err)
		}

		if !contains(output, "v1.0.0") {
			t.Errorf("expected version 'v1.0.0' to be printed, but it was not found in output: %s", output)
		}
	})

	t.Run("does nothing if no flags are set", func(t *testing.T) {

		flagsConfig := &FlagsConfig{}
		result := HandleFlags(flagsConfig)
		if result {
			t.Error("expected HandleFlags to return false, but got true")
		}
	})
}

// Helper function to capture output printed to stdout
func captureOutput(f func()) (string, error) {
	var buf bytes.Buffer
	stdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = stdout
	_, err := buf.ReadFrom(r)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

// Helper function to check if a string contains a substring
func contains(str, substr string) bool {
	return bytes.Contains([]byte(str), []byte(substr))
}
