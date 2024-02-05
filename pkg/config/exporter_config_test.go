package config

import (
	"os"
	"testing"
)

// mockDoubleSetLogger to capture warning logs for testing
type mockDoubleSetLogger struct {
	Warnings map[string][]interface{} // Maps property names to warnings to allow more detailed assertions
}

func (m *mockDoubleSetLogger) Warn(propertyName string, keysAndValues ...interface{}) {
	if m.Warnings == nil {
		m.Warnings = make(map[string][]interface{})
	}
	m.Warnings[propertyName] = append(m.Warnings[propertyName], keysAndValues...)
}

func createTemporaryConfigFile(contents string) (string, error) {
	tmpFile, err := os.CreateTemp("", "config-*.yml")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	_, err = tmpFile.WriteString(contents)
	if err != nil {
		return "", err
	}

	return tmpFile.Name(), nil
}

func setEnvironmentVariable(key, value string) func() {
	originalValue, isSet := os.LookupEnv(key)
	os.Setenv(key, value)
	return func() {
		if isSet {
			os.Setenv(key, originalValue)
		} else {
			os.Unsetenv(key)
		}
	}
}

func TestDefaultDoubleSetLogger(t *testing.T) {
	t.Parallel()
	logger := defaultDoubleSetLogger{}

	t.Run("warn", func(t *testing.T) {
		t.Parallel()
		logger.Warn("test")
	})
}

func TestGetConfig(t *testing.T) {
	t.Run("with invalid path", func(t *testing.T) {
		t.Parallel()
		logger := &mockDoubleSetLogger{}
		_, err := GetConfig("invalidpath", logger)
		if err == nil {
			t.Fatal("expected error, got none")
		}
	})

	t.Run("without providing logger", func(t *testing.T) {
		t.Parallel()
		_, err := GetConfig("", nil)
		if err == nil {
			t.Fatal("expected error, got none")
		}
	})

	t.Run("with invalid port", func(t *testing.T) {
		configContent := ""
		configFileName, err := createTemporaryConfigFile(configContent)
		if err != nil {
			t.Fatalf("failed to create temp config file: %v", err)
		}
		defer os.Remove(configFileName)

		clearEnvPort := setEnvironmentVariable(envPort, "invalidport")
		defer clearEnvPort()

		logger := &mockDoubleSetLogger{}
		_, err = GetConfig(configFileName, logger)
		if err == nil {
			t.Fatal("expected error, got none")
		}
	})

	t.Run("with invalid environment config", func(t *testing.T) {
		t.Parallel()
		configContent := ""
		configFileName, err := createTemporaryConfigFile(configContent)
		if err != nil {
			t.Fatalf("failed to create temp config file: %v", err)
		}
		defer os.Remove(configFileName)

		clearEnvPort := setEnvironmentVariable(envPort, "invalidport")
		defer clearEnvPort()

		logger := &mockDoubleSetLogger{}
		_, err = GetConfig(configFileName, logger)
		if err == nil {
			t.Fatal("expected error, got none")
		}
	})
}

