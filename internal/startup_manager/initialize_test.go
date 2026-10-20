package startup_manager

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kuskoman/logstash-exporter/internal/flags"
	"github.com/prometheus/client_golang/prometheus"
)

func TestStartupManager_Initialize(t *testing.T) {
	t.Parallel()

	t.Run("starts_the_server_with_the_loaded_config", func(t *testing.T) {
		t.Parallel()

		cfg := createTestConfig()
		source := newMockConfigSource(cfg)
		factory := newMockServerFactory()
		sm, registry := newTestStartupManager(source, factory)

		runInitialize(t, sm)
		server := factory.waitForServer(t)

		if configs := factory.Configs(); len(configs) != 1 || configs[0] != cfg {
			t.Errorf("expected the server to be created once with the loaded config, got %v", configs)
		}
		if server.UsedTLS() {
			t.Error("expected a plain HTTP server without TLS configuration")
		}
		if !isRegistered(registry, sm.prometheusCollector) {
			t.Error("expected the collector to be registered")
		}
		if source.ReloadCalls() != 0 {
			t.Error("expected the initial load not to use the reload path")
		}
	})

	t.Run("shuts_down_gracefully_when_the_context_is_cancelled", func(t *testing.T) {
		t.Parallel()

		factory := newMockServerFactory()
		sm, registry := newTestStartupManager(newMockConfigSource(createTestConfig()), factory)

		cancel, result := runInitialize(t, sm)
		server := factory.waitForServer(t)
		collector := sm.prometheusCollector

		cancel()

		if err := waitForResult(t, result); err != nil {
			t.Errorf("expected Initialize to return nil, got %v", err)
		}
		if server.ShutdownCalls() != 1 {
			t.Errorf("expected the server to be shut down once, got %d calls", server.ShutdownCalls())
		}
		if isRegistered(registry, collector) {
			t.Error("expected the collector to be unregistered")
		}
	})

	t.Run("returns_an_error_when_the_server_stops_unexpectedly", func(t *testing.T) {
		t.Parallel()

		factory := newMockServerFactory()
		sm, _ := newTestStartupManager(newMockConfigSource(createTestConfig()), factory)

		_, result := runInitialize(t, sm)
		server := factory.waitForServer(t)

		// Something other than the startup manager stops the server
		if err := server.Shutdown(context.Background()); err != nil {
			t.Fatal(err)
		}

		if err := waitForResult(t, result); !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("expected ErrServerClosed, got %v", err)
		}
	})

	t.Run("returns_the_error_when_the_server_cannot_listen", func(t *testing.T) {
		t.Parallel()

		listenErr := errors.New("listen tcp 127.0.0.1:9198: bind: address already in use")
		factory := newMockServerFactory()
		factory.listenErr = listenErr
		sm, _ := newTestStartupManager(newMockConfigSource(createTestConfig()), factory)

		_, result := runInitialize(t, sm)

		if err := waitForResult(t, result); !errors.Is(err, listenErr) {
			t.Errorf("expected %v, got %v", listenErr, err)
		}
	})

	t.Run("returns_the_error_when_the_server_cannot_be_created", func(t *testing.T) {
		t.Parallel()

		createErr := errors.New("invalid TLS configuration")
		factory := newMockServerFactory()
		factory.err = createErr
		sm, _ := newTestStartupManager(newMockConfigSource(createTestConfig()), factory)

		_, result := runInitialize(t, sm)

		if err := waitForResult(t, result); !errors.Is(err, createErr) {
			t.Errorf("expected %v, got %v", createErr, err)
		}
	})

	t.Run("returns_the_config_load_error", func(t *testing.T) {
		t.Parallel()

		loadErr := errors.New("config load error")
		source := newMockConfigSource(nil)
		source.set(nil, false, loadErr)
		factory := newMockServerFactory()
		sm, _ := newTestStartupManager(source, factory)

		if err := sm.Initialize(context.Background()); !errors.Is(err, loadErr) {
			t.Errorf("expected %v, got %v", loadErr, err)
		}
		if len(factory.Servers()) != 0 {
			t.Error("expected no server to be started")
		}
	})

	t.Run("returns_an_error_when_the_config_is_nil", func(t *testing.T) {
		t.Parallel()

		factory := newMockServerFactory()
		sm, _ := newTestStartupManager(newMockConfigSource(nil), factory)

		if err := sm.Initialize(context.Background()); err == nil {
			t.Error("expected an error for a nil config")
		}
		if len(factory.Servers()) != 0 {
			t.Error("expected no server to be started")
		}
	})

	t.Run("returns_an_error_when_already_initialized", func(t *testing.T) {
		t.Parallel()

		factory := newMockServerFactory()
		sm, _ := newTestStartupManager(newMockConfigSource(createTestConfig()), factory)

		cancel, result := runInitialize(t, sm)
		factory.waitForServer(t)
		cancel()
		if err := waitForResult(t, result); err != nil {
			t.Fatalf("expected the first Initialize to succeed, got %v", err)
		}

		if err := sm.Initialize(context.Background()); !errors.Is(err, ErrAlreadyInitialized) {
			t.Errorf("expected ErrAlreadyInitialized, got %v", err)
		}
		if len(factory.Servers()) != 1 {
			t.Errorf("expected only one server to be started, got %d", len(factory.Servers()))
		}
	})
}

