package logstashclient

import (
	"net/http"
	"testing"

	"github.com/kuskoman/logstash-exporter/httpclient"
)

func TestNewClientDefaultHandler(t *testing.T) {
	client := NewClient(nil)
	httpClient := client.httpClient
	if httpClient == nil {
		t.Error("Expected httpClient to be set")
	}

	_, isDefaultHandler := httpClient.(*httpclient.DefaultHTTPHandler)
	if !isDefaultHandler {
		t.Error("Expected httpClient to be of type HTTPHandler")
	}
}

func TestNewClientCustomHandler(t *testing.T) {
	handler := &mockHTTPHandler{}
	client := NewClient(handler)
	httpClient := client.httpClient
	if httpClient == nil {
		t.Error("Expected httpClient to be set")
	}

	_, isCustomHandler := httpClient.(*mockHTTPHandler)
	if !isCustomHandler {
		t.Error("Expected httpClient to be of type *mockHTTPHandler")
	}

	_, isDefaultHandler := httpClient.(*httpclient.DefaultHTTPHandler)
	if isDefaultHandler {
		t.Error("Expected httpClient to not be of type *httpclient.DefaultHTTPHandler")
	}
}

type mockHTTPHandler struct{}

func (m *mockHTTPHandler) Get() (*http.Response, error) {
	return nil, nil
}
