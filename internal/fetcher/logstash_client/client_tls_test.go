package logstash_client

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kuskoman/logstash-exporter/internal/file_utils"
)

func TestNewClientWithTLS(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "client-tls-test")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer file_utils.HandleTempDirRemoval(t, tempDir)

	testCerts := GetTestCertificates()

	caPath := filepath.Join(tempDir, "ca.pem")
	if err := os.WriteFile(caPath, []byte(testCerts.CAPEM), 0600); err != nil {
		t.Fatalf("failed to write CA file: %v", err)
	}

	timeout := time.Duration(TestTimeout) * time.Second

	testCases := []struct {
		name               string
		caFile             string
		serverName         string
		insecureSkipVerify bool
		expectError        bool
	}{
		{
			name:               "basic configuration",
			caFile:             "",
			serverName:         "",
			insecureSkipVerify: false,
			expectError:        false,
		},
		{
			name:               "with insecure skip verify",
			caFile:             "",
			serverName:         "",
			insecureSkipVerify: true,
			expectError:        false,
		},
		{
			name:               "with valid CA file",
			caFile:             caPath,
			serverName:         "",
			insecureSkipVerify: false,
			expectError:        false,
		},
		{
			name:               "with invalid CA file",
			caFile:             TestNonexistentCA,
			serverName:         "",
			insecureSkipVerify: false,
			expectError:        true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, err := NewClientWithTLS(
				TestBaseURL,
				timeout,
				tc.caFile,
				tc.serverName,
				tc.insecureSkipVerify,
			)

			if tc.expectError && err == nil {
				t.Errorf("expected error, got nil")
				return
			}

			if !tc.expectError && err != nil {
				t.Errorf("expected no error, got: %v", err)
				return
			}

			if tc.expectError {
				return
			}

			if client == nil {
				t.Errorf("expected non-nil client, got nil")
				return
			}
		})
	}
}

func TestNewClientWithBasicAuth(t *testing.T) {
	timeout := time.Duration(TestTimeout) * time.Second

	client, err := NewClientWithBasicAuth(
		TestBaseURL,
		timeout,
		TestUsername,
		TestPassword,
		"",    // No CA file
		"",    // No server name
		false, // Don't skip verification
	)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if client == nil {
		t.Fatalf("expected non-nil client, got nil")
	}

	if client.httpClient.Transport == nil {
		t.Errorf("expected transport to be configured, got nil")
	}
}
