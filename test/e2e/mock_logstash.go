package e2e

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
)

// MockLogstashServer represents a mock Logstash HTTP server for testing
type MockLogstashServer struct {
	Server      *httptest.Server
	NodeInfoJSON  []byte
	NodeStatsJSON []byte
	RequestCount  int
	FailNextRequest bool
}

// NewMockLogstashServer creates a new mock Logstash server that serves fixture data
func NewMockLogstashServer() (*MockLogstashServer, error) {
	mock := &MockLogstashServer{}

	// Load fixtures from the fixtures directory
	fixturesDir := filepath.Join("..", "..", "fixtures")

	nodeInfo, err := os.ReadFile(filepath.Join(fixturesDir, "node_info.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to read node_info.json: %w", err)
	}
	mock.NodeInfoJSON = nodeInfo

	nodeStats, err := os.ReadFile(filepath.Join(fixturesDir, "node_stats.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to read node_stats.json: %w", err)
	}
	mock.NodeStatsJSON = nodeStats

	// Create HTTP test server
	mux := http.NewServeMux()

	// Logstash API endpoints
	// Note: /_node/stats must be registered before / to avoid being caught by the root handler
	mux.HandleFunc("/_node/stats", func(w http.ResponseWriter, r *http.Request) {
		mock.RequestCount++

		if mock.FailNextRequest {
			mock.FailNextRequest = false
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(mock.NodeStatsJSON)
	})

	// Root endpoint serves node info
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Only handle exact root path for node info
		if r.URL.Path != "/" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		mock.RequestCount++

		if mock.FailNextRequest {
			mock.FailNextRequest = false
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(mock.NodeInfoJSON)
	})

	mock.Server = httptest.NewServer(mux)

	return mock, nil
}

// URL returns the base URL of the mock server
func (m *MockLogstashServer) URL() string {
	return m.Server.URL
}

// Close shuts down the mock server
func (m *MockLogstashServer) Close() {
	m.Server.Close()
}

// Reset resets the request counter and fail flag
func (m *MockLogstashServer) Reset() {
	m.RequestCount = 0
	m.FailNextRequest = false
}

// SetNodeInfo sets custom node info JSON response
func (m *MockLogstashServer) SetNodeInfo(data []byte) {
	m.NodeInfoJSON = data
}

// SetNodeStats sets custom node stats JSON response
func (m *MockLogstashServer) SetNodeStats(data []byte) {
	m.NodeStatsJSON = data
}
