package e2e

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

// assertInstanceUp checks the logstash_info_up value for an instance
func assertInstanceUp(t *testing.T, metrics *Metrics, url string, expected float64) {
	t.Helper()

	value, err := metrics.Value("logstash_info_up", map[string]string{"instance_name": InstanceName(url)})
	if err != nil {
		t.Errorf("instance %s: %v", url, err)
		return
	}
	if value != expected {
		t.Errorf("expected logstash_info_up for %s to be %v, got %v", url, expected, value)
	}
}

func TestLogstashFailures(t *testing.T) {
	t.Parallel()

	failures := []struct {
		name            string
		breakInstance   func(*MockLogstashServer)
		expectedHealthy bool
	}{
		{
			name:          "unreachable",
			breakInstance: func(m *MockLogstashServer) { m.Close() },
		},
		{
			name:          "returns_http_500",
			breakInstance: func(m *MockLogstashServer) { m.SetStatusCode(http.StatusInternalServerError) },
		},
		{
			name:          "returns_http_401",
			breakInstance: func(m *MockLogstashServer) { m.SetStatusCode(http.StatusUnauthorized) },
		},
		{
			name: "returns_invalid_json",
			breakInstance: func(m *MockLogstashServer) {
				m.SetResponse(nodeInfoPath, []byte("not json"))
				m.SetResponse(nodeStatsPath, []byte("not json"))
			},
			// The healthcheck only checks that Logstash answers with HTTP 200
			expectedHealthy: true,
		},
		{
			name: "returns_truncated_json",
			breakInstance: func(m *MockLogstashServer) {
				m.SetResponse(nodeInfoPath, []byte(`{"host": "logstash", "version": `))
				m.SetResponse(nodeStatsPath, []byte(`{"jvm": {"threads": {`))
			},
			expectedHealthy: true,
		},
	}

	for _, failure := range failures {
		t.Run(failure.name, func(t *testing.T) {
			t.Parallel()

			broken := NewMockLogstashServer(t)
			healthy := NewMockLogstashServer(t)
			failure.breakInstance(broken)

			exporter := StartExporter(t, NewConfigBuilder().
				AddInstance(broken.URL()).
				AddInstance(healthy.URL()))

			metrics := exporter.WaitForInstances(2)

			t.Run("reports_the_broken_instance_as_down", func(t *testing.T) {
				assertInstanceUp(t, metrics, broken.URL(), 0)
			})

			t.Run("keeps_exporting_the_healthy_instance", func(t *testing.T) {
				assertInstanceUp(t, metrics, healthy.URL(), 1)
				labels := map[string]string{"instance_name": InstanceName(healthy.URL())}
				if _, err := metrics.Value("logstash_stats_jvm_threads_count", labels); err != nil {
					t.Error(err)
				}
			})

			t.Run("reports_failed_scrapes", func(t *testing.T) {
				count, err := metrics.Value("logstash_exporter_scrape_duration_seconds", map[string]string{
					"collector": "nodestats",
					"result":    "error",
				})
				if err != nil {
					t.Fatal(err)
				}
				if count == 0 {
					t.Error("expected failed scrapes to be counted")
				}
			})

			t.Run("does_not_export_stats_for_the_broken_instance", func(t *testing.T) {
				labels := map[string]string{"instance_name": InstanceName(broken.URL())}
				if _, err := metrics.Value("logstash_stats_jvm_threads_count", labels); err == nil {
					t.Error("expected no JVM stats for the broken instance")
				}
			})

			t.Run("healthcheck_reflects_the_failure", func(t *testing.T) {
				resp, err := exporter.Get("/healthcheck")
				if err != nil {
					t.Fatal(err)
				}
				healthy := resp.StatusCode == http.StatusOK
				if healthy != failure.expectedHealthy {
					t.Errorf("expected healthy=%v, got status %d: %s", failure.expectedHealthy, resp.StatusCode, resp.Body)
				}
				if !failure.expectedHealthy && !strings.Contains(resp.Body, broken.URL()) {
					t.Errorf("expected the healthcheck error to name %s, got %q", broken.URL(), resp.Body)
				}
			})
		})
	}
}

