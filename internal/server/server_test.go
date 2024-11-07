package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kuskoman/logstash-exporter/pkg/config"
)

const defaultHttpTimeout = 2 * time.Second

func TestNewAppServer(t *testing.T) {
	defaultConfig := &config.Config{
		Server: config.ServerConfig{
			Host: "",
			Port: 8080,
		},
	}
	t.Run("test handling of /metrics endpoint", func(t *testing.T) {
		server := NewAppServer(defaultConfig)
		req, err := http.NewRequest("GET", "/metrics", nil)
		if err != nil {
			t.Fatal(fmt.Errorf("error creating request: %v", err))
		}
		rr := httptest.NewRecorder()
		server.Handler.ServeHTTP(rr, req)
		if status := rr.Code; status != http.StatusOK {
			t.Errorf("unexpected status code: got %v want %v", status, http.StatusOK)
		}
	})

	t.Run("test handling of / endpoint", func(t *testing.T) {
		server := NewAppServer(defaultConfig)
		req, err := http.NewRequest("GET", "/", nil)
		if err != nil {
			t.Fatal(fmt.Errorf("error creating request: %v", err))
		}
		rr := httptest.NewRecorder()
		server.Handler.ServeHTTP(rr, req)
		if status := rr.Code; status != http.StatusMovedPermanently {
			t.Errorf("unexpected status code: got %v want %v", status, http.StatusMovedPermanently)
		}
		if location := rr.Header().Get("Location"); location != "/metrics" {
			t.Errorf("unexpected redirect location: got %v want %v", location, "/metrics")
		}
	})

	t.Run("test handling of /healthcheck endpoint", func(t *testing.T) {
		cfg := &config.Config{
			Logstash: config.LogstashConfig{
				Servers: []*config.LogstashServer{
					{Host: "http://localhost:1234"},
				},
			},
			Server: defaultConfig.Server,
		}
		server := NewAppServer(cfg)
		req, err := http.NewRequest("GET", "/healthcheck", nil)
		if err != nil {
			t.Fatal(fmt.Errorf("error creating request: %v", err))
		}
		rr := httptest.NewRecorder()
		server.Handler.ServeHTTP(rr, req)
		// Assuming the localhost:1234 is not serving, so the healthcheck should return InternalServerError
		if status := rr.Code; status != http.StatusInternalServerError {
			t.Errorf("unexpected status code: got %v want %v", status, http.StatusInternalServerError)
		}
	})

	t.Run("test handling of /version endpoint", func(t *testing.T) {
		server := NewAppServer(defaultConfig)
		req, err := http.NewRequest("GET", "/version", nil)
		if err != nil {
			t.Fatal(fmt.Errorf("error creating request: %v", err))
		}
		rr := httptest.NewRecorder()
		server.Handler.ServeHTTP(rr, req)
		if status := rr.Code; status != http.StatusOK {
			t.Errorf("unexpected status code: got %v want %v", status, http.StatusOK)
		}
	})
}
