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

func TestNodeStatsInfinityResponse(t *testing.T) {
	type Data  struct {
		F responses.InfinityFloat
	}
	var d Data

	okData := [][]byte{
		[]byte(`{"F": "Infinity"}`),
		[]byte(`{"F": "-Infinity"}`),
		[]byte(`{"F": 13.37}`),
	}
	for _, e := range okData {
		if err := json.Unmarshal(e, &d); err != nil {
			t.Errorf("unexpected error: %s", err)
		}
	}

	badData := [][]byte{
		[]byte(`{"F": "-infinity"}`),
		[]byte(`{"F": "--Infinity"}`),
		[]byte(`{"F": "13.3"}`),
	}
	for _, e := range badData {
		if err := json.Unmarshal(e, &d); err == nil {
			t.Errorf("expected error for: %s, got: %+v", string(e), d)
		}
	}


}
