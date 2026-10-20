package e2e

import (
	"strings"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
)

// normalizeMetrics keeps only the Logstash metrics and replaces the mock
// server URL, which contains a random port. The exporter's own metrics and
// the Go runtime and process metrics change between runs.
func normalizeMetrics(text, logstashURL string) string {
	var lines []string
	for _, line := range strings.Split(text, "\n") {
		name := strings.TrimPrefix(strings.TrimPrefix(line, "# HELP "), "# TYPE ")
		if !strings.HasPrefix(name, "logstash_") || strings.HasPrefix(name, "logstash_exporter_") {
			continue
		}
		line = strings.ReplaceAll(line, InstanceName(logstashURL), "INSTANCE_NAME")
		line = strings.ReplaceAll(line, logstashURL, "LOGSTASH_URL")
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func TestMetricsOutput(t *testing.T) {
	t.Parallel()

	mockLogstash := NewMockLogstashServer(t)
	exporter := StartExporter(t, NewConfigBuilder().AddInstance(mockLogstash.URL()))
	exporter.WaitForInstances(1)

	resp, err := exporter.Get("/metrics")
	if err != nil {
		t.Fatal(err)
	}
	metrics, err := ParseMetrics(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("matches_the_snapshot_for_the_fixtures", func(t *testing.T) {
		snaps.MatchSnapshot(t, normalizeMetrics(resp.Body, mockLogstash.URL()))
	})

	instance := map[string]string{"instance_name": InstanceName(mockLogstash.URL())}

	t.Run("exports_values_from_node_stats", func(t *testing.T) {
		expected := map[string]float64{
			"logstash_stats_events_in":         3751,
			"logstash_stats_events_filtered":   1250,
			"logstash_stats_events_out":        1250,
			"logstash_stats_jvm_threads_count": 65,
		}
		for name, value := range expected {
			actual, err := metrics.Value(name, instance)
			if err != nil {
				t.Error(err)
				continue
			}
			if actual != value {
				t.Errorf("expected %s to be %v, got %v", name, value, actual)
			}
		}
	})

	t.Run("exports_node_info_as_labels", func(t *testing.T) {
		labels := map[string]string{
			"instance_name": InstanceName(mockLogstash.URL()),
			"version":       "8.15.2",
			"id":            "690de5cc-deb1-48d9-ba02-d4ec1b22e62a",
		}
		if _, err := metrics.Value("logstash_info_node", labels); err != nil {
			t.Error(err)
		}
	})

	t.Run("exports_one_series_per_pipeline", func(t *testing.T) {
		for _, pipeline := range []string{"main", ".monitoring-logstash"} {
			labels := map[string]string{"instance_name": InstanceName(mockLogstash.URL()), "pipeline": pipeline}
			if _, err := metrics.Value("logstash_stats_pipeline_up", labels); err != nil {
				t.Error(err)
			}
		}
	})

}
