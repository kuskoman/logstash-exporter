package config

import (
	"os"
	"strconv"
	"testing"
)

// mockDoubleSetLogger to capture warning logs for testing
type mockDoubleSetLogger struct {
	Warnings []string // Captures warnings to allow assertions
}

func (m *mockDoubleSetLogger) Warn(propertyName string, keysAndValues ...interface{}) {
	m.Warnings = append(m.Warnings, propertyName) // Simplified for demonstration; could be expanded to include all inputs
}

func createTemporaryConfigFile(contents string) (string, error) {
	tmpFile, err := os.CreateTemp("", "config-*.yml")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	_, err = tmpFile.WriteString(contents)
	return tmpFile.Name(), err
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

func TestDefaultWarningLogger(t *testing.T) {
	logger := &defaultDoubleSetLogger{}
	logger.Warn("test")
}

func TestGetConfigWithoutProvidingLogger(t *testing.T) {
	_, err := GetConfig("", nil)
	if err == nil {
		t.Fatal("expected error when getting config without providing logger, got none")
	}
}

func TestGetConfigInvalidPath(t *testing.T) {
	logger := &mockDoubleSetLogger{}
	_, err := GetConfig("invalidpath", logger)
	if err == nil {
		t.Fatal("expected error when getting config with invalid path, got none")
	}
}

func TestGetConfigInvalidPort(t *testing.T) {
	var configContent string
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
		t.Fatal("expected error when getting config with invalid port, got none")
	}
}

func TestGetConfigMergeError(t *testing.T) {
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
		t.Fatal("expected error when merging config, got none")
	}
}

func TestMergeWithDefaultNilConfig(t *testing.T) {
	logger := &mockDoubleSetLogger{}
	_, err := mergeWithDefault(nil, logger)
	if err != nil {
		t.Fatalf("failed to merge with default config: %v", err)
	}
}

func TestGetConfigInvalidHttpTimeoutOverride(t *testing.T) {
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
		t.Fatal("expected error when getting config with invalid http timeout override, got none")
	}
}

func TestLoadConfigFromFile(t *testing.T) {
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
		t.Fatalf("failed to load config: %v", err)
	}

	if config.Server.Port != 8080 {
		t.Errorf("expected port 8080, got %d", config.Server.Port)
	}
}

func TestEnvironmentVariableOverridesFileConfig(t *testing.T) {
	configContent := `
logstash:
  servers:
    - url: "http://original-url:9600"
server:
  port: 8080
logging:
  level: "info"
  format: "text"
`
	configFileName, err := createTemporaryConfigFile(configContent)
	if err != nil {
		t.Fatalf("failed to create temp config file: %v", err)
	}
	defer os.Remove(configFileName)

	cleanupPort := setEnvironmentVariable(envPort, "9090")
	defer cleanupPort()

	logger := &mockDoubleSetLogger{}
	config, err := GetConfig(configFileName, logger)
	if err != nil {
		t.Fatalf("failed to load config with environment override: %v", err)
	}

	if len(logger.Warnings) == 0 {
		t.Fatal("expected warnings when environment variables override file configs")
	}

	expectedPort := 8080
	if config.Server.Port != expectedPort {
		t.Errorf("expected port %d, got %d", expectedPort, config.Server.Port)
	}
}

func TestNonExistentConfigFileReturnsError(t *testing.T) {
	_, err := loadConfig("nonexistentconfig.yml")
	if err == nil {
		t.Fatal("expected error when loading from a non-existent file, got none")
	}
}

func TestInvalidYAMLConfigFileReturnsError(t *testing.T) {
	invalidContent := "invalid: [yaml: format"
	fileName, err := createTemporaryConfigFile(invalidContent)
	if err != nil {
		t.Fatalf("failed to create temp invalid config file: %v", err)
	}
	defer os.Remove(fileName)

	_, err = loadConfig(fileName)
	if err == nil {
		t.Fatal("expected error when loading invalid YAML, got none")
	}
}

func TestMergeWithDefaultConfig(t *testing.T) {
	config := &Config{}
	envCleanup := setEnvironmentVariable(envPort, "9090")
	defer envCleanup()

	logger := &mockDoubleSetLogger{}
	mergedConfig, err := mergeWithDefault(config, logger)
	if err != nil {
		t.Fatalf("failed to merge with default config: %v", err)
	}

	expectedPort, _ := strconv.Atoi(os.Getenv(envPort))
	if mergedConfig.Server.Port != expectedPort {
		t.Errorf("expected port to be %d after merge, got %d", expectedPort, mergedConfig.Server.Port)
	}
}

func TestConfigLoadAndMergeProcess(t *testing.T) {
	configContent := `
server:
  port: 8080
`
	configFileName, err := createTemporaryConfigFile(configContent)
	if err != nil {
		t.Fatalf("failed to create temp config file for load and merge test: %v", err)
	}
	defer os.Remove(configFileName)

	logger := &mockDoubleSetLogger{}
	config, err := GetConfig(configFileName, logger)
	if err != nil {
		t.Fatalf("failed to get config in load and merge process: %v", err)
	}

	if len(logger.Warnings) > 0 {
		t.Fatal("did not expect warnings for port as it is not set in both env and file")
	}

	expectedPort := 8080
	if config.Server.Port != expectedPort {
		t.Errorf("expected port %d from file, got %d", expectedPort, config.Server.Port)
	}
}
