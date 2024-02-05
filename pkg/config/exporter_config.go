package config

import (
	"log/slog"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v2"
)

const (
	envConfigLocation = "EXPORTER_CONFIG_LOCATION"
	envPort           = "PORT"
	envLogLevel       = "LOG_LEVEL"
	envLogFormat      = "LOG_FORMAT"
	envLogstashURL    = "LOGSTASH_URL"
	envHttpTimeout    = "HTTP_TIMEOUT"

	defaultConfigLocation = "config.yml"
	defaultPort           = "9198"
	defaultLogLevel       = "info"
	defaultLogFormat      = "text"
	defaultLogstashURL    = "http://localhost:9600"
	defaultHttpTimeout    = "2s"
)

var (
	ExporterConfigLocation = getEnvWithDefault("EXPORTER_CONFIG_LOCATION", defaultConfigLocation)
)

// LogstashServer represents individual Logstash server configuration
type LogstashServer struct {
	Host string `yaml:"url"`
}

// LogstashConfig holds the configuration for all Logstash servers
type LogstashConfig struct {
	Servers     []*LogstashServer `yaml:"servers"`
	HttpTimeout time.Duration     `yaml:"httpTimeout"`
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
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// Config represents the overall configuration loaded from the YAML file
type Config struct {
	Logstash LogstashConfig `yaml:"logstash"`
	Server   ServerConfig   `yaml:"server"`
	Logging  LoggingConfig  `yaml:"logging"`
}

type environmentMetadata struct {
	envName    string
	defaultVal string
	value      string
	isSetInEnv bool
}

// IsSet returns true if the value is set by the user in the environment
func (e *environmentMetadata) IsSet() bool {
	return e.isSetInEnv
}

// Value returns the value of the environment variable, or the default value if not set
func (e *environmentMetadata) Value() string {
	return e.value
}

// Load loads the value from the environment variable, or sets the default value if not set
func (e *environmentMetadata) Load() {
	envVal, isSet := os.LookupEnv(e.envName)
	if isSet {
		e.value = envVal
		e.isSetInEnv = true
	} else {
		e.value = e.defaultVal
	}
}

// environmentConfig represents the configuration loaded from the environment variables.
// The configuration is used when properties are not set in the YAML file.
// In case the properties are set in the YAML file, the environment variables are ignored.
type environmentConfig struct {
	Port        *environmentMetadata
	LogLevel    *environmentMetadata
	LogFormat   *environmentMetadata
	LogstashURL *environmentMetadata
	HttpTimeout *environmentMetadata
}

func loadEnvironmentConfig() *environmentConfig {
	envConfig := &environmentConfig{
		Port: &environmentMetadata{
			envName:    envPort,
			defaultVal: defaultPort,
		},
		LogLevel: &environmentMetadata{
			envName:    envLogLevel,
			defaultVal: defaultLogLevel,
		},
		LogFormat: &environmentMetadata{
			envName:    envLogFormat,
			defaultVal: defaultLogFormat,
		},
		LogstashURL: &environmentMetadata{
			envName:    envLogstashURL,
			defaultVal: defaultLogstashURL,
		},
		HttpTimeout: &environmentMetadata{
			envName:    envHttpTimeout,
			defaultVal: defaultHttpTimeout,
		},
	}

	envConfig.Port.Load()
	envConfig.LogLevel.Load()
	envConfig.LogFormat.Load()
	envConfig.LogstashURL.Load()
	envConfig.HttpTimeout.Load()

	return envConfig
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

// printWarningIfDoubleSetString prints a warning if the value is set in both the environment and the config file.
func printWarningIfDoubleSetString(name, configFileValue string, envMetadata *environmentMetadata) {
	if configFileValue != "" && envMetadata.IsSet() {
		slog.Warn("value set in both environment and config file, using config file value", "configFileValue", configFileValue, "envValue", envMetadata.Value())
	}
}

// printWarningIfDoubleSetInt prints a warning if the value is set in both the environment and the config file.
func printWarningIfDoubleSetInt(name string, configFileValue int, envMetadata *environmentMetadata) {
	if configFileValue != 0 && envMetadata.IsSet() {
		slog.Warn("value set in both environment and config file, using config file value", "configFileValue", configFileValue, "envValue", envMetadata.Value())
	}
}

// mergeWithDefault merges the loaded configuration with the default configuration values.
func mergeWithDefault(config *Config) (*Config, error) {
	if config == nil {
		config = &Config{}
	}

	envConfig := loadEnvironmentConfig()

	printWarningIfDoubleSetInt("port", config.Server.Port, envConfig.Port)

	if config.Server.Port == 0 {
		portString := envConfig.Port.Value()
		port, err := strconv.Atoi(portString)
		if err != nil {
			return nil, err
		}

		config.Server.Port = port
	}

	printWarningIfDoubleSetString("log level", config.Logging.Level, envConfig.LogLevel)
	if config.Logging.Level == "" {
		config.Logging.Level = envConfig.LogLevel.Value()
	}

	printWarningIfDoubleSetString("log format", config.Logging.Format, envConfig.LogFormat)
	if config.Logging.Format == "" {
		config.Logging.Format = envConfig.LogFormat.Value()
	}

	if config.Logstash.Servers == nil || len(config.Logstash.Servers) == 0 {
		config.Logstash.Servers = append(config.Logstash.Servers, &LogstashServer{
			Host: envConfig.LogstashURL.Value(),
		})
	} else if envConfig.LogstashURL.IsSet() {
		printWarningIfDoubleSetString("logstash url", config.Logstash.Servers[0].Host, envConfig.LogstashURL)
	}

	printWarningIfDoubleSetString("http timeout", config.Logstash.HttpTimeout.String(), envConfig.HttpTimeout)
	if config.Logstash.HttpTimeout == 0 {
		httpTimeout, err := time.ParseDuration(envConfig.HttpTimeout.Value())
		if err != nil {
			return nil, err
		}

		config.Logstash.HttpTimeout = httpTimeout
	}

	return config, nil
}

// GetConfig combines loadConfig and mergeWithDefault to get the final configuration.
func GetConfig(location string) (*Config, error) {
	config, err := loadConfig(location)
	if err != nil {
		return nil, err
	}

	mergedConfig, err := mergeWithDefault(config)

	if err != nil {
		return nil, err
	}

	return mergedConfig, nil
}
