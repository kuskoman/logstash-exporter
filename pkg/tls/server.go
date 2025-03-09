package tls

import (
	"crypto/tls"
	"log/slog"

	"github.com/kuskoman/logstash-exporter/pkg/config"
)

// ConfigureServerTLS configures TLS for the server.
func ConfigureServerTLS(cfg *config.Config) (*tls.Config, error) {
	if cfg.Server.TLSConfig != nil {
		return ConfigureAdvancedServerTLS(cfg.Server.TLSConfig)
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

	return config, nil
}
