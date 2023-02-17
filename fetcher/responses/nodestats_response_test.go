package responses_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/kuskoman/logstash-exporter/fetcher/responses"
)

func TestNodeStatsResponseStructure(t *testing.T) {
	fixtureContent, err := os.ReadFile("../../fixtures/node_stats.json")
	if err != nil {
		t.Fatalf("Error reading fixture file: %v", err)
	}

	var target responses.NodeStatsResponse
	err = json.Unmarshal(fixtureContent, &target)
	if err != nil {
		t.Fatalf("Error unmarshalling fixture: %v", err)
	}

	snaps.MatchSnapshot(t, "Unmarshalled NodestatsResponse", target)
}
