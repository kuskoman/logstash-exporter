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
	"github.com/kuskoman/logstash-exporter/pkg/config"
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

	slog.Debug("starting initialization")
	if err := manager.LoadConfig(ctx); err != nil {
		slog.Error("failed to load config", "err", err)
		return err
	}

	err := manager.setupSlog()
	if err != nil {
		slog.Error("failed to setup slog", "err", err)
		return err
	}

	// Setup server and Prometheus first
	if err := manager.SetupServerAndPrometheus(ctx); err != nil {
		return err
	}

	// Check for hot reload flag
	slog.Debug("checking if hot reload is enabled", "enabled", manager.flagsConfig.HotReload)
	if manager.flagsConfig.HotReload {
		if err := manager.setupHotReload(ctx); err != nil {
			return err
		}
	}

	// Run the group (it will block until one of the processes terminates)
	slog.Debug("running runGroup")
	return manager.runGroup.Run()
}

func (manager *StartupManager) setupSlog() error {
	cfg := manager.configComparator.GetCurrentConfig()
	loggingConfig := cfg.Logging

	logger, err := config.SetupSlog(loggingConfig.Level, loggingConfig.Format)
	if err != nil {
		return err
	}

	slog.SetDefault(logger)
	return nil
}

// SetupServer sets up the server lifecycle management
func (manager *StartupManager) SetupServer(ctx context.Context) error {
	config := manager.configComparator.GetCurrentConfig()
	host := config.Server.Host
	port := strconv.Itoa(config.Server.Port)

	appServer := server.NewAppServer(host, port, config, config.Logstash.HttpTimeout)
	manager.serverInstance = appServer

	// Add named functions for server run and shutdown
	manager.runGroup.Add(
		manager.runServer,
		manager.shutdownServer,
	)

	slog.Debug("server setup complete", "host", host, "port", port)
	return nil
}

// runServer starts the server
func (manager *StartupManager) runServer() error {
	cfg := manager.configComparator.GetCurrentConfig()
	slog.Info("starting server", "host", cfg.Server.Host, "port", cfg.Server.Port)
	return manager.serverInstance.ListenAndServe()
}

// shutdownServer gracefully shuts down the server
func (manager *StartupManager) shutdownServer(err error) {
	cfg := manager.configComparator.GetCurrentConfig()
	slog.Info("shutting down server", "host", cfg.Server.Host, "port", cfg.Server.Port)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), ServerShutdownTimeout)
	defer cancel()

	if err := manager.serverInstance.Shutdown(shutdownCtx); err != nil {
		slog.Error("failed to shutdown server", "err", err)
	}
}

// SetupServerAndPrometheus sets up the Prometheus collector and adds the server to the runGroup
func (manager *StartupManager) SetupServerAndPrometheus(ctx context.Context) error {
	// Setup Prometheus first
	manager.setupPrometheus()

	// Then add the server to the runGroup
	if err := manager.SetupServer(ctx); err != nil {
		slog.Error("failed to setup server", "err", err)
		return err
	}

	return nil
}

// setupHotReload sets up file watching and hot-reload functionality
func (manager *StartupManager) setupHotReload(ctx context.Context) error {
	slog.Debug("setting up hot reload", "config file", manager.flagsConfig.ConfigLocation)
	watcher, err := NewFileWatcher(manager.flagsConfig.ConfigLocation, manager.reloadManager)
	if err != nil {
		return err
	}
	manager.fileWatcher = watcher

	manager.setupPrometheusReloader()

	// Watch file changes
	err = manager.fileWatcher.Watch(ctx, manager.configComparator)
	if err != nil {
		slog.Error("failed to setup file watcher", "err", err)
		return err
	}

	// Add the reload manager to the runGroup
	manager.runGroup.Add(
		manager.startReloadManager,
		manager.handleReloadManagerShutdown,
	)

	return nil
}

// startReloadManager starts the reload manager
func (manager *StartupManager) startReloadManager() error {
	slog.Info("starting reload manager")
	return manager.reloadManager.Run(context.Background())
}

// handleReloadManagerShutdown gracefully stops the reload manager
func (manager *StartupManager) handleReloadManagerShutdown(err error) {
	slog.Info("stopping reload manager")
	// Add any necessary shutdown logic here
}

// setupPrometheus sets up the Prometheus exporter
func (manager *StartupManager) setupPrometheus() {
	config := manager.configComparator.GetCurrentConfig()

	collector := collector_manager.NewCollectorManager(
		config.Logstash.Servers,
		config.Logstash.HttpTimeout,
	)

	prometheus.MustRegister(collector)
}

// setupPrometheusReloader configures the reloader to handle Prometheus reload events
func (manager *StartupManager) setupPrometheusReloader() {
	manager.reloadManager.Add(0, reload.ReloaderFunc(func(ctx context.Context, event string) error {
		if event == NoEvent {
			return nil
		}

		// Reload Prometheus configuration
		manager.setupPrometheus()
		slog.Info("prometheus reloaded")
		return nil
	}))
}

// LoadConfig loads the configuration for the application
func (manager *StartupManager) LoadConfig(ctx context.Context) error {
	slog.Info("loading config", "file", manager.flagsConfig.ConfigLocation)
	_, err := manager.configComparator.LoadAndCompareConfig(ctx)
	if err != nil {
		slog.Error("failed to load config", "err", err)
	}
	return err
}
