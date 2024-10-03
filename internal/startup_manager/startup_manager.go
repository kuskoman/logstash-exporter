package startup_manager

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/joho/godotenv"
	"github.com/kuskoman/logstash-exporter/internal/server"
	"github.com/kuskoman/logstash-exporter/pkg/collector_manager"
	"github.com/kuskoman/logstash-exporter/pkg/config"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/fsnotify/fsnotify"
	"github.com/oklog/run"
	"github.com/slok/reload"
)

// StartupManager is a struct that holds the startup manager
// responsible for handling the startup of the application
// and its components
type StartupManager struct {
	isInitialized bool
	mutex         *sync.Mutex
	flagsConfig   *flagsConfig
	appConfig     *config.Config
	runGroup      run.Group
	reloadManager reload.Manager
}

// NewStartupManager returns a new instance of the StartupManager
func NewStartupManager() *StartupManager {
	return &StartupManager{
		isInitialized: false,
		mutex:         &sync.Mutex{},
	}
}

var previousCollector prometheus.Collector

const (
	NoEvent      = "no-event"
	EventError   = ""
	FileModified = "modified"
)

func isConfigNotChanged(event string) bool {
	return event != FileModified
}

func (manager *StartupManager) SetupAppServer() {
	config := manager.appConfig

	host := config.Server.Host
	port := strconv.Itoa(config.Server.Port)
	appServer := server.NewAppServer(host, port, config, config.Logstash.HttpTimeout)

	manager.isInitialized = true
	slog.Info("starting server on", "host", host, "port", port)
	if err := appServer.ListenAndServe(); err != nil {
		slog.Error("failed to listen and serve", "err", err)
		os.Exit(1)
	}
}

// Setups Prometeus reloader in a hot-reload fashion.
func (manager *StartupManager) SetupPrometeusReloader(ctx context.Context) {
	manager.reloadManager.Add(0, reload.ReloaderFunc(func(ctx context.Context, event string) error {
		if isConfigNotChanged(event) {
			return nil
		}

		manager.SetupPrometheus(ctx)
		slog.Info("prometeus reloaded")
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

// Setups App Server in parallel.
// Used for hot-reload functionality.
func (manager *StartupManager) SetupAppServerParallel(ctx context.Context) {
	config := manager.appConfig
	host := config.Server.Host
	port := strconv.Itoa(config.Server.Port)
	appServer := server.NewAppServer(host, port, config, config.Logstash.HttpTimeout)
	manager.runGroup.Add(
		func() error {
			slog.Info("starting server on", "host", host, "port", port)
			return appServer.ListenAndServe()
		},
		func(_ error) {
			slog.Info("stopping HTTP server")
			ctx, cancel := context.WithTimeout(context.Background(), config.Logstash.HttpTimeout)
			defer cancel()
			err := appServer.Shutdown(ctx)
			if err != nil {
				slog.Error("could not shut down http server", "err", err)
			}
			os.Exit(1)
		},
	)
}

// Setups file watcher for the config.yml file for hot-reload.
// It is blocked forever until the watcher stops.
func (manager *StartupManager) SetupFileWatcher(ctx context.Context) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	err = watcher.Add(filepath.Dir(*manager.flagsConfig.configLocation))
	if err != nil {
		slog.Error("could not add file watcher for %s: %s", *manager.flagsConfig.configLocation, err)
		return err
	}

	configFname := filepath.Base(*manager.flagsConfig.configLocation)
	manager.reloadManager.On(reload.NotifierFunc(func(ctx context.Context) (string, error) {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return EventError, nil
				}

				if !strings.Contains(event.Name, configFname) {
					return NoEvent, nil
				}

				tmpConfig := manager.appConfig
				err = manager.LoadConfig(ctx)
				if err != nil {
					slog.Error("config could not be reloaded", "err", err)
					return EventError, err
				}
				if reflect.DeepEqual(tmpConfig, manager.appConfig) {
					return NoEvent, nil
				}

				slog.Info("config modified", "config fname", event.Name)
				return FileModified, nil
			case err := <-watcher.Errors:
				slog.Error("file watcher could not handle events", "err", err)
				return EventError, err
			}
		}
	}))

	ctx, cancel := context.WithCancel(ctx)
	manager.runGroup.Add(
		func() error {
			slog.Info("file watcher running with","config file", *manager.flagsConfig.configLocation)
			<-ctx.Done()
			return nil
		},
		func(_ error) {
			slog.Info("stopping file watcher")
			watcher.Close()
			cancel()
		},
	)

	return nil
}

// Initializes the startup manager in hot-reloading mode.
// Should be only called once.
func (manager *StartupManager) SetupHotReload(ctx context.Context) error {
	manager.reloadManager = reload.NewManager()

	manager.SetupPrometeusReloader(ctx)
	manager.SetupAppServerParallel(ctx)
	err := manager.SetupFileWatcher(ctx)
	if err != nil {
		slog.Error("failed to setup file watcher", "err", err)
		return err
	}

	manager.isInitialized = true

	err = manager.runGroup.Run()
	if err != nil {
		slog.Error("failed run reload group", "err", err)
		return err
	}

	return nil
}

// Initialize is a method that initializes the startup manager.
// Should be only called once.
func (manager *StartupManager) Initialize(ctx context.Context) error {
	ctx, rootCancel := context.WithCancel(ctx)
	defer rootCancel()

	warn := godotenv.Load()

	if manager.isInitialized {
		return errors.New("startup manager is already initialized")
	}

	manager.LoadFlags()

	err := manager.LoadConfig(ctx)

	if warn != nil {
		slog.Warn("failed to load .env file", "err", warn)
	}

	if err != nil {
		slog.Error("failed to load config.yml file", "err", err)
		return err
	}

	printInitialMessage()
	manager.SetupPrometheus(ctx)

	if (*manager.flagsConfig.hotReload) {
		err := manager.SetupHotReload(ctx)
		if err != nil {
			slog.Error("failed to set up hot reload", "err", err)
			return err
		}
	} else {
		manager.SetupAppServer()
	}

	return nil
}

func (manager *StartupManager) LoadFlags() {
	flagsConfig, shouldExit := handleFlags()
	if shouldExit {
		os.Exit(0)
	}

	manager.flagsConfig = flagsConfig
}

func (manager *StartupManager) LoadConfig(ctx context.Context) error {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	exporterConfig, err := config.GetConfig(*manager.flagsConfig.configLocation)
	if err != nil {
		return err
	}

	if err := setupLogging(&exporterConfig.Logging); err != nil {
		return err
	}

	manager.appConfig = exporterConfig
	return nil
}

func setupLogging(loggingConfig *config.LoggingConfig) error {
	logger, err := config.SetupSlog(loggingConfig.Level, loggingConfig.Format)
	if err != nil {
		return err
	}

	slog.SetDefault(logger)
	return nil
}

func (startupManager *StartupManager) SetupPrometheus(ctx context.Context) {
	startupManager.mutex.Lock()
	defer startupManager.mutex.Unlock()

	config := startupManager.appConfig
	slog.Debug("http timeout", "timeout", config.Logstash.HttpTimeout)

	if (previousCollector != nil) {
		slog.Debug("should unregister")
		prometheus.Unregister(previousCollector)
	}

	collectorManager := collector_manager.NewCollectorManager(
		config.Logstash.Servers,
		config.Logstash.HttpTimeout,
	)

	prometheus.MustRegister(collectorManager)
	previousCollector = collectorManager
}

func printInitialMessage() {
	slog.Debug("application starting... ")
	versionInfo := config.GetVersionInfo()
	slog.Info(versionInfo.String())
}
