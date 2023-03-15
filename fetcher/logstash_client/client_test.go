package logstash_client

import (
	"testing"
)

func TestNewClient(t *testing.T) {
	t.Run("should return a new client for the default endpoint", func(t *testing.T) {
		client := NewClient("")

		if client.(*DefaultClient).endpoint != defaultLogstashEndpoint {
			t.Errorf("expected endpoint to be %s, got %s", defaultLogstashEndpoint, client.(*DefaultClient).endpoint)
		}
	})

	t.Run("should return a new client for the given endpoint", func(t *testing.T) {
		endpoint := "http://localhost:9601"
		client := NewClient(endpoint)

		if client.(*DefaultClient).endpoint != endpoint {
			t.Errorf("expected endpoint to be %s, got %s", endpoint, client.(*DefaultClient).endpoint)
		}
	})
}