func TestLogstashUnavailableAtStartup(t *testing.T) {
	t.Parallel()

	mockLogstash := NewMockLogstashServer(t)
	mockLogstash.SetStatusCode(http.StatusServiceUnavailable)

	exporter := StartExporter(t, NewConfigBuilder().AddInstance(mockLogstash.URL()))

	t.Run("serves_metrics_while_logstash_is_down", func(t *testing.T) {
		metrics := exporter.WaitForInstances(1)
		assertInstanceUp(t, metrics, mockLogstash.URL(), 0)
		if !metrics.Has("logstash_exporter_build_info") {
			t.Error("expected exporter metrics while Logstash is down")
		}
	})

	t.Run("recovers_when_logstash_comes_back", func(t *testing.T) {
		mockLogstash.SetStatusCode(http.StatusOK)

		metrics := exporter.WaitForMetrics(func(m *Metrics) error {
			up, err := m.Value("logstash_info_up", map[string]string{"instance_name": InstanceName(mockLogstash.URL())})
			if err != nil {
				return err
			}
			if up != 1 {
				return fmt.Errorf("expected logstash_info_up to be 1, got %v", up)
			}
			return nil
		})
		if !metrics.Has("logstash_stats_jvm_threads_count") {
			t.Error("expected stats to be exported after recovery")
		}

		resp, err := exporter.Get("/healthcheck")
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected healthcheck to recover, got %d: %s", resp.StatusCode, resp.Body)
		}
	})
}

func TestLogstashGoesDownWhileRunning(t *testing.T) {
	t.Parallel()

	mockLogstash := NewMockLogstashServer(t)
	exporter := StartExporter(t, NewConfigBuilder().AddInstance(mockLogstash.URL()))
	assertInstanceUp(t, exporter.WaitForInstances(1), mockLogstash.URL(), 1)

	mockLogstash.Close()

	metrics := exporter.WaitForMetrics(func(m *Metrics) error {
		up, err := m.Value("logstash_info_up", map[string]string{"instance_name": InstanceName(mockLogstash.URL())})
		if err != nil {
			return err
		}
		if up != 0 {
			return fmt.Errorf("expected logstash_info_up to be 0, got %v", up)
		}
		return nil
	})
	if metrics.Has("logstash_stats_jvm_threads_count") {
		t.Error("expected stats to stop being exported once Logstash is down")
	}

	resp, err := exporter.Get("/healthcheck")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected healthcheck to return 500, got %d", resp.StatusCode)
	}
}

func TestSlowLogstash(t *testing.T) {
	t.Parallel()

	const httpTimeout = 500 * time.Millisecond

	slow := NewMockLogstashServer(t)
	fast := NewMockLogstashServer(t)
	slow.SetDelay(10 * time.Second)

	exporter := StartExporter(t, NewConfigBuilder().
		AddInstance(slow.URL()).
		AddInstance(fast.URL()).
		WithHTTPTimeout(httpTimeout.String()))

	t.Run("metrics_respond_within_the_http_timeout", func(t *testing.T) {
		start := time.Now()
		metrics, err := exporter.Metrics()
		if err != nil {
			t.Fatal(err)
		}
		// Node info and node stats are requested one after another for each
		// instance, so a slow instance can take up to twice the timeout.
		if elapsed := time.Since(start); elapsed > 4*httpTimeout {
			t.Errorf("expected /metrics to respond within %s, took %s", 4*httpTimeout, elapsed)
		}

		assertInstanceUp(t, metrics, slow.URL(), 0)
		assertInstanceUp(t, metrics, fast.URL(), 1)
	})

	t.Run("healthcheck_times_out_and_reports_the_slow_instance", func(t *testing.T) {
		start := time.Now()
		resp, err := exporter.Get("/healthcheck")
		if err != nil {
			t.Fatal(err)
		}
		if elapsed := time.Since(start); elapsed > 4*httpTimeout {
			t.Errorf("expected /healthcheck to respond within %s, took %s", 4*httpTimeout, elapsed)
		}
		if resp.StatusCode != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", resp.StatusCode)
		}
		if !strings.Contains(resp.Body, slow.URL()) {
			t.Errorf("expected the error to name %s, got %q", slow.URL(), resp.Body)
		}
	})
}
