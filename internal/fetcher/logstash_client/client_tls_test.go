package logstash_client

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewClientWithTLS(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "client-tls-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Get test certificate data
	testCerts := GetTestCertificates()

	// Create a valid CA certificate file
	caPath := filepath.Join(tempDir, "ca.pem")
	if err := os.WriteFile(caPath, []byte(testCerts.CAPEM), 0600); err != nil {
		t.Fatalf("Failed to write CA file: %v", err)
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
				t.Errorf("Expected error, got nil")
				return
			}

			if !tc.expectError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
				return
			}

			// Skip further checks if we expected an error
			if tc.expectError {
				return
			}

			if client == nil {
				t.Errorf("Expected non-nil client, got nil")
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
		t.Fatalf("Expected no error, got: %v", err)
	}

	if client == nil {
		t.Fatalf("Expected non-nil client, got nil")
	}

	if client.httpClient.Transport == nil {
		t.Errorf("Expected transport to be configured, got nil")
	}
}
