package tls

import (
	"crypto/tls"
)

// TLS version constants used in configuration files.
// These string constants are mapped to the corresponding tls package constants.
const (
	// TLSVersion10 represents TLS 1.0 (DEPRECATED - use only if legacy support is required)
	TLSVersion10 = "TLS10"

	// TLSVersion11 represents TLS 1.1 (DEPRECATED - use only if legacy support is required)
	TLSVersion11 = "TLS11"

	// TLSVersion12 represents TLS 1.2 (recommended minimum version)
	TLSVersion12 = "TLS12"

	// TLSVersion13 represents TLS 1.3 (most secure option)
	TLSVersion13 = "TLS13"

	// DefaultMinTLSVersion is the default minimum TLS version when not specified
	// Uses TLS 1.2 for backward compatibility and security
	DefaultMinTLSVersion = tls.VersionTLS12
)

// Client authentication type constants define how server validates client certificates.
// These string constants are mapped to the corresponding tls.ClientAuthType values.
const (
	// ClientAuthNone - server does not request client cert (tls.NoClientCert)
	ClientAuthNone = "NOCLIENTCERT"

	// ClientAuthRequestClient - server requests client cert but doesn't require it (tls.RequestClientCert)
	ClientAuthRequestClient = "REQUESTCLIENTCERT"

	// ClientAuthRequireAny - server requires any client cert without validation (tls.RequireAnyClientCert)
	ClientAuthRequireAny = "REQUIREANYCLIENTCERT"

	// ClientAuthVerifyIfGiven - server verifies client cert only if provided (tls.VerifyClientCertIfGiven)
	ClientAuthVerifyIfGiven = "VERIFYCLIENTCERTIFGIVEN"

	// ClientAuthRequireAndVerify - server requires and validates client cert (tls.RequireAndVerifyClientCert)
	// Most secure option for client certificate authentication
	ClientAuthRequireAndVerify = "REQUIREANDVERIFYCLIENTCERT"
)

// HTTP authentication constants
const (
	// BasicAuthRealm is the realm name used in HTTP Basic Authentication
	BasicAuthRealm = "Restricted"
)
