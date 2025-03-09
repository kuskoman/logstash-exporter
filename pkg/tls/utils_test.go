package tls

import (
	"crypto/tls"
	"testing"
)

func TestParseTLSVersion(t *testing.T) {
	testCases := []struct {
		input       string
		expected    uint16
		expectError bool
	}{
		{TLSVersion10, tls.VersionTLS10, false},
		{"tls10", tls.VersionTLS10, false},
		{TLSVersion11, tls.VersionTLS11, false},
		{TLSVersion12, tls.VersionTLS12, false},
		{TLSVersion13, tls.VersionTLS13, false},
		{"invalid", 0, true},
		{"", 0, true},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			version, err := ParseTLSVersion(tc.input)

			if tc.expectError && err == nil {
				t.Errorf("Expected error for input %s, got nil", tc.input)
			}

			if !tc.expectError && err != nil {
				t.Errorf("Expected no error for input %s, got: %v", tc.input, err)
			}

			if !tc.expectError && version != tc.expected {
				t.Errorf("Expected version %d for input %s, got: %d", tc.expected, tc.input, version)
			}
		})
	}
}

func TestParseClientAuthType(t *testing.T) {
	testCases := []struct {
		input       string
		expected    tls.ClientAuthType
		expectError bool
	}{
		{ClientAuthNone, tls.NoClientCert, false},
		{"noclientcert", tls.NoClientCert, false},
		{ClientAuthRequestClient, tls.RequestClientCert, false},
		{ClientAuthRequireAny, tls.RequireAnyClientCert, false},
		{ClientAuthVerifyIfGiven, tls.VerifyClientCertIfGiven, false},
		{ClientAuthRequireAndVerify, tls.RequireAndVerifyClientCert, false},
		{"invalid", tls.NoClientCert, true},
		{"", tls.NoClientCert, true},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			authType, err := ParseClientAuthType(tc.input)

			if tc.expectError && err == nil {
				t.Errorf("Expected error for input %s, got nil", tc.input)
			}

			if !tc.expectError && err != nil {
				t.Errorf("Expected no error for input %s, got: %v", tc.input, err)
			}

			if !tc.expectError && authType != tc.expected {
				t.Errorf("Expected auth type %v for input %s, got: %v", tc.expected, tc.input, authType)
			}
		})
	}
}
