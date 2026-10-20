package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v2"
)

// Instance describes a Logstash instance in the exporter configuration
type Instance struct {
	URL                string
	Name               string
	Username           string
	Password           string
	CAFile             string
	InsecureSkipVerify bool
}

// ConfigBuilder builds exporter configuration files
type ConfigBuilder struct {
	instances      []Instance
	useLegacyKey   bool
	httpTimeout    string
	serverPort     int
	logLevel       string
	basicAuthUsers map[string]string
	tlsCertFile    string
	tlsKeyFile     string
	readTimeout    int
	writeTimeout   int
}

// NewConfigBuilder creates a ConfigBuilder with test-friendly defaults
func NewConfigBuilder() *ConfigBuilder {
	return &ConfigBuilder{
		httpTimeout: "2s",
		logLevel:    "debug",
	}
}

// AddInstance adds a Logstash instance by URL
func (cb *ConfigBuilder) AddInstance(url string) *ConfigBuilder {
	return cb.AddInstanceWith(Instance{URL: url})
}

// AddInstanceWith adds a Logstash instance with additional settings
func (cb *ConfigBuilder) AddInstanceWith(instance Instance) *ConfigBuilder {
	cb.instances = append(cb.instances, instance)
	return cb
}

// UseLegacyServersKey writes instances under the deprecated `servers` key
func (cb *ConfigBuilder) UseLegacyServersKey() *ConfigBuilder {
	cb.useLegacyKey = true
	return cb
}

// WithServerPort sets the port the exporter listens on
func (cb *ConfigBuilder) WithServerPort(port int) *ConfigBuilder {
	cb.serverPort = port
	return cb
}

// WithLogLevel sets the exporter log level
func (cb *ConfigBuilder) WithLogLevel(level string) *ConfigBuilder {
	cb.logLevel = level
	return cb
}

// WithHTTPTimeout sets the timeout for requests to Logstash, e.g. "500ms"
func (cb *ConfigBuilder) WithHTTPTimeout(timeout string) *ConfigBuilder {
	cb.httpTimeout = timeout
	return cb
}

// WithBasicAuthUsers protects the metrics endpoint with basic authentication
func (cb *ConfigBuilder) WithBasicAuthUsers(users map[string]string) *ConfigBuilder {
	cb.basicAuthUsers = users
	return cb
}

// WithServerTLS serves the exporter over HTTPS
func (cb *ConfigBuilder) WithServerTLS(certFile, keyFile string) *ConfigBuilder {
	cb.tlsCertFile = certFile
	cb.tlsKeyFile = keyFile
	return cb
}

// WithServerTimeouts sets the read and write timeouts in seconds
func (cb *ConfigBuilder) WithServerTimeouts(readTimeout, writeTimeout int) *ConfigBuilder {
	cb.readTimeout = readTimeout
	cb.writeTimeout = writeTimeout
	return cb
}

// Build generates the YAML configuration
func (cb *ConfigBuilder) Build() string {
	instances := make([]yaml.MapSlice, 0, len(cb.instances))
	for _, instance := range cb.instances {
		entry := yaml.MapSlice{{Key: "url", Value: instance.URL}}
		if instance.Name != "" {
			entry = append(entry, yaml.MapItem{Key: "name", Value: instance.Name})
		}
		if instance.Username != "" || instance.Password != "" {
			entry = append(entry, yaml.MapItem{Key: "basic_auth", Value: yaml.MapSlice{
				{Key: "username", Value: instance.Username},
				{Key: "password", Value: instance.Password},
			}})
		}
		if instance.CAFile != "" || instance.InsecureSkipVerify {
			tlsConfig := yaml.MapSlice{}
			if instance.CAFile != "" {
				tlsConfig = append(tlsConfig, yaml.MapItem{Key: "ca_file", Value: instance.CAFile})
			}
			if instance.InsecureSkipVerify {
				tlsConfig = append(tlsConfig, yaml.MapItem{Key: "insecure_skip_verify", Value: true})
			}
			entry = append(entry, yaml.MapItem{Key: "tls_config", Value: tlsConfig})
		}
		instances = append(instances, entry)
	}

	instancesKey := "instances"
	if cb.useLegacyKey {
		instancesKey = "servers"
	}

	server := yaml.MapSlice{
		{Key: "host", Value: "127.0.0.1"},
		{Key: "port", Value: cb.serverPort},
	}
	if cb.readTimeout > 0 {
		server = append(server, yaml.MapItem{Key: "read_timeout_seconds", Value: cb.readTimeout})
	}
	if cb.writeTimeout > 0 {
		server = append(server, yaml.MapItem{Key: "write_timeout_seconds", Value: cb.writeTimeout})
	}
	if len(cb.basicAuthUsers) > 0 {
		server = append(server, yaml.MapItem{Key: "basic_auth", Value: yaml.MapSlice{
			{Key: "users", Value: cb.basicAuthUsers},
		}})
	}
	if cb.tlsCertFile != "" {
		server = append(server, yaml.MapItem{Key: "tls_server_config", Value: yaml.MapSlice{
			{Key: "cert_file", Value: cb.tlsCertFile},
			{Key: "key_file", Value: cb.tlsKeyFile},
		}})
	}

	config := yaml.MapSlice{
		{Key: "logstash", Value: yaml.MapSlice{
			{Key: instancesKey, Value: instances},
			{Key: "httpTimeout", Value: cb.httpTimeout},
		}},
		{Key: "server", Value: server},
		{Key: "logging", Value: yaml.MapSlice{
			{Key: "level", Value: cb.logLevel},
			{Key: "format", Value: "json"},
		}},
	}

	out, err := yaml.Marshal(config)
	if err != nil {
		panic(err)
	}
	return string(out)
}

// WriteConfigFile writes content to a new config file in a temporary
// directory and returns its path
func WriteConfigFile(t *testing.T, content string) string {
	t.Helper()

	configPath := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}
	return configPath
}
