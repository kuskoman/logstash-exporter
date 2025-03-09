package server

import (
	"crypto/tls"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/kuskoman/logstash-exporter/pkg/config"
	customtls "github.com/kuskoman/logstash-exporter/pkg/tls"
)

// TestConfigureTLS tests the TLS configuration functionality
func TestConfigureTLS(t *testing.T) {
	// Create temporary directory for test certificates
	tempDir, err := os.MkdirTemp("", "tls-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Get test certificate data
	testCerts := customtls.GetTestCertificates()

	// Write certificates to files
	certPath := filepath.Join(tempDir, "cert.pem")
	keyPath := filepath.Join(tempDir, "key.pem")
	caPath := filepath.Join(tempDir, "ca.pem")

	if err := os.WriteFile(certPath, []byte(testCerts.CertPEM), 0600); err != nil {
		t.Fatalf("Failed to write cert file: %v", err)
	}
	if err := os.WriteFile(keyPath, []byte(testCerts.KeyPEM), 0600); err != nil {
		t.Fatalf("Failed to write key file: %v", err)
	}
	if err := os.WriteFile(caPath, []byte(testCerts.CAPEM), 0600); err != nil {
		t.Fatalf("Failed to write CA file: %v", err)
	}

	// Test cases
	testCases := []struct {
		name         string
		config       *config.Config
		expectError  bool
		validateFunc func(*testing.T, *tls.Config)
	}{
		{
			name: "no TLS config",
			config: &config.Config{
				Server: config.ServerConfig{},
			},
			expectError: false,
			validateFunc: func(t *testing.T, tlsConfig *tls.Config) {
				if tlsConfig != nil {
					t.Errorf("Expected nil TLS config, got non-nil")
				}
			},
		},
		{
			name: "invalid certificate path",
			config: &config.Config{
				Server: config.ServerConfig{
					TLSConfig: &config.TLSServerConfig{
						CertFile: "/nonexistent/cert.pem",
						KeyFile:  "/nonexistent/key.pem",
					},
				},
			},
			expectError: true,
		},
		{
			name: "valid TLS config",
			config: &config.Config{
				Server: config.ServerConfig{
					TLSConfig: &config.TLSServerConfig{
						CertFile:   certPath,
						KeyFile:    keyPath,
						MinVersion: customtls.TLSVersion13,
						ClientAuth: customtls.ClientAuthNone,
					},
				},
			},
			expectError: false,
			validateFunc: func(t *testing.T, tlsConfig *tls.Config) {
				if tlsConfig == nil {
					t.Errorf("Expected non-nil TLS config, got nil")
					return
				}
				if tlsConfig.MinVersion != tls.VersionTLS13 {
					t.Errorf("Expected MinVersion to be TLS 1.3, got %v", tlsConfig.MinVersion)
				}
				if tlsConfig.ClientAuth != tls.NoClientCert {
					t.Errorf("Expected ClientAuth to be NoClientCert, got %v", tlsConfig.ClientAuth)
				}
				if len(tlsConfig.Certificates) != 1 {
					t.Errorf("Expected 1 certificate, got %d", len(tlsConfig.Certificates))
				}
			},
		},
		{
			name: "invalid min version",
			config: &config.Config{
				Server: config.ServerConfig{
					TLSConfig: &config.TLSServerConfig{
						CertFile:   certPath,
						KeyFile:    keyPath,
						MinVersion: "INVALID",
					},
				},
			},
			expectError: true,
		},
		{
			name: "invalid client auth type",
			config: &config.Config{
				Server: config.ServerConfig{
					TLSConfig: &config.TLSServerConfig{
						CertFile:   certPath,
						KeyFile:    keyPath,
						ClientAuth: "INVALID",
					},
				},
			},
			expectError: true,
		},
		{
			name: "client CA config",
			config: &config.Config{
				Server: config.ServerConfig{
					TLSConfig: &config.TLSServerConfig{
						CertFile:   certPath,
						KeyFile:    keyPath,
						ClientCA:   caPath,
						ClientAuth: customtls.ClientAuthRequireAndVerify,
					},
				},
			},
			expectError: false,
			validateFunc: func(t *testing.T, tlsConfig *tls.Config) {
				if tlsConfig == nil {
					t.Errorf("Expected non-nil TLS config, got nil")
					return
				}
				if tlsConfig.ClientCAs == nil {
					t.Errorf("Expected non-nil ClientCAs, got nil")
				}
				if tlsConfig.ClientAuth != tls.RequireAndVerifyClientCert {
					t.Errorf("Expected ClientAuth to be RequireAndVerifyClientCert, got %v", tlsConfig.ClientAuth)
				}
			},
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tlsConfig, err := customtls.ConfigureServerTLS(tc.config)

			if tc.expectError && err == nil {
				t.Errorf("Expected error, got nil")
			}

			if !tc.expectError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}

			if !tc.expectError && err == nil && tc.validateFunc != nil {
				tc.validateFunc(t, tlsConfig)
			}
		})
	}
}

// TestMultiUserAuthMiddleware tests the multi-user auth middleware
func TestMultiUserAuthMiddleware(t *testing.T) {
	users := map[string]string{"user": "pass"}
	handler := customtls.MultiUserAuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}), users)

	if handler == nil {
		t.Error("Expected non-nil handler, got nil")
	}
}
