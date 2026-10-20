package startup_manager

import (
	"context"
	"errors"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kuskoman/logstash-exporter/pkg/config"
)

func TestStartServer(t *testing.T) {
	t.Parallel()

	t.Run("serves_https_when_tls_is_configured", func(t *testing.T) {
		t.Parallel()

		cfg := createTestConfig()
		cfg.Server.TLSConfig = &config.TLSServerConfig{CertFile: "server.crt", KeyFile: "server.key"}

		factory := newMockServerFactory()
		sm, _ := newTestStartupManager(newMockConfigSource(cfg), factory)
		sm.serverErrorChan = make(chan error, 1)

		sm.startServer(cfg)
		server := factory.waitForServer(t)

		if !server.UsedTLS() {
			t.Error("expected ListenAndServeTLS to be used")
		}
		if sm.server != server {
			t.Error("expected the started server to be kept for shutdown")
		}
	})

	t.Run("serves_http_without_tls", func(t *testing.T) {
		t.Parallel()

		cfg := createTestConfig()
		factory := newMockServerFactory()
		sm, _ := newTestStartupManager(newMockConfigSource(cfg), factory)
		sm.serverErrorChan = make(chan error, 1)

		sm.startServer(cfg)
		server := factory.waitForServer(t)

		if server.UsedTLS() {
			t.Error("expected ListenAndServe to be used")
		}
	})

	t.Run("reports_errors_creating_the_server", func(t *testing.T) {
		t.Parallel()

		createErr := errors.New("invalid TLS configuration")
		factory := newMockServerFactory()
		factory.err = createErr
		sm, _ := newTestStartupManager(newMockConfigSource(createTestConfig()), factory)
		sm.serverErrorChan = make(chan error, 1)

		sm.startServer(createTestConfig())

		select {
		case err := <-sm.serverErrorChan:
			if !errors.Is(err, createErr) {
				t.Errorf("expected %v, got %v", createErr, err)
			}
		case <-time.After(testTimeout):
			t.Fatal("expected the error to be reported")
		}
		if sm.server != nil {
			t.Error("expected no server to be kept")
		}
	})

	t.Run("reports_errors_from_the_running_server", func(t *testing.T) {
		t.Parallel()

		listenErr := errors.New("bind: address already in use")
		factory := newMockServerFactory()
		factory.listenErr = listenErr
		sm, _ := newTestStartupManager(newMockConfigSource(createTestConfig()), factory)
		sm.serverErrorChan = make(chan error, 1)

		sm.startServer(createTestConfig())

		select {
		case err := <-sm.serverErrorChan:
			if !errors.Is(err, listenErr) {
				t.Errorf("expected %v, got %v", listenErr, err)
			}
		case <-time.After(testTimeout):
			t.Fatal("expected the error to be reported")
		}
	})
}

func TestNewAppServer(t *testing.T) {
	t.Parallel()

	fixture := func(name string) string {
		return filepath.Join("..", "..", "fixtures", "https", name)
	}

	t.Run("creates_an_http_server_for_the_configured_address", func(t *testing.T) {
		t.Parallel()

		cfg := createTestConfig()
		cfg.Server.Port = 9300

		appServer, err := newAppServer(cfg)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		httpServer, ok := appServer.(*http.Server)
		if !ok {
			t.Fatalf("expected an *http.Server, got %T", appServer)
		}
		if httpServer.Addr != "127.0.0.1:9300" {
			t.Errorf("expected address 127.0.0.1:9300, got %s", httpServer.Addr)
		}
		if httpServer.TLSConfig != nil {
			t.Error("expected no TLS configuration")
		}
	})

	t.Run("configures_tls_from_the_certificate_files", func(t *testing.T) {
		t.Parallel()

		cfg := createTestConfig()
		cfg.Server.TLSConfig = &config.TLSServerConfig{
			CertFile: fixture("server.crt"),
			KeyFile:  fixture("server.key"),
		}

		appServer, err := newAppServer(cfg)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		httpServer := appServer.(*http.Server)
		if httpServer.TLSConfig == nil || len(httpServer.TLSConfig.Certificates) != 1 {
			t.Error("expected the certificate to be loaded into the TLS configuration")
		}
	})

	t.Run("rejects_tls_without_a_key_file", func(t *testing.T) {
		t.Parallel()

		cfg := createTestConfig()
		cfg.Server.TLSConfig = &config.TLSServerConfig{CertFile: fixture("server.crt")}

		_, err := newAppServer(cfg)
		if err == nil || !strings.Contains(err.Error(), "invalid TLS configuration") {
			t.Errorf("expected an invalid TLS configuration error, got %v", err)
		}
	})

	t.Run("rejects_unreadable_certificate_files", func(t *testing.T) {
		t.Parallel()

		cfg := createTestConfig()
		cfg.Server.TLSConfig = &config.TLSServerConfig{
			CertFile: fixture("missing.crt"),
			KeyFile:  fixture("missing.key"),
		}

		_, err := newAppServer(cfg)
		if err == nil || !strings.Contains(err.Error(), "failed to configure TLS") {
			t.Errorf("expected a TLS configuration error, got %v", err)
		}
	})
}

