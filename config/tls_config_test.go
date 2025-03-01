package config

import (
	"crypto/tls"
	"testing"
)

func TestSetupTLS(t *testing.T) {
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
}