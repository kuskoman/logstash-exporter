package tls

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCertificateFromFile(t *testing.T) {
	// Create temporary directory for test certificates
	tempDir, err := os.MkdirTemp("", "cert-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Get test certificates and save them to files
	testCerts := GetTestCertificates()
	certPath := filepath.Join(tempDir, "cert.pem")
	keyPath := filepath.Join(tempDir, "key.pem")

	if err := os.WriteFile(certPath, []byte(testCerts.CertPEM), 0600); err != nil {
		t.Fatalf("Failed to write cert file: %v", err)
	}
	if err := os.WriteFile(keyPath, []byte(testCerts.KeyPEM), 0600); err != nil {
		t.Fatalf("Failed to write key file: %v", err)
	}

	// Test successful loading
	cert, err := LoadCertificateFromFile(certPath, keyPath)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if len(cert.Certificate) == 0 {
		t.Error("Expected certificate to be loaded, got empty certificate")
	}

	// Test invalid certificate path
	_, err = LoadCertificateFromFile("/nonexistent/cert.pem", keyPath)
	if err == nil {
		t.Error("Expected error for invalid certificate path, got nil")
	}

	// Test invalid key path
	_, err = LoadCertificateFromFile(certPath, "/nonexistent/key.pem")
	if err == nil {
		t.Error("Expected error for invalid key path, got nil")
	}
}

func TestLoadCertificateAuthority(t *testing.T) {
	// Create temporary directory for test certificates
	tempDir, err := os.MkdirTemp("", "ca-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Get test certificates and save them to files
	testCerts := GetTestCertificates()
	caPath := filepath.Join(tempDir, "ca.pem")
	invalidCAPath := filepath.Join(tempDir, "invalid-ca.pem")

	if err := os.WriteFile(caPath, []byte(testCerts.CAPEM), 0600); err != nil {
		t.Fatalf("Failed to write CA file: %v", err)
	}

	// Write invalid CA (not a valid PEM format)
	if err := os.WriteFile(invalidCAPath, []byte("NOT A VALID CA CERTIFICATE"), 0600); err != nil {
		t.Fatalf("Failed to write invalid CA file: %v", err)
	}

	// Test successful loading
	certPool, err := LoadCertificateAuthority(caPath)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if certPool == nil {
		t.Error("Expected cert pool to be loaded, got nil")
	}

	// Test invalid CA path
	_, err = LoadCertificateAuthority("/nonexistent/ca.pem")
	if err == nil {
		t.Error("Expected error for invalid CA path, got nil")
	}

	// Test invalid CA content (not a valid PEM format)
	_, err = LoadCertificateAuthority(invalidCAPath)
	if err == nil {
		t.Error("Expected error for invalid CA content, got nil")
	}
}
