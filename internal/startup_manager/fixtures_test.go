package startup_manager

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kuskoman/logstash-exporter/pkg/config"
)

// createValidConfigFile creates a valid config file for testing
func createValidConfigFile(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yml")

	// Use a free port so that tests do not conflict with each other, with
	// servers left over from a previous run (-count) or with other processes
	configContent := fmt.Sprintf(`
logstash:
  instances:
    - host: http://localhost:9600
      name: logstash-test
  httpTimeout: 2s
server:
  host: 127.0.0.1
  port: %d
logging:
  level: info
  format: text
`, getFreePort(t))

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}

	return configPath
}

// getFreePort returns a TCP port that is currently free
func getFreePort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find a free port: %v", err)
	}
	defer func() { _ = listener.Close() }()

	return listener.Addr().(*net.TCPAddr).Port
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
			HttpTimeout: 2 * time.Second,
		},
		Server: config.ServerConfig{
			Host: "127.0.0.1",
			Port: 9198,
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "text",
		},
	}
}
