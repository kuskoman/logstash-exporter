package nodeinfo

import (
	"encoding/json"
	"os"
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

func TestCollectNotNil(t *testing.T) {
	collector := NewNodestatsCollector(&mockClient{})
	ch := make(chan prometheus.Metric)

	go func() {
		err := collector.Collect(ch)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	}()

	for i := 0; i < 6; i++ {
		metric := <-ch
		if metric == nil {
			t.Errorf("Expected metric %s to be not nil", metric.Desc().String())
		}
	}
}
