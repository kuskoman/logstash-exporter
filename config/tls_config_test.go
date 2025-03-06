package config

import (
	"crypto/tls"
	"testing"
)

func TestSetupTLS(t *testing.T) {
	t.Run("default configuration", func(t *testing.T) {
		tlsConfig, err := SetupTLS()

		if err != nil {
			t.Fatalf("Unexpected error setting up TLS: %v", err)
		}

		if tlsConfig == nil {
			t.Fatal("Expected TLS config, got nil")
		}

		if tlsConfig.MinVersion != tls.VersionTLS12 {
			t.Errorf("Expected MinVersion TLS 1.2, got %d", tlsConfig.MinVersion)
		}

		expectedCipherSuites := []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		}
		
		if len(tlsConfig.CipherSuites) != len(expectedCipherSuites) {
			t.Errorf("Expected %d cipher suites, got %d", len(expectedCipherSuites), len(tlsConfig.CipherSuites))
		}
	})

	t.Run("custom TLS version", func(t *testing.T) {
		// Save original values
		originalMinVersion := SSLMinVersion
		defer func() {
			SSLMinVersion = originalMinVersion
		}()

		// Test TLS 1.1
		SSLMinVersion = "1.1"
		tlsConfig, err := SetupTLS()
		if err != nil {
			t.Fatalf("Unexpected error setting up TLS with version 1.1: %v", err)
		}
		if tlsConfig.MinVersion != tls.VersionTLS11 {
			t.Errorf("Expected MinVersion TLS 1.1, got %d", tlsConfig.MinVersion)
		}

		// Test TLS 1.0
		SSLMinVersion = "1.0"
		tlsConfig, err = SetupTLS()
		if err != nil {
			t.Fatalf("Unexpected error setting up TLS with version 1.0: %v", err)
		}
		if tlsConfig.MinVersion != tls.VersionTLS10 {
			t.Errorf("Expected MinVersion TLS 1.0, got %d", tlsConfig.MinVersion)
		}

		// Test TLS 1.3
		SSLMinVersion = "1.3"
		tlsConfig, err = SetupTLS()
		if err != nil {
			t.Fatalf("Unexpected error setting up TLS with version 1.3: %v", err)
		}
		if tlsConfig.MinVersion != tls.VersionTLS13 {
			t.Errorf("Expected MinVersion TLS 1.3, got %d", tlsConfig.MinVersion)
		}

		// Test invalid version
		SSLMinVersion = "invalid"
		_, err = SetupTLS()
		if err == nil {
			t.Errorf("Expected error for invalid TLS version, got nil")
		}
	})

	t.Run("custom cipher suite", func(t *testing.T) {
		// Save original values
		originalCipherList := SSLCipherList
		defer func() {
			SSLCipherList = originalCipherList
		}()

		// Test empty cipher list (should default to Go's defaults)
		SSLCipherList = ""
		tlsConfig, err := SetupTLS()
		if err != nil {
			t.Fatalf("Unexpected error setting up TLS with empty cipher list: %v", err)
		}
		if tlsConfig.CipherSuites != nil {
			t.Errorf("Expected nil CipherSuites for empty list, got %v", tlsConfig.CipherSuites)
		}

		// Test single cipher
		SSLCipherList = "TLS_RSA_WITH_AES_256_CBC_SHA"
		tlsConfig, err = SetupTLS()
		if err != nil {
			t.Fatalf("Unexpected error setting up TLS with single cipher: %v", err)
		}
		if len(tlsConfig.CipherSuites) != 1 {
			t.Errorf("Expected 1 cipher suite, got %d", len(tlsConfig.CipherSuites))
		}

		// Test invalid cipher
		SSLCipherList = "INVALID_CIPHER"
		_, err = SetupTLS()
		if err == nil {
			t.Errorf("Expected error for invalid cipher suite, got nil")
		}
	})
}