package nodestats

import (
	"encoding/json"
	"math"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	"github.com/kuskoman/logstash-exporter/internal/fetcher/responses"
	"github.com/kuskoman/logstash-exporter/internal/prometheus_helper"
)

func collectPipelineMetricNames(t *testing.T, pipeStats *responses.SinglePipelineResponse) []string {
	t.Helper()

	ch := make(chan prometheus.Metric, 200)
	NewPipelineSubcollector().Collect(pipeStats, "main", ch, "http://localhost:9600", "logstash")
	close(ch)

	var names []string
	for metric := range ch {
		name, err := prometheus_helper.ExtractFqName(metric.Desc().String())
		if err != nil {
			t.Fatalf("failed to extract fqName: %v", err)
		}
		names = append(names, name)
	}
	return names
}

func TestPipelineSubcollector_ReloadTimestamps(t *testing.T) {
	t.Parallel()

	timestampMetrics := []string{
		"logstash_stats_pipeline_reloads_last_success_timestamp",
		"logstash_stats_pipeline_reloads_last_failure_timestamp",
	}

	t.Run("exports_the_reload_timestamps_when_logstash_reports_them", func(t *testing.T) {
		t.Parallel()

		success, failure := time.Now(), time.Now().Add(-time.Hour)
		names := collectPipelineMetricNames(t, &responses.SinglePipelineResponse{
			Reloads: responses.PipelineReloadResponse{LastSuccessTimestamp: &success, LastFailureTimestamp: &failure},
		})

		for _, metric := range timestampMetrics {
			if !slices.Contains(names, metric) {
				t.Errorf("expected metric %s to be exported", metric)
			}
		}
	})

	t.Run("omits_the_reload_timestamps_before_the_first_reload", func(t *testing.T) {
		t.Parallel()

		names := collectPipelineMetricNames(t, &responses.SinglePipelineResponse{})

		for _, metric := range timestampMetrics {
			if slices.Contains(names, metric) {
				t.Errorf("expected metric %s not to be exported", metric)
			}
		}
	})
}

func TestIsPipelineHealthy(t *testing.T) {
	t.Parallel()
	collector := NewPipelineSubcollector()

	now := time.Now()
	oneHourBefore := now.Add(-1 * time.Hour)
	oneHourAfter := now.Add(1 * time.Hour)

	tests := []struct {
		name     string
		stats    responses.PipelineReloadResponse
		expected float64
	}{
		{
			name: "Both timestamps nil",
			stats: responses.PipelineReloadResponse{
				LastFailureTimestamp: nil,
				LastSuccessTimestamp: nil,
			},
			expected: CollectorHealthy,
		},
		{
			name: "Failure timestamp set",
			stats: responses.PipelineReloadResponse{
				LastFailureTimestamp: &now,
				LastSuccessTimestamp: nil,
			},
			expected: CollectorUnhealthy,
		},
		{
			name: "Success timestamp earlier than failure timestamp",
			stats: responses.PipelineReloadResponse{
				LastFailureTimestamp: &now,
				LastSuccessTimestamp: &oneHourBefore,
			},
			expected: CollectorUnhealthy,
		},
		{
			name: "Success timestamp later than failure timestamp",
			stats: responses.PipelineReloadResponse{
				LastFailureTimestamp: &now,
				LastSuccessTimestamp: &oneHourAfter,
			},
			expected: CollectorHealthy,
		},
		{
			name:     "Missing fields, assume healthy",
			stats:    responses.PipelineReloadResponse{},
			expected: CollectorHealthy,
		},
		{
			name: "Success timestamp equal to failure timestamp",
			stats: responses.PipelineReloadResponse{
				LastFailureTimestamp: &now,
				LastSuccessTimestamp: &now,
			},
			expected: CollectorHealthy,
		},
	}

	// Run test cases
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			localTestCase := testCase
			t.Parallel()
			result := collector.isPipelineHealthy(localTestCase.stats)
			if result != localTestCase.expected {
				t.Errorf("expected %v, but got %v", localTestCase.expected, result)
				return
			}
		})
	}
}

