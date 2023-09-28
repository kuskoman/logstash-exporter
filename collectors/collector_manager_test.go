package collectors

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestNewCollectorManager(t *testing.T) {
	mockEndpoint := "http://localhost:9600"
	cm := NewCollectorManager(mockEndpoint)

	if cm == nil {
		t.Error("Expected collector manager to be initialized")
	}
}

type mockCollector struct {
	shouldFail bool
}

func newMockCollector(shouldFail bool) *mockCollector {
	return &mockCollector{
		shouldFail: shouldFail,
	}
}

func (m *mockCollector) Collect(ctx context.Context, ch chan<- prometheus.Metric) error {
	if m.shouldFail {
		return errors.New("mock collector failed")
	}

	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc("mock_metric", "mock metric description", nil, nil),
		prometheus.GaugeValue,
		1,
	)
	return nil
}

func TestCollect(t *testing.T) {
	t.Run("should fail", func(t *testing.T) {
		cm := &CollectorManager{
			collectors: map[string]Collector{
				"mock": newMockCollector(true),
			},
			scrapeDurations: getScrapeDurationsCollector(),
		}

		ch := make(chan prometheus.Metric)

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			cm.Collect(ch)
			wg.Done()
		}()

		select {
		case <-ch:
			t.Error("Expected no metric to be sent to the channel")
		case <-func() chan struct{} {
			done := make(chan struct{})
			go func() {
				wg.Wait()
				close(done)
			}()
			return done
		}():
			// No metric was sent to the channel, which is expected.
		}
	})

	t.Run("should succeed", func(t *testing.T) {
		cm := &CollectorManager{
			collectors: map[string]Collector{
				"mock": newMockCollector(false),
			},
			scrapeDurations: getScrapeDurationsCollector(),
		}

		ch := make(chan prometheus.Metric)
		go cm.Collect(ch)

		metric := <-ch

		desc := metric.Desc()
		expectedDesc := "Desc{fqName: \"mock_metric\", help: \"mock metric description\", constLabels: {}, variableLabels: {}}"
		if desc.String() != expectedDesc {
			t.Errorf("Expected metric description to be '%s', got %s", expectedDesc, desc.String())
		}
	})
}

func TestDescribe(t *testing.T) {
	cm := &CollectorManager{
		collectors: map[string]Collector{
			"mock": newMockCollector(false),
		},
		scrapeDurations: getScrapeDurationsCollector(),
	}

	ch := make(chan *prometheus.Desc, 1)
	cm.Describe(ch)

	desc := <-ch
	expectedDesc := "Desc{fqName: \"logstash_exporter_scrape_duration_seconds\", help: \"logstash_exporter: Duration of a scrape job.\", constLabels: {}, variableLabels: {collector,result}}"
	if desc.String() != expectedDesc {
		t.Errorf("Expected metric description to be '%s', got %s", expectedDesc, desc.String())
	}
}
