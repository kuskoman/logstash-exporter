package config

import (
	"reflect"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	t.Parallel()
	t.Run("loads valid config", func(t *testing.T) {
		t.Parallel()

		location := "../fixtures/valid_config.yml"
		config, err := loadConfig(location)

		if err != nil {
			t.Fatalf("got an error %v", err)
		}
		if config == nil {
			t.Fatal("expected config to be non-nil")
		}
		if config.Logstash.Servers[0].URL != "http://localhost:9601" {
			t.Errorf("expected URL to be %v, got %v", "http://localhost:9601", config.Logstash.Servers[0].URL)
		}
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		t.Parallel()

		location := "../fixtures/non_existent.yml"
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

		location := "../fixtures/invalid_config.toml"
		config, err := loadConfig(location)

		if err == nil {
			t.Fatal("expected error, got none")
		}

		if config != nil {
			t.Fatal("expected config to be nil")
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
		if mergedConfig.Logstash.Servers[0].URL != defaultLogstashURL {
			t.Errorf("expected URL to be %v, got %v", defaultLogstashURL, mergedConfig.Logstash.Servers[0].URL)
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
		if mergedConfig.Logstash.Servers[0].URL != defaultLogstashURL {
			t.Errorf("expected URL to be %v, got %v", defaultLogstashURL, mergedConfig.Logstash.Servers[0].URL)
		}
	})

	t.Run("merge with non-empty config", func(t *testing.T) {
		t.Parallel()

		config := &Config{
			Server: ServerConfig{
				Port: 1234,
			},
			Logging: LoggingConfig{
				Level: "debug",
			},
			Logstash: LogstashConfig{
				Servers: []LogstashServer{
					{URL: "http://localhost:9601"},
					{URL: "http://localhost:9602"},
				},
			},
		}

		mergedConfig := mergeWithDefault(config)

		if mergedConfig.Server.Port != 1234 {
			t.Errorf("expected port to be %v, got %v", 1234, mergedConfig.Server.Port)
		}

		if mergedConfig.Logging.Level != "debug" {
			t.Errorf("expected level to be %v, got %v", "debug", mergedConfig.Logging.Level)
		}

		if mergedConfig.Logstash.Servers[0].URL != "http://localhost:9601" {
			t.Errorf("expected URL to be %v, got %v", "http://localhost:9601", mergedConfig.Logstash.Servers[0].URL)
		}

		if mergedConfig.Logstash.Servers[1].URL != "http://localhost:9602" {
			t.Errorf("expected URL to be %v, got %v", "http://localhost:9602", mergedConfig.Logstash.Servers[1].URL)
		}
	})
}

func TestGetConfig(t *testing.T) {
	t.Run("returns valid config", func(t *testing.T) {

		location := "../fixtures/valid_config.yml"
		config, err := GetConfig(location)

		if err != nil {
			t.Fatalf("got an error %v", err)
		}
		if config == nil {
			t.Fatal("expected config to be non-nil")
		}
	})

	t.Run("returns error for invalid config", func(t *testing.T) {
		location := "../fixtures/invalid_config.yml"
		config, err := GetConfig(location)

		if err == nil {
			t.Fatal("expected error, got none")
		}
		if config != nil {
			t.Fatal("expected config to be nil")
		}
	})
}

func TestGetLogstashUrls(t *testing.T) {
	t.Run("gets Logstash URLs correctly", func(t *testing.T) {
		config := &Config{
			Logstash: LogstashConfig{
				Servers: []LogstashServer{{URL: "http://localhost:9601"}},
			},
		}

		urls := config.GetLogstashUrls()
		expectedUrls := []string{"http://localhost:9601"}

		if !reflect.DeepEqual(urls, expectedUrls) {
			t.Errorf("expected urls to be %v, got %v", expectedUrls, urls)
		}
	})
}
