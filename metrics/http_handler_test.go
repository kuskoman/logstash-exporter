package metrics_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/kuskoman/logstash-exporter/collector/nodeinfo"
	"github.com/kuskoman/logstash-exporter/metrics"
)

type mockHTTPHandler struct {
	Response *http.Response
	Error    error
}

func (m *mockHTTPHandler) Get() (*http.Response, error) {
	return m.Response, m.Error
}

func TestGetMetrics(t *testing.T) {
	// Create a sample JSON response
	sampleResponse := `{"host": "localhost", "version": "7.0.0", "http_address": "127.0.0.1:9600", "id": "example", "name": "example-node"}`

	// Create a test response with the sample JSON response
	testResponse := &http.Response{
		StatusCode: http.StatusOK,
		Body:       nopCloser{bytes.NewBufferString(sampleResponse)},
	}

	// Create a mockHTTPHandler with the test response
	mock := &mockHTTPHandler{
		Response: testResponse,
		Error:    nil,
	}

	// Create a target struct to hold the response
	var target nodeinfo.NodeInfoResponse

	// Call the getMetrics function with the mockHTTPHandler
	err := metrics.GetMetrics(mock, &target)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Unmarshal the sample JSON response to a map
	var expected map[string]interface{}
	json.Unmarshal([]byte(sampleResponse), &expected)

	// Compare the target struct and the expected map
	if target.Host != expected["host"] {
		t.Errorf("Expected host %v, got %v", expected["host"], target.Host)
	}
	if target.Version != expected["version"] {
		t.Errorf("Expected version %v, got %v", expected["version"], target.Version)
	}
	if target.HTTPAddress != expected["http_address"] {
		t.Errorf("Expected http_address %v, got %v", expected["http_address"], target.HTTPAddress)
	}
	if target.ID != expected["id"] {
		t.Errorf("Expected id %v, got %v", expected["id"], target.ID)
	}
	if target.Name != expected["name"] {
		t.Errorf("Expected name %v, got %v", expected["name"], target.Name)
	}
}

type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() error { return nil }
