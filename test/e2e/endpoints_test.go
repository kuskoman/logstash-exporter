package e2e

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestEndpoints(t *testing.T) {
	t.Parallel()

	mockLogstash := NewMockLogstashServer(t)
	exporter := StartExporter(t, NewConfigBuilder().AddInstance(mockLogstash.URL()))

	t.Run("healthcheck_returns_200_when_logstash_is_reachable", func(t *testing.T) {
		resp, err := exporter.Get("/healthcheck")
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", resp.StatusCode, resp.Body)
		}
	})

	t.Run("version_returns_build_information_as_json", func(t *testing.T) {
		resp, err := exporter.Get("/version")
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d", resp.StatusCode)
		}
		if contentType := resp.Header.Get("Content-Type"); contentType != "application/json" {
			t.Errorf("expected content type application/json, got %q", contentType)
		}

		var versionInfo map[string]any
		if err := json.Unmarshal([]byte(resp.Body), &versionInfo); err != nil {
			t.Fatalf("version response is not valid JSON: %v\n%s", err, resp.Body)
		}
		for _, field := range []string{"Version", "GitCommit", "GoVersion", "BuildArch", "BuildOS", "BuildDate"} {
			if _, ok := versionInfo[field]; !ok {
				t.Errorf("version response is missing %s: %s", field, resp.Body)
			}
		}
	})

	t.Run("build_info_matches_the_version_endpoint", func(t *testing.T) {
		resp, err := exporter.Get("/version")
		if err != nil {
			t.Fatal(err)
		}
		var versionInfo struct{ Version, GitCommit string }
		if err := json.Unmarshal([]byte(resp.Body), &versionInfo); err != nil {
			t.Fatalf("version response is not valid JSON: %v", err)
		}
		if versionInfo.Version != "e2e-test" {
			t.Fatalf("expected the e2e binary to be built with version e2e-test, got %q (run make e2e-prepare)", versionInfo.Version)
		}

		metrics, err := exporter.Metrics()
		if err != nil {
			t.Fatal(err)
		}
		labels := map[string]string{"version": versionInfo.Version, "revision": versionInfo.GitCommit}
		if _, err := metrics.Value("logstash_exporter_build_info", labels); err != nil {
			t.Errorf("expected build info labelled with version %q and revision %q: %v",
				versionInfo.Version, versionInfo.GitCommit, err)
		}
	})

	t.Run("metrics_returns_prometheus_text_format", func(t *testing.T) {
		resp, err := exporter.Get("/metrics")
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d", resp.StatusCode)
		}
		if contentType := resp.Header.Get("Content-Type"); !strings.HasPrefix(contentType, "text/plain") {
			t.Errorf("expected text/plain content type, got %q", contentType)
		}
		if _, err := ParseMetrics(resp.Body); err != nil {
			t.Error(err)
		}
		if !strings.Contains(resp.Body, "# HELP logstash_info_build") || !strings.Contains(resp.Body, "# TYPE logstash_info_build") {
			t.Error("expected HELP and TYPE comments for logstash_info_build")
		}
	})

	t.Run("root_redirects_to_metrics", func(t *testing.T) {
		resp, err := exporter.Get("/")
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusMovedPermanently {
			t.Errorf("expected status 301, got %d", resp.StatusCode)
		}
		if location := resp.Header.Get("Location"); location != "/metrics" {
			t.Errorf("expected redirect to /metrics, got %q", location)
		}
	})
}
