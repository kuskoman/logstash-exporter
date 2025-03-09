package tls

import (
	"crypto/tls"
	"fmt"
	"strings"
)

// ParseTLSVersion converts a string TLS version to a uint16 value.
func ParseTLSVersion(version string) (uint16, error) {
	switch strings.ToUpper(version) {
	case TLSVersion10:
		return tls.VersionTLS10, nil
	case TLSVersion11:
		return tls.VersionTLS11, nil
	case TLSVersion12:
		return tls.VersionTLS12, nil
	case TLSVersion13:
		return tls.VersionTLS13, nil
	default:
		return 0, fmt.Errorf("unsupported TLS version: %s", version)
	}
}

// ParseClientAuthType converts a string client auth type to a tls.ClientAuthType.
func ParseClientAuthType(authType string) (tls.ClientAuthType, error) {
	switch strings.ToUpper(authType) {
	case ClientAuthNone:
		return tls.NoClientCert, nil
	case ClientAuthRequestClient:
		return tls.RequestClientCert, nil
	case ClientAuthRequireAny:
		return tls.RequireAnyClientCert, nil
	case ClientAuthVerifyIfGiven:
		return tls.VerifyClientCertIfGiven, nil
	case ClientAuthRequireAndVerify:
		return tls.RequireAndVerifyClientCert, nil
	default:
		return tls.NoClientCert, fmt.Errorf("unsupported client auth type: %s", authType)
	}
}
