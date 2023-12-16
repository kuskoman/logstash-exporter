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

		client := NewClient("", nil)

		if client.(*DefaultClient).endpoint != defaultLogstashEndpoint {
			t.Errorf("expected endpoint to be %s, got %s", defaultLogstashEndpoint, client.(*DefaultClient).endpoint)
		}
	})

	t.Run("should return a new client for the given endpoint", func(t *testing.T) {
		t.Parallel()

		expectedEndpoint := "http://localhost:9601"
		client := NewClient(expectedEndpoint, nil)

		receivedEndpoint := client.GetEndpoint()
		if receivedEndpoint != expectedEndpoint {
			t.Errorf("expected endpoint to be %s, got %s", expectedEndpoint, receivedEndpoint)
		}
	})

	t.Run("should return a new client with the given labels", func(t *testing.T) {
		t.Parallel()

		expectedLabels := map[string]string{"foo": "bar"}
		client := NewClient("", expectedLabels)

		receivedLabels := client.GetLabels()
		if receivedLabels["foo"] != expectedLabels["foo"] {
			t.Errorf("expected label to be %s, got %s", expectedLabels["foo"], receivedLabels["foo"])
		}
	})

	t.Run("should return a new client with an empty map if no labels are given", func(t *testing.T) {
		t.Parallel()

		client := NewClient("", nil)

		receivedLabels := client.GetLabels()
		if len(receivedLabels) != 0 {
			t.Errorf("expected labels to be empty, got %v", receivedLabels)
		}
	})
}

func TestGetMetrics(t *testing.T) {
	t.Run("should return an error if the URL is invalid", func(t *testing.T) {
		httpClient := &http.Client{}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		// Pass an invalid URL
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

		if result.Foo != "bar" {
			t.Errorf("expected foo to be bar, got %s", result.Foo)
		}
	})

	t.Run("should return an error if the response is invalid", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`{"foo": "bar"`))
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

		if result.Foo != "bar" {
			t.Errorf("expected foo to be bar, got %s", result.Foo)
		}
	})

	t.Run("should return an error if the response is invalid", func(t *testing.T) {
		httpResponseMock := &http.Response{
			Body: io.NopCloser(strings.NewReader(`{"foo": "bar"`)),
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
