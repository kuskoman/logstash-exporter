package nodestats

import (
	"testing"
	"time"

	"github.com/kuskoman/logstash-exporter/internal/fetcher/responses"
)

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
