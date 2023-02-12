package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewAppServer(t *testing.T) {
	t.Run("Test handling of /metrics endpoint", func(t *testing.T) {
		server := NewAppServer("8080")
		req, err := http.NewRequest("GET", "/metrics", nil)
		if err != nil {
			t.Fatal(err)
		}
		rr := httptest.NewRecorder()
		server.Handler.ServeHTTP(rr, req)
		if status := rr.Code; status != http.StatusOK {
			t.Errorf("unexpected status code: got %v want %v", status, http.StatusOK)
		}
	})

	t.Run("Test handling of / endpoint", func(t *testing.T) {
		server := NewAppServer("8080")
		req, err := http.NewRequest("GET", "/", nil)
		if err != nil {
			t.Fatal(err)
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
}
