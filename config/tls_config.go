package config

import (
    "fmt"
    "crypto/tls"
	"strings"
)

var(
	SSLCipherList = getEnvWithDefault("SSL_CIPHER_LIST","TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,TLS_RSA_WITH_AES_256_GCM_SHA384,TLS_RSA_WITH_AES_256_CBC_SHA")
	SSLMinVersion = getEnvWithDefault("SSL_MIN_VERSION","1.2")
)

func SetupTLS() (*tls.Config, error) {
	var cipherSuites []uint16
	if SSLCipherList != "" {
		cipherMap := make(map[string]uint16)
		for _, suite := range tls.CipherSuites() {
			cipherMap[suite.Name] = suite.ID
		}
		for _, suite := range tls.InsecureCipherSuites() {
			cipherMap[suite.Name] = suite.ID
		}

		for _, cipher := range strings.Split(SSLCipherList, ",") {
			cipher = strings.TrimSpace(cipher)
			if id, exists := cipherMap[cipher]; exists {
				cipherSuites = append(cipherSuites, id)
			} else {
				return nil, fmt.Errorf("unsupported cipher suite: %s", cipher)
			}
		}
	}

	var minVersion uint16
	switch SSLMinVersion {
	case "1.0":
		minVersion = tls.VersionTLS10
	case "1.1":
		minVersion = tls.VersionTLS11
	case "1.2":
		minVersion = tls.VersionTLS12
	case "1.3", "":
		minVersion = tls.VersionTLS13 
	default:
		return nil, fmt.Errorf("invalid TLS version: %s", SSLMinVersion)
	}

	tlsConfig := &tls.Config{
		MinVersion:   minVersion,
		CipherSuites: cipherSuites,
	}

	return tlsConfig, nil
}