package tls

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMultiUserAuthMiddleware(t *testing.T) {
	// Define users for testing
	users := map[string]string{
		"testuser":  "testpass",
		"otheruser": "otherpass",
	}

	// Create a test handler that just returns 200 OK
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap the handler with the MultiUserAuthMiddleware
	handler := MultiUserAuthMiddleware(nextHandler, users)

	// Test case: no auth header
	t.Run("no auth header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://example.com/foo", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status code %d, got %d", http.StatusUnauthorized, w.Code)
		}

		if w.Header().Get("WWW-Authenticate") == "" {
			t.Error("Expected WWW-Authenticate header, got none")
		}
	})

	// Test case: invalid username
	t.Run("invalid username", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://example.com/foo", nil)
		req.SetBasicAuth("wronguser", "testpass")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status code %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})

	// Test case: invalid password
	t.Run("invalid password", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://example.com/foo", nil)
		req.SetBasicAuth("testuser", "wrongpass")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status code %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})

	// Test case: valid auth for first user
	t.Run("valid auth for first user", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://example.com/foo", nil)
		req.SetBasicAuth("testuser", "testpass")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
		}
	})

	// Test case: valid auth for second user
	t.Run("valid auth for second user", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://example.com/foo", nil)
		req.SetBasicAuth("otheruser", "otherpass")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
		}
	})
}
