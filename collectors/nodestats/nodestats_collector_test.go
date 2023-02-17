package nodestats

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/kuskoman/logstash-exporter/fetcher/responses"
	"github.com/prometheus/client_golang/prometheus"
)

type mockClient struct{}

func (m *mockClient) GetNodeStats() (*responses.NodestatsResponse, error) {
	b, err := os.ReadFile("../../fixtures/node_stats.json")
	if err != nil {
		return nil, err
	}

	var nodestats responses.NodestatsResponse
	err = json.Unmarshal(b, &nodestats)
	if err != nil {
		return nil, err
	}

	return &nodestats, nil
}

func (m *mockClient) GetNodeInfo() (*responses.NodeInfoResponse, error) {
	return nil, nil
}

func TestCollectNotNil(t *testing.T) {
	collector := NewNodestatsCollector(&mockClient{})
	ch := make(chan prometheus.Metric)

	go func() {
		err := collector.Collect(ch)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	}()

	expectedMetrics := []string{
		"logstash_stats_jvm_mem_heap_committed_bytes",
		"logstash_stats_jvm_mem_heap_max_bytes",
		"logstash_stats_jvm_mem_heap_used_bytes",
		"logstash_stats_jvm_mem_heap_used_percent",
		"logstash_stats_jvm_mem_non_heap_committed_bytes",
		"logstash_stats_jvm_threads_count",
		"logstash_stats_jvm_threads_peak_count",
		"logstash_stats_jvm_uptime_millis",
		"logstash_stats_pipeline_events_duration",
		"logstash_stats_pipeline_events_filtered",
		"logstash_stats_pipeline_events_in",
		"logstash_stats_pipeline_events_out",
		"logstash_stats_pipeline_events_queue_push_duration",
		"logstash_stats_pipeline_queue_events_count",
		"logstash_stats_pipeline_queue_events_queue_size",
		"logstash_stats_pipeline_queue_max_size_in_bytes",
		"logstash_stats_pipeline_reloads_failures",
		"logstash_stats_pipeline_reloads_successes",
		"logstash_stats_process_cpu_percent",
		"logstash_stats_process_cpu_total_millis",
		"logstash_stats_process_max_file_descriptors",
		"logstash_stats_process_mem_total_virtual",
		"logstash_stats_process_open_file_descriptors",
		"logstash_stats_queue_events_count",
		"logstash_stats_reload_failures",
		"logstash_stats_reload_successes",
	}

	var foundMetrics []string
	for i := 0; i < len(expectedMetrics); i++ {
		metric := <-ch
		if metric == nil {
			t.Errorf("expected metric %s not to be nil", metric.Desc().String())
		}

		foundMetrics = append(foundMetrics, metric.Desc().String())
	}

	for _, expectedMetric := range expectedMetrics {
		found := false
		for _, foundMetric := range foundMetrics {
			// todo: find a better way to compare metrics, unfortunetely Prometheus doesn't provide easy way to extract fqdn
			if strings.Contains(foundMetric, expectedMetric) {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("Expected metric %s to be found", expectedMetric)
		}
	}
}
