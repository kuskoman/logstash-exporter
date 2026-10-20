package startup_manager

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kuskoman/logstash-exporter/internal/file_watcher"
	"github.com/kuskoman/logstash-exporter/internal/flags"
	"github.com/kuskoman/logstash-exporter/internal/k8s_controller"
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
	ListenAndServeTLS(certFile, keyFile string) error
	Shutdown(ctx context.Context) error
}

// serverFactory creates a server for the given configuration, ready to listen
type serverFactory func(cfg *config.Config) (AppServer, error)

// configSource loads the configuration and reports whether it changed.
// It is implemented by ConfigManager.
type configSource interface {
	LoadAndCompareConfig(ctx context.Context) (bool, error)
	ReloadAndCompareConfig(ctx context.Context) (bool, error)
	GetCurrentConfig() *config.Config
}

// StartupManager is responsible for managing the application lifecycle,
// including initialization, configuration loading, and hot reloading.
type StartupManager struct {
	mutex                sync.Mutex
	watchEnabled         bool
	isInitialized        bool
	configPath           string
	server               AppServer
	newServer            serverFactory
	configManager        configSource
	watcher              *file_watcher.FileWatcher
	registerer           prometheus.Registerer
	prometheusCollector  prometheus.Collector
	kubernetesController *k8s_controller.Controller
	serverErrorChan      chan error
	withController       bool
	shuttingDown         atomic.Bool
}

// NewStartupManager creates a new StartupManager with the given configuration.
func NewStartupManager(configPath string, flagsCfg *flags.FlagsConfig) (*StartupManager, error) {
	return newStartupManager(configPath, flagsCfg, false)
}

// NewStartupManagerWithController creates a new StartupManager with the Kubernetes controller enabled
func NewStartupManagerWithController(configPath string, flagsCfg *flags.FlagsConfig) (*StartupManager, error) {
	return newStartupManager(configPath, flagsCfg, true)
}

func newStartupManager(configPath string, flagsCfg *flags.FlagsConfig, withController bool) (*StartupManager, error) {
	sm := &StartupManager{
		configPath:      configPath,
		configManager:   NewConfigManager(configPath),
		newServer:       newAppServer,
		registerer:      prometheus.DefaultRegisterer,
		watchEnabled:    flagsCfg.HotReload,
		serverErrorChan: make(chan error),
		withController:  withController,
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
// It blocks until the server fails, or until ctx is cancelled, in which case
// it shuts the application down gracefully and returns nil on success.
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
		slog.Info("watching for config changes", "configPath", sm.configPath)
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
	
	if sm.withController {
		sm.startKubernetesController(cfg)
	}
	
	sm.startServer(cfg)

	slog.Info("application initialized")
	slog.Debug("starting server error handler in a separate goroutine")

	applicationErrorChan := make(chan error, 1)
	go sm.handleServerErrors(applicationErrorChan)

	select {
	case err = <-applicationErrorChan:
		return err
	case <-ctx.Done():
		slog.Info("shutdown requested")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), ServerShutdownTimeout)
		defer cancel()
		return sm.shutdownComponents(shutdownCtx)
	}
}

// Shutdown cleanly stops all components of the application.
func (sm *StartupManager) Shutdown(ctx context.Context) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	slog.Info("shutting down application")

	if !sm.isInitialized {
		return errors.New("startup manager not initialized")
	}

	return sm.shutdownComponents(ctx)
}

// shutdownComponents stops the server, the Kubernetes controller and the
// Prometheus collector. The caller must hold sm.mutex.
func (sm *StartupManager) shutdownComponents(ctx context.Context) error {
	sm.shuttingDown.Store(true)

	err := sm.shutdownServer(ctx)
	if err != nil {
		return err
	}

	if sm.withController {
		sm.shutdownKubernetesController(ctx)
	}

	sm.shutdownPrometheus()

	return nil
}
