package tls

import (
	"os"
	"path/filepath"
	"runtime"
)

// TestCertificates contains certificate data for testing.
// These are self-signed certificates for testing purposes only.
// DO NOT USE IN PRODUCTION.
type TestCertificates struct {
	// Certificate in PEM format
	CertPEM string
	// Private key in PEM format
	KeyPEM string
	// CA certificate in PEM format
	CAPEM string
}

// GetTestCertificates returns test certificate data for use in tests.
// It loads the certificates from the fixtures/https directory.
func GetTestCertificates() TestCertificates {
	// Get the path to the fixtures directory
	_, thisFile, _, _ := runtime.Caller(0)
	fixturesDir := filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(thisFile))), "fixtures", "https")

	// Load the certificates
	certPEM, _ := os.ReadFile(filepath.Join(fixturesDir, "server.crt"))
	keyPEM, _ := os.ReadFile(filepath.Join(fixturesDir, "server.key"))
	caPEM, _ := os.ReadFile(filepath.Join(fixturesDir, "ca.crt"))

	return TestCertificates{
		CertPEM: string(certPEM),
		KeyPEM:  string(keyPEM),
		CAPEM:   string(caPEM),
	}
}

// Constants for test configuration
const (
	TestBaseURL       = "https://example.com:9600"
	TestUsername      = "testuser"
	TestPassword      = "testpass"
	TestTimeout       = 5 // seconds
	TestServerName    = "custom.example.com"
	TestNonexistentCA = "/nonexistent/ca.pem"
)
