package logstash_client

import (
	"os"
	"path/filepath"
	"runtime"
)

// TestCertificates contains certificate data for testing.
// These are self-signed certificates for testing purposes only.
// DO NOT USE IN PRODUCTION.
type TestCertificates struct {
	// CA certificate in PEM format
	CAPEM string
}

// GetTestCertificates returns test certificate data for use in tests.
func GetTestCertificates() TestCertificates {
	// Get the path to the fixtures directory
	_, thisFile, _, _ := runtime.Caller(0)
	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(thisFile))))
	fixturesDir := filepath.Join(projectRoot, "fixtures", "https")

	// Load the CA certificate
	caPEM, err := os.ReadFile(filepath.Join(fixturesDir, "ca.crt"))
	if err != nil {
		// We should never get here in normal operation, but this provides a safety net
		panic("Failed to read CA certificate file: " + err.Error())
	}

	return TestCertificates{
		CAPEM: string(caPEM),
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
