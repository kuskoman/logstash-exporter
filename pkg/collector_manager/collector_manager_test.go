package collector_manager

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/kuskoman/logstash-exporter/pkg/config"
	"github.com/prometheus/client_golang/prometheus"
)

const httpTimeout = 2 * time.Second

func TestNewCollectorManager(t *testing.T) {
	t.Parallel()

	t.Run("multiple endpoints", func(t *testing.T) {
		endpoint1 := &config.LogstashServer{
			Host: "http://localhost:9600",
		}

		endpoint2 := &config.LogstashServer{
			Host: "http://localhost:9601",
		}

		mockEndpoints := []*config.LogstashServer{endpoint1, endpoint2}
		cm := NewCollectorManager(mockEndpoints, httpTimeout)

		if cm == nil {
			t.Error("expected collector manager to be initialized")
		}
	})

	// prometheus has a global state, so we cannot register the same collector twice, therefore there is no single endpoint test
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
			t.Error("expected no metric to be sent to the channel")
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
			t.Errorf("expected metric description to be '%s', got %s", expectedDesc, desc.String())
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
		t.Errorf("expected metric description to be '%s', got %s", expectedDesc, desc.String())
	}
}
