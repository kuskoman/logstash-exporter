package startup_manager

import (
	"context"
	"log/slog"
	"sync"

	"github.com/kuskoman/logstash-exporter/pkg/config"
)

// ConfigManager handles config loading and comparing for changes
type ConfigManager struct {
	currentConfig *config.Config
	mutex         sync.Mutex
	configPath    string
}

// NewConfigManager creates a new ConfigManager
func NewConfigManager(configPath string) *ConfigManager {
	return &ConfigManager{
		configPath: configPath,
	}
}

// LoadAndCompareConfig loads the configuration and compares it with the current one
func (cc *ConfigManager) LoadAndCompareConfig(ctx context.Context) (bool, error) {
	slog.Debug("loading and comparing config")
	cc.mutex.Lock()
	defer cc.mutex.Unlock()

	newConfig, err := config.GetConfig(cc.configPath)
	if err != nil {
		return false, err
	}

	if cc.currentConfig == nil || !cc.currentConfig.Equals(newConfig) {
		cc.currentConfig = newConfig
		return true, nil
	} else {
		slog.Debug("config is unchanged")
	}

	return false, nil
}

// GetCurrentConfig returns the current loaded configuration
func (cc *ConfigManager) GetCurrentConfig() *config.Config {
	cc.mutex.Lock()
	defer cc.mutex.Unlock()

	return cc.currentConfig
}
