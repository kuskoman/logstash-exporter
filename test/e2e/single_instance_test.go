package e2e

import (
	"strings"
	"testing"
	"time"
)

func TestSingleInstance(t *testing.T) {
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

	// Wait a bit for metrics to be collected
	time.Sleep(2 * time.Second)

	t.Run("metrics_endpoint_returns_data", func(t *testing.T) {
		metricsText := FetchMetrics(t, exporter.GetMetricsURL())

		// Verify we got metrics
		if len(metricsText) == 0 {
			t.Fatal("metrics endpoint returned empty response")
		}

		// Check for expected metric families (not specific metrics which may vary)
		AssertMetricExists(t, metricsText, "logstash_info")
		AssertMetricExists(t, metricsText, "logstash_exporter_build_info")
	})

	t.Run("metrics_contain_instance_labels", func(t *testing.T) {
		metricsText := FetchMetrics(t, exporter.GetMetricsURL())

		// Should have at least some metrics with instance_name or hostname labels
		foundInstanceMetric := strings.Contains(metricsText, "instance_name=") ||
			strings.Contains(metricsText, "hostname=")

		if !foundInstanceMetric {
			t.Error("no metrics with instance_name or hostname labels found")
		}
	})

	t.Run("exporter_queries_logstash_api", func(t *testing.T) {
		if mockLogstash.RequestCount == 0 {
			t.Error("mock Logstash server received no requests")
		}
	})

	t.Run("metrics_are_valid_prometheus_format", func(t *testing.T) {
		metricsText := FetchMetrics(t, exporter.GetMetricsURL())

		// Try to parse metrics
		metrics := ParseMetrics(t, metricsText)

		if len(metrics) == 0 {
			t.Error("no metrics parsed from response")
		}

		// Check for required exporter metrics
		foundBuildInfo := false
		for key := range metrics {
			if strings.Contains(key, "logstash_exporter_build_info") {
				foundBuildInfo = true
				break
			}
		}

		if !foundBuildInfo {
			t.Error("required metric logstash_exporter_build_info not found")
		}
	})
}
