package startup_manager

import (
	"errors"
	"log"
	"log/slog"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/kuskoman/logstash-exporter/internal/server"
	"github.com/kuskoman/logstash-exporter/pkg/collector_manager"
	"github.com/kuskoman/logstash-exporter/pkg/config"
	"github.com/prometheus/client_golang/prometheus"
)

// StartupManager is a struct that holds the startup manager
// responsible for handling the startup of the application
// and its components
type StartupManager struct {
	isInitialized bool
	flagsConfig   *flagsConfig
	appConfig     *config.Config
}

// NewStartupManager returns a new instance of the StartupManager
func NewStartupManager() *StartupManager {
	return &StartupManager{
		isInitialized: false,
	}
}

// Initialize is a method that initializes the startup manager
func (manager *StartupManager) Initialize() error {
	warn := godotenv.Load()

	if manager.isInitialized {
		return errors.New("startup manager is already initialized")
	}

	flagsConfig, shouldExit := handleFlags()
	if shouldExit {
		os.Exit(0)
	}

	manager.flagsConfig = flagsConfig
	manager.isInitialized = true

	err := manager.loadConfig()
	if err != nil {
		if warn != nil {
			log.Printf("failed to load .env file: %v", warn)
		}
		return err
	}

	if warn != nil {
		slog.Warn("failed to load .env file", "err", warn)
	}

	printInitialMessage()
	manager.setupPrometheus()
	manager.startAppServer()

	return nil
}

func (manager *StartupManager) loadConfig() error {
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

func (manager *StartupManager) startAppServer() {
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

func (startupManager *StartupManager) setupPrometheus() {
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