// TestInitializeWithRealComponents runs Initialize with the real config
// manager, file watcher and HTTP server
func TestInitializeWithRealComponents(t *testing.T) {
	t.Parallel()

	t.Run("serves_http_and_shuts_down_when_the_context_is_cancelled", func(t *testing.T) {
		t.Parallel()

		port := getFreePort(t)
		configPath := filepath.Join(t.TempDir(), "config.yml")
		configContent := fmt.Sprintf("server:\n  host: 127.0.0.1\n  port: %d\n", port)
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		sm, err := NewStartupManager(configPath, &flags.FlagsConfig{})
		if err != nil {
			t.Fatalf("failed to create startup manager: %v", err)
		}
		// Keep the default Prometheus registry free for other tests
		sm.registerer = prometheus.NewRegistry()

		cancel, result := runInitialize(t, sm)

		address := fmt.Sprintf("http://127.0.0.1:%d/version", port)
		deadline := time.Now().Add(testTimeout)
		for {
			resp, err := http.Get(address)
			if err == nil {
				_ = resp.Body.Close()
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("server did not start: %v", err)
			}
			time.Sleep(10 * time.Millisecond)
		}

		cancel()

		if err := waitForResult(t, result); err != nil {
			t.Errorf("expected Initialize to return nil after cancellation, got %v", err)
		}
		if resp, err := http.Get(address); err == nil {
			_ = resp.Body.Close()
			t.Error("expected the server to be stopped")
		}
	})

	t.Run("restarts_the_server_when_the_watched_config_changes", func(t *testing.T) {
		t.Parallel()

		configPath := filepath.Join(t.TempDir(), "config.yml")
		writeConfig := func(port int) {
			content := fmt.Sprintf("server:\n  host: 127.0.0.1\n  port: %d\n", port)
			if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
				t.Fatalf("failed to write config: %v", err)
			}
		}
		writeConfig(10001)

		sm, err := NewStartupManager(configPath, &flags.FlagsConfig{HotReload: true})
		if err != nil {
			t.Fatalf("failed to create startup manager: %v", err)
		}
		factory := newMockServerFactory()
		sm.newServer = factory.create
		sm.registerer = prometheus.NewRegistry()

		runInitialize(t, sm)
		first := factory.waitForServer(t)

		writeConfig(10002)
		second := factory.waitForServer(t)

		if first.ShutdownCalls() != 1 {
			t.Errorf("expected the first server to be shut down, got %d calls", first.ShutdownCalls())
		}
		configs := factory.Configs()
		if port := configs[len(configs)-1].Server.Port; port != 10002 {
			t.Errorf("expected the new server to use port 10002, got %d", port)
		}
		if second.ShutdownCalls() != 0 {
			t.Error("expected the new server to keep running")
		}
	})
}
