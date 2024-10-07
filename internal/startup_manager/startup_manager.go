package startup_manager

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"time"

	"github.com/kuskoman/logstash-exporter/internal/flags"
	"github.com/kuskoman/logstash-exporter/internal/server"
	"github.com/kuskoman/logstash-exporter/pkg/collector_manager"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/slok/reload"
)

const ServerShutdownTimeout = 10 * time.Second

// AppServer defines the behavior of an application server
type AppServer interface {
	ListenAndServe() error
	Shutdown(ctx context.Context) error
}

// StartupManager manages the startup, reload, and shutdown of the application
type StartupManager struct {
	isInitialized    bool
	flagsConfig      *flags.FlagsConfig
	configComparator *ConfigComparator
	reloadManager    reload.Manager
	fileWatcher      *FileWatcher
	runGroup         run.Group
	serverInstance   AppServer
}

// NewStartupManager creates a new StartupManager
func NewStartupManager(flagsConfig *flags.FlagsConfig) *StartupManager {
	configPath := flagsConfig.ConfigLocation
	return &StartupManager{
		isInitialized:    false,
		flagsConfig:      flagsConfig,
		configComparator: NewConfigComparator(configPath),
		reloadManager:    reload.NewManager(),
	}
}

// Initialize initializes the startup manager, loading configuration and setting up the server
func (manager *StartupManager) Initialize(ctx context.Context) error {
	if manager.isInitialized {
		return errors.New("startup manager is already initialized")
	}

	if err := manager.LoadConfig(ctx); err != nil {
		slog.Error("failed to load config", "err", err)
		return err
	}

	manager.SetupServerAndPrometheus(ctx)

	if manager.flagsConfig.HotReload {
		return manager.SetupHotReload(ctx)
	}

	return manager.StartServer(ctx)
}

// StartServer starts the application server and stops the previous instance if running
func (manager *StartupManager) StartServer(ctx context.Context) error {
	if err := manager.StopServer(ctx); err != nil {
		slog.Error("failed to stop existing server", "err", err)
		return err
	}

	config := manager.configComparator.GetCurrentConfig()
	host := config.Server.Host
	port := strconv.Itoa(config.Server.Port)

	appServer := server.NewAppServer(host, port, config, config.Logstash.HttpTimeout)
	manager.serverInstance = appServer

	slog.Info("starting server", "host", host, "port", port)
	if err := appServer.ListenAndServe(); err != nil {
		slog.Error("failed to listen and serve", "err", err)
		return err
	}
	return nil
}

// StopServer stops the running server instance, if one exists
func (manager *StartupManager) StopServer(ctx context.Context) error {
	if manager.serverInstance == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, ServerShutdownTimeout)
	defer cancel()

	slog.Info("stopping server")
	if err := manager.serverInstance.Shutdown(ctx); err != nil {
		slog.Error("failed to shut down server", "err", err)
		return err
	}
	manager.serverInstance = nil
	return nil
}

// SetupServerAndPrometheus sets up the Prometheus collector and starts the server
func (manager *StartupManager) SetupServerAndPrometheus(ctx context.Context) {
	manager.SetupPrometheus(ctx)
	manager.StartServer(ctx)
}

// SetupHotReload sets up file watching and hot-reload functionality
func (manager *StartupManager) SetupHotReload(ctx context.Context) error {
	watcher, err := NewFileWatcher(manager.flagsConfig.ConfigLocation, manager.reloadManager)
	if err != nil {
		return err
	}
	manager.fileWatcher = watcher

	manager.SetupPrometheusReloader(ctx)

	err = manager.fileWatcher.Watch(ctx, manager.configComparator)
	if err != nil {
		slog.Error("failed to setup file watcher", "err", err)
		return err
	}

	return manager.runGroup.Run()
}

// SetupPrometheus sets up the Prometheus exporter
func (manager *StartupManager) SetupPrometheus(ctx context.Context) {
	config := manager.configComparator.GetCurrentConfig()

	collector := collector_manager.NewCollectorManager(
		config.Logstash.Servers,
		config.Logstash.HttpTimeout,
	)

	prometheus.MustRegister(collector)
}

// SetupPrometheusReloader configures the reloader to handle Prometheus reload events
func (manager *StartupManager) SetupPrometheusReloader(ctx context.Context) {
	manager.reloadManager.Add(0, reload.ReloaderFunc(func(ctx context.Context, event string) error {
		if event == NoEvent {
			return nil
		}

		manager.SetupPrometheus(ctx)
		slog.Info("prometheus reloaded")
		return nil
	}))

	ctx, cancel := context.WithCancel(ctx)
	manager.runGroup.Add(
		func() error {
			slog.Info("starting reload manager")
			return manager.reloadManager.Run(ctx)
		},
		func(_ error) {
			slog.Info("stopping reload manager")
			cancel()
		},
	)
}

// LoadConfig loads the configuration for the application
func (manager *StartupManager) LoadConfig(ctx context.Context) error {
	_, err := manager.configComparator.LoadAndCompareConfig(ctx)
	return err
}
