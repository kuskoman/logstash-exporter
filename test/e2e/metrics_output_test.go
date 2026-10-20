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
			"logstash_stats_events_in":         3001,
			"logstash_stats_events_filtered":   1000,
			"logstash_stats_events_out":        1000,
			"logstash_stats_jvm_threads_count": 53,
			"logstash_stats_pipeline_workers":  8,
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
			"version":       "9.5.5",
			"id":            "ec53064e-be14-4264-8f3f-10d502c84cbc",
		}
		if _, err := metrics.Value("logstash_info_node", labels); err != nil {
			t.Error(err)
		}
	})

	t.Run("exports_the_pipeline_and_its_plugins", func(t *testing.T) {
		pipeline := map[string]string{"instance_name": InstanceName(mockLogstash.URL()), "pipeline": "main"}
		if _, err := metrics.Value("logstash_stats_pipeline_up", pipeline); err != nil {
			t.Error(err)
		}

		output := map[string]string{"instance_name": InstanceName(mockLogstash.URL()), "pipeline": "main", "plugin": "elasticsearch"}
		value, err := metrics.Value("logstash_stats_pipeline_plugin_documents_successes", output)
		if err != nil {
			t.Fatal(err)
		}
		if value != 1000 {
			t.Errorf("expected 1000 documents written by the elasticsearch output, got %v", value)
		}
	})

}
