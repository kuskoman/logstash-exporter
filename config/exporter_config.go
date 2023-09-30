package config

import (
	"log/slog"
	"os"

	"gopkg.in/yaml.v2"
)

const (
	defaultConfigLocation = "config.yml"
	defaultPort           = 9198
	defaultLogLevel       = "info"
	defaultLogstashURL    = "http://localhost:9600"
)

var (
	ExporterConfigLocation = getEnvWithDefault("EXPORTER_CONFIG_LOCATION", defaultConfigLocation)
)

// LogstashServer represents individual Logstash server configuration
type LogstashServer struct {
	URL string `yaml:"url"`
}

// LogstashConfig holds the configuration for all Logstash servers
type LogstashConfig struct {
	Servers []LogstashServer `yaml:"servers"`
}

// ServerConfig represents the server configuration
type ServerConfig struct {
	// Host is the host the exporter will listen on.
	// Defaults to an empty string, which will listen on all interfaces
	// Can be overridden by setting the HOST environment variable
	// For windows, use "localhost", because an empty string will not work
	// with the default windows firewall configuration.
	// Alternatively you can change the firewall configuration to allow
	// connections to the port from all interfaces.
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// LoggingConfig represents the logging configuration
type LoggingConfig struct {
	Level string `yaml:"level"`
}

// Config represents the overall configuration loaded from the YAML file
type Config struct {
	Logstash LogstashConfig `yaml:"logstash"`
	Server   ServerConfig   `yaml:"server"`
	Logging  LoggingConfig  `yaml:"logging"`
}

// loadConfig loads the configuration from the YAML file.
func loadConfig(location string) (*Config, error) {
	data, err := os.ReadFile(location)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// mergeWithDefault merges the loaded configuration with the default configuration values.
func mergeWithDefault(config *Config) *Config {
	if config == nil {
		config = &Config{}
	}

	if config.Server.Port == 0 {
		slog.Debug("using default port", "port", defaultPort)
		config.Server.Port = defaultPort
	}

	if config.Logging.Level == "" {
		slog.Debug("using default log level", "level", defaultLogLevel)
		config.Logging.Level = defaultLogLevel
	}

	if len(config.Logstash.Servers) == 0 {
		slog.Debug("using default logstash server", "url", defaultLogstashURL)
		config.Logstash.Servers = append(config.Logstash.Servers, LogstashServer{
			URL: defaultLogstashURL,
		})
	}

	return config
}

// GetConfig combines loadConfig and mergeWithDefault to get the final configuration.
func GetConfig(location string) (*Config, error) {
	config, err := loadConfig(location)
	if err != nil {
		return nil, err
	}

	mergedConfig := mergeWithDefault(config)
	return mergedConfig, nil
}

func (cfg *Config) GetLogstashUrls() []string {
	urls := make([]string, len(cfg.Logstash.Servers))
	for i, server := range cfg.Logstash.Servers {
		urls[i] = server.URL
	}
	return urls
}
