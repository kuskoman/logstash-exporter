package e2e

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestConfigReload(t *testing.T) {
	t.Parallel()

	// Get a free port for this test
	port, err := GetFreePort()
	if err != nil {
		t.Fatalf("failed to get free port: %v", err)
	}

	// Create initial mock Logstash server
	mockLogstash1, err := NewMockLogstashServer()
	if err != nil {
		t.Fatalf("failed to create first mock Logstash server: %v", err)
	}
	defer mockLogstash1.Close()

	// Build initial configuration with one instance
	initialConfigYAML := NewConfigBuilder().
		AddInstance(mockLogstash1.URL()).
		WithServerPort(port).
		WithLogLevel("error").
		Build()

	configPath := TestConfig(t, initialConfigYAML)

	// Start exporter with hot-reload enabled (-watch flag)
	binaryPath := GetBinaryPath(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := exec.CommandContext(ctx, binaryPath, "-config", configPath, "-watch")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start exporter: %v", err)
	}

	// Ensure cleanup
	defer func() {
		cancel()
		if err := cmd.Wait(); err != nil {
			t.Logf("error waiting for command: %v", err)
		}
	}()

	// Wait for server to start
	serverURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	if !waitForServer(serverURL, 10*time.Second) {
		t.Fatal("server did not start in time")
	}

	// Wait for initial metrics collection
	time.Sleep(2 * time.Second)

	t.Run("initial_configuration_one_instance", func(t *testing.T) {
		metricsURL := fmt.Sprintf("http://127.0.0.1:%d/metrics", port)
		metricsText := FetchMetrics(t, metricsURL)

		// Count instances in metrics
		instancesFound := CountInstancesInMetrics(metricsText)
		if instancesFound != 1 {
			t.Errorf("expected 1 instance initially, found %d", instancesFound)
		}
	})

	// Create a second mock Logstash server
	mockLogstash2, err := NewMockLogstashServer()
	if err != nil {
		t.Fatalf("failed to create second mock Logstash server: %v", err)
	}
	defer mockLogstash2.Close()

	// Update configuration with two instances
	updatedConfigYAML := NewConfigBuilder().
		AddInstance(mockLogstash1.URL()).
		AddInstance(mockLogstash2.URL()).
		WithServerPort(port).
		WithLogLevel("error").
		Build()

	// Write updated config to the same file
	if err := os.WriteFile(configPath, []byte(updatedConfigYAML), 0644); err != nil {
		t.Fatalf("failed to write updated config: %v", err)
	}

	t.Run("reload_adds_second_instance", func(t *testing.T) {
		metricsURL := fmt.Sprintf("http://127.0.0.1:%d/metrics", port)
		healthURL := fmt.Sprintf("http://127.0.0.1:%d/healthcheck", port)

		// Wait for file watcher to detect change and reload
		// The file watcher typically checks every 1-2 seconds
		// First, wait for the server to come back up after reload
		time.Sleep(2 * time.Second)

		// Wait for server to be healthy again after reload
		for i := 0; i < 15; i++ {
			resp, err := httpGet(healthURL)
			if err == nil {
				if err := resp.Body.Close(); err != nil {
					t.Logf("error closing response body: %v", err)
				}
				if resp.StatusCode == 200 {
					break
				}
			}
			time.Sleep(1 * time.Second)
		}

		// Now retry fetching metrics to check for the second instance
		var instancesFound int
		var metricsText string
		var lastErr error

		for i := 0; i < 15; i++ {
			// Try to fetch metrics, but don't fail immediately
			resp, err := httpGet(metricsURL)
			if err != nil {
				lastErr = err
				time.Sleep(1 * time.Second)
				continue
			}

			body, err := readResponseBody(resp)
			if err := resp.Body.Close(); err != nil {
				t.Logf("error closing response body: %v", err)
			}
			if err != nil {
				lastErr = err
				time.Sleep(1 * time.Second)
				continue
			}

			metricsText = body
			instancesFound = CountInstancesInMetrics(metricsText)

			if instancesFound >= 2 {
				break
			}

			time.Sleep(1 * time.Second)
		}

		if instancesFound < 2 {
			if lastErr != nil {
				t.Errorf("expected 2 instances after reload, found %d (last error: %v)", instancesFound, lastErr)
			} else {
				t.Errorf("expected 2 instances after reload, found %d", instancesFound)
			}
		}
	})

	t.Run("new_instance_is_queried", func(t *testing.T) {
		// Wait a bit more to ensure scraping happens
		time.Sleep(3 * time.Second)

		if mockLogstash2.RequestCount == 0 {
			t.Error("second Logstash instance not queried after config reload")
		}
	})
}
