package config

import (
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"strings"
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
	Host string `yaml:"url"`
	Name string `yaml:"name"`

	// TLS configuration for the HTTP client
	TLSConfig *TLSClientConfig `yaml:"tls_config,omitempty"`

	// Basic authentication for the HTTP client
	BasicAuth *ClientAuthConfig `yaml:"basic_auth,omitempty"`
}

// TLSClientConfig configures TLS for the HTTP client connecting to Logstash.
type TLSClientConfig struct {
	// CAFile is the path to the certificate authority file for custom certificates.
	CAFile string `yaml:"ca_file,omitempty"`

	// ServerName is used to verify the hostname on the certificate.
	ServerName string `yaml:"server_name,omitempty"`

	// InsecureSkipVerify disables verification of the certificate.
	InsecureSkipVerify bool `yaml:"insecure_skip_verify,omitempty"`
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
	Host string `yaml:"host"`
	Port int    `yaml:"port"`

	// TLS Configuration
	TLSConfig *TLSServerConfig `yaml:"tls_server_config,omitempty"`

	// Basic authentication configuration
	BasicAuth *BasicAuthConfig `yaml:"basic_auth,omitempty"`

	// ReadTimeout defines the maximum duration for reading the entire request.
	ReadTimeout int `yaml:"read_timeout_seconds,omitempty"`

	// WriteTimeout defines the maximum duration before timing out writes of the response.
	WriteTimeout int `yaml:"write_timeout_seconds,omitempty"`

	// MaxConnections defines the maximum number of simultaneous connections.
	MaxConnections int `yaml:"max_connections,omitempty"`
}

// TLSServerConfig configures TLS for the server.
// This is similar to Prometheus exporter-toolkit's web configuration,
// but implemented independently as exporter-toolkit is not considered stable.
type TLSServerConfig struct {
	// CertFile is the path to the certificate file.
	CertFile string `yaml:"cert_file,omitempty"`

	// KeyFile is the path to the key file.
	KeyFile string `yaml:"key_file,omitempty"`

	// ClientAuth configures client authentication policy.
	// One of: "NoClientCert", "RequestClientCert", "RequireAnyClientCert",
	// "VerifyClientCertIfGiven", "RequireAndVerifyClientCert"
	ClientAuth string `yaml:"client_auth_type,omitempty"`

	// ClientCAs is the path to the CA certificates file used for client authentication.
	ClientCA string `yaml:"client_ca_file,omitempty"`

	// MinVersion is the minimum TLS version.
	// One of: "TLS10", "TLS11", "TLS12", "TLS13"
	MinVersion string `yaml:"min_version,omitempty"`

	// MaxVersion is the maximum TLS version.
	// One of: "TLS10", "TLS11", "TLS12", "TLS13"
	MaxVersion string `yaml:"max_version,omitempty"`

	// CipherSuites is the list of supported cipher suites.
	CipherSuites []string `yaml:"cipher_suites,omitempty"`

	// CurvePreferences is the list of supported curve preferences.
	CurvePreferences []string `yaml:"curve_preferences,omitempty"`
}

// BasicAuthConfig configures basic authentication for the server.
// This supports multiple users for server authentication.
type BasicAuthConfig struct {
	// Users is a map of username to password.
	Users map[string]string `yaml:"users,omitempty"`

	// UsersFile is the path to a file containing a map of username to password.
	// The file should be in the format: username:password
	// with one user per line.
	UsersFile string `yaml:"users_file,omitempty"`
}

