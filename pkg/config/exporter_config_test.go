package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	t.Parallel()
	t.Run("loads valid config", func(t *testing.T) {
		t.Parallel()

		location := "../../fixtures/valid_config.yml"
		config, err := loadConfig(location)

		if err != nil {
			t.Fatalf("got an error %v", err)
		}
		if config == nil {
			t.Fatal("expected config to be non-nil")
		}
		if config.Logstash.Instances[0].Host != "http://localhost:9601" {
			t.Errorf("expected URL to be %v, got %v", "http://localhost:9601", config.Logstash.Instances[0].Host)
		}
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		t.Parallel()

		location := "../../fixtures/non_existent.yml"
		config, err := loadConfig(location)

		if err == nil {
			t.Fatal("expected error, got none")
		}
		if config != nil {
			t.Fatal("expected config to be nil")
		}
	})

	t.Run("returns error for invalid config", func(t *testing.T) {
		t.Parallel()

		location := "../../fixtures/invalid_config.toml"
		config, err := loadConfig(location)

		if err == nil {
			t.Fatal("expected error, got none")
		}

		if config != nil {
			t.Fatal("expected config to be nil")
		}
	})
}

func TestConfigEquals(t *testing.T) {
	t.Parallel()

	t.Run("returns true for equal configs", func(t *testing.T) {
		config1 := &Config{
			Server: ServerConfig{
				Port: 1234,
			},
			Logging: LoggingConfig{
				Level:  "debug",
				Format: "json",
			},
			Logstash: LogstashConfig{
				Instances: []*LogstashInstance{
					{Host: "http://localhost:9601"},
					{Host: "http://localhost:9602"},
				},
			},
		}

		config2 := &Config{
			Server: ServerConfig{
				Port: 1234,
			},
			Logging: LoggingConfig{
				Level:  "debug",
				Format: "json",
			},
			Logstash: LogstashConfig{
				Instances: []*LogstashInstance{
					{Host: "http://localhost:9601"},
					{Host: "http://localhost:9602"},
				},
			},
		}

		if !config1.Equals(config2) {
			t.Error("expected configs to be equal")
		}
	})

	t.Run("returns false for unequal configs (when nested fields differ)", func(t *testing.T) {
		config1 := &Config{
			Server: ServerConfig{
				Port: 1234,
			},
			Logging: LoggingConfig{
				Level:  "debug",
				Format: "json",
			},
			Logstash: LogstashConfig{
				Instances: []*LogstashInstance{
					{Host: "http://localhost:9601"},
					{Host: "http://localhost:9603"},
				},
			},
		}

		config2 := &Config{
			Server: ServerConfig{
				Port: 1234,
			},
			Logging: LoggingConfig{
				Level:  "debug",
				Format: "json",
			},
			Logstash: LogstashConfig{
				Instances: []*LogstashInstance{
					{Host: "http://localhost:9601"},
					{Host: "http://localhost:9602"},
				},
			},
		}

		if config1.Equals(config2) {
			t.Error("expected configs to be unequal")
		}
	})

	t.Run("returns false for nil config", func(t *testing.T) {
		config1 := &Config{
			Server: ServerConfig{
				Port: 1234,
			},
			Logging: LoggingConfig{
				Level:  "debug",
				Format: "json",
			},
			Logstash: LogstashConfig{
				Instances: []*LogstashInstance{
					{Host: "http://localhost:9601"},
					{Host: "http://localhost:9602"},
				},
			},
		}

		if config1.Equals(nil) {
			t.Error("expected configs to be unequal")
		}
	})

	t.Run("returns false for unequal configs (when one field is empty)", func(t *testing.T) {
		config1 := &Config{
			Server: ServerConfig{
				Port: 1234,
			},
			Logging: LoggingConfig{
				Level:  "debug",
				Format: "json",
			},
			Logstash: LogstashConfig{
				Instances: []*LogstashInstance{
					{Host: "http://localhost:9601"},
					{Host: "http://localhost:9602"},
				},
			},
		}

		config2 := &Config{
			Server: ServerConfig{
				Port: 1234,
			},
			Logging: LoggingConfig{
				Level:  "debug",
				Format: "json",
			},
			Logstash: LogstashConfig{
				Instances: nil,
			},
		}

		if config1.Equals(config2) {
			t.Error("expected configs to be unequal")
		}
	})
}

