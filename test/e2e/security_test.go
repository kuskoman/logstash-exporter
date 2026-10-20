package e2e

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func fixturePath(t *testing.T, parts ...string) string {
	t.Helper()

	path, err := filepath.Abs(filepath.Join(append([]string{"..", "..", "fixtures"}, parts...)...))
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func TestServerBasicAuth(t *testing.T) {
	t.Parallel()

	mockLogstash := NewMockLogstashServer(t)
	exporter := StartExporter(t, NewConfigBuilder().
		AddInstance(mockLogstash.URL()).
		WithBasicAuthUsers(map[string]string{"prometheus": "secret"}))

	t.Run("rejects_requests_without_credentials", func(t *testing.T) {
		resp, err := exporter.Get("/metrics")
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", resp.StatusCode)
		}
		if challenge := resp.Header.Get("WWW-Authenticate"); challenge == "" {
			t.Error("expected a WWW-Authenticate header")
		}
	})

	t.Run("rejects_wrong_password", func(t *testing.T) {
		resp, err := exporter.GetWithAuth("/metrics", "prometheus", "wrong")
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", resp.StatusCode)
		}
	})

	t.Run("rejects_unknown_user", func(t *testing.T) {
		resp, err := exporter.GetWithAuth("/metrics", "someone", "secret")
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", resp.StatusCode)
		}
	})

	t.Run("serves_metrics_with_valid_credentials", func(t *testing.T) {
		Eventually(t, defaultWaitTimeout, func() error {
			resp, err := exporter.GetWithAuth("/metrics", "prometheus", "secret")
			if err != nil {
				return err
			}
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}
			metrics, err := ParseMetrics(resp.Body)
			if err != nil {
				return err
			}
			if !metrics.Has("logstash_info_build") {
				return fmt.Errorf("expected logstash_info_build to be exported")
			}
			return nil
		})
	})

	t.Run("keeps_version_and_healthcheck_public", func(t *testing.T) {
		for _, path := range []string{"/version", "/healthcheck"} {
			resp, err := exporter.Get(path)
			if err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != http.StatusOK {
				t.Errorf("expected %s to return 200 without credentials, got %d", path, resp.StatusCode)
			}
		}
	})
}

func TestServerTLS(t *testing.T) {
	t.Parallel()

	mockLogstash := NewMockLogstashServer(t)
	exporter := StartExporter(t, NewConfigBuilder().
		AddInstance(mockLogstash.URL()).
		WithServerTLS(fixturePath(t, "https", "server.crt"), fixturePath(t, "https", "server.key")),
		WithHTTPS())

	t.Run("serves_metrics_over_https", func(t *testing.T) {
		exporter.WaitForInstances(1)
	})

	t.Run("presents_a_certificate_signed_by_the_test_ca", func(t *testing.T) {
		caCert, err := os.ReadFile(fixturePath(t, "https", "ca.crt"))
		if err != nil {
			t.Fatal(err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caCert) {
			t.Fatal("failed to parse the test CA certificate")
		}

		client := &http.Client{Transport: &http.Transport{
			TLSClientConfig: &tls.Config{RootCAs: pool, ServerName: "localhost"},
		}}
		resp, err := client.Get(exporter.URL("/version"))
		if err != nil {
			t.Fatalf("failed to verify the server certificate: %v", err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("rejects_plain_http", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/version", exporter.Port))
		if err != nil {
			return
		}
		_ = resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			t.Error("expected plain HTTP requests to fail")
		}
	})
}

func TestLogstashBasicAuth(t *testing.T) {
	t.Parallel()

	mockLogstash := NewMockLogstashServer(t)
	exporter := StartExporter(t, NewConfigBuilder().AddInstanceWith(Instance{
		URL:      mockLogstash.URL(),
		Username: "logstash_user",
		Password: "logstash_password",
	}))
	exporter.WaitForInstances(1)

	requests := mockLogstash.Requests()
	if len(requests) == 0 {
		t.Fatal("expected requests to Logstash")
	}
	for _, req := range requests {
		if !req.HasAuth || req.Username != "logstash_user" || req.Password != "logstash_password" {
			t.Errorf("expected request to %s to use the configured credentials, got user %q (auth=%v)",
				req.Path, req.Username, req.HasAuth)
		}
	}
}

func TestLogstashTLS(t *testing.T) {
	t.Parallel()

	t.Run("skips_verification_when_configured", func(t *testing.T) {
		t.Parallel()

		mockLogstash := NewMockLogstashServer(t, WithMockTLS())
		exporter := StartExporter(t, NewConfigBuilder().AddInstanceWith(Instance{
			URL:                mockLogstash.URL(),
			InsecureSkipVerify: true,
		}))

		assertInstanceUp(t, exporter.WaitForInstances(1), mockLogstash.URL(), 1)
	})

	t.Run("verifies_the_certificate_with_a_custom_ca", func(t *testing.T) {
		t.Parallel()

		mockLogstash := NewMockLogstashServer(t, WithMockTLS())
		caFile := filepath.Join(t.TempDir(), "ca.crt")
		certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: mockLogstash.Server.Certificate().Raw})
		if err := os.WriteFile(caFile, certPEM, 0644); err != nil {
			t.Fatal(err)
		}

		exporter := StartExporter(t, NewConfigBuilder().AddInstanceWith(Instance{
			URL:    mockLogstash.URL(),
			CAFile: caFile,
		}))

		assertInstanceUp(t, exporter.WaitForInstances(1), mockLogstash.URL(), 1)
	})

	t.Run("rejects_an_untrusted_certificate", func(t *testing.T) {
		t.Parallel()

		mockLogstash := NewMockLogstashServer(t, WithMockTLS())
		exporter := StartExporter(t, NewConfigBuilder().AddInstance(mockLogstash.URL()))

		assertInstanceUp(t, exporter.WaitForInstances(1), mockLogstash.URL(), 0)
		if count := mockLogstash.RequestCount(""); count != 0 {
			t.Errorf("expected no request to reach Logstash over an untrusted connection, got %d", count)
		}
	})
}
