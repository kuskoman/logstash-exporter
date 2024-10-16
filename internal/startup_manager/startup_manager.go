package startup_manager

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/kuskoman/logstash-exporter/internal/file_watcher"
	"github.com/kuskoman/logstash-exporter/internal/flags"
	"github.com/kuskoman/logstash-exporter/internal/server"
	"github.com/kuskoman/logstash-exporter/pkg/collector_manager"
	"github.com/kuskoman/logstash-exporter/pkg/config"
	"github.com/prometheus/client_golang/prometheus"
)

const ServerShutdownTimeout = 10 * time.Second

var (
	ErrAlreadyInitialized = errors.New("startup manager already initialized")
)

// AppServer defines the behavior of an application server
type AppServer interface {
	ListenAndServe() error
	Shutdown(ctx context.Context) error
}

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

	err = sm.setLogger(cfg)
	if err != nil {
		return err
	}

	if sm.watchEnabled {
		slog.Info("watching for config changes", "configPath", sm.configManager.configPath)
		err := sm.watcher.Watch(ctx)
		if err != nil {
			return err
		}
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

func (sm *StartupManager) setLogger(cfg *config.Config) error {
	logLevel, logFormat := cfg.Logging.Level, cfg.Logging.Format
	logger, err := config.SetupSlog(logLevel, logFormat)
	if err != nil {
		return err
	}

	slog.SetDefault(logger)
	return nil
}

func (sm *StartupManager) shutdownPrometheus() {
	if sm.prometheusCollector != nil {
		slog.Info("unregistering prometheus collector")
		prometheus.Unregister(sm.prometheusCollector)
	} else {
		slog.Debug("prometheus collector is nil")
	}
}

func (sm *StartupManager) shutdownServer(ctx context.Context) error {
	if sm.server == nil {
		slog.Debug("server is nil")
		return nil
	}

	slog.Info("shutting down server")
	err := sm.server.Shutdown(ctx)
	if errors.Is(err, http.ErrServerClosed) {
		slog.Debug("server closed gracefully")
		return nil
	}

	if err != nil {
		slog.Debug("server shutdown error", "error", err)
		return err
	}

	slog.Debug("server closed")
	return nil
}

func (sm *StartupManager) startPrometheus(cfg *config.Config) {
	collectorManager := collector_manager.NewCollectorManager(
		cfg.Logstash.Servers,
		cfg.Logstash.HttpTimeout,
	)

	sm.prometheusCollector = collectorManager
	prometheus.MustRegister(sm.prometheusCollector)
}

func (sm *StartupManager) startServer(cfg *config.Config) {
	slog.Debug("creating new app server instance", "config", fmt.Sprintf("%+v", cfg.Server))
	appServer := server.NewAppServer(cfg)
	sm.server = appServer

	go func() {
		slog.Info("starting server", "host", cfg.Server.Host, "port", cfg.Server.Port)
		err := appServer.ListenAndServe()
		sm.serverErrorChan <- err
	}()
}

func (sm *StartupManager) handleConfigChange() error {
	ctx, cancel := context.WithTimeout(context.Background(), ServerShutdownTimeout)
	defer cancel()

	err := sm.Reload(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (sm *StartupManager) Reload(ctx context.Context) error {
	changed, err := sm.configManager.LoadAndCompareConfig(ctx)
	if err != nil {
		return err
	}

	if changed {
		cfg := sm.configManager.GetCurrentConfig()
		if cfg == nil {
			return errors.New("config is nil")
		}

		slog.Info("config has changed, reloading server")

		sm.shutdownPrometheus()
		sm.shutdownServer(ctx)

		sm.startPrometheus(cfg)
		sm.startServer(cfg)

		slog.Info("application reloaded")
	} else {
		slog.Debug("skipping reload, config is unchanged")
	}

	return nil
}

func (sm *StartupManager) handleServerErrors(applicationErrorChan chan error) {
	for err := range sm.serverErrorChan {
		slog.Debug("server error occurred", "error", err)

		if errors.Is(err, http.ErrServerClosed) {
			if sm.watchEnabled {
				slog.Info("server closed for hot reload")
				continue
			} else {
				slog.Error("server closed while hot reload is disabled")
				applicationErrorChan <- err
			}
		} else if err != nil {
			applicationErrorChan <- err
		}
	}
}
