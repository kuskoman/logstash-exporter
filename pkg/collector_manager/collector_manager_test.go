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

// mockCollector implements the Collector interface for testing
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

func TestNewCollectorManager(t *testing.T) {
	t.Parallel()

	t.Run("with_multiple_endpoints", func(t *testing.T) {
		t.Parallel()

		// Setup
		endpoint1 := &config.LogstashInstance{
			Host: "http://localhost:9600",
		}

		endpoint2 := &config.LogstashInstance{
			Host: "http://localhost:9601",
		}

		mockEndpoints := []*config.LogstashInstance{endpoint1, endpoint2}
		
		// Execute
		cm := NewCollectorManager(mockEndpoints, httpTimeout)

		// Verify
		if cm == nil {
			t.Errorf("expected collector manager to be initialized, got nil")
		}
	})

	// Note: prometheus has a global state, so we cannot register the same collector twice, 
	// therefore there is no single endpoint test
}

func TestCollect(t *testing.T) {
	t.Parallel()

	t.Run("should_fail_when_collector_returns_error", func(t *testing.T) {
		t.Parallel()

		// Setup
		cm := &CollectorManager{
			collectors: map[string]Collector{
				"mock": newMockCollector(true),
			},
			scrapeDurations: getScrapeDurationsCollector(),
		}

		ch := make(chan prometheus.Metric)

		var wg sync.WaitGroup
		wg.Add(1)
		
		// Execute
		go func() {
			cm.Collect(ch)
			wg.Done()
		}()

		// Verify
		select {
		case <-ch:
			t.Errorf("expected no metric to be sent to the channel, but received one")
		case <-func() chan struct{} {
			done := make(chan struct{})
			go func() {
				wg.Wait()
				close(done)
			}()
			return done
		}():
			// Success: No metric was sent to the channel, which is expected
		}
	})

	t.Run("should_succeed_when_collector_returns_metric", func(t *testing.T) {
		t.Parallel()

		// Setup
		cm := &CollectorManager{
			collectors: map[string]Collector{
				"mock": newMockCollector(false),
			},
			scrapeDurations: getScrapeDurationsCollector(),
		}

		ch := make(chan prometheus.Metric)
		
		// Execute
		go cm.Collect(ch)

		// Verify
		metric := <-ch
		if metric == nil {
			t.Errorf("expected metric to be non-nil")
			return
		}

		desc := metric.Desc()
		expectedDesc := "Desc{fqName: \"mock_metric\", help: \"mock metric description\", constLabels: {}, variableLabels: {}}"
		if desc.String() != expectedDesc {
			t.Errorf("expected metric description to be %q, got %q", expectedDesc, desc.String())
		}
	})
}

func TestDescribe(t *testing.T) {
	t.Parallel()

	// Setup
	cm := &CollectorManager{
		collectors: map[string]Collector{
			"mock": newMockCollector(false),
		},
		scrapeDurations: getScrapeDurationsCollector(),
	}

	ch := make(chan *prometheus.Desc, 1)
	
	// Execute
	cm.Describe(ch)

	// Verify
	desc := <-ch
	if desc == nil {
		t.Errorf("expected description to be non-nil")
		return
	}
	
	expectedDesc := "Desc{fqName: \"logstash_exporter_scrape_duration_seconds\", help: \"logstash_exporter: Duration of a scrape job.\", constLabels: {}, variableLabels: {collector,result}}"
	if desc.String() != expectedDesc {
		t.Errorf("expected metric description to be %q, got %q", expectedDesc, desc.String())
	}
}