func TestStartPrometheus(t *testing.T) {
	t.Parallel()

	t.Run("registers_a_collector", func(t *testing.T) {
		t.Parallel()

		sm, registry := newTestStartupManager(newMockConfigSource(createTestConfig()), newMockServerFactory())
		sm.startPrometheus(createTestConfig())

		if sm.prometheusCollector == nil || !isRegistered(registry, sm.prometheusCollector) {
			t.Error("expected the collector to be registered")
		}
	})

	t.Run("replaces_the_previous_collector", func(t *testing.T) {
		t.Parallel()

		sm, registry := newTestStartupManager(newMockConfigSource(createTestConfig()), newMockServerFactory())
		sm.startPrometheus(createTestConfig())
		first := sm.prometheusCollector

		sm.startPrometheus(createTestConfig())

		if isRegistered(registry, first) {
			t.Error("expected the previous collector to be unregistered")
		}
		if !isRegistered(registry, sm.prometheusCollector) {
			t.Error("expected the new collector to be registered")
		}
	})

	t.Run("shutdown_unregisters_the_collector", func(t *testing.T) {
		t.Parallel()

		sm, registry := newTestStartupManager(newMockConfigSource(createTestConfig()), newMockServerFactory())
		sm.startPrometheus(createTestConfig())

		sm.shutdownPrometheus()

		if isRegistered(registry, sm.prometheusCollector) {
			t.Error("expected the collector to be unregistered")
		}
	})
}

// blockingServer is an AppServer whose Shutdown blocks until the context is done
type blockingServer struct{ mockServer }

func (s *blockingServer) Shutdown(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}

func TestShutdownServer(t *testing.T) {
	t.Parallel()

	newManagerWithServer := func(server AppServer) *StartupManager {
		sm, _ := newTestStartupManager(newMockConfigSource(createTestConfig()), newMockServerFactory())
		sm.server = server
		return sm
	}

	t.Run("treats_an_already_closed_server_as_success", func(t *testing.T) {
		t.Parallel()

		server := newMockServer()
		server.shutdownErr = http.ErrServerClosed

		if err := newManagerWithServer(server).shutdownServer(context.Background()); err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if server.ShutdownCalls() != 1 {
			t.Error("expected the server to be shut down")
		}
	})

	t.Run("returns_other_shutdown_errors", func(t *testing.T) {
		t.Parallel()

		shutdownErr := errors.New("custom server error")
		server := newMockServer()
		server.shutdownErr = shutdownErr

		if err := newManagerWithServer(server).shutdownServer(context.Background()); !errors.Is(err, shutdownErr) {
			t.Errorf("expected %v, got %v", shutdownErr, err)
		}
	})

	t.Run("returns_when_the_context_expires", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := newManagerWithServer(&blockingServer{}).shutdownServer(ctx)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("expected context.DeadlineExceeded, got %v", err)
		}
	})

	t.Run("succeeds_without_a_server", func(t *testing.T) {
		t.Parallel()

		if err := newManagerWithServer(nil).shutdownServer(context.Background()); err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
}
