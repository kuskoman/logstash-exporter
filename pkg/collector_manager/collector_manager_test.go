package collector_manager

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kuskoman/logstash-exporter/pkg/config"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
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

	t.Run("can_be_created_concurrently", func(t *testing.T) {
		t.Parallel()

		instances := []*config.LogstashInstance{{Host: "http://localhost:9600"}}

		var wg sync.WaitGroup
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if cm := NewCollectorManager(instances, httpTimeout); cm == nil {
					t.Error("expected collector manager to be initialized, got nil")
				}
			}()
		}
		wg.Wait()
	})

	t.Run("labels_build_info_with_the_exporter_version", func(t *testing.T) {
		// Not parallel: changes the global version variables
		originalVersion, originalSemanticVersion, originalCommit := config.Version, config.SemanticVersion, config.GitCommit
		t.Cleanup(func() {
			config.Version, config.SemanticVersion, config.GitCommit = originalVersion, originalSemanticVersion, originalCommit
		})

		tests := []struct {
			name            string
			version         string
			semanticVersion string
			expectedVersion string
		}{
			{name: "uses_the_semantic_version", version: "master", semanticVersion: "v2.0.0-3-gabc1234", expectedVersion: "v2.0.0-3-gabc1234"},
			{name: "falls_back_to_the_version", version: "v2.0.0", semanticVersion: "unknown", expectedVersion: "v2.0.0"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				config.Version, config.SemanticVersion, config.GitCommit = tt.version, tt.semanticVersion, "abc1234"

				registry := prometheus.NewRegistry()
				registry.MustRegister(newBuildInfoCollector())
				families, err := registry.Gather()
				if err != nil {
					t.Fatal(err)
				}

				labels := map[string]string{}
				for _, pair := range families[0].GetMetric()[0].GetLabel() {
					labels[pair.GetName()] = pair.GetValue()
				}
				if labels["version"] != tt.expectedVersion {
					t.Errorf("expected version label %q, got %q", tt.expectedVersion, labels["version"])
				}
				if labels["revision"] != "abc1234" {
					t.Errorf("expected revision label %q, got %q", "abc1234", labels["revision"])
				}
			})
		}
	})

	t.Run("registers_build_info_in_the_default_registry", func(t *testing.T) {
		t.Parallel()

		NewCollectorManager([]*config.LogstashInstance{{Host: "http://localhost:9600"}}, httpTimeout)

		families, err := prometheus.DefaultGatherer.Gather()
		if err != nil {
			t.Fatal(err)
		}
		for _, family := range families {
			if family.GetName() == "logstash_exporter_build_info" {
				return
			}
		}
		t.Error("expected logstash_exporter_build_info to be registered")
	})
}

func TestCollect(t *testing.T) {
	t.Parallel()

	t.Run("should_report_failed_scrape_duration_when_collector_returns_error", func(t *testing.T) {
		t.Parallel()

		cm := &CollectorManager{
			collectors: map[string]Collector{
				"mock": newMockCollector(true),
			},
			scrapeDurations: getScrapeDurationsCollector(),
		}

		metrics := collectAll(cm)

		if len(metrics) != 1 {
			t.Fatalf("expected only the scrape duration metric, got %d metrics", len(metrics))
		}
		assertScrapeDuration(t, metrics[0], "mock", "error")
	})

	t.Run("should_send_collector_metrics_and_scrape_duration", func(t *testing.T) {
		t.Parallel()

		cm := &CollectorManager{
			collectors: map[string]Collector{
				"mock": newMockCollector(false),
			},
			scrapeDurations: getScrapeDurationsCollector(),
		}

		metrics := collectAll(cm)

		if len(metrics) != 2 {
			t.Fatalf("expected the collector metric and the scrape duration, got %d metrics", len(metrics))
		}

		expectedDesc := "Desc{fqName: \"mock_metric\", help: \"mock metric description\", unit: \"\", constLabels: {}, variableLabels: {}}"
		if desc := metrics[0].Desc().String(); desc != expectedDesc {
			t.Errorf("expected metric description to be %q, got %q", expectedDesc, desc)
		}
		assertScrapeDuration(t, metrics[1], "mock", "success")
	})
}

// collectAll runs Collect and returns every metric it sent, in order
func collectAll(cm *CollectorManager) []prometheus.Metric {
	ch := make(chan prometheus.Metric)
	go func() {
		cm.Collect(ch)
		close(ch)
	}()

	var metrics []prometheus.Metric
	for metric := range ch {
		metrics = append(metrics, metric)
	}
	return metrics
}

func assertScrapeDuration(t *testing.T, metric prometheus.Metric, collector, result string) {
	t.Helper()

	if !strings.Contains(metric.Desc().String(), "logstash_exporter_scrape_duration_seconds") {
		t.Fatalf("expected the scrape duration metric, got %s", metric.Desc())
	}

	var written dto.Metric
	if err := metric.Write(&written); err != nil {
		t.Fatalf("failed to write metric: %v", err)
	}

	labels := map[string]string{}
	for _, pair := range written.GetLabel() {
		labels[pair.GetName()] = pair.GetValue()
	}
	if labels["collector"] != collector || labels["result"] != result {
		t.Errorf("expected collector=%q result=%q, got %v", collector, result, labels)
	}
	if count := written.GetSummary().GetSampleCount(); count != 1 {
		t.Errorf("expected one observed scrape, got %d", count)
	}
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
	
	expectedDesc := "Desc{fqName: \"logstash_exporter_scrape_duration_seconds\", help: \"logstash_exporter: Duration of a scrape job.\", unit: \"\", constLabels: {}, variableLabels: {collector,result}}"
	if desc.String() != expectedDesc {
		t.Errorf("expected metric description to be %q, got %q", expectedDesc, desc.String())
	}
}
