package nodestats

import (
	"testing"
	"time"

	"github.com/kuskoman/logstash-exporter/fetcher/responses"
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
				t.Errorf("Expected %v, but got %v", localTestCase.expected, result)
				return
			}
		})
	}
}

func TestTruncatePluginId(t *testing.T) {
	testCases := []struct {
		input  string
		output string
	}{
		{"plain_2c897236-b1fd-42e6-ab7a-f468-b6e6-e404", "b6e6e404"},
		{"552b7810244be6259a4cc88fe34833088a23437c5ee9b4c788b2ec4e502c819f", "502c819f"},
		{"pipeline_custom_filter_foobar", "pipeline_custom_filter_foobar"},
		{"filter_0001", "filter_0001"},
	}

	for _, tc := range testCases {
		got := TruncatePluginId(tc.input)
		if got != tc.output {
			t.Errorf("TruncatePluginId(%v) = %v; want %v", tc.input, got, tc.output)
		}
	}
}
