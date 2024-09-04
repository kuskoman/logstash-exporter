package startup_manager

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strconv"
	"sync"
	"path/filepath"
	"strings"


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
}

// NewStartupManager returns a new instance of the StartupManager
func NewStartupManager() *StartupManager {
	return &StartupManager{
		isInitialized: false,
		mutex:         &sync.Mutex{},
	}
}

var previousCollector prometheus.Collector = nil

func (manager *StartupManager) StartAppServer() {
	config := manager.appConfig

	host := config.Server.Host
	port := strconv.Itoa(config.Server.Port)
	appServer := server.NewAppServer(host, port, config, config.Logstash.HttpTimeout)

	slog.Info("starting server on", "host", host, "port", port)
	if err := appServer.ListenAndServe(); err != nil {
		slog.Error("failed to listen and serve", "err", err)
		os.Exit(1)
	}
}

func (manager *StartupManager) SetupHotReload(ctx context.Context) error {
	var (
		runGroup      run.Group
		reloadManager = reload.NewManager()
	)

	// Add all app reloaders in order.
	reloadManager.Add(0, reload.ReloaderFunc(func(ctx context.Context, id string) error {
		// If configuration fails ignore reload with a warning.
		slog.Info("ev", "event", id)
		if (id == "no-event") {
			return nil
		}

		err := manager.LoadConfig(ctx)
		if err != nil {
			slog.Warn("config could not be reloaded", "err", err)
			return err
		}

		slog.Info("config reloaded")
		return nil
	}))

	reloadManager.Add(100, reload.ReloaderFunc(func(ctx context.Context, id string) error {
		if (id == "no-event") {
			return nil
		}

		manager.SetupPrometheus(ctx)
		slog.Info("prometeus reloaded")
		return nil
	}))

	ctx, cancel := context.WithCancel(ctx)
	runGroup.Add(
		func() error {
			slog.Info("starting reload manager")
			return reloadManager.Run(ctx)
		},
		func(_ error) {
			slog.Info("stopping reload manager")
			cancel()
		},
	)

	config := manager.appConfig
	host := config.Server.Host
	port := strconv.Itoa(config.Server.Port)
	appServer := server.NewAppServer(host, port, config, config.Logstash.HttpTimeout)
	runGroup.Add(
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


	// File watcher:

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	err = watcher.Add(filepath.Dir(*manager.flagsConfig.configLocation))
	if err != nil {
		slog.Warn("could not add file watcher for %s: %s", *manager.flagsConfig.configLocation, err)
		return err
	}

	fname := filepath.Base(*manager.flagsConfig.configLocation)
	initialStat, err := os.Stat(*manager.flagsConfig.configLocation)
	if err != nil {
		return err
	}
	// Add file watcher based reload notifier.
	reloadManager.On(reload.NotifierFunc(func(ctx context.Context) (string, error) {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return "", err
				}

				if strings.Contains(event.Name, fname) {
					stat, err := os.Stat(*manager.flagsConfig.configLocation)
					if err != nil {
						return "", err
					}

					if stat.Size() != initialStat.Size() || stat.ModTime() != initialStat.ModTime() {
						slog.Info("config modified", "config fname", event.Name)
						return "file-watch", nil
					}
                }
				return "no-event", err
			case err := <-watcher.Errors:
				return "", err
			}
		}
	}))

	ctx, cancel = context.WithCancel(ctx)
	runGroup.Add(
		func() error {
			// Block forever until the watcher stops.
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

	manager.isInitialized = true

	runGroup.Run()

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
		return err
	}


	printInitialMessage()
	manager.SetupPrometheus(ctx)

	if (*manager.flagsConfig.hotReload) {
		manager.SetupHotReload(ctx)
	} else {
		manager.StartAppServer()
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
		slog.Info("should unregister")
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
