package e2e

import (
	"strings"
	"testing"
	"time"
)

func TestHealthCheckEndpoint(t *testing.T) {
	t.Parallel()

	// Get a free port for this test
	port, err := GetFreePort()
	if err != nil {
		t.Fatalf("failed to get free port: %v", err)
	}

	// Create mock Logstash server
	mockLogstash, err := NewMockLogstashServer()
	if err != nil {
		t.Fatalf("failed to create mock Logstash server: %v", err)
	}
	defer mockLogstash.Close()

	// Build configuration
	configYAML := NewConfigBuilder().
		AddInstance(mockLogstash.URL()).
		WithServerPort(port).
		WithLogLevel("error").
		Build()

	configPath := TestConfig(t, configYAML)

	// Start exporter
	exporter := StartExporter(t, configPath, port)
	defer func() {
		if err := exporter.Stop(); err != nil {
			t.Logf("error stopping exporter: %v", err)
		}
	}()

	time.Sleep(1 * time.Second)

	t.Run("returns_200_when_healthy", func(t *testing.T) {
		resp, err := httpGet(exporter.GetHealthCheckURL())
		if err != nil {
			t.Fatalf("failed to get healthcheck: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("responds_quickly", func(t *testing.T) {
		start := time.Now()
		resp, err := httpGet(exporter.GetHealthCheckURL())
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("failed to get healthcheck: %v", err)
		}
		defer resp.Body.Close()

		// Health check should respond within reasonable time
		if duration > 5*time.Second {
			t.Errorf("healthcheck took too long: %v", duration)
		}
	})
}

func TestVersionEndpoint(t *testing.T) {
	t.Parallel()

	// Get a free port for this test
	port, err := GetFreePort()
	if err != nil {
		t.Fatalf("failed to get free port: %v", err)
	}

	// Create mock Logstash server
	mockLogstash, err := NewMockLogstashServer()
	if err != nil {
		t.Fatalf("failed to create mock Logstash server: %v", err)
	}
	defer mockLogstash.Close()

	// Build configuration
	configYAML := NewConfigBuilder().
		AddInstance(mockLogstash.URL()).
		WithServerPort(port).
		WithLogLevel("error").
		Build()

	configPath := TestConfig(t, configYAML)

	// Start exporter
	exporter := StartExporter(t, configPath, port)
	defer func() {
		if err := exporter.Stop(); err != nil {
			t.Logf("error stopping exporter: %v", err)
		}
	}()

	time.Sleep(1 * time.Second)

	t.Run("returns_build_info", func(t *testing.T) {
		resp, err := httpGet(exporter.GetVersionURL())
		if err != nil {
			t.Fatalf("failed to get version: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		// Read response
		body, err := readResponseBody(resp)
		if err != nil {
			t.Fatalf("failed to read response: %v", err)
		}

		// Should contain version information (JSON with capital V)
		if !strings.Contains(body, "Version") && !strings.Contains(body, "version") {
			t.Errorf("version endpoint response does not contain version info: %s", body)
		}

		// Should be JSON
		if !strings.HasPrefix(strings.TrimSpace(body), "{") {
			t.Error("version endpoint should return JSON")
		}
	})

	t.Run("accessible_without_auth", func(t *testing.T) {
		// Version endpoint should not require authentication
		resp, err := httpGet(exporter.GetVersionURL())
		if err != nil {
			t.Fatalf("failed to get version: %v", err)
		}
		defer resp.Body.Close()

		// Should get 200, not 401 Unauthorized
		if resp.StatusCode == 401 {
			t.Error("version endpoint requires authentication but should be public")
		}
	})
}

func TestMetricsEndpoint(t *testing.T) {
	t.Parallel()

	// Get a free port for this test
	port, err := GetFreePort()
	if err != nil {
		t.Fatalf("failed to get free port: %v", err)
	}

	// Create mock Logstash server
	mockLogstash, err := NewMockLogstashServer()
	if err != nil {
		t.Fatalf("failed to create mock Logstash server: %v", err)
	}
	defer mockLogstash.Close()

	// Build configuration
	configYAML := NewConfigBuilder().
		AddInstance(mockLogstash.URL()).
		WithServerPort(port).
		WithLogLevel("error").
		Build()

	configPath := TestConfig(t, configYAML)

	// Start exporter
	exporter := StartExporter(t, configPath, port)
	defer func() {
		if err := exporter.Stop(); err != nil {
			t.Logf("error stopping exporter: %v", err)
		}
	}()

	time.Sleep(2 * time.Second)

	t.Run("returns_prometheus_format", func(t *testing.T) {
		resp, err := httpGet(exporter.GetMetricsURL())
		if err != nil {
			t.Fatalf("failed to get metrics: %v", err)
		}
		defer resp.Body.Close()

		// Check content type
		contentType := resp.Header.Get("Content-Type")
		if !strings.Contains(contentType, "text/plain") {
			t.Errorf("unexpected content type: %s", contentType)
		}
	})

	t.Run("includes_exporter_metadata", func(t *testing.T) {
		metricsText := FetchMetrics(t, exporter.GetMetricsURL())

		// Should include HELP and TYPE comments
		if !strings.Contains(metricsText, "# HELP") {
			t.Error("metrics missing HELP comments")
		}

		if !strings.Contains(metricsText, "# TYPE") {
			t.Error("metrics missing TYPE comments")
		}
	})

	t.Run("includes_logstash_metrics", func(t *testing.T) {
		metricsText := FetchMetrics(t, exporter.GetMetricsURL())

		// Should include various Logstash metrics
		expectedMetrics := []string{
			"logstash_info",
			"logstash_exporter",
		}

		for _, metric := range expectedMetrics {
			if !strings.Contains(metricsText, metric) {
				t.Errorf("missing expected metric family: %s", metric)
			}
		}
	})
}
