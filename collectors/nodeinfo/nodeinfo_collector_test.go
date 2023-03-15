package nodeinfo

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/kuskoman/logstash-exporter/fetcher/responses"
	"github.com/kuskoman/logstash-exporter/prometheus_helper"
)

type mockClient struct{}

func (m *mockClient) GetNodeInfo(ctx context.Context) (*responses.NodeInfoResponse, error) {
	b, err := os.ReadFile("../../fixtures/node_info.json")
	if err != nil {
		return nil, err
	}

	var nodeInfo responses.NodeInfoResponse
	err = json.Unmarshal(b, &nodeInfo)
	if err != nil {
		return nil, err
	}

	return &nodeInfo, nil
}

func (m *mockClient) GetNodeStats(ctx context.Context) (*responses.NodeStatsResponse, error) {
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

	expectedMetrics := []string{
		"logstash_info_build",
		"logstash_info_node",
		"logstash_info_pipeline_batch_delay",
		"logstash_info_pipeline_batch_size",
		"logstash_info_pipeline_workers",
		"logstash_info_status",
		"logstash_info_up",
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

	for _, expectedMetric := range expectedMetrics {
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
