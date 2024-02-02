package nodestats

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/kuskoman/logstash-exporter/internal/fetcher/logstash_client"
	"github.com/kuskoman/logstash-exporter/internal/fetcher/responses"
	"github.com/kuskoman/logstash-exporter/internal/prometheus_helper"
)

type mockClient struct{}

func (m *mockClient) GetNodeStats(ctx context.Context) (*responses.NodeStatsResponse, error) {
	b, err := os.ReadFile("../../../fixtures/node_stats.json")
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

func (m *mockClient) GetEndpoint() string {
	return ""
}

type errorMockClient struct{}

func (m *errorMockClient) GetNodeInfo(ctx context.Context) (*responses.NodeInfoResponse, error) {
	return nil, nil
}

func (m *errorMockClient) GetNodeStats(ctx context.Context) (*responses.NodeStatsResponse, error) {
	return nil, errors.New("could not connect to instance")
}

func (m *errorMockClient) GetEndpoint() string {
	return ""
}

func TestCollectNotNil(t *testing.T) {
	t.Parallel()

	clients := []logstash_client.Client{&mockClient{}, &mockClient{}}
	collector := NewNodestatsCollector(clients)
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
		"logstash_stats_pipeline_reloads_last_success_timestamp",
		"logstash_stats_pipeline_reloads_last_failure_timestamp",
		"logstash_stats_pipeline_plugin_events_in",
		"logstash_stats_pipeline_plugin_events_out",
		"logstash_stats_pipeline_plugin_events_duration",
		"logstash_stats_pipeline_plugin_events_queue_push_duration",
		"logstash_stats_pipeline_plugin_documents_successes",
		"logstash_stats_pipeline_plugin_documents_non_retryable_failures",
		"logstash_stats_pipeline_plugin_bulk_requests_errors",
		"logstash_stats_pipeline_plugin_bulk_requests_responses",
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

		// todo: optimize this
		found := false
		for _, foundMetric := range foundMetrics {
			if foundMetric == foundMetricFqName {
				found = true
				break
			}
		}

		if !found {
			foundMetrics = append(foundMetrics, foundMetricFqName)
		}
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

func TestCollectsErrors(t *testing.T) {
	t.Parallel()

	testCollectorForClients := func(clients []logstash_client.Client) {
		collector := NewNodestatsCollector(clients)
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

	t.Run("should return an error if the only client returns an error", func(t *testing.T) {
		t.Parallel()
		testCollectorForClients([]logstash_client.Client{&errorMockClient{}})
	})

	t.Run("should return an error if one of the clients returns an error", func(t *testing.T) {
		t.Parallel()
		testCollectorForClients([]logstash_client.Client{&mockClient{}, &errorMockClient{}})
	})

	t.Run("should return an error if all clients return an error", func(t *testing.T) {
		t.Parallel()
		testCollectorForClients([]logstash_client.Client{&errorMockClient{}, &errorMockClient{}})
	})
}
