package logstash_client

// TestCertificates contains certificate data for testing.
// These are self-signed certificates for testing purposes only.
// DO NOT USE IN PRODUCTION.
type TestCertificates struct {
	// CA certificate in PEM format
	CAPEM string
}

// GetTestCertificates returns test certificate data for use in tests.
func GetTestCertificates() TestCertificates {
	return TestCertificates{
		CAPEM: `-----BEGIN CERTIFICATE-----
MIIBhTCCASugAwIBAgIQIRi6zePL6mKjOipn+dNuaTAKBggqhkjOPQQDAjASMRAw
DgYDVQQKEwdBY21lIENvMB4XDTE3MTAyMDE5NDMwNloXDTE4MTAyMDE5NDMwNlow
EjEQMA4GA1UEChMHQWNtZSBDbzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABD0d
7VNhbWvZLWPuj/RtHFjvtJBEwOkhbN/BnnE8rnZR8+sbwnc/KhCk3FhnpHZnQz7B
5aETbbIgmuvewdjvSBSjYzBhMA4GA1UdDwEB/wQEAwICpDATBgNVHSUEDDAKBggr
BgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MCkGA1UdEQQiMCCCDmxvY2FsaG9zdDo1
NDUzgg4xMjcuMC4wLjE6NTQ1MzAKBggqhkjOPQQDAgNIADBFAiEA2zpJEPQyz6/l
Wf86aX6PepsntZv2GYlA5UpabfT2EZICICpJ5h/iI+i341gBmLiAFQOyTDT+/wQc
6MF9+Yw1Yy0t
-----END CERTIFICATE-----`,
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
