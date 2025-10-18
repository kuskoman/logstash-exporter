package e2e

import (
	"strings"
	"testing"
	"time"
)

func TestMultiInstance(t *testing.T) {
	t.Parallel()

	// Get a free port for this test
	port, err := GetFreePort()
	if err != nil {
		t.Fatalf("failed to get free port: %v", err)
	}

	// Create two mock Logstash servers
	mockLogstash1, err := NewMockLogstashServer()
	if err != nil {
		t.Fatalf("failed to create first mock Logstash server: %v", err)
	}
	defer mockLogstash1.Close()

	mockLogstash2, err := NewMockLogstashServer()
	if err != nil {
		t.Fatalf("failed to create second mock Logstash server: %v", err)
	}
	defer mockLogstash2.Close()

	// Build configuration with two instances
	configYAML := NewConfigBuilder().
		AddInstance(mockLogstash1.URL()).
		AddInstance(mockLogstash2.URL()).
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

	// Wait for metrics to be collected from both instances
	time.Sleep(3 * time.Second)

	t.Run("metrics_from_all_instances", func(t *testing.T) {
		var instancesFound int
		var metricsText string

		// Retry a few times to allow for both instances to be scraped
		for i := 0; i < 5; i++ {
			metricsText = FetchMetrics(t, exporter.GetMetricsURL())
			instancesFound = CountInstancesInMetrics(metricsText)

			if instancesFound >= 2 {
				break
			}
			time.Sleep(1 * time.Second)
		}

		if instancesFound < 2 {
			t.Errorf("expected metrics from 2 instances, found %d", instancesFound)
		}
	})

	t.Run("all_instances_queried", func(t *testing.T) {
		// Wait a bit more and retry to ensure both instances have been queried
		for i := 0; i < 5; i++ {
			if mockLogstash1.RequestCount > 0 && mockLogstash2.RequestCount > 0 {
				break
			}
			time.Sleep(1 * time.Second)
		}

		if mockLogstash1.RequestCount == 0 {
			t.Error("first mock Logstash server received no requests")
		}
		if mockLogstash2.RequestCount == 0 {
			t.Error("second mock Logstash server received no requests")
		}
	})

	t.Run("instance_labels_distinguish_sources", func(t *testing.T) {
		var buildMetricCount int
		var metricsText string

		// Retry to ensure both instances have been scraped
		for i := 0; i < 5; i++ {
			metricsText = FetchMetrics(t, exporter.GetMetricsURL())

			// Verify that we can distinguish between instances
			// Count logstash_info_build metrics (one per instance)
			lines := strings.Split(metricsText, "\n")
			buildMetricCount = 0

			for _, line := range lines {
				if strings.Contains(line, "logstash_info_build{") &&
					(strings.Contains(line, "instance_name=") || strings.Contains(line, "hostname=")) {
					buildMetricCount++
				}
			}

			if buildMetricCount >= 2 {
				break
			}
			time.Sleep(1 * time.Second)
		}

		// Should have at least one build metric per instance
		if buildMetricCount < 2 {
			t.Errorf("expected at least 2 instance-specific build metrics, found %d", buildMetricCount)
		}
	})
}
