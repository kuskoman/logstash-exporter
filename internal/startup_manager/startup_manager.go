package startup_manager

import (
	"context"
	"errors"
	"time"

	"github.com/kuskoman/logstash-exporter/internal/file_watcher"
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
	isInitialized bool
	server        AppServer
	configManager *ConfigManager
	watcher       *file_watcher.FileWatcher
}

func NewStartupManager(configPath string) (*StartupManager, error) {

	sm := &StartupManager{
		configManager: NewConfigManager(configPath),
		isInitialized: false,
	}

	watcher, err := file_watcher.NewFileWatcher(configPath, sm.reload)
	if err != nil {
		return nil, err
	}

	sm.watcher = watcher

	return sm, nil
}

func (sm *StartupManager) Initialize(ctx context.Context) error {
	if sm.isInitialized {
		return ErrAlreadyInitialized
	}

	sm.isInitialized = true
	sm.watcher.Watch(ctx)

	return nil
}

func (cm *StartupManager) reload() error {
	// Reload the configuration
	return nil
}
