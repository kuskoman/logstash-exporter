package logstash_client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestGetNodeInfo(t *testing.T) {
	t.Run("should return a valid NodeInfoResponse when the request is successful", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fixtureBytes, err := loadFixture("node_info.json")
			if err != nil {
				t.Fatalf("error loading fixture: %s", err)
			}

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(fixtureBytes)
		}))
		defer ts.Close()

		client := NewClient(ts.URL)

		response, err := client.GetNodeInfo(context.Background())
		if err != nil {
			t.Fatalf("error getting node info: %s", err)
		}

		if response.Status != "green" {
			t.Fatalf("expected status to be properly read as green, got %s", response.Status)
		}
		// detailed checks are done in the responses package
	})
}

func TestGetNodeStats(t *testing.T) {
	t.Run("should return a valid NodestatsResponse when the request is successful", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fixtureBytes, err := loadFixture("node_stats.json")
			if err != nil {
				t.Fatalf("error loading fixture: %s", err)
			}

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(fixtureBytes)
		}))
		defer ts.Close()

		client := NewClient(ts.URL)

		response, err := client.GetNodeStats(context.Background())
		if err != nil {
			t.Fatalf("error getting node stats: %s", err)
		}

		if response.Status != "green" {
			t.Fatalf("expected status to be properly read as green, got %s", response.Status)
		}
		// detailed checks are done in the responses package
	})
}

func loadFixture(filename string) ([]byte, error) {
	fullPath := fmt.Sprintf("../../../fixtures/%s", filename)
	fixtureBytes, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, err
	}

	return fixtureBytes, nil
}
