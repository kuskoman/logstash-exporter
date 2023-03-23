package nodestats

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/kuskoman/logstash-exporter/fetcher/responses"
	"github.com/kuskoman/logstash-exporter/prometheus_helper"
)

type mockClient struct{}

func (m *mockClient) GetNodeStats(ctx context.Context) (*responses.NodeStatsResponse, error) {
	b, err := os.ReadFile("../../fixtures/node_stats.json")
	if err != nil {
		return nil, err
	}

	var nodestats responses.NodeStatsResponse
	err = json.Unmarshal(b, &nodestats)
	if err != nil {
		return nil, err
	}

	return &nodestats, nil
}

func (m *mockClient) GetNodeInfo(ctx context.Context) (*responses.NodeInfoResponse, error) {
	return nil, nil
}

func TestCollectNotNil(t *testing.T) {
	collector := NewNodestatsCollector(&mockClient{})
	ch := make(chan prometheus.Metric)
	ctx := context.Background()

	go func() {
		err := collector.Collect(ctx, ch)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		close(ch)
	}()

	expectedBaseMetrics := []string{
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
		"logstash_stats_jvm_mem_pool_peak_used_bytes",
		"logstash_stats_jvm_mem_pool_used_bytes",
		"logstash_stats_jvm_mem_pool_peak_max_bytes",
		"logstash_stats_jvm_mem_pool_max_bytes",
		"logstash_stats_jvm_mem_pool_committed_bytes",
	}

	var foundMetrics []string
	for metric := range ch {
		if metric == nil {
			t.Errorf("expected metric %s not to be nil", metric.Desc().String())
		}

		foundMetricDesc := metric.Desc().String()
		foundMetricFqName, err := prometheus_helper.ExtractFqName(foundMetricDesc)
		if err != nil {
			t.Errorf("failed to extract fqName from metric %s", foundMetricDesc)
		}

		foundMetrics = append(foundMetrics, foundMetricFqName)
	}

	for _, expectedMetric := range expectedBaseMetrics {
		found := false
		for _, foundMetric := range foundMetrics {
			if foundMetric == expectedMetric {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("Expected metric %s to be found", expectedMetric)
		}
	}
}

func TestParseSubcollectorErrors(t *testing.T) {
	t.Run("should return nil for 0 errors", func(t *testing.T) {
		errs := make(map[string]error)
		parsedErrors := parseSubcollectorErrors(errs)
		if parsedErrors != nil {
			t.Errorf("Expected parsedErrors to be nil, received %s", parsedErrors)
		}
	})

	t.Run("should return error when exactly 1 error is provided", func(t *testing.T) {
		errs := make(map[string]error)
		exampleErr := errors.New("test error")
		errs["pipe"] = exampleErr

		parsedErr := parseSubcollectorErrors(errs)
		if parsedErr == nil {
			t.Error("Expected parsedErr to contain an error, received nil")
		}
	})

	t.Run("should return error when more than 1 error is provided", func(t *testing.T) {
		errs := make(map[string]error)
		errs["pipe"] = errors.New("test error")
		errs["pipe2"] = errors.New("test error2")

		// todo: check for an exact error
		parsedErr := parseSubcollectorErrors(errs)
		if parsedErr == nil {
			t.Error("Expected parsedErr to contain an error, received nil")
		}
	})
}