func TestPipelineSubcollector_PluginFlow(t *testing.T) {
	t.Parallel()

	pipelineJSON := `{
		"plugins": {
			"filters": [
				{"id": "busy", "name": "ruby", "flow": {
					"worker_utilization": {"current": 97.5, "lifetime": 96.2},
					"worker_millis_per_event": {"current": "Infinity", "lifetime": 112.1}
				}},
				{"id": "idle", "name": "drop", "flow": {
					"worker_utilization": {"current": 0, "lifetime": 0},
					"worker_millis_per_event": {}
				}}
			],
			"outputs": [
				{"id": "es", "name": "elasticsearch", "flow": {
					"worker_utilization": {"current": 2.25, "lifetime": 1.15},
					"worker_millis_per_event": {"current": 2.68, "lifetime": 2.68}
				}}
			]
		}
	}`

	var pipeStats responses.SinglePipelineResponse
	if err := json.Unmarshal([]byte(pipelineJSON), &pipeStats); err != nil {
		t.Fatalf("failed to parse pipeline: %v", err)
	}

	ch := make(chan prometheus.Metric, 200)
	NewPipelineSubcollector().Collect(&pipeStats, "main", ch, "http://localhost:9600", "logstash")
	close(ch)

	// value by "metric plugin_id"
	values := map[string]float64{}
	for metric := range ch {
		name, err := prometheus_helper.ExtractFqName(metric.Desc().String())
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(name, "logstash_stats_pipeline_plugin_flow_") {
			continue
		}
		var written dto.Metric
		if err := metric.Write(&written); err != nil {
			t.Fatal(err)
		}
		for _, label := range written.GetLabel() {
			if label.GetName() == "plugin_id" {
				values[strings.TrimPrefix(name, "logstash_stats_pipeline_plugin_flow_")+" "+label.GetValue()] = written.GetGauge().GetValue()
			}
		}
	}

	expected := map[string]float64{
		"worker_utilization_current busy":       97.5,
		"worker_utilization_lifetime busy":      96.2,
		"worker_millis_per_event_current busy":  math.Inf(1),
		"worker_millis_per_event_lifetime busy": 112.1,
		"worker_utilization_current idle":       0,
		"worker_utilization_lifetime idle":      0,
		"worker_utilization_current es":         2.25,
		"worker_utilization_lifetime es":        1.15,
		"worker_millis_per_event_current es":    2.68,
		"worker_millis_per_event_lifetime es":   2.68,
	}

	for key, value := range expected {
		actual, ok := values[key]
		if !ok {
			t.Errorf("expected %s to be exported", key)
		} else if actual != value {
			t.Errorf("expected %s to be %v, got %v", key, value, actual)
		}
	}
	for key := range values {
		if _, ok := expected[key]; !ok {
			t.Errorf("did not expect %s to be exported", key)
		}
	}
}

func TestPipelineSubcollector_Settings(t *testing.T) {
	t.Parallel()

	settings := []string{
		"logstash_stats_pipeline_workers",
		"logstash_stats_pipeline_batch_size",
		"logstash_stats_pipeline_batch_delay",
	}

	t.Run("exports_the_pipeline_settings_when_logstash_reports_them", func(t *testing.T) {
		t.Parallel()

		workers, batchSize, batchDelay := 8, 125, 50
		names := collectPipelineMetricNames(t, &responses.SinglePipelineResponse{
			Settings: responses.PipelineSettingsResponse{Workers: &workers, BatchSize: &batchSize, BatchDelay: &batchDelay},
		})

		for _, metric := range settings {
			if !slices.Contains(names, metric) {
				t.Errorf("expected metric %s to be exported", metric)
			}
		}
	})

	t.Run("omits_the_pipeline_settings_for_older_logstash_versions", func(t *testing.T) {
		t.Parallel()

		names := collectPipelineMetricNames(t, &responses.SinglePipelineResponse{})

		for _, metric := range settings {
			if slices.Contains(names, metric) {
				t.Errorf("expected metric %s not to be exported", metric)
			}
		}
	})
}
