package startup_manager

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/kuskoman/logstash-exporter/internal/file_watcher"
	"github.com/kuskoman/logstash-exporter/internal/flags"
	"github.com/kuskoman/logstash-exporter/pkg/config"
	"github.com/prometheus/client_golang/prometheus"
)

const ServerShutdownTimeout = 10 * time.Second

var (
	ErrAlreadyInitialized = errors.New("startup manager already initialized")
	ErrServerClosed       = http.ErrServerClosed
)

// AppServer defines the behavior of an application server
type AppServer interface {
	ListenAndServe() error
	Shutdown(ctx context.Context) error
}

// StartupManager is responsible for managing the application lifecycle,
// including initialization, configuration loading, and hot reloading.
type StartupManager struct {
	mutex               sync.Mutex
	watchEnabled        bool
	isInitialized       bool
	server              AppServer
	configManager       *ConfigManager
	watcher             *file_watcher.FileWatcher
	prometheusCollector prometheus.Collector
	serverErrorChan     chan error
}

// NewStartupManager creates a new StartupManager with the given configuration.
func NewStartupManager(configPath string, flagsCfg *flags.FlagsConfig) (*StartupManager, error) {
	sm := &StartupManager{
		configManager:   NewConfigManager(configPath),
		isInitialized:   false,
		mutex:           sync.Mutex{},
		watchEnabled:    flagsCfg.HotReload,
		serverErrorChan: make(chan error),
	}

	watcher, err := file_watcher.NewFileWatcher(configPath, sm.handleConfigChange)
	if err != nil {
		return nil, err
	}

	sm.watcher = watcher

	return sm, nil
}

// Initialize loads the configuration, sets up logging, and starts the server.
// If hot reload is enabled, it also starts watching the configuration file.
func (sm *StartupManager) Initialize(ctx context.Context) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if sm.isInitialized {
		return ErrAlreadyInitialized
	}

	sm.isInitialized = true

	_, err := sm.configManager.LoadAndCompareConfig(ctx)
	if err != nil {
		return err
	}

	cfg := sm.configManager.GetCurrentConfig()
	if cfg == nil {
		return errors.New("config is nil")
	}

	err = config.SetupSlog(cfg)
	if err != nil {
		return err
	}

	if sm.watchEnabled {
		slog.Info("watching for config changes", "configPath", sm.configManager.configPath)
		readyCh, err := sm.watcher.Watch(ctx)
		if err != nil {
			return err
		}
		// Wait for the watcher to be ready
		<-readyCh
	} else {
		slog.Debug("watching for config changes is disabled")
	}

	slog.Debug("starting application components")
	sm.startPrometheus(cfg)
	sm.startServer(cfg)

	slog.Info("application initialized")
	slog.Debug("starting server error handler in a separate goroutine")

	applicationErrorChan := make(chan error)
	go sm.handleServerErrors(applicationErrorChan)

	err = <-applicationErrorChan
	return err
}

// Shutdown cleanly stops all components of the application.
func (sm *StartupManager) Shutdown(ctx context.Context) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	slog.Info("shutting down application")

	if !sm.isInitialized {
		return errors.New("startup manager not initialized")
	}

	err := sm.shutdownServer(ctx)
	if err != nil {
		return err
	}

	sm.shutdownPrometheus()

	return nil
}
