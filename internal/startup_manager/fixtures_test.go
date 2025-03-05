package startup_manager

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kuskoman/logstash-exporter/pkg/config"
)

// createValidConfigFile creates a valid config file for testing
func createValidConfigFile(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yml")

	configContent := `
logstash:
  instances:
    - host: http://localhost:9600
      name: logstash-test
  httpTimeout: 2s
server:
  host: 0.0.0.0
  port: 9198
logging:
  level: info
  format: text
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}

	return configPath
}

// createTestConfig returns a test config
func createTestConfig() *config.Config {
	return &config.Config{
		Logstash: config.LogstashConfig{
			Instances: []*config.LogstashInstance{
				{
					Host: "http://localhost:9600",
					Name: "logstash-test",
				},
			},
			HttpTimeout: 2,
		},
		Server: config.ServerConfig{
			Host: "0.0.0.0",
			Port: 9198,
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "text",
		},
	}
}