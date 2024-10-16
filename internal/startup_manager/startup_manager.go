package startup_manager

import (
	"context"
	"errors"
	"log/slog"
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
}

func NewStartupManager(configPath string, flagsCfg *flags.FlagsConfig) (*StartupManager, error) {
	sm := &StartupManager{
		configManager: NewConfigManager(configPath),
		isInitialized: false,
		mutex:         sync.Mutex{},
		watchEnabled:  flagsCfg.HotReload,
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

	sm.startPrometheus(cfg)
	err = sm.startServer(cfg)
	if err != nil {
		return err
	}

	return nil
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

func (cm *StartupManager) shutdownPrometheus() {
	if cm.prometheusCollector != nil {
		slog.Info("unregistering prometheus collector")
		prometheus.Unregister(cm.prometheusCollector)
	} else {
		slog.Debug("prometheus collector is nil")
	}
}

func (cm *StartupManager) shutdownServer(ctx context.Context) error {
	if cm.server != nil {
		slog.Info("shutting down server")
		return cm.server.Shutdown(ctx)
	} else {
		slog.Debug("server is nil")
	}

	return nil
}

func (cm *StartupManager) startPrometheus(cfg *config.Config) {
	collectorManager := collector_manager.NewCollectorManager(
		cfg.Logstash.Servers,
		cfg.Logstash.HttpTimeout,
	)

	cm.prometheusCollector = collectorManager
	prometheus.MustRegister(cm.prometheusCollector)
}

func (cm *StartupManager) startServer(cfg *config.Config) error {
	appServer := server.NewAppServer(cfg)
	cm.server = appServer

	go func() {
		err := appServer.ListenAndServe()
		if err != nil {
			slog.Error("server error", "error", err)
		}
	}()

	return nil
}

func (cm *StartupManager) handleConfigChange() error {
	ctx, cancel := context.WithTimeout(context.Background(), ServerShutdownTimeout)
	defer cancel()

	err := cm.Reload(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (cm *StartupManager) Reload(ctx context.Context) error {
	changed, err := cm.configManager.LoadAndCompareConfig(ctx)
	if err != nil {
		return err
	}

	if changed {
		cfg := cm.configManager.GetCurrentConfig()
		if cfg == nil {
			return errors.New("config is nil")
		}

		slog.Info("config has changed, reloading server")

		cm.shutdownPrometheus()
		cm.shutdownServer(ctx)

		cm.startPrometheus(cfg)
		err = cm.startServer(cfg)
		if err != nil {
			return err
		}

		slog.Info("server reloaded")
	} else {
		slog.Debug("config is unchanged")
	}

	return nil
}
