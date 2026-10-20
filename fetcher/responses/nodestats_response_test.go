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
	type Data struct {
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

// TestNodeStatsAcceptsInfinityInAllFlowValues checks that every flow value
// in the node stats response accepts "Infinity" and "-Infinity". Logstash
// reports them as strings, for example when no events were processed yet,
// and a single plain float64 field makes the whole response fail to parse.
func TestNodeStatsAcceptsInfinityInAllFlowValues(t *testing.T) {
	fixtureContent, err := os.ReadFile("../../fixtures/node_stats.json")
	if err != nil {
		t.Fatalf("Error reading fixture file: %v", err)
	}

	for _, value := range []string{"Infinity", "-Infinity"} {
		t.Run(value, func(t *testing.T) {
			var document any
			if err := json.Unmarshal(fixtureContent, &document); err != nil {
				t.Fatalf("Error unmarshalling fixture: %v", err)
			}

			replaced := replaceFlowNumbers(document, false, value)
			if replaced == 0 {
				t.Fatal("expected the fixture to contain flow values")
			}

			modified, err := json.Marshal(document)
			if err != nil {
				t.Fatalf("Error marshalling modified fixture: %v", err)
			}

			var target responses.NodeStatsResponse
			if err := json.Unmarshal(modified, &target); err != nil {
				t.Errorf("expected %d flow values set to %q to be accepted, got: %v", replaced, value, err)
			}
		})
	}
}

// replaceFlowNumbers replaces every number inside a "flow" object with value
// and returns how many numbers were replaced
func replaceFlowNumbers(node any, inFlow bool, value string) int {
	replaced := 0
	switch typed := node.(type) {
	case map[string]any:
		for key, child := range typed {
			if _, isNumber := child.(float64); isNumber && inFlow {
				typed[key] = value
				replaced++
				continue
			}
			replaced += replaceFlowNumbers(child, inFlow || key == "flow", value)
		}
	case []any:
		for _, child := range typed {
			replaced += replaceFlowNumbers(child, inFlow, value)
		}
	}
	return replaced
}
