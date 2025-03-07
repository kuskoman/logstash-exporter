package config

import (
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"time"

	"gopkg.in/yaml.v2"
)

const (
	defaultConfigLocation = "config.yml"
	defaultPort           = 9198
	defaultLogLevel       = "info"
	defaultLogFormat      = "text"
	defaultLogstashURL    = "http://localhost:9600"
	defaultHttpTimeout    = time.Second * 2
	defaultHttpInsecure   = false
)

var (
	ExporterConfigLocation = getEnvWithDefault("EXPORTER_CONFIG_LOCATION", defaultConfigLocation)
)

// LogstashInstance represents individual Logstash server configuration
type LogstashInstance struct {
	Host         string `yaml:"url"`
	HttpInsecure bool   `yaml:"httpInsecure"`
	Name         string `yaml:"name"`
}

// LogstashConfig holds the configuration for all Logstash instances
type LogstashConfig struct {
	// LegacyServers is a deprecated field, use Instances instead
	LegacyServers []*LogstashInstance `yaml:"servers"`

	Instances   []*LogstashInstance `yaml:"instances"`
	HttpTimeout time.Duration       `yaml:"httpTimeout"`
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
	Host      string `yaml:"host"`
	Port      int    `yaml:"port"`
	CertFile  string `yaml:"certFile"`
	KeyFile   string `yaml:"keyFile"`
	EnableSSL bool   `yaml:"enableSSL"`
}

// LoggingConfig represents the logging configuration
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// Config represents the overall configuration loaded from the YAML file
type Config struct {
	Logstash   LogstashConfig   `yaml:"logstash"`
	Server     ServerConfig     `yaml:"server"`
	Logging    LoggingConfig    `yaml:"logging"`
	Kubernetes KubernetesConfig `yaml:"kubernetes"`
}

func (config *Config) Equals(other *Config) bool {
	return reflect.DeepEqual(config, other)
}

// handleLegacyServersProperty handles the deprecated 'servers' property.
// This method will log a warning and append the legacy servers to the new 'instances' property.
func (config *Config) handleLegacyServersProperty() {
	if len(config.Logstash.LegacyServers) > 0 {
		slog.Warn("The 'servers' property is deprecated, please use 'instances' instead", "servers", fmt.Sprintf("%v", config.Logstash.LegacyServers))
		config.Logstash.Instances = append(config.Logstash.Instances, config.Logstash.LegacyServers...)
	}
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

	config.handleLegacyServersProperty()

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

	if config.Logging.Format == "" {
		slog.Debug("using default log format", "format", defaultLogLevel)
		config.Logging.Format = defaultLogFormat
	}

	if len(config.Logstash.Instances) == 0 {
		slog.Debug("using default logstash server", "url", defaultLogstashURL)
		config.Logstash.Instances = append(config.Logstash.Instances, &LogstashInstance{
			Host: defaultLogstashURL,
		})
	}

	if config.Logstash.HttpTimeout == 0 {
		slog.Debug("using default http timeout", "httpTimeout", defaultHttpTimeout)
		config.Logstash.HttpTimeout = defaultHttpTimeout
	}

	// Set default Kubernetes configuration
	defaultK8sConfig := DefaultKubernetesConfig()
	if config.Kubernetes.PodAnnotationPrefix == "" {
		config.Kubernetes.PodAnnotationPrefix = defaultK8sConfig.PodAnnotationPrefix
	}
	if config.Kubernetes.ResyncPeriod == 0 {
		config.Kubernetes.ResyncPeriod = defaultK8sConfig.ResyncPeriod
	}
	if config.Kubernetes.ScrapeInterval == 0 {
		config.Kubernetes.ScrapeInterval = defaultK8sConfig.ScrapeInterval
	}
	if config.Kubernetes.LogstashURLAnnotation == "" {
		config.Kubernetes.LogstashURLAnnotation = defaultK8sConfig.LogstashURLAnnotation
	}
	if config.Kubernetes.LogstashUsernameAnnotation == "" {
		config.Kubernetes.LogstashUsernameAnnotation = defaultK8sConfig.LogstashUsernameAnnotation
	}
	if config.Kubernetes.LogstashPasswordAnnotation == "" {
		config.Kubernetes.LogstashPasswordAnnotation = defaultK8sConfig.LogstashPasswordAnnotation
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
