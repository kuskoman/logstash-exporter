package nodeinfo

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/kuskoman/logstash-exporter/internal/fetcher/logstash_client"
	"github.com/kuskoman/logstash-exporter/internal/fetcher/responses"
	"github.com/kuskoman/logstash-exporter/internal/prometheus_helper"
)

type mockClient struct{}

func (m *mockClient) GetNodeInfo(ctx context.Context) (*responses.NodeInfoResponse, error) {
	b, err := os.ReadFile("../../../fixtures/node_info.json")
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

func (m *mockClient) GetEndpoint() string {
	return ""
}

func (m *mockClient) Name() string {
	return ""
}

type errorMockClient struct{}

func (m *errorMockClient) GetNodeInfo(ctx context.Context) (*responses.NodeInfoResponse, error) {
	return nil, errors.New("could not connect to instance")
}

func (m *errorMockClient) GetNodeStats(ctx context.Context) (*responses.NodeStatsResponse, error) {
	return nil, nil
}

func (m *errorMockClient) GetEndpoint() string {
	return ""
}

func (m *errorMockClient) Name() string {
	return ""
}

func TestCollectNotNil(t *testing.T) {
	runTest := func(t *testing.T, clients []logstash_client.Client) {
		collector := NewNodeinfoCollector(clients)
		ch := make(chan prometheus.Metric)
		ctx := context.Background()

		go func() {
			err := collector.Collect(ctx, ch)
			if err != nil {
				t.Errorf("expected no error, got %v", err)
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
				t.Error("expected metric not to be nil")
			}

			foundMetricDesc := metric.Desc().String()
			foundMetricFqName, err := prometheus_helper.ExtractFqName(foundMetricDesc)
			if err != nil {
				t.Errorf("failed to extract fqName from metric %s", foundMetricDesc)
			}

			foundMetrics = append(foundMetrics, foundMetricFqName)
		}

		for _, expectedMetric := range expectedMetrics {
			if !slices.Contains(foundMetrics, expectedMetric) {
				t.Errorf("expected metric %s to be found", expectedMetric)
			}
		}
	}

	t.Run("single client", func(t *testing.T) {
		t.Parallel()

		runTest(t, []logstash_client.Client{&mockClient{}})
	})

	t.Run("multiple clients", func(t *testing.T) {
		t.Parallel()

		runTest(t, []logstash_client.Client{&mockClient{}, &mockClient{}})
	})
}

func TestCollectError(t *testing.T) {
	runTest := func(t *testing.T, clients []logstash_client.Client) {
		collector := NewNodeinfoCollector(clients)
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
			t.Error("expected err not to be nil")
		}
	}

	t.Run("single faulty client", func(t *testing.T) {
		t.Parallel()

		runTest(t, []logstash_client.Client{&errorMockClient{}})
	})

	t.Run("multiple faulty clients", func(t *testing.T) {
		t.Parallel()

		runTest(t, []logstash_client.Client{&errorMockClient{}, &errorMockClient{}})
	})

	t.Run("multiple clients, one faulty", func(t *testing.T) {
		t.Parallel()

		runTest(t, []logstash_client.Client{&mockClient{}, &errorMockClient{}})
	})
}

func TestGetUpStatus(t *testing.T) {
	clients := []logstash_client.Client{&mockClient{}}
	collector := NewNodeinfoCollector(clients)

	tests := []struct {
		name     string
		nodeInfo *responses.NodeInfoResponse
		err      error
		expected int
	}{
		{
			name:     "nil error and green status",
			nodeInfo: &responses.NodeInfoResponse{Status: "green"},
			err:      nil,
			expected: 1,
		},
		{
			name:     "nil error and yellow status",
			nodeInfo: &responses.NodeInfoResponse{Status: "yellow"},
			err:      nil,
			expected: 1,
		},
		{
			name:     "nil error and red status",
			nodeInfo: &responses.NodeInfoResponse{Status: "red"},
			err:      nil,
			expected: 0,
		},
		{
			name:     "error",
			nodeInfo: nil,
			err:      errors.New("test error"),
			expected: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			status := collector.getUpStatus(test.nodeInfo, test.err)

			if status != test.expected {
				t.Errorf("expected up value to be %v, got %v", test.expected, status)
			}
		})
	}
}
