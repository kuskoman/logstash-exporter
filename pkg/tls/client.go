package tls

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/kuskoman/logstash-exporter/pkg/config"
)

// ConfigureClientTLS creates a TLS configuration for a client connection.
func ConfigureClientTLS(caFile, serverName string, insecureSkipVerify bool) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: insecureSkipVerify,
	}

	// If server name is specified, set it in the TLS config
	if serverName != "" {
		tlsConfig.ServerName = serverName
	}

	// If CA file is specified, load it
	if caFile != "" {
		certPool, err := LoadCertificateAuthority(caFile)
		if err != nil {
			return nil, err
		}
		tlsConfig.RootCAs = certPool
	}

	return tlsConfig, nil
}

// ConfigureHTTPClientWithTLS creates an HTTP client with TLS configuration.
func ConfigureHTTPClientWithTLS(timeout time.Duration, caFile, serverName string, insecureSkipVerify bool) (*http.Client, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()

	// Configure TLS
	tlsConfig, err := ConfigureClientTLS(caFile, serverName, insecureSkipVerify)
	if err != nil {
		return nil, err
	}

	// Set the TLS config in the transport
	transport.TLSClientConfig = tlsConfig

	// Create and return the HTTP client
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}, nil
}

// ConfigureHTTPClientFromLogstashInstance creates an HTTP client from a Logstash instance configuration.
func ConfigureHTTPClientFromLogstashInstance(instance *config.LogstashInstance, timeout time.Duration) (*http.Client, error) {
	// Determine if we should use TLS configuration
	if instance.TLSConfig != nil {
		// Use advanced TLS configuration
		return ConfigureHTTPClientWithTLS(
			timeout,
			instance.TLSConfig.CAFile,
			instance.TLSConfig.ServerName,
			instance.TLSConfig.InsecureSkipVerify || instance.HttpInsecure, // Support both new and legacy config
		)
	}

	// Fall back to legacy configuration
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: instance.HttpInsecure,
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}, nil
}

// ConfigureBasicAuth adds basic authentication to an HTTP client's transport.
// This method is for single user authentication only.
// NOTE: This will be updated in a future release to support multiple users.
func ConfigureBasicAuth(client *http.Client, username, password string) *http.Client {
	if client == nil {
		return nil
	}

	// Create a new transport that wraps the existing one and adds basic auth
	client.Transport = &basicAuthTransport{
		username:  username,
		password:  password,
		transport: client.Transport,
	}

	return client
}

// basicAuthTransport adds basic authentication to requests.
type basicAuthTransport struct {
	username  string
	password  string
	transport http.RoundTripper
}

// RoundTrip implements the http.RoundTripper interface.
func (t *basicAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request to avoid modifying the original
	req2 := req.Clone(req.Context())

	// Add basic auth header
	req2.SetBasicAuth(t.username, t.password)

	// Pass the request to the underlying transport
	return t.transport.RoundTrip(req2)
}
