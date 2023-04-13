package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kuskoman/logstash-exporter/config"
)

func TestHandleVersionInfo(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(handleVersionInfo))
	defer ts.Close()

	resp, err := http.Get(ts.URL)
	if err != nil {
		t.Fatalf("Failed to make a request to the test server: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read the response body: %v", err)
	}

	var versionInfo config.VersionInfo
	err = json.Unmarshal(body, &versionInfo)
	if err != nil {
		t.Fatalf("Failed to decode JSON: %v", err)
	}

	expectedVersionInfo := config.GetBuildInfo()
	if versionInfo != *expectedVersionInfo {
		t.Errorf("Expected version info: %+v, but got: %+v", *expectedVersionInfo, versionInfo)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, but got: %d", http.StatusOK, resp.StatusCode)
	}

	if contentType := resp.Header.Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Expected Content-Type header to be 'application/json', but got: %s", contentType)
	}
}
