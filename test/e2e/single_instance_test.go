package e2e

import (
	"fmt"
	"testing"
)

func TestSingleInstance(t *testing.T) {
	t.Parallel()

	mockLogstash := NewMockLogstashServer(t)
	exporter := StartExporter(t, NewConfigBuilder().AddInstance(mockLogstash.URL()))

	metrics := exporter.WaitForInstances(1)

	t.Run("exports_logstash_and_exporter_metrics", func(t *testing.T) {
		for _, name := range []string{
			"logstash_info_build",
			"logstash_info_up",
			"logstash_stats_jvm_threads_count",
			"logstash_stats_pipeline_up",
			"logstash_exporter_build_info",
		} {
			if !metrics.Has(name) {
				t.Errorf("expected metric %s to be exported", name)
			}
		}
	})

	t.Run("labels_metrics_with_the_instance", func(t *testing.T) {
		instances := metrics.Instances()
		if len(instances) != 1 || instances[0] != InstanceName(mockLogstash.URL()) {
			t.Errorf("expected instance_name label %q, got %v", InstanceName(mockLogstash.URL()), instances)
		}
	})

	t.Run("queries_both_logstash_endpoints", func(t *testing.T) {
		for _, path := range []string{nodeInfoPath, nodeStatsPath} {
			if mockLogstash.RequestCount(path) == 0 {
				t.Errorf("expected exporter to query %s", path)
			}
		}
	})

	t.Run("scrapes_logstash_on_every_request", func(t *testing.T) {
		before := mockLogstash.RequestCount(nodeStatsPath)
		if _, err := exporter.Metrics(); err != nil {
			t.Fatal(err)
		}
		if after := mockLogstash.RequestCount(nodeStatsPath); after <= before {
			t.Errorf("expected a new request to Logstash, request count stayed at %d", before)
		}
	})
}

func TestNamedInstance(t *testing.T) {
	t.Parallel()

	mockLogstash := NewMockLogstashServer(t)
	exporter := StartExporter(t, NewConfigBuilder().AddInstanceWith(Instance{
		URL:  mockLogstash.URL(),
		Name: "primary",
	}))

	exporter.WaitForMetrics(func(m *Metrics) error {
		instances := m.Instances()
		if len(instances) != 1 || instances[0] != "primary" {
			return fmt.Errorf("expected instance_name label %q, got %v", "primary", instances)
		}
		return nil
	})
}
