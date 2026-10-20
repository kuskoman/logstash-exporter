package e2e

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

const (
	nodeInfoPath  = "/"
	nodeStatsPath = "/_node/stats"
)

// RecordedRequest is a request received by the mock Logstash server
type RecordedRequest struct {
	Path     string
	Username string
	Password string
	HasAuth  bool
}

// MockLogstashServer is a mock Logstash HTTP API serving fixture data.
// It is safe for concurrent use.
type MockLogstashServer struct {
	Server *httptest.Server

	mu         sync.Mutex
	responses  map[string][]byte
	statusCode int
	delay      time.Duration
	failNext   int
	requests   []RecordedRequest
}

// MockOption configures a MockLogstashServer
type MockOption func(*mockSettings)

type mockSettings struct {
	tls bool
}

// WithMockTLS serves the mock Logstash API over HTTPS
func WithMockTLS() MockOption {
	return func(s *mockSettings) { s.tls = true }
}

// NewMockLogstashServer starts a mock Logstash server serving the fixtures
// from the repository's fixtures directory. It is closed when the test ends.
func NewMockLogstashServer(t *testing.T, opts ...MockOption) *MockLogstashServer {
	t.Helper()

	settings := &mockSettings{}
	for _, opt := range opts {
		opt(settings)
	}

	mock := &MockLogstashServer{
		responses: map[string][]byte{
			nodeInfoPath:  readFixture(t, "node_info.json"),
			nodeStatsPath: readFixture(t, "node_stats.json"),
		},
		statusCode: http.StatusOK,
	}

	handler := http.HandlerFunc(mock.serveHTTP)
	if settings.tls {
		mock.Server = httptest.NewTLSServer(handler)
	} else {
		mock.Server = httptest.NewServer(handler)
	}
	t.Cleanup(mock.Server.Close)

	return mock
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("..", "..", "fixtures", name))
	if err != nil {
		t.Fatalf("failed to read fixture %s: %v", name, err)
	}
	return data
}

func (m *MockLogstashServer) serveHTTP(w http.ResponseWriter, r *http.Request) {
	username, password, hasAuth := r.BasicAuth()

	m.mu.Lock()
	m.requests = append(m.requests, RecordedRequest{
		Path:     r.URL.Path,
		Username: username,
		Password: password,
		HasAuth:  hasAuth,
	})
	body, found := m.responses[r.URL.Path]
	statusCode := m.statusCode
	if m.failNext > 0 {
		m.failNext--
		statusCode = http.StatusInternalServerError
	}
	delay := m.delay
	m.mu.Unlock()

	if delay > 0 {
		select {
		case <-time.After(delay):
		case <-r.Context().Done():
			return
		}
	}

	if !found {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if statusCode != http.StatusOK {
		w.WriteHeader(statusCode)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(body); err != nil {
		fmt.Printf("mock logstash: error writing response: %v\n", err)
	}
}

// URL returns the base URL of the mock server
func (m *MockLogstashServer) URL() string {
	return m.Server.URL
}

// Close shuts down the mock server, making the instance unreachable
func (m *MockLogstashServer) Close() {
	m.Server.Close()
}

// SetResponse replaces the body served for the given path
func (m *MockLogstashServer) SetResponse(path string, body []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses[path] = body
}

// SetStatusCode makes every response use the given status code
func (m *MockLogstashServer) SetStatusCode(code int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.statusCode = code
}

// SetDelay delays every response by the given duration
func (m *MockLogstashServer) SetDelay(delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.delay = delay
}

// FailNext makes the next n requests return HTTP 500
func (m *MockLogstashServer) FailNext(n int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failNext = n
}

// Requests returns a copy of all requests received so far
func (m *MockLogstashServer) Requests() []RecordedRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]RecordedRequest(nil), m.requests...)
}

// RequestCount returns the number of requests received for the given path,
// or for all paths when path is empty
func (m *MockLogstashServer) RequestCount(path string) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	if path == "" {
		return len(m.requests)
	}

	count := 0
	for _, req := range m.requests {
		if req.Path == path {
			count++
		}
	}
	return count
}

// ResetRequests forgets all recorded requests
func (m *MockLogstashServer) ResetRequests() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requests = nil
}
