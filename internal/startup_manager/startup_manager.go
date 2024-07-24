package startup_manager

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strconv"
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
}

// NewStartupManager returns a new instance of the StartupManager
func NewStartupManager() *StartupManager {
	return &StartupManager{
		isInitialized: false,
		mutex:         &sync.Mutex{},
	}
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

	manager.loadFlags()

	err := manager.loadConfig(ctx)

	if warn != nil {
		slog.Warn("failed to load .env file", "err", warn)
	}

	if err != nil {
		return err
	}


	printInitialMessage()
	slog.Warn("test1")
	manager.setupPrometheus(ctx)
	slog.Warn("test2")

	var (
		runGroup      run.Group
		reloadManager = reload.NewManager()
	)

	// Add all app reloaders in order.
	reloadManager.Add(0, reload.ReloaderFunc(func(ctx context.Context, id string) error {
		// If configuration fails ignore reload with a warning.
		err := manager.loadConfig(ctx)
		if err != nil {
			slog.Warn("Config could not be reloaded: %s", err)
			return err
		}

		slog.Info("Config reloaded")
		return nil
	}))

	reloadManager.Add(100, reload.ReloaderFunc(func(ctx context.Context, id string) error {
		slog.Info("Prometeus reloaded")
		manager.setupPrometheus(ctx)
		return nil
	}))

	reloadManager.Add(100, reload.ReloaderFunc(func(ctx context.Context, id string) error {
		slog.Info("App server reloaded")
		manager.startAppServer(ctx)
		return nil
	}))

	ctx, cancel := context.WithCancel(ctx)

	runGroup.Add(
		func() error {
			slog.Info("Starting reload manager")
			return reloadManager.Run(ctx)
		},
		func(_ error) {
			slog.Info("Stopping reload manager")
			cancel()
		},
	)

	// File watcher:

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	err = watcher.Add(*manager.flagsConfig.configLocation)
	if err != nil {
		slog.Warn("could not add file watcher for %s: %s", *manager.flagsConfig.configLocation, err)
		return err
	}

	// Add file watcher based reload notifier.
	reloadManager.On(reload.NotifierFunc(func(ctx context.Context) (string, error) {
		select {
		case <-watcher.Events:
			return "file-watch", nil
		case err := <-watcher.Errors:
			return "", err
		}
	}))

	ctx, cancel = context.WithCancel(ctx)
	runGroup.Add(
		func() error {
			// Block forever until the watcher stops.
			slog.Info("File watcher with %s config file running", *manager.flagsConfig.configLocation)
			<-ctx.Done()
			return nil
		},
		func(_ error) {
			slog.Info("Stopping file watcher")
			watcher.Close()
			cancel()
		},
	)


	manager.isInitialized = true

	runGroup.Run()
	
	slog.Warn("test3")
	manager.startAppServer(ctx)
	return nil
}

func (manager *StartupManager) loadFlags() {
	flagsConfig, shouldExit := handleFlags()
	if shouldExit {
		os.Exit(0)
	}

	manager.flagsConfig = flagsConfig
}

func (manager *StartupManager) loadConfig(ctx context.Context) error {
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

func (manager *StartupManager) startAppServer(ctx context.Context) {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

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

func (startupManager *StartupManager) setupPrometheus(ctx context.Context) {
	startupManager.mutex.Lock()
	defer startupManager.mutex.Unlock()

	config := startupManager.appConfig
	slog.Debug("http timeout", "timeout", config.Logstash.HttpTimeout)

	collectorManager := collector_manager.NewCollectorManager(
		config.Logstash.Servers,
		config.Logstash.HttpTimeout,
	)
	prometheus.MustRegister(collectorManager)
}

func printInitialMessage() {
	slog.Debug("application starting... ")
	versionInfo := config.GetVersionInfo()
	slog.Info(versionInfo.String())
}
