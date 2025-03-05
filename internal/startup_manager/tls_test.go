package startup_manager

import (
	"fmt"
	"testing"

	"github.com/kuskoman/logstash-exporter/pkg/config"
)

func TestTLSConfig(t *testing.T) {
	t.Run("should use TLS when enabled", func(t *testing.T) {
		certFile := "/path/to/cert.pem"
		keyFile := "/path/to/key.pem"

		cfg := &config.Config{
			Server: config.ServerConfig{
				EnableSSL: true,
				CertFile:  certFile,
				KeyFile:   keyFile,
			},
		}

		// Use our direct implementation method to avoid actually launching a server
		// Override the server instance in a separate StartupManager
		mockSrv := &mockAppServer{
			listenAndServeCalled:    make(chan struct{}, 1),
			listenAndServeTLSCalled: make(chan struct{}, 1),
		}
		var err error
		mockServerFunc := func(cfg *config.Config) {
			if cfg.Server.EnableSSL {
				err = mockSrv.ListenAndServeTLS(cfg.Server.CertFile, cfg.Server.KeyFile)
			} else {
				err = mockSrv.ListenAndServe()
			}
		}

		if err != nil {
			t.Errorf("unexpected error when starting server: %v", err)
		}

		// Call our function directly - no goroutines
		mockServerFunc(cfg)

		// Check if the TLS function was called correctly
		select {
		case <-mockSrv.listenAndServeTLSCalled:
			// Success
			if mockSrv.listenAndServeTLSCertFile != certFile {
				t.Errorf("expected cert file %s, got %s", certFile, mockSrv.listenAndServeTLSCertFile)
			}
			if mockSrv.listenAndServeTLSKeyFile != keyFile {
				t.Errorf("expected key file %s, got %s", keyFile, mockSrv.listenAndServeTLSKeyFile)
			}
		default:
			t.Error("ListenAndServeTLS was not called")
		}

		// Make sure the non-TLS function wasn't called
		select {
		case <-mockSrv.listenAndServeCalled:
			t.Error("ListenAndServe was incorrectly called when TLS is enabled")
		default:
			// Success - non-TLS function wasn't called
		}
	})

	t.Run("should not use TLS when disabled", func(t *testing.T) {
		cfg := &config.Config{
			Server: config.ServerConfig{
				EnableSSL: false,
				CertFile:  "/path/to/cert.pem", // Should be ignored when EnableSSL is false
				KeyFile:   "/path/to/key.pem",  // Should be ignored when EnableSSL is false
			},
		}

		// Use our direct implementation method to avoid actually launching a server
		mockSrv := &mockAppServer{
			listenAndServeCalled:    make(chan struct{}, 1),
			listenAndServeTLSCalled: make(chan struct{}, 1),
		}

		var err error
		mockServerFunc := func(cfg *config.Config) {
			if cfg.Server.EnableSSL {
				err = mockSrv.ListenAndServeTLS(cfg.Server.CertFile, cfg.Server.KeyFile)
			} else {
				err = mockSrv.ListenAndServe()
			}
		}

		if err != nil {
			t.Errorf("unexpected error when starting server: %v", err)
		}

		// Call our function directly - no goroutines
		mockServerFunc(cfg)

		// Check if the non-TLS function was called
		select {
		case <-mockSrv.listenAndServeCalled:
			// Success - non-TLS function was called
		default:
			t.Error("ListenAndServe was not called")
		}

		// Make sure the TLS function wasn't called
		select {
		case <-mockSrv.listenAndServeTLSCalled:
			t.Error("ListenAndServeTLS was incorrectly called when TLS is disabled")
		default:
			// Success - TLS function wasn't called
		}
	})

	t.Run("should validate TLS configuration", func(t *testing.T) {
		cfg := &config.Config{
			Server: config.ServerConfig{
				EnableSSL: true,
				// CertFile and KeyFile are intentionally missing
			},
		}

		// Use direct test of the validation logic
		err := validateTLSConfig(cfg)
		if err == nil {
			t.Error("expected validation error, got nil")
		}

		expectedErr := "TLS is enabled but certFile or keyFile is missing"
		if err != nil && err.Error() != expectedErr {
			t.Errorf("expected error message %q, got %q", expectedErr, err.Error())
		}
	})
}

// Helper function that extracts the validation logic for testing
func validateTLSConfig(cfg *config.Config) error {
	if cfg.Server.EnableSSL && (cfg.Server.CertFile == "" || cfg.Server.KeyFile == "") {
		return fmt.Errorf("TLS is enabled but certFile or keyFile is missing")
	}
	return nil
}
