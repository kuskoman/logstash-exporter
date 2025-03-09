package tls

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kuskoman/logstash-exporter/pkg/config"
)

func TestConfigureClientTLS(t *testing.T) {
	// Create temporary directory for test certificates
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
			tlsConfig, err := ConfigureClientTLS(tc.caFile, tc.serverName, tc.insecureSkipVerify)

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

			if tlsConfig == nil {
				t.Errorf("Expected non-nil TLS config, got nil")
				return
			}

			// Verify TLS config properties
			if tlsConfig.InsecureSkipVerify != tc.insecureSkipVerify {
				t.Errorf("Expected InsecureSkipVerify to be %v, got %v",
					tc.insecureSkipVerify,
					tlsConfig.InsecureSkipVerify)
			}

			if tc.serverName != "" && tlsConfig.ServerName != tc.serverName {
				t.Errorf("Expected ServerName to be %s, got %s",
					tc.serverName,
					tlsConfig.ServerName)
			}

			if tc.caFile != "" && tlsConfig.RootCAs == nil {
				t.Errorf("Expected non-nil RootCAs when CA file is provided")
			}
		})
	}
}

func TestConfigureHTTPClientFromLogstashInstance(t *testing.T) {
	// Create temporary directory for test certificates
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
		name     string
		instance *config.LogstashInstance
	}{
		{
			name: "legacy config",
			instance: &config.LogstashInstance{
				Host: TestBaseURL,
				Name: "legacy",
			},
		},
		{
			name: "TLS config with CA",
			instance: &config.LogstashInstance{
				Host: TestBaseURL,
				Name: "with-ca",
				TLSConfig: &config.TLSClientConfig{
					CAFile:             caPath,
					ServerName:         TestServerName,
					InsecureSkipVerify: false,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, err := ConfigureHTTPClientFromLogstashInstance(tc.instance, timeout)

			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
				return
			}

			if client == nil {
				t.Errorf("Expected non-nil HTTP client, got nil")
				return
			}

			if client.Timeout != timeout {
				t.Errorf("Expected timeout %v, got %v", timeout, client.Timeout)
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	// Create a test server that echoes back the Authorization header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		w.Header().Set("Echo-Auth", auth)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create the transport under test
	original := http.DefaultTransport
	transport := &basicAuthTransport{
		username:  "testuser",
		password:  "testpass",
		transport: original,
	}

	// Create a request
	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Perform the request
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip failed: %v", err)
	}
	defer resp.Body.Close()

	// Check that the auth header was added correctly
	echoedAuth := resp.Header.Get("Echo-Auth")
	if echoedAuth == "" {
		t.Error("Expected Authorization header to be added, but it wasn't")
	}

	// Ensure the original request wasn't modified
	if req.Header.Get("Authorization") != "" {
		t.Error("Original request was modified, it shouldn't be")
	}
}

func TestConfigureBasicAuth(t *testing.T) {
	t.Run("nil client", func(t *testing.T) {
		result := ConfigureBasicAuth(nil, "user", "pass")
		if result != nil {
			t.Errorf("Expected nil result for nil client, got %v", result)
		}
	})

	t.Run("valid client", func(t *testing.T) {
		client := &http.Client{}
		result := ConfigureBasicAuth(client, "user", "pass")

		if result != client {
			t.Errorf("Expected client to be returned, got different instance")
		}

		// Check that the transport was set
		if result.Transport == nil {
			t.Error("Expected Transport to be set, got nil")
		}

		// Check that it's our basicAuthTransport
		_, ok := result.Transport.(*basicAuthTransport)
		if !ok {
			t.Errorf("Expected Transport to be basicAuthTransport, got %T", result.Transport)
		}
	})

	t.Run("with existing transport", func(t *testing.T) {
		// Create a client with a custom transport
		existingTransport := http.DefaultTransport
		client := &http.Client{
			Transport: existingTransport,
		}

		result := ConfigureBasicAuth(client, "user", "pass")

		// Check that the transport was wrapped
		transport, ok := result.Transport.(*basicAuthTransport)
		if !ok {
			t.Errorf("Expected Transport to be basicAuthTransport, got %T", result.Transport)
		}

		// Check that the original transport was preserved
		if transport.transport != existingTransport {
			t.Errorf("Expected original transport to be preserved")
		}
	})
}

func TestConfigureHTTPClientWithTLS(t *testing.T) {
	// Create temporary directory for test certificates
	tempDir, err := os.MkdirTemp("", "tls-client-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Get test certificates and save them to files
	testCerts := GetTestCertificates()
	caPath := filepath.Join(tempDir, "ca.pem")

	if err := os.WriteFile(caPath, []byte(testCerts.CAPEM), 0600); err != nil {
		t.Fatalf("Failed to write CA file: %v", err)
	}

	// Non-existent CA file path
	badPath := filepath.Join(tempDir, "nonexistent.pem")

	tests := []struct {
		name               string
		timeout            time.Duration
		caFile             string
		serverName         string
		insecureSkipVerify bool
		expectError        bool
		validateFunc       func(*testing.T, *http.Client)
	}{
		{
			name:               "basic client",
			timeout:            time.Second * 5,
			caFile:             "",
			serverName:         "",
			insecureSkipVerify: false,
			expectError:        false,
			validateFunc: func(t *testing.T, c *http.Client) {
				if c.Timeout != time.Second*5 {
					t.Errorf("Expected timeout %v, got %v", time.Second*5, c.Timeout)
				}

				// Check transport is set
				transport, ok := c.Transport.(*http.Transport)
				if !ok {
					t.Fatalf("Expected transport to be *http.Transport, got %T", c.Transport)
				}

				// Check TLS config is set
				if transport.TLSClientConfig == nil {
					t.Error("Expected TLSClientConfig to be set, got nil")
				}
			},
		},
		{
			name:               "with CA file",
			timeout:            time.Second * 5,
			caFile:             caPath,
			serverName:         "",
			insecureSkipVerify: false,
			expectError:        false,
			validateFunc: func(t *testing.T, c *http.Client) {
				transport, ok := c.Transport.(*http.Transport)
				if !ok {
					t.Fatalf("Expected transport to be *http.Transport, got %T", c.Transport)
				}

				if transport.TLSClientConfig.RootCAs == nil {
					t.Error("Expected RootCAs to be set, got nil")
				}
			},
		},
		{
			name:               "with server name",
			timeout:            time.Second * 5,
			caFile:             "",
			serverName:         "example.com",
			insecureSkipVerify: false,
			expectError:        false,
			validateFunc: func(t *testing.T, c *http.Client) {
				transport, ok := c.Transport.(*http.Transport)
				if !ok {
					t.Fatalf("Expected transport to be *http.Transport, got %T", c.Transport)
				}

				if transport.TLSClientConfig.ServerName != "example.com" {
					t.Errorf("Expected ServerName to be example.com, got %s", transport.TLSClientConfig.ServerName)
				}
			},
		},
		{
			name:               "with insecure skip verify",
			timeout:            time.Second * 5,
			caFile:             "",
			serverName:         "",
			insecureSkipVerify: true,
			expectError:        false,
			validateFunc: func(t *testing.T, c *http.Client) {
				transport, ok := c.Transport.(*http.Transport)
				if !ok {
					t.Fatalf("Expected transport to be *http.Transport, got %T", c.Transport)
				}

				if !transport.TLSClientConfig.InsecureSkipVerify {
					t.Error("Expected InsecureSkipVerify to be true, got false")
				}
			},
		},
		{
			name:               "with invalid CA file",
			timeout:            time.Second * 5,
			caFile:             badPath,
			serverName:         "",
			insecureSkipVerify: false,
			expectError:        true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client, err := ConfigureHTTPClientWithTLS(tc.timeout, tc.caFile, tc.serverName, tc.insecureSkipVerify)

			if tc.expectError && err == nil {
				t.Error("Expected error, got nil")
			}

			if !tc.expectError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}

			if !tc.expectError && err == nil && tc.validateFunc != nil {
				tc.validateFunc(t, client)
			}
		})
	}
}
