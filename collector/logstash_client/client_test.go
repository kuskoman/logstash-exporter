package logstashclient

import (
	"testing"

	"github.com/kuskoman/logstash-exporter/httpclient"
)

func TestNewClientDefault(t *testing.T) {
	client := NewClient(nil)
	httpClient := client.httpClient
	if httpClient == nil {
		t.Error("Expected httpClient to be set")
	}

	_, ok := httpClient.(*httpclient.DefaultHTTPHandler)
	if !ok {
		t.Errorf("Expected httpClient to be of type DefaultHTTPHandler, got %T", httpClient)
	}
}