func TestMergeWithDefault(t *testing.T) {
	t.Parallel()

	t.Run("merge with empty config", func(t *testing.T) {
		t.Parallel()

		config := &Config{}
		mergedConfig := mergeWithDefault(config)

		if mergedConfig.Server.Port != defaultPort {
			t.Errorf("expected port to be %v, got %v", defaultPort, mergedConfig.Server.Port)
		}
		if mergedConfig.Logging.Level != defaultLogLevel {
			t.Errorf("expected level to be %v, got %v", defaultLogLevel, mergedConfig.Logging.Level)
		}
		if mergedConfig.Logging.Format != defaultLogFormat {
			t.Errorf("expected format to be %v, got %v", defaultLogFormat, mergedConfig.Logging.Format)
		}
		if mergedConfig.Logstash.Instances[0].Host != defaultLogstashURL {
			t.Errorf("expected URL to be %v, got %v", defaultLogstashURL, mergedConfig.Logstash.Instances[0].Host)
		}
		if mergedConfig.Logstash.HttpTimeout != defaultHttpTimeout {
			t.Errorf("expected http timeout to be %v, got %v", defaultHttpTimeout, mergedConfig.Logstash.HttpTimeout)
		}
	})

	t.Run("does not add the default instance in kubernetes controller mode", func(t *testing.T) {
		t.Parallel()

		mergedConfig := mergeWithDefault(&Config{Kubernetes: KubernetesConfig{Enabled: true}})

		if len(mergedConfig.Logstash.Instances) != 0 {
			t.Errorf("expected no static instances, got %v", mergedConfig.Logstash.Instances)
		}
	})

	t.Run("keeps configured instances in kubernetes controller mode", func(t *testing.T) {
		t.Parallel()

		instance := &LogstashInstance{Host: "http://logstash:9600"}
		mergedConfig := mergeWithDefault(&Config{
			Kubernetes: KubernetesConfig{Enabled: true},
			Logstash:   LogstashConfig{Instances: []*LogstashInstance{instance}},
		})

		if len(mergedConfig.Logstash.Instances) != 1 || mergedConfig.Logstash.Instances[0] != instance {
			t.Errorf("expected the configured instance to be kept, got %v", mergedConfig.Logstash.Instances)
		}
	})

	t.Run("merge with nil config", func(t *testing.T) {
		t.Parallel()

		mergedConfig := mergeWithDefault(nil)

		if mergedConfig.Server.Port != defaultPort {
			t.Errorf("expected port to be %v, got %v", defaultPort, mergedConfig.Server.Port)
		}
		if mergedConfig.Logging.Level != defaultLogLevel {
			t.Errorf("expected level to be %v, got %v", defaultLogLevel, mergedConfig.Logging.Level)
		}
		if mergedConfig.Logging.Format != defaultLogFormat {
			t.Errorf("expected format to be %v, got %v", defaultLogFormat, mergedConfig.Logging.Format)
		}
		if mergedConfig.Logstash.Instances[0].Host != defaultLogstashURL {
			t.Errorf("expected URL to be %v, got %v", defaultLogstashURL, mergedConfig.Logstash.Instances[0].Host)
		}
		if mergedConfig.Logstash.HttpTimeout != defaultHttpTimeout {
			t.Errorf("expected http timeout to be %v, got %v", defaultHttpTimeout, mergedConfig.Logstash.HttpTimeout)
		}
	})

	t.Run("merge with non-empty config", func(t *testing.T) {
		t.Parallel()

		config := &Config{
			Server: ServerConfig{
				Port: 1234,
			},
			Logging: LoggingConfig{
				Level:  "debug",
				Format: "json",
			},
			Logstash: LogstashConfig{
				Instances: []*LogstashInstance{
					{Host: "http://localhost:9601"},
					{Host: "http://localhost:9602"},
				},
				HttpTimeout: 3 * time.Second,
			},
		}

		mergedConfig := mergeWithDefault(config)

		if mergedConfig.Server.Port != 1234 {
			t.Errorf("expected port to be %v, got %v", 1234, mergedConfig.Server.Port)
		}

		if mergedConfig.Logging.Level != "debug" {
			t.Errorf("expected level to be %v, got %v", "debug", mergedConfig.Logging.Level)
		}

		if mergedConfig.Logging.Format != "json" {
			t.Errorf("expected format to be %v, got %v", "json", mergedConfig.Logging.Format)
		}

		if mergedConfig.Logstash.Instances[0].Host != "http://localhost:9601" {
			t.Errorf("expected URL to be %v, got %v", "http://localhost:9601", mergedConfig.Logstash.Instances[0].Host)
		}

		if mergedConfig.Logstash.Instances[1].Host != "http://localhost:9602" {
			t.Errorf("expected URL to be %v, got %v", "http://localhost:9602", mergedConfig.Logstash.Instances[1].Host)
		}
		if mergedConfig.Logstash.HttpTimeout != 3*time.Second {
			t.Errorf("expected http timeout to be %v, got %v", 3*time.Second, mergedConfig.Logstash.HttpTimeout)
		}
	})
}

func TestGetConfig(t *testing.T) {
	t.Run("returns valid config", func(t *testing.T) {

		location := "../../fixtures/valid_config.yml"
		config, err := GetConfig(location)

		if err != nil {
			t.Fatalf("got an error %v", err)
		}
		if config == nil {
			t.Fatal("expected config to be non-nil")
		}
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		location := "../../fixtures/non_existent.yml"
		config, err := GetConfig(location)

		if err == nil {
			t.Fatal("expected error, got none")
		}

		if config != nil {
			t.Fatal("expected config to be nil")
		}
	})

	t.Run("ignores the removed max_connections setting", func(t *testing.T) {
		location := filepath.Join(t.TempDir(), "config.yml")
		content := "server:\n  port: 9300\n  max_connections: 512\n"
		if err := os.WriteFile(location, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		config, err := GetConfig(location)
		if err != nil {
			t.Fatalf("expected configs that still set max_connections to load, got %v", err)
		}
		if config.Server.Port != 9300 {
			t.Errorf("expected port 9300, got %d", config.Server.Port)
		}
	})

	t.Run("returns error for a config that fails validation", func(t *testing.T) {
		location := filepath.Join(t.TempDir(), "config.yml")
		content := "logstash:\n  instances:\n    - url: http://logstash:9600\n      basic_auth:\n        username: monitoring\n"
		if err := os.WriteFile(location, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		config, err := GetConfig(location)
		if err == nil || !strings.Contains(err.Error(), "invalid Logstash instance 0") {
			t.Errorf("expected a validation error for instance 0, got %v", err)
		}
		if config != nil {
			t.Error("expected config to be nil")
		}
	})

	t.Run("returns error for invalid config", func(t *testing.T) {
		location := "../../fixtures/invalid_config.toml"
		config, err := GetConfig(location)

		if err == nil {
			t.Fatal("expected error, got none")
		}
		if config != nil {
			t.Fatal("expected config to be nil")
		}
	})
}
