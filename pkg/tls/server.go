package tls

import (
	"crypto/tls"
	"log/slog"

	"github.com/kuskoman/logstash-exporter/pkg/config"
)

// ConfigureServerTLS configures TLS for the server.
// It handles both the legacy and new TLS configuration formats.
func ConfigureServerTLS(cfg *config.Config) (*tls.Config, error) {
	if cfg.Server.TLSConfig != nil {
		return ConfigureAdvancedServerTLS(cfg.Server.TLSConfig)
	}

	if cfg.Server.EnableSSL {
		return ConfigureLegacyServerTLS(cfg.Server.CertFile, cfg.Server.KeyFile)
	}

	return nil, nil
}

// ConfigureAdvancedServerTLS configures TLS with advanced options.
func ConfigureAdvancedServerTLS(tlsConfig *config.TLSServerConfig) (*tls.Config, error) {
	config := &tls.Config{}

	cert, err := LoadCertificateFromFile(tlsConfig.CertFile, tlsConfig.KeyFile)
	if err != nil {
		return nil, err
	}

	config.Certificates = []tls.Certificate{cert}

	if tlsConfig.ClientCA != "" {
		clientCAs, err := LoadCertificateAuthority(tlsConfig.ClientCA)
		if err != nil {
			return nil, err
		}
		config.ClientCAs = clientCAs
	}

	if tlsConfig.ClientAuth != "" {
		clientAuthType, err := ParseClientAuthType(tlsConfig.ClientAuth)
		if err != nil {
			return nil, err
		}
		config.ClientAuth = clientAuthType
	}

	if tlsConfig.MinVersion != "" {
		minVersion, err := ParseTLSVersion(tlsConfig.MinVersion)
		if err != nil {
			return nil, err
		}
		config.MinVersion = minVersion
	}

	if tlsConfig.MaxVersion != "" {
		maxVersion, err := ParseTLSVersion(tlsConfig.MaxVersion)
		if err != nil {
			return nil, err
		}
		config.MaxVersion = maxVersion
	}

	if len(tlsConfig.CipherSuites) > 0 {
		slog.Warn("cipher suites configuration is not implemented")
	}

	if len(tlsConfig.CurvePreferences) > 0 {
		slog.Warn("curve preferences configuration is not implemented")
	}

	config.PreferServerCipherSuites = tlsConfig.PreferServerCipherSuites

	return config, nil
}

// ConfigureLegacyServerTLS configures TLS with the legacy format.
func ConfigureLegacyServerTLS(certFile, keyFile string) (*tls.Config, error) {
	cert, err := LoadCertificateFromFile(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   DefaultMinTLSVersion,
	}, nil
}
