package e2e

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/common/expfmt"
)

// TestConfig creates a temporary config file with the given content
func TestConfig(t *testing.T, content string) string {
	t.Helper()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	err := os.WriteFile(configPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	return configPath
}

// ConfigBuilder helps build test configurations programmatically
type ConfigBuilder struct {
	instances      []string
	serverPort     int
	logLevel       string
	httpTimeout    string
	readTimeout    int
	writeTimeout   int
	maxConnections int
}

// NewConfigBuilder creates a new configuration builder
func NewConfigBuilder() *ConfigBuilder {
	return &ConfigBuilder{
		instances:      []string{},
		serverPort:     0,
		logLevel:       "error",
		httpTimeout:    "5s",
		readTimeout:    30,
		writeTimeout:   30,
		maxConnections: 512,
	}
}

// AddInstance adds a Logstash instance URL
func (cb *ConfigBuilder) AddInstance(url string) *ConfigBuilder {
	cb.instances = append(cb.instances, url)
	return cb
}

// WithServerPort sets the server port
func (cb *ConfigBuilder) WithServerPort(port int) *ConfigBuilder {
	cb.serverPort = port
	return cb
}

// WithLogLevel sets the log level
func (cb *ConfigBuilder) WithLogLevel(level string) *ConfigBuilder {
	cb.logLevel = level
	return cb
}

// WithHTTPTimeout sets the HTTP timeout for Logstash requests
func (cb *ConfigBuilder) WithHTTPTimeout(timeout string) *ConfigBuilder {
	cb.httpTimeout = timeout
	return cb
}

// Build generates the YAML configuration string
func (cb *ConfigBuilder) Build() string {
	var sb strings.Builder

	sb.WriteString("logstash:\n")
	sb.WriteString("  instances:\n")
	for _, instance := range cb.instances {
		sb.WriteString(fmt.Sprintf("    - url: %s\n", instance))
	}
	sb.WriteString(fmt.Sprintf("  httpTimeout: %s\n", cb.httpTimeout))
	sb.WriteString("\n")

	sb.WriteString("server:\n")
	sb.WriteString("  host: 127.0.0.1\n")
	sb.WriteString(fmt.Sprintf("  port: %d\n", cb.serverPort))
	sb.WriteString(fmt.Sprintf("  read_timeout_seconds: %d\n", cb.readTimeout))
	sb.WriteString(fmt.Sprintf("  write_timeout_seconds: %d\n", cb.writeTimeout))
	sb.WriteString(fmt.Sprintf("  max_connections: %d\n", cb.maxConnections))
	sb.WriteString("\n")

	sb.WriteString("logging:\n")
	sb.WriteString(fmt.Sprintf("  level: %s\n", cb.logLevel))
	sb.WriteString("  format: json\n")

	return sb.String()
}

// ExporterInstance represents a running exporter binary for testing
type ExporterInstance struct {
	BinaryPath string
	ConfigPath string
	Port       int
	cmd        *exec.Cmd
	ctx        context.Context
	cancel     context.CancelFunc
}

// GetBinaryPath returns the path to the exporter binary
// The binary should be built using 'make e2e-prepare' before running tests
func GetBinaryPath(t *testing.T) string {
	t.Helper()

	// Path to the e2e binary (built by make e2e-prepare)
	binaryPath := filepath.Join("..", "..", "test", "e2e", "bin", "logstash-exporter-e2e")

	// Check if binary exists
	if _, err := os.Stat(binaryPath); err != nil {
		t.Fatalf("e2e binary not found at %s. Please run 'make e2e-prepare' first", binaryPath)
	}

	return binaryPath
}

// GetFreePort finds an available TCP port in the range 1000-9999
// This ensures the port is within commonly allowed ranges in CI environments
func GetFreePort() (int, error) {
	const minPort = 1000
	const maxPort = 9999

	for port := minPort; port <= maxPort; port++ {
		address := fmt.Sprintf("127.0.0.1:%d", port)
		listener, err := net.Listen("tcp", address)
		if err != nil {
			// Port is in use or unavailable, try next one
			continue
		}

		// Port is available, close and return it
		if err := listener.Close(); err != nil {
			fmt.Printf("error closing listener: %v\n", err)
		}
		return port, nil
	}

	return 0, fmt.Errorf("no free port found in range %d-%d", minPort, maxPort)
}

// StartExporter starts an exporter binary with the given configuration
func StartExporter(t *testing.T, configPath string, port int) *ExporterInstance {
	t.Helper()

	binaryPath := GetBinaryPath(t)

	ctx, cancel := context.WithCancel(context.Background())

	// Start the binary
	cmd := exec.CommandContext(ctx, binaryPath, "-config", configPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		cancel()
		t.Fatalf("failed to start exporter: %v", err)
	}

	instance := &ExporterInstance{
		BinaryPath: binaryPath,
		ConfigPath: configPath,
		Port:       port,
		cmd:        cmd,
		ctx:        ctx,
		cancel:     cancel,
	}

	// Wait for server to be ready
	serverURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	if !waitForServer(serverURL, 10*time.Second) {
		if err := instance.Stop(); err != nil {
			t.Logf("error stopping instance: %v", err)
		}
		t.Fatal("server did not start in time")
	}

	return instance
}

// Stop stops the exporter instance
func (e *ExporterInstance) Stop() error {
	if e.cancel != nil {
		e.cancel()
	}

	if e.cmd != nil && e.cmd.Process != nil {
		// Give it time to shut down gracefully
		done := make(chan error, 1)
		go func() {
			done <- e.cmd.Wait()
		}()

		select {
		case <-done:
			return nil
		case <-time.After(5 * time.Second):
			// Force kill if it doesn't stop gracefully
			if err := e.cmd.Process.Kill(); err != nil {
				return fmt.Errorf("failed to kill process: %w", err)
			}
			<-done // Wait for Wait() to return
			return nil
		}
	}

	return nil
}

// GetMetricsURL returns the metrics endpoint URL
func (e *ExporterInstance) GetMetricsURL() string {
	return fmt.Sprintf("http://127.0.0.1:%d/metrics", e.Port)
}

// GetHealthCheckURL returns the health check endpoint URL
func (e *ExporterInstance) GetHealthCheckURL() string {
	return fmt.Sprintf("http://127.0.0.1:%d/healthcheck", e.Port)
}

// GetVersionURL returns the version endpoint URL
func (e *ExporterInstance) GetVersionURL() string {
	return fmt.Sprintf("http://127.0.0.1:%d/version", e.Port)
}

// Reload sends a reload signal by updating the config file
// Note: This requires the exporter to be started with -watch flag
func (e *ExporterInstance) Reload(newConfig string) error {
	return os.WriteFile(e.ConfigPath, []byte(newConfig), 0644)
}

// waitForServer waits for the server to be ready
func waitForServer(url string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	client := &http.Client{
		Timeout: 1 * time.Second,
	}

	for time.Now().Before(deadline) {
		resp, err := client.Get(url + "/version")
		if err == nil {
			err = resp.Body.Close()
			if err != nil {
				fmt.Printf("error closing response body: %v\n", err)
			}
			if resp.StatusCode == http.StatusOK {
				return true
			}
		}
		time.Sleep(100 * time.Millisecond)
	}

	return false
}

// FetchMetrics fetches and returns the metrics from the exporter
func FetchMetrics(t *testing.T, url string) string {
	t.Helper()

	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("failed to fetch metrics: %v", err)
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("error closing response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	return string(body)
}

// ParseMetrics parses Prometheus metrics from the response
func ParseMetrics(t *testing.T, metricsText string) map[string]float64 {
	t.Helper()

	parser := expfmt.TextParser{}
	metricFamilies, err := parser.TextToMetricFamilies(strings.NewReader(metricsText))
	if err != nil {
		t.Fatalf("failed to parse metrics: %v", err)
	}

	metrics := make(map[string]float64)
	for name, mf := range metricFamilies {
		for _, m := range mf.Metric {
			// Build metric key with labels
			key := name
			if len(m.Label) > 0 {
				var labels []string
				for _, l := range m.Label {
					labels = append(labels, fmt.Sprintf("%s=%q", l.GetName(), l.GetValue()))
				}
				key = fmt.Sprintf("%s{%s}", name, strings.Join(labels, ","))
			}

			// Extract value based on metric type
			if m.Gauge != nil {
				metrics[key] = m.Gauge.GetValue()
			} else if m.Counter != nil {
				metrics[key] = m.Counter.GetValue()
			} else if m.Untyped != nil {
				metrics[key] = m.Untyped.GetValue()
			}
		}
	}

	return metrics
}

// AssertMetricExists checks if a metric exists in the metrics text
func AssertMetricExists(t *testing.T, metricsText, metricName string) {
	t.Helper()

	if !strings.Contains(metricsText, metricName) {
		t.Errorf("metric %q not found in metrics output", metricName)
	}
}

// AssertMetricWithLabel checks if a metric with a specific label exists
func AssertMetricWithLabel(t *testing.T, metricsText, metricName, labelName, labelValue string) {
	t.Helper()

	// Look for pattern like: metricName{labelName="labelValue"}
	expectedPattern := fmt.Sprintf("%s{", metricName)
	labelPattern := fmt.Sprintf(`%s="%s"`, labelName, labelValue)

	lines := strings.Split(metricsText, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, expectedPattern) && strings.Contains(line, labelPattern) {
			return // Found it
		}
	}

	t.Errorf("metric %q with label %s=%q not found in metrics output", metricName, labelName, labelValue)
}

// httpGet performs an HTTP GET request and returns the response
func httpGet(url string) (*http.Response, error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	return client.Get(url)
}

// readResponseBody reads the entire response body and returns it as a string
func readResponseBody(resp *http.Response) (string, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// CountInstancesInMetrics counts unique instances in metrics output
// Looks for either "instance_name=" or "hostname=" labels
func CountInstancesInMetrics(metricsText string) int {
	instancesFound := make(map[string]bool)
	for _, line := range strings.Split(metricsText, "\n") {
		// Skip comments and empty lines
		if strings.HasPrefix(line, "#") || len(strings.TrimSpace(line)) == 0 {
			continue
		}

		// Look for instance_name label (primary)
		if strings.Contains(line, "instance_name=\"") {
			parts := strings.Split(line, "instance_name=\"")
			if len(parts) > 1 {
				instanceValue := strings.Split(parts[1], "\"")[0]
				instancesFound[instanceValue] = true
			}
		} else if strings.Contains(line, "hostname=\"") {
			// Fallback to hostname label
			parts := strings.Split(line, "hostname=\"")
			if len(parts) > 1 {
				instanceValue := strings.Split(parts[1], "\"")[0]
				instancesFound[instanceValue] = true
			}
		}
	}
	return len(instancesFound)
}
