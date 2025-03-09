package tls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

// LoadCertificateFromFile loads a certificate from the given file path.
func LoadCertificateFromFile(certFile, keyFile string) (tls.Certificate, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("error loading key pair: %w", err)
	}
	return cert, nil
}

// LoadCertificateAuthority loads a CA certificate from the given file path.
func LoadCertificateAuthority(caFile string) (*x509.CertPool, error) {
	caData, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA file: %w", err)
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(caData) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	return certPool, nil
}