// ClientAuthConfig configures basic authentication for connecting to Logstash.
// This only supports a single user/password.
type ClientAuthConfig struct {
	// Username for basic authentication.
	Username string `yaml:"username"`

	// Password for basic authentication.
	Password string `yaml:"password"`

	// PasswordFile is the path to a file containing the password.
	// This is mutually exclusive with Password.
	PasswordFile string `yaml:"password_file,omitempty"`
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

// GetUsers returns a map of username to password for basic authentication.
func (c *BasicAuthConfig) GetUsers() (map[string]string, error) {
	// If Users map is provided, use it
	if len(c.Users) > 0 {
		return c.Users, nil
	}

	// If UsersFile is provided, read it
	if c.UsersFile != "" {
		users, err := readUsersFile(c.UsersFile)
		if err != nil {
			return nil, err
		}
		return users, nil
	}

	return nil, fmt.Errorf("no authentication configuration provided")
}

// readUsersFile reads a file containing username:password pairs.
func readUsersFile(filepath string) (map[string]string, error) {
	content, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read users file: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	users := make(map[string]string)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid line in users file: %s", line)
		}

		username := strings.TrimSpace(parts[0])
		password := strings.TrimSpace(parts[1])
		users[username] = password
	}

	return users, nil
}

// ValidateBasicAuth validates the basic authentication configuration.
func (c *BasicAuthConfig) ValidateBasicAuth() error {
	if c.UsersFile != "" && len(c.Users) > 0 {
		return fmt.Errorf("users and users_file are mutually exclusive")
	}

	if len(c.Users) == 0 && c.UsersFile == "" {
		return fmt.Errorf("no authentication configuration provided")
	}

	return nil
}

// ValidateServerTLS validates the server TLS configuration.
func (c *ServerConfig) ValidateServerTLS() error {
	// Check TLS configuration
	if c.TLSConfig != nil {
		if c.TLSConfig.CertFile == "" {
			return fmt.Errorf("cert_file must be specified when TLS is enabled")
		}
		if c.TLSConfig.KeyFile == "" {
			return fmt.Errorf("key_file must be specified when TLS is enabled")
		}
	}

	// Check basic auth configuration
	if c.BasicAuth != nil {
		return c.BasicAuth.ValidateBasicAuth()
	}

	return nil
}

// ValidateClientTLS validates the client TLS configuration for a Logstash instance.
func (instance *LogstashInstance) ValidateClientTLS() error {
	if instance.TLSConfig != nil {
		// If CAFile is specified, make sure it exists
		if instance.TLSConfig.CAFile != "" {
			if _, err := os.Stat(instance.TLSConfig.CAFile); os.IsNotExist(err) {
				return fmt.Errorf("CA file %s does not exist", instance.TLSConfig.CAFile)
			}
		}
	}

	// Check basic auth configuration
	if instance.BasicAuth != nil {
		return instance.BasicAuth.ValidateClientAuth()
	}

	return nil
}

// Validate validates the entire configuration.
func (config *Config) Validate() error {
	// Validate server TLS configuration
	if err := config.Server.ValidateServerTLS(); err != nil {
		return fmt.Errorf("invalid server TLS configuration: %w", err)
	}

	// Validate each Logstash instance
	for i, instance := range config.Logstash.Instances {
		if err := instance.ValidateClientTLS(); err != nil {
			return fmt.Errorf("invalid Logstash instance %d TLS configuration: %w", i, err)
		}
	}

	return nil
}

// GetPassword returns the password for basic authentication.
func (c *ClientAuthConfig) GetPassword() (string, error) {
	if c.Password != "" {
		return c.Password, nil
	}

	if c.PasswordFile != "" {
		content, err := os.ReadFile(c.PasswordFile)
		if err != nil {
			return "", fmt.Errorf("failed to read password file: %w", err)
		}
		return string(content), nil
	}

	return "", fmt.Errorf("neither password nor password_file specified")
}

// ValidateClientAuth validates the client authentication configuration.
func (c *ClientAuthConfig) ValidateClientAuth() error {
	if c.Username == "" {
		return fmt.Errorf("username must be specified when basic auth is enabled")
	}

	if c.Password == "" && c.PasswordFile == "" {
		return fmt.Errorf("either password or password_file must be specified when basic auth is enabled")
	}

	if c.Password != "" && c.PasswordFile != "" {
		return fmt.Errorf("password and password_file are mutually exclusive")
	}

	return nil
}
