package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthCheck(t *testing.T) {
	runTest := func(mockStatus int, expectedStatus int) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(mockStatus)
		}))
		defer mockServer.Close()

		handler := getHealthCheck(mockServer.URL, defaultHttpTimeout)
		req, err := http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.Fatalf("Error creating request: %v", err)
		}
		rr := httptest.NewRecorder()

		handler(rr, req)

		if status := rr.Code; status != expectedStatus {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, expectedStatus)
		}
	}

	t.Run("500 status", func(t *testing.T) {
		runTest(http.StatusInternalServerError, http.StatusInternalServerError)
	})

	t.Run("200 status", func(t *testing.T) {
		runTest(http.StatusOK, http.StatusOK)
	})

	t.Run("404 status", func(t *testing.T) {
		runTest(http.StatusNotFound, http.StatusInternalServerError)
	})

	t.Run("no response", func(t *testing.T) {
		handler := getHealthCheck("http://localhost:12345", defaultHttpTimeout)
		req, err := http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.Fatalf("Error creating request: %v", err)
		}
		rr := httptest.NewRecorder()

		handler(rr, req)

		if status := rr.Code; status != http.StatusInternalServerError {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusInternalServerError)
		}
	})

	t.Run("invalid url", func(t *testing.T) {
		handler := getHealthCheck("http://localhost:96010:invalidurl", defaultHttpTimeout)
		req, err := http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.Fatalf("Error creating request: %v", err)
		}
		rr := httptest.NewRecorder()

		handler(rr, req)
		if status := rr.Code; status != http.StatusInternalServerError {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusInternalServerError)
		}
	})
}
