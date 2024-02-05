package config

import (
	"os"
	"strconv"
	"testing"
)

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

func TestGetConfigInvalidPath(t *testing.T) {
	_, err := GetConfig("invalidpath")
	if err == nil {
		t.Fatal("expected error when getting config with invalid path, got none")
	}
}

func TestGetConfigInvalidPort(t *testing.T) {
	configContent := `
server:
  port: "0"
`
	configFileName, err := createTemporaryConfigFile(configContent)
	if err != nil {
		t.Fatalf("failed to create temp config file: %v", err)
	}

	clearEnvPort := setEnvironmentVariable(envPort, "0")
	defer clearEnvPort()

	_, err = GetConfig(configFileName)
	if err == nil {
		t.Fatal("expected error when getting config with invalid port, got none")
	}
}

func TestGetConfigMergeError(t *testing.T) {
	var configContent string

	configFileName, err := createTemporaryConfigFile(configContent)
	if err != nil {
		t.Fatalf("failed to create temp config file: %v", err)
	}

	clearEnvPort := setEnvironmentVariable(envPort, "invalidport")
	defer clearEnvPort()

	_, err = GetConfig(configFileName)
	if err == nil {
		t.Fatal("expected error when merging config, got none")
	}
}

func TestMergeWithDefaultNilConfig(t *testing.T) {
	_, err := mergeWithDefault(nil)
	if err != nil {
		t.Fatalf("failed to merge with default config: %v", err)
	}
}

func TestGetConfigInvalidHttpTimeoutOverride(t *testing.T) {
	var configContent string

	configFileName, err := createTemporaryConfigFile(configContent)
	if err != nil {
		t.Fatalf("failed to create temp config file: %v", err)
	}

	clearEnvHttpTimeout := setEnvironmentVariable(envHttpTimeout, "invalidtimeout")
	defer clearEnvHttpTimeout()

	_, err = GetConfig(configFileName)
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
`
	configFileName, err := createTemporaryConfigFile(configContent)
	if err != nil {
		t.Fatalf("failed to create temp config file: %v", err)
	}
	defer os.Remove(configFileName)

	envPortValue := 9090
	envPortValueStr := strconv.Itoa(envPortValue)
	cleanupPort := setEnvironmentVariable(envPort, envPortValueStr)
	defer cleanupPort()

	config, err := GetConfig(configFileName)
	if err != nil {
		t.Fatalf("failed to load config with environment override: %v", err)
	}

	expectedPort := 8080
	if config.Server.Port != expectedPort {
		t.Errorf("expected port not to be overridden to %s, got %d", envPortValueStr, config.Server.Port)
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

	mergedConfig, err := mergeWithDefault(config)
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

	cleanupPort := setEnvironmentVariable(envPort, "8081")
	defer cleanupPort()

	config, err := GetConfig(configFileName)
	if err != nil {
		t.Fatalf("failed to get config in load and merge process: %v", err)
	}

	expectedPort := 8080
	if config.Server.Port != expectedPort {
		t.Errorf("expected port %d from file, got %d", expectedPort, config.Server.Port)
	}
}
