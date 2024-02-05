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

// doubleSetLogger interface for logging double set value warnings
type doubleSetLogger interface {
	Warn(msg string, keysAndValues ...interface{})
}

// defaultDoubleSetLogger using slog for logging double set value warnings
type defaultDoubleSetLogger struct{}

func (l defaultDoubleSetLogger) Warn(propertyName string, keysAndValues ...interface{}) {
	msg := "value set in both environment and config file, using config file value"
	keysAndValues = append(keysAndValues, "propertyName", propertyName)
	slog.Warn(msg, keysAndValues...)
}

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

type environmentConfig struct {
	Port        *environmentMetadata
	LogLevel    *environmentMetadata
	LogFormat   *environmentMetadata
	LogstashURL *environmentMetadata
	HttpTimeout *environmentMetadata
}

func loadEnvironmentConfig() *environmentConfig {
	envConfig := &environmentConfig{
		Port:        &environmentMetadata{envName: envPort, defaultVal: defaultPort},
		LogLevel:    &environmentMetadata{envName: envLogLevel, defaultVal: defaultLogLevel},
		LogFormat:   &environmentMetadata{envName: envLogFormat, defaultVal: defaultLogFormat},
		LogstashURL: &environmentMetadata{envName: envLogstashURL, defaultVal: defaultLogstashURL},
		HttpTimeout: &environmentMetadata{envName: envHttpTimeout, defaultVal: defaultHttpTimeout},
	}

	envConfig.Port.Load()
	envConfig.LogLevel.Load()
	envConfig.LogFormat.Load()
	envConfig.LogstashURL.Load()
	envConfig.HttpTimeout.Load()

	return envConfig
}

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

// Refactored warning logic to return boolean instead of logging directly
func shouldWarnIfDoubleSetString(configFileValue string, envMetadata *environmentMetadata) bool {
	return configFileValue != "" && envMetadata.IsSet()
}

func shouldWarnIfDoubleSetInt(configFileValue int, envMetadata *environmentMetadata) bool {
	return configFileValue != 0 && envMetadata.IsSet()
}

func mergeWithDefault(config *Config, logger doubleSetLogger) (*Config, error) {
	if config == nil {
		config = &Config{}
	}

	envConfig := loadEnvironmentConfig()

	if shouldWarnIfDoubleSetInt(config.Server.Port, envConfig.Port) {
		logger.Warn("port", "configFileValue", config.Server.Port, "envValue", envConfig.Port.Value())
	}

	if config.Server.Port == 0 {
		portString := envConfig.Port.Value()
		port, err := strconv.Atoi(portString)
		if err != nil {
			return nil, err
		}

		config.Server.Port = port
	}

	if shouldWarnIfDoubleSetString(config.Logging.Level, envConfig.LogLevel) {
		logger.Warn("log level", "configFileValue", config.Logging.Level, "envValue", envConfig.LogLevel.Value())
	}
	if config.Logging.Level == "" {
		config.Logging.Level = envConfig.LogLevel.Value()
	}

	if shouldWarnIfDoubleSetString(config.Logging.Format, envConfig.LogFormat) {
		logger.Warn("log format", "configFileValue", config.Logging.Format, "envValue", envConfig.LogFormat.Value())
	}
	if config.Logging.Format == "" {
		config.Logging.Format = envConfig.LogFormat.Value()
	}

	if config.Logstash.Servers == nil || len(config.Logstash.Servers) == 0 {
		config.Logstash.Servers = append(config.Logstash.Servers, &LogstashServer{
			Host: envConfig.LogstashURL.Value(),
		})
	} else if shouldWarnIfDoubleSetString(config.Logstash.Servers[0].Host, envConfig.LogstashURL) {
		logger.Warn("logstash URL", "configFileValue", config.Logstash.Servers[0].Host, "envValue", envConfig.LogstashURL.Value())
	}

	if shouldWarnIfDoubleSetString(config.Logstash.HttpTimeout.String(), envConfig.HttpTimeout) {
		logger.Warn("logstash http timeout", "configFileValue", config.Logstash.HttpTimeout.String(), "envValue", envConfig.HttpTimeout.Value())
	}
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
func GetConfig(location string, logger doubleSetLogger) (*Config, error) {
	if logger == nil {
		logger = defaultDoubleSetLogger{} // Use default logger if none provided
	}

	config, err := loadConfig(location)
	if err != nil {
		return nil, err
	}

	mergedConfig, err := mergeWithDefault(config, logger)
	if err != nil {
		return nil, err
	}

	return mergedConfig, nil
}
