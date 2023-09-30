package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthCheck(t *testing.T) {
	runTest := func(mockStatuses []int, expectedStatus int) {
		var urls []string
		for _, status := range mockStatuses {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
			}))
			defer server.Close()
			urls = append(urls, server.URL)
		}

		handler := getHealthCheck(urls)
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

	t.Run("single 500 status", func(t *testing.T) {
		runTest([]int{http.StatusInternalServerError}, http.StatusInternalServerError)
	})

	t.Run("single 200 status", func(t *testing.T) {
		runTest([]int{http.StatusOK}, http.StatusOK)
	})

	t.Run("single 404 status", func(t *testing.T) {
		runTest([]int{http.StatusNotFound}, http.StatusInternalServerError)
	})

	t.Run("multiple instances, mixed statuses", func(t *testing.T) {
		runTest([]int{http.StatusOK, http.StatusNotFound, http.StatusInternalServerError}, http.StatusInternalServerError)
	})

	t.Run("multiple instances, all OK", func(t *testing.T) {
		runTest([]int{http.StatusOK, http.StatusOK, http.StatusOK}, http.StatusOK)
	})

	t.Run("no response", func(t *testing.T) {
		handler := getHealthCheck([]string{"http://localhost:12345"})
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
		handler := getHealthCheck([]string{"http://localhost:96010:invalidurl"})
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
