package tls

import (
	"crypto/tls"
	"os"
	"path/filepath"
	"testing"

	"github.com/kuskoman/logstash-exporter/pkg/config"
)

func TestConfigureServerTLS(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tls-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Get test certificate data
	testCerts := GetTestCertificates(t)

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
		name        string
		config      *config.Config
		expectError bool
	}{
		{
			name: "no TLS config",
			config: &config.Config{
				Server: config.ServerConfig{},
			},
			expectError: false,
		},
		{
			name: "valid TLS config",
			config: &config.Config{
				Server: config.ServerConfig{
					TLSConfig: &config.TLSServerConfig{
						CertFile:   certPath,
						KeyFile:    keyPath,
						MinVersion: TLSVersion13,
						ClientAuth: ClientAuthNone,
					},
				},
			},
			expectError: false,
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
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tlsConfig, err := ConfigureServerTLS(tc.config)

			if tc.expectError && err == nil {
				t.Errorf("Expected error, got nil")
			}

			if !tc.expectError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}

			// For the "no TLS config" case, we expect a nil result
			if tc.name == "no TLS config" && tlsConfig != nil {
				t.Errorf("Expected nil TLS config, got non-nil")
			}

			// For other non-error cases, we expect a non-nil result
			if !tc.expectError && tc.name != "no TLS config" && tlsConfig == nil {
				t.Errorf("Expected non-nil TLS config, got nil")
			}
		})
	}
}

func TestConfigureAdvancedServerTLS(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tls-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Get test certificates and save them to files
	testCerts := GetTestCertificates(t)
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

	tests := []struct {
		name         string
		config       *config.TLSServerConfig
		expectError  bool
		validateFunc func(*testing.T, *tls.Config)
	}{
		{
			name: "basic config",
			config: &config.TLSServerConfig{
				CertFile: certPath,
				KeyFile:  keyPath,
			},
			expectError: false,
			validateFunc: func(t *testing.T, c *tls.Config) {
				if len(c.Certificates) != 1 {
					t.Errorf("Expected 1 certificate, got %d", len(c.Certificates))
				}
			},
		},
		{
			name: "with client CA",
			config: &config.TLSServerConfig{
				CertFile: certPath,
				KeyFile:  keyPath,
				ClientCA: caPath,
			},
			expectError: false,
			validateFunc: func(t *testing.T, c *tls.Config) {
				if c.ClientCAs == nil {
					t.Error("Expected ClientCAs to be set, got nil")
				}
			},
		},
		{
			name: "with client auth",
			config: &config.TLSServerConfig{
				CertFile:   certPath,
				KeyFile:    keyPath,
				ClientAuth: ClientAuthRequireAndVerify,
			},
			expectError: false,
			validateFunc: func(t *testing.T, c *tls.Config) {
				if c.ClientAuth != tls.RequireAndVerifyClientCert {
					t.Errorf("Expected ClientAuth to be RequireAndVerifyClientCert, got %v", c.ClientAuth)
				}
			},
		},
		{
			name: "with min version",
			config: &config.TLSServerConfig{
				CertFile:   certPath,
				KeyFile:    keyPath,
				MinVersion: TLSVersion13,
			},
			expectError: false,
			validateFunc: func(t *testing.T, c *tls.Config) {
				if c.MinVersion != tls.VersionTLS13 {
					t.Errorf("Expected MinVersion to be TLS 1.3, got %v", c.MinVersion)
				}
			},
		},
		{
			name: "with max version",
			config: &config.TLSServerConfig{
				CertFile:   certPath,
				KeyFile:    keyPath,
				MaxVersion: TLSVersion12,
			},
			expectError: false,
			validateFunc: func(t *testing.T, c *tls.Config) {
				if c.MaxVersion != tls.VersionTLS12 {
					t.Errorf("Expected MaxVersion to be TLS 1.2, got %v", c.MaxVersion)
				}
			},
		},
		{
			name: "with invalid client auth type",
			config: &config.TLSServerConfig{
				CertFile:   certPath,
				KeyFile:    keyPath,
				ClientAuth: "INVALID",
			},
			expectError: true,
		},
		{
			name: "with invalid min version",
			config: &config.TLSServerConfig{
				CertFile:   certPath,
				KeyFile:    keyPath,
				MinVersion: "INVALID",
			},
			expectError: true,
		},
		{
			name: "with invalid max version",
			config: &config.TLSServerConfig{
				CertFile:   certPath,
				KeyFile:    keyPath,
				MaxVersion: "INVALID",
			},
			expectError: true,
		},
		{
			name: "with cipher suites",
			config: &config.TLSServerConfig{
				CertFile:     certPath,
				KeyFile:      keyPath,
				CipherSuites: []string{"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"},
			},
			expectError: false,
		},
		{
			name: "with curve preferences",
			config: &config.TLSServerConfig{
				CertFile:         certPath,
				KeyFile:          keyPath,
				CurvePreferences: []string{"P256"},
			},
			expectError: false,
		}
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tlsConfig, err := ConfigureAdvancedServerTLS(tc.config)

			if tc.expectError && err == nil {
				t.Error("Expected error, got nil")
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
