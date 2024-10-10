package config

import (
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
		if config.Logstash.Servers[0].Host != "http://localhost:9601" {
			t.Errorf("expected URL to be %v, got %v", "http://localhost:9601", config.Logstash.Servers[0].Host)
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
				Servers: []*LogstashServer{
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
				Servers: []*LogstashServer{
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
				Servers: []*LogstashServer{
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
				Servers: []*LogstashServer{
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
				Servers: []*LogstashServer{
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
				Servers: []*LogstashServer{
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
				Servers: nil,
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
		if mergedConfig.Logstash.Servers[0].Host != defaultLogstashURL {
			t.Errorf("expected URL to be %v, got %v", defaultLogstashURL, mergedConfig.Logstash.Servers[0].Host)
		}
		if mergedConfig.Logstash.HttpTimeout != defaultHttpTimeout {
			t.Errorf("expected http timeout to be %v, got %v", defaultHttpTimeout, mergedConfig.Logstash.HttpTimeout)
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
		if mergedConfig.Logstash.Servers[0].Host != defaultLogstashURL {
			t.Errorf("expected URL to be %v, got %v", defaultLogstashURL, mergedConfig.Logstash.Servers[0].Host)
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
				Servers: []*LogstashServer{
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

		if mergedConfig.Logstash.Servers[0].Host != "http://localhost:9601" {
			t.Errorf("expected URL to be %v, got %v", "http://localhost:9601", mergedConfig.Logstash.Servers[0].Host)
		}

		if mergedConfig.Logstash.Servers[1].Host != "http://localhost:9602" {
			t.Errorf("expected URL to be %v, got %v", "http://localhost:9602", mergedConfig.Logstash.Servers[1].Host)
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
