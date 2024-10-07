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

	slog.Debug("hot reload enabled", "enabled", manager.flagsConfig.HotReload)
	if manager.flagsConfig.HotReload {
		return manager.setupHotReload(ctx)
	}

	return manager.runGroup.Run() // Run the group of processes, including the server
}

// StartServer adds the server execution to the runGroup and handles shutdown properly
func (manager *StartupManager) StartServer(ctx context.Context) error {
	config := manager.configComparator.GetCurrentConfig()
	host := config.Server.Host
	port := strconv.Itoa(config.Server.Port)

	appServer := server.NewAppServer(host, port, config, config.Logstash.HttpTimeout)
	manager.serverInstance = appServer

	manager.runGroup.Add(
		// Server running function
		func() error {
			slog.Info("starting server", "host", host, "port", port)
			return appServer.ListenAndServe()
		},
		// Server shutdown function
		func(err error) {
			slog.Info("stopping server", "host", host, "port", port)
			shutdownCtx, cancel := context.WithTimeout(context.Background(), ServerShutdownTimeout)
			defer cancel()
			if err := appServer.Shutdown(shutdownCtx); err != nil {
				slog.Error("failed to shutdown server", "err", err)
			}
		},
	)

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

// SetupServerAndPrometheus sets up the Prometheus collector and adds the server to the runGroup
func (manager *StartupManager) SetupServerAndPrometheus(ctx context.Context) {
	manager.SetupPrometheus(ctx)
	manager.StartServer(ctx) // Add the server execution to runGroup instead of running it directly
}

// setupHotReload sets up file watching and hot-reload functionality
func (manager *StartupManager) setupHotReload(ctx context.Context) error {
	slog.Debug("setting up hot reload", "config file", manager.flagsConfig.ConfigLocation)
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

	return manager.runGroup.Run() // Run the group including hot reload, server, etc.
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

	manager.runGroup.Add(
		func() error {
			slog.Info("starting reload manager")
			return manager.reloadManager.Run(ctx)
		},
		func(err error) {
			slog.Info("stopping reload manager")
			// No specific shutdown needed here, but we can add a cancel
		},
	)
}

// LoadConfig loads the configuration for the application
func (manager *StartupManager) LoadConfig(ctx context.Context) error {
	slog.Info("loading config", "file", manager.flagsConfig.ConfigLocation)
	_, err := manager.configComparator.LoadAndCompareConfig(ctx)
	return err
}
