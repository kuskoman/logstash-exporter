package logstash_client

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type TestResponse struct {
	Foo string `json:"foo"`
}

func TestNewClient(t *testing.T) {
	t.Parallel()

	t.Run("should return a new client for the default endpoint", func(t *testing.T) {
		t.Parallel()

		client := NewClient("", "default")

		if client.(*DefaultClient).endpoint != defaultLogstashEndpoint {
			t.Errorf("expected endpoint to be %s, got %s", defaultLogstashEndpoint, client.(*DefaultClient).endpoint)
		}
	})

	t.Run("should return a new client for the given endpoint", func(t *testing.T) {
		t.Parallel()

		expectedEndpoint := "http://localhost:9601"
		client := NewClient(expectedEndpoint, "custom")

		receivedEndpoint := client.GetEndpoint()
		if receivedEndpoint != expectedEndpoint {
			t.Errorf("expected endpoint to be %s, got %s", expectedEndpoint, receivedEndpoint)
		}
	})

	t.Run("should derive the name from the endpoint when no name is set", func(t *testing.T) {
		t.Parallel()

		tests := map[string]string{
			"http://localhost:9600":             "http_localhost_9600",
			"https://logstash.example.com:9600": "https_logstash_example_com_9600",
			"http://10.0.0.5:9600/":             "http_10_0_0_5_9600",
		}
		for endpoint, expected := range tests {
			if name := NewClient(endpoint, "").Name(); name != expected {
				t.Errorf("expected name %q for %s, got %q", expected, endpoint, name)
			}
		}
	})

	t.Run("should use the given http client", func(t *testing.T) {
		t.Parallel()

		httpClient := &http.Client{}
		client := NewClientWithHTTPClient("http://logstash:9600", httpClient, "custom")

		if client.(*DefaultClient).httpClient != httpClient {
			t.Error("expected the given http client to be used")
		}
		if client.GetEndpoint() != "http://logstash:9600" || client.Name() != "custom" {
			t.Errorf("unexpected endpoint %q or name %q", client.GetEndpoint(), client.Name())
		}
	})

	t.Run("should set the client name correctly", func(t *testing.T) {
		t.Parallel()

		client := NewClient("", "testName")
		if client.Name() != "testName" {
			t.Errorf("expected client name to be %q, got %q", "testName", client.Name())
		}
	})
}

func TestGetMetrics(t *testing.T) {
	t.Run("should return an error if the URL is invalid", func(t *testing.T) {
		httpClient := &http.Client{}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		invalidURL := "http://localhost:96010:invalidurl"
		result, err := getMetrics[TestResponse](ctx, httpClient, invalidURL)

		if err == nil {
			t.Errorf("expected error, got nil")
		}

		if result != nil {
			t.Errorf("expected result to be nil, got %v", result)
		}
	})

	t.Run("should return a valid response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`{"foo": "bar"}`))
			if err != nil {
				t.Errorf("error writing response: %s", err)
			}
		}))
		defer server.Close()

		httpClient := &http.Client{}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		result, err := getMetrics[TestResponse](ctx, httpClient, server.URL)
		if err != nil {
			t.Errorf("expected no error, got %s", err)
		}

		if result == nil || result.Foo != "bar" {
			t.Errorf("expected foo to be 'bar', got %v", result)
		}
	})

	t.Run("should return an error if the response is invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`{"foo": "bar"`)) // Invalid JSON
			if err != nil {
				t.Errorf("error writing response: %s", err)
			}
		}))
		defer server.Close()

		httpClient := &http.Client{}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		result, err := getMetrics[TestResponse](ctx, httpClient, server.URL)
		if err == nil {
			t.Errorf("expected error, got nil")
		}

		if result != nil {
			t.Errorf("expected result to be nil, got %v", result)
		}
	})

	t.Run("should return an error if the request fails", func(t *testing.T) {
		httpClient := &http.Client{}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		result, err := getMetrics[TestResponse](ctx, httpClient, "http://localhost:96010")
		if err == nil {
			t.Errorf("expected error, got nil")
		}

		if result != nil {
			t.Errorf("expected result to be nil, got %v", result)
		}
	})

	t.Run("should return an error if the context is cancelled", func(t *testing.T) {
		httpClient := &http.Client{}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		result, err := getMetrics[TestResponse](ctx, httpClient, "http://localhost:96010")
		if err == nil {
			t.Errorf("expected error, got nil")
		}

		if result != nil {
			t.Errorf("expected result to be nil, got %v", result)
		}
	})
}

func TestDeserializeHttpResponse(t *testing.T) {
	t.Run("should properly deserialize a valid response", func(t *testing.T) {
		httpResponseMock := &http.Response{
			Body: io.NopCloser(strings.NewReader(`{"foo": "bar"}`)),
		}

		result, err := deserializeHttpResponse[TestResponse](httpResponseMock)
		if err != nil {
			t.Errorf("expected no error, got %s", err)
		}

		if result == nil || result.Foo != "bar" {
			t.Errorf("expected foo to be 'bar', got %v", result)
		}
	})

	t.Run("should return an error if the response is invalid JSON", func(t *testing.T) {
		httpResponseMock := &http.Response{
			Body: io.NopCloser(strings.NewReader(`{"foo": "bar"`)), // Invalid JSON
		}

		result, err := deserializeHttpResponse[TestResponse](httpResponseMock)
		if err == nil {
			t.Errorf("expected error, got nil")
		}

		if result != nil {
			t.Errorf("expected result to be nil, got %v", result)
		}
	})

	t.Run("should return an error if the response body cannot be read", func(t *testing.T) {
		httpResponseMock := &http.Response{
			Body: io.NopCloser(&errorReader{}),
		}

		result, err := deserializeHttpResponse[TestResponse](httpResponseMock)
		if err == nil {
			t.Errorf("expected error, got nil")
		}

		if result != nil {
			t.Errorf("expected result to be nil, got %v", result)
		}
	})
}

// errorReader is a mock that simulates a read error
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}

func (e *errorReader) Close() error {
	return nil
}