func TestMergeWithDefault(t *testing.T) {
	t.Run("with nil config", func(t *testing.T) {
		t.Parallel()
		logger := &mockDoubleSetLogger{}
		_, err := mergeWithDefault(nil, logger)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("with invalid http timeout override", func(t *testing.T) {
		t.Parallel()
		configContent := ""
		configFileName, err := createTemporaryConfigFile(configContent)
		if err != nil {
			t.Fatalf("failed to create temp config file: %v", err)
		}
		defer os.Remove(configFileName)

		clearEnvHttpTimeout := setEnvironmentVariable(envHttpTimeout, "invalidtimeout")
		defer clearEnvHttpTimeout()

		logger := &mockDoubleSetLogger{}
		_, err = GetConfig(configFileName, logger)
		if err == nil {
			t.Fatal("expected error, got none")
		}
	})
}

func TestLoggingWarnings(t *testing.T) {
	t.Run("with double set port", func(t *testing.T) {
		configContent := `
server:
  port: 8080
`
		configFileName, err := createTemporaryConfigFile(configContent)
		if err != nil {
			t.Fatalf("failed to create temp config file: %v", err)
		}
		defer os.Remove(configFileName)

		clearEnvPort := setEnvironmentVariable(envPort, "9090")
		defer clearEnvPort()

		logger := &mockDoubleSetLogger{}
		_, _ = GetConfig(configFileName, logger)

		if warnings, ok := logger.Warnings["port"]; !ok || len(warnings) == 0 {
			t.Fatal("expected warnings for double set port, got none")
		}
	})

	t.Run("with double set logstash url", func(t *testing.T) {
		configContent := `
logstash:
  servers:
    - url: "http://file-url:9600"
`
		configFileName, err := createTemporaryConfigFile(configContent)
		if err != nil {
			t.Fatalf("failed to create temp config file: %v", err)
		}
		defer os.Remove(configFileName)

		clearEnvLogstashURL := setEnvironmentVariable(envLogstashURL, "http://env-url:9600")
		defer clearEnvLogstashURL()

		logger := &mockDoubleSetLogger{}
		_, _ = GetConfig(configFileName, logger)

		if warnings, ok := logger.Warnings["logstash URL"]; !ok || len(warnings) == 0 {
			t.Fatal("expected warnings for double set logstash URL, got none")
		}
	})

	t.Run("with double set log format", func(t *testing.T) {
		configContent := `
logging:
  format: "json"
`
		configFileName, err := createTemporaryConfigFile(configContent)
		if err != nil {
			t.Fatalf("failed to create temp config file: %v", err)
		}
		defer os.Remove(configFileName)

		clearEnvLogFormat := setEnvironmentVariable(envLogFormat, "text")
		defer clearEnvLogFormat()

		logger := &mockDoubleSetLogger{}
		_, _ = GetConfig(configFileName, logger)

		if warnings, ok := logger.Warnings["log format"]; !ok || len(warnings) == 0 {
			t.Fatal("expected warnings for double set log format, got none")
		}
	})

	t.Run("with double set log level", func(t *testing.T) {
		configContent := `
logging:
  level: "debug"
`
		configFileName, err := createTemporaryConfigFile(configContent)
		if err != nil {
			t.Fatalf("failed to create temp config file: %v", err)
		}
		defer os.Remove(configFileName)

		clearEnvLogLevel := setEnvironmentVariable(envLogLevel, "info")
		defer clearEnvLogLevel()

		logger := &mockDoubleSetLogger{}
		_, _ = GetConfig(configFileName, logger)

		if warnings, ok := logger.Warnings["log level"]; !ok || len(warnings) == 0 {
			t.Fatal("expected warnings for double set log level, got none")
		}
	})
}

func TestLoadConfig(t *testing.T) {
	t.Run("with valid file", func(t *testing.T) {
		t.Parallel()
		configContent := `
logstash:
  servers:
    - url: "http://test-url:9600"
  httpTimeout: "5s"
server:
  host: "localhost"
  port: 8080
logging:
  level: "debug"
  format: "json"
`
		configFileName, err := createTemporaryConfigFile(configContent)
		if err != nil {
			t.Fatalf("failed to create temp config file: %v", err)
		}
		defer os.Remove(configFileName)

		config, err := loadConfig(configFileName)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if config.Server.Port != 8080 {
			t.Errorf("expected port 8080, got %d", config.Server.Port)
		}
	})

	t.Run("with non existent file", func(t *testing.T) {
		t.Parallel()
		_, err := loadConfig("nonexistentconfig.yml")
		if err == nil {
			t.Fatal("expected error, got none")
		}
	})

	t.Run("with invalid yaml", func(t *testing.T) {
		t.Parallel()
		invalidContent := "invalid: [yaml: format"
		fileName, err := createTemporaryConfigFile(invalidContent)
		if err != nil {
			t.Fatalf("failed to create temp invalid config file: %v", err)
		}
		defer os.Remove(fileName)

		_, err = loadConfig(fileName)
		if err == nil {
			t.Fatal("expected error, got none")
		}
	})
}
