package nodestats

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

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

type errorMockClient struct{}

func (m *errorMockClient) GetNodeInfo(ctx context.Context) (*responses.NodeInfoResponse, error) {
	return nil, nil
}

func (m *errorMockClient) GetNodeStats(ctx context.Context) (*responses.NodeStatsResponse, error) {
	return nil, errors.New("could not connect to instance")
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
		"logstash_stats_pipeline_up",
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
		"logstash_stats_process_cpu_load_average_1m",
		"logstash_stats_process_cpu_load_average_5m",
		"logstash_stats_process_cpu_load_average_15m",
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

func TestCollectError(t *testing.T) {
	collector := NewNodestatsCollector(&errorMockClient{})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	ch := make(chan prometheus.Metric)

	go func() {
		for range ch {
			// simulate reading from the channel
		}
	}()

	err := collector.Collect(ctx, ch)
	close(ch)

	if err == nil {
		t.Error("Expected err not to be nil")
	}
}

func TestIsPipelineHealthy(t *testing.T) {
	collector := NewPipelineSubcollector()
	tests := []struct {
		name     string
		stats    responses.PipelineReloadResponse
		expected float64
	}{
		{
			name: "Both timestamps nil",
			stats: responses.PipelineReloadResponse{
				LastFailureTimestamp:  nil,
				LastSuccessTimestamp: nil,
			},
			expected: 1,
		},
		{
			name: "Failure timestamp set",
			stats: responses.PipelineReloadResponse{
				LastFailureTimestamp:  &time.Time{},
				LastSuccessTimestamp: nil,
			},
			expected: 0,
		},
		{
			name: "Success timestamp earlier than failure timestamp",
			stats: responses.PipelineReloadResponse{
				LastFailureTimestamp:  &time.Time{},
				LastSuccessTimestamp: func() *time.Time { t := time.Time{}.Add(-1 * time.Hour); return &t }(),
			},
			expected: 0,
		},
		{
			name: "Success timestamp later than failure timestamp",
			stats: responses.PipelineReloadResponse{
				LastFailureTimestamp:  &time.Time{},
				LastSuccessTimestamp: func() *time.Time { t := time.Time{}.Add(1 * time.Hour); return &t }(),
			},
			expected: 1,
		},
		{
			name: "Missing fields, assume healthy",
			stats: responses.PipelineReloadResponse{},
			expected: 1,
		},
	}

	// Run test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collector.isPipelineHealthy(tt.stats)
			if result != tt.expected {
				t.Errorf("Expected %v, but got %v", tt.expected, result)
				return
			}
		})
	}

}