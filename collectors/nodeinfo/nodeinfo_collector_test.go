package nodeinfo

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/kuskoman/logstash-exporter/fetcher/responses"
	"github.com/prometheus/client_golang/prometheus"
)

type mockClient struct{}

func (m *mockClient) GetNodeInfo() (*responses.NodeInfoResponse, error) {
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

func (m *mockClient) GetNodeStats() (*responses.NodestatsResponse, error) {
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
		"logstash_info_build",
		"logstash_info_node",
		"logstash_info_pipeline_batch_delay",
		"logstash_info_pipeline_batch_size",
		"logstash_info_pipeline_workers",
		"logstash_info_status",
		"logstash_info_up",
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
