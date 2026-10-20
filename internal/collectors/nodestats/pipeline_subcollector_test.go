package nodestats

import (
	"slices"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"

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
