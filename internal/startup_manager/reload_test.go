package startup_manager

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kuskoman/logstash-exporter/internal/file_watcher"
	"github.com/kuskoman/logstash-exporter/internal/flags"
	"github.com/prometheus/client_golang/prometheus"
)

// startTestManager runs Initialize with a mock config source and server
// factory and waits until the first server is listening
func startTestManager(t *testing.T) (*StartupManager, *mockConfigSource, *mockServerFactory, *prometheus.Registry, *mockServer) {
	t.Helper()

	source := newMockConfigSource(createTestConfig())
	factory := newMockServerFactory()
	sm, registry := newTestStartupManager(source, factory)

	// Reloads restart the server, which Initialize only tolerates with hot
	// reload enabled. The watched file is never changed by these tests.
	watcher, err := file_watcher.NewFileWatcher(createValidConfigFile(t), sm.handleConfigChange)
	if err != nil {
		t.Fatalf("failed to create file watcher: %v", err)
	}
	sm.watcher = watcher
	sm.watchEnabled = true

	runInitialize(t, sm)
	server := factory.waitForServer(t)

	return sm, source, factory, registry, server
}

func TestStartupManager_Reload(t *testing.T) {
	t.Parallel()

	t.Run("restarts_the_server_and_collector_when_the_config_changed", func(t *testing.T) {
		t.Parallel()

		sm, source, factory, registry, oldServer := startTestManager(t)
		oldCollector := sm.prometheusCollector

		newConfig := createTestConfig()
		newConfig.Server.Port = 9300
		source.set(newConfig, true, nil)

		if err := sm.Reload(context.Background()); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		newServer := factory.waitForServer(t)

		if oldServer.ShutdownCalls() != 1 {
			t.Errorf("expected the old server to be shut down, got %d calls", oldServer.ShutdownCalls())
		}
		if configs := factory.Configs(); configs[len(configs)-1] != newConfig {
			t.Error("expected the new server to use the new config")
		}
		if newServer.ShutdownCalls() != 0 {
			t.Error("expected the new server to keep running")
		}
		if isRegistered(registry, oldCollector) {
			t.Error("expected the old collector to be unregistered")
		}
		if !isRegistered(registry, sm.prometheusCollector) {
			t.Error("expected the new collector to be registered")
		}
		if sm.prometheusCollector == oldCollector {
			t.Error("expected a new collector for the new config")
		}
	})

	t.Run("uses_the_reload_path_of_the_config_source", func(t *testing.T) {
		t.Parallel()

		sm, source, _, _, _ := startTestManager(t)
		source.set(createTestConfig(), false, nil)

		if err := sm.Reload(context.Background()); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if source.ReloadCalls() != 1 {
			t.Errorf("expected ReloadAndCompareConfig to be called once, got %d", source.ReloadCalls())
		}
	})

	t.Run("keeps_the_server_when_the_config_is_unchanged", func(t *testing.T) {
		t.Parallel()

		sm, source, factory, registry, server := startTestManager(t)
		collector := sm.prometheusCollector
		source.set(createTestConfig(), false, nil)

		if err := sm.Reload(context.Background()); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if server.ShutdownCalls() != 0 {
			t.Error("expected the server not to be restarted")
		}
		if len(factory.Servers()) != 1 {
			t.Errorf("expected no new server, got %d servers", len(factory.Servers()))
		}
		if !isRegistered(registry, collector) {
			t.Error("expected the collector to stay registered")
		}
	})

	t.Run("keeps_the_server_when_the_config_cannot_be_loaded", func(t *testing.T) {
		t.Parallel()

		sm, source, factory, _, server := startTestManager(t)
		source.set(nil, false, ErrEmptyConfigFile)

		if err := sm.Reload(context.Background()); !errors.Is(err, ErrEmptyConfigFile) {
			t.Errorf("expected %v, got %v", ErrEmptyConfigFile, err)
		}
		if server.ShutdownCalls() != 0 || len(factory.Servers()) != 1 {
			t.Error("expected the running server to be kept")
		}
	})

	t.Run("returns_an_error_when_the_changed_config_is_nil", func(t *testing.T) {
		t.Parallel()

		sm, source, factory, _, server := startTestManager(t)
		source.set(nil, true, nil)

		if err := sm.Reload(context.Background()); err == nil {
			t.Error("expected an error for a nil config")
		}
		if server.ShutdownCalls() != 0 || len(factory.Servers()) != 1 {
			t.Error("expected the running server to be kept")
		}
	})

	t.Run("returns_the_shutdown_error_and_does_not_start_a_new_server", func(t *testing.T) {
		t.Parallel()

		sm, source, factory, _, server := startTestManager(t)
		shutdownErr := errors.New("server shutdown error")
		server.mu.Lock()
		server.shutdownErr = shutdownErr
		server.mu.Unlock()
		source.set(createTestConfig(), true, nil)

		if err := sm.Reload(context.Background()); !errors.Is(err, shutdownErr) {
			t.Errorf("expected %v, got %v", shutdownErr, err)
		}
		if len(factory.Servers()) != 1 {
			t.Errorf("expected no new server, got %d servers", len(factory.Servers()))
		}
	})
}

