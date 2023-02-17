package logstashclient

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
)

type MockHTTPClient struct {
	Response *http.Response
	Err      error
}

func (m *MockHTTPClient) Get(url string) (*http.Response, error) {
	return m.Response, m.Err
}

func TestGetNodeInfo(t *testing.T) {
	t.Run("should return a valid NodeInfoResponse when the request is successful", func(t *testing.T) {
		mockClient, err := setupSuccessfulHttpMock("node_info.json")
		if err != nil {
			t.Fatalf("error setting up mock http client: %s", err)
		}

		client := &DefaultClient{
			httpClient: mockClient,
			endpoint:   "http://localhost:9600",
		}

		response, err := client.GetNodeInfo()
		if err != nil {
			t.Fatalf("error getting node info: %s", err)
		}

		if response.Status != "green" {
			t.Fatalf("expected status to be properly read as green, got %s", response.Status)
		}
		// detailed checks are done in the responses package
	})

	t.Run("should return an error when the request fails", func(t *testing.T) {
		mockClient := &MockHTTPClient{
			Response: nil,
			Err:      fmt.Errorf("error"),
		}

		client := &DefaultClient{
			httpClient: mockClient,
			endpoint:   "http://localhost:9600",
		}

		_, err := client.GetNodeInfo()
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
	})
}

func TestGetNodeStats(t *testing.T) {
	t.Run("should return a valid NodestatsResponse when the request is successful", func(t *testing.T) {
		mockClient, err := setupSuccessfulHttpMock("node_stats.json")
		if err != nil {
			t.Fatalf("error setting up mock http client: %s", err)
		}

		client := &DefaultClient{
			httpClient: mockClient,
			endpoint:   "http://localhost:9600",
		}

		response, err := client.GetNodeStats()
		if err != nil {
			t.Fatalf("error getting node stats: %s", err)
		}

		if response.Status != "green" {
			t.Fatalf("expected status to be properly read as green, got %s", response.Status)
		}
		// detailed checks are done in the responses package
	})

	t.Run("should return an error when the request fails", func(t *testing.T) {
		mockClient := &MockHTTPClient{
			Response: nil,
			Err:      fmt.Errorf("error"),
		}

		client := &DefaultClient{
			httpClient: mockClient,
			endpoint:   "http://localhost:9600",
		}

		_, err := client.GetNodeStats()
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
	})
}

func loadFixture(filename string) ([]byte, error) {
	fullPath := fmt.Sprintf("../../fixtures/%s", filename)
	fixtureBytes, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, err
	}

	return fixtureBytes, nil
}

func setupSuccessfulHttpMock(filename string) (*MockHTTPClient, error) {
	fixtureBytes, err := loadFixture(filename)
	if err != nil {
		return nil, err
	}

	fixtureReader := bytes.NewReader(fixtureBytes)

	mockResponse := &http.Response{
		Body:       io.NopCloser(fixtureReader),
		StatusCode: 200,
	}

	return &MockHTTPClient{
		Response: mockResponse,
		Err:      nil,
	}, nil
}
