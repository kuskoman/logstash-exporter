package startup_manager

import (
	"context"
	"log/slog"
	"sync"

	"github.com/kuskoman/logstash-exporter/pkg/config"
)

// ConfigComparator handles config loading and comparing for changes
type ConfigComparator struct {
	currentConfig *config.Config
	mutex         sync.Mutex
	configPath    string
}

// NewConfigComparator creates a new ConfigComparator
func NewConfigComparator(configPath string) *ConfigComparator {
	return &ConfigComparator{
		configPath: configPath,
	}
}

// LoadAndCompareConfig loads the configuration and compares it with the current one
func (cc *ConfigComparator) LoadAndCompareConfig(ctx context.Context) (bool, error) {
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
func (cc *ConfigComparator) GetCurrentConfig() *config.Config {
	cc.mutex.Lock()
	defer cc.mutex.Unlock()

	return cc.currentConfig
}
