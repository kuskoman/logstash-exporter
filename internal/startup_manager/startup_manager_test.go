package startup_manager

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/kuskoman/logstash-exporter/internal/flags"
	"github.com/prometheus/client_golang/prometheus"
)

func TestNewStartupManager(t *testing.T) {
	t.Parallel()

	t.Run("creates_a_startup_manager_with_the_default_components", func(t *testing.T) {
		t.Parallel()

		configPath := createValidConfigFile(t)
		sm, err := NewStartupManager(configPath, &flags.FlagsConfig{HotReload: true})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if _, ok := sm.configManager.(*ConfigManager); !ok {
			t.Errorf("expected a *ConfigManager, got %T", sm.configManager)
		}
		if sm.newServer == nil {
			t.Error("expected a server factory")
		}
		if sm.registerer != prometheus.DefaultRegisterer {
			t.Error("expected collectors to be registered in the default registry")
		}
		if sm.watcher == nil {
			t.Error("expected a file watcher")
		}
		if !sm.watchEnabled {
			t.Error("expected hot reload to be enabled by the flag")
		}
		if sm.withController {
			t.Error("expected the Kubernetes controller to be disabled")
		}
		if sm.isInitialized {
			t.Error("expected the startup manager not to be initialized yet")
		}
	})

	t.Run("enables_the_kubernetes_controller", func(t *testing.T) {
		t.Parallel()

		sm, err := NewStartupManagerWithController(createValidConfigFile(t), &flags.FlagsConfig{})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !sm.withController {
			t.Error("expected the Kubernetes controller to be enabled")
		}
	})

	t.Run("returns_an_error_for_a_missing_config_file", func(t *testing.T) {
		t.Parallel()

		sm, err := NewStartupManager("/path/that/does/not/exist.yml", &flags.FlagsConfig{HotReload: true})
		if err == nil {
			t.Error("expected an error for a missing config file")
		}
		if sm != nil {
			t.Errorf("expected no startup manager, got %+v", sm)
		}
	})
}

func TestStartupManager_Shutdown(t *testing.T) {
	t.Parallel()

	t.Run("stops_the_server_and_unregisters_the_collector", func(t *testing.T) {
		t.Parallel()

		cfg := createTestConfig()
		factory := newMockServerFactory()
		sm, registry := newTestStartupManager(newMockConfigSource(cfg), factory)
		sm.serverErrorChan = make(chan error, 1)
		sm.isInitialized = true
		sm.startPrometheus(cfg)
		sm.startServer(cfg)
		server := factory.waitForServer(t)
		collector := sm.prometheusCollector

		if err := sm.Shutdown(context.Background()); err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if calls := server.ShutdownCalls(); calls != 1 {
			t.Errorf("expected the server to be shut down once, got %d calls", calls)
		}
		if isRegistered(registry, collector) {
			t.Error("expected the collector to be unregistered")
		}
	})

	t.Run("returns_an_error_when_not_initialized", func(t *testing.T) {
		t.Parallel()

		sm, _ := newTestStartupManager(newMockConfigSource(createTestConfig()), newMockServerFactory())
		if err := sm.Shutdown(context.Background()); err == nil {
			t.Error("expected an error when shutting down an uninitialized manager")
		}
	})

	t.Run("returns_the_server_shutdown_error", func(t *testing.T) {
		t.Parallel()

		shutdownErr := errors.New("shutdown error")
		server := newMockServer()
		server.shutdownErr = shutdownErr

		sm, _ := newTestStartupManager(newMockConfigSource(createTestConfig()), newMockServerFactory())
		sm.isInitialized = true
		sm.server = server

		if err := sm.Shutdown(context.Background()); !errors.Is(err, shutdownErr) {
			t.Errorf("expected %v, got %v", shutdownErr, err)
		}
	})

	t.Run("succeeds_without_a_server", func(t *testing.T) {
		t.Parallel()

		sm, _ := newTestStartupManager(newMockConfigSource(createTestConfig()), newMockServerFactory())
		sm.isInitialized = true

		if err := sm.Shutdown(context.Background()); err != nil {
			t.Errorf("expected no error without a server, got %v", err)
		}
	})
}

func TestStartupManager_handleServerErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		watchEnabled bool
		shuttingDown bool
		serverErr    error
		propagated   bool
	}{
		{
			name:       "propagates_server_errors",
			serverErr:  errors.New("listen tcp: address already in use"),
			propagated: true,
		},
		{
			name:       "propagates_an_unexpected_server_close_without_hot_reload",
			serverErr:  http.ErrServerClosed,
			propagated: true,
		},
		{
			name:         "ignores_server_close_during_shutdown",
			shuttingDown: true,
			serverErr:    http.ErrServerClosed,
		},
		{
			name:         "ignores_server_close_for_hot_reload",
			watchEnabled: true,
			serverErr:    http.ErrServerClosed,
		},
		{
			name:         "propagates_server_errors_with_hot_reload",
			watchEnabled: true,
			serverErr:    errors.New("listen tcp: address already in use"),
			propagated:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sm, _ := newTestStartupManager(newMockConfigSource(createTestConfig()), newMockServerFactory())
			sm.watchEnabled = tt.watchEnabled
			sm.shuttingDown.Store(tt.shuttingDown)

			applicationErrorChan := make(chan error, 1)
			go sm.handleServerErrors(applicationErrorChan)
			sm.serverErrorChan <- tt.serverErr

			select {
			case err := <-applicationErrorChan:
				if !tt.propagated {
					t.Errorf("expected the error not to be propagated, got %v", err)
				} else if !errors.Is(err, tt.serverErr) {
					t.Errorf("expected %v, got %v", tt.serverErr, err)
				}
			case <-time.After(100 * time.Millisecond):
				if tt.propagated {
					t.Error("expected the error to be propagated")
				}
			}
		})
	}
}