func TestStartupManager_handleConfigChange(t *testing.T) {
	t.Parallel()

	t.Run("reloads_with_the_shutdown_timeout", func(t *testing.T) {
		t.Parallel()

		sm, source, _, _, _ := startTestManager(t)
		source.set(createTestConfig(), false, nil)

		before := time.Now()
		if err := sm.handleConfigChange(); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		source.mu.Lock()
		ctx := source.reloadCtx
		source.mu.Unlock()
		if ctx == nil {
			t.Fatal("expected the config to be reloaded")
		}
		deadline, ok := ctx.Deadline()
		if !ok {
			t.Fatal("expected the reload context to have a deadline")
		}
		if deadline.Before(before.Add(ServerShutdownTimeout)) || deadline.After(time.Now().Add(ServerShutdownTimeout)) {
			t.Errorf("expected a deadline %s from now, got %s", ServerShutdownTimeout, time.Until(deadline))
		}
	})

	t.Run("returns_the_reload_error", func(t *testing.T) {
		t.Parallel()

		sm, source, _, _, _ := startTestManager(t)
		reloadErr := errors.New("reload error")
		source.set(nil, false, reloadErr)

		if err := sm.handleConfigChange(); !errors.Is(err, reloadErr) {
			t.Errorf("expected %v, got %v", reloadErr, err)
		}
	})
}

// TestReloadWithConfigFile reloads using the real config manager and a config
// file on disk
func TestReloadWithConfigFile(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "config.yml")
	writeConfig := func(content string) {
		if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}
	}
	writeConfig(`
logstash:
  instances:
    - url: http://localhost:9600
      name: initial
server:
  host: 127.0.0.1
  port: 10001
`)

	sm, err := NewStartupManager(configPath, &flags.FlagsConfig{})
	if err != nil {
		t.Fatalf("failed to create startup manager: %v", err)
	}
	factory := newMockServerFactory()
	sm.newServer = factory.create
	sm.registerer = prometheus.NewRegistry()
	sm.serverErrorChan = make(chan error, 10)

	// Start the components like Initialize does, without blocking and without
	// the file watcher, so that only the explicit Reload calls below reload
	if _, err := sm.configManager.LoadAndCompareConfig(context.Background()); err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	sm.startPrometheus(sm.configManager.GetCurrentConfig())
	sm.startServer(sm.configManager.GetCurrentConfig())
	factory.waitForServer(t)

	t.Run("applies_the_changed_file", func(t *testing.T) {
		writeConfig(`
logstash:
  instances:
    - url: http://localhost:9600
      name: updated
    - url: http://localhost:9601
      name: second
server:
  host: 127.0.0.1
  port: 10002
logging:
  level: debug
`)

		if err := sm.Reload(context.Background()); err != nil {
			t.Fatalf("reload failed: %v", err)
		}
		factory.waitForServer(t)

		configs := factory.Configs()
		cfg := configs[len(configs)-1]
		if len(cfg.Logstash.Instances) != 2 || cfg.Logstash.Instances[0].Name != "updated" {
			t.Errorf("expected the two updated instances, got %+v", cfg.Logstash.Instances)
		}
		if cfg.Server.Port != 10002 {
			t.Errorf("expected port 10002, got %d", cfg.Server.Port)
		}
		if cfg.Logging.Level != "debug" {
			t.Errorf("expected log level debug, got %s", cfg.Logging.Level)
		}
	})

	t.Run("keeps_the_running_config_when_the_file_is_emptied", func(t *testing.T) {
		writeConfig("")

		if err := sm.Reload(context.Background()); !errors.Is(err, ErrEmptyConfigFile) {
			t.Errorf("expected %v, got %v", ErrEmptyConfigFile, err)
		}
		if len(factory.Servers()) != 2 {
			t.Errorf("expected no new server, got %d servers", len(factory.Servers()))
		}
	})
}
