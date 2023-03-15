package collectors

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

type mockCollector struct{}

func (m *mockCollector) Collect(ctx context.Context, ch chan<- prometheus.Metric) error {
	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc("mock_metric", "mock metric description", nil, nil),
		prometheus.GaugeValue,
		1,
	)
	return nil
}

func TestCollect(t *testing.T) {
	cm := &CollectorManager{
		collectors: map[string]Collector{
			"mock": &mockCollector{},
		},
		scrapeDurations: getScrapeDurationsCollector(),
	}

	ch := make(chan prometheus.Metric)
	go cm.Collect(ch)

	metric := <-ch

	desc := metric.Desc()
	expectedDesc := "Desc{fqName: \"mock_metric\", help: \"mock metric description\", constLabels: {}, variableLabels: []}"
	if desc.String() != expectedDesc {
		t.Errorf("Expected metric description to be '%s', got %s", expectedDesc, desc.String())
	}
}
