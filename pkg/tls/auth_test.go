package tls

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMultiUserAuthMiddleware(t *testing.T) {
	users := map[string]string{
		"testuser":  "testpass",
		"otheruser": "otherpass",
	}

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := MultiUserAuthMiddleware(nextHandler, users)

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

	t.Run("invalid username", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://example.com/foo", nil)
		req.SetBasicAuth("wronguser", "testpass")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status code %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})

	t.Run("invalid password", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://example.com/foo", nil)
		req.SetBasicAuth("testuser", "wrongpass")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status code %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})

	t.Run("valid auth for first user", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://example.com/foo", nil)
		req.SetBasicAuth("testuser", "testpass")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
		}
	})

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
