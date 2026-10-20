package startup_manager

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
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

// LoadAndCompareConfig loads the configuration and compares it with the current one.
// If the configuration has changed, it returns true, otherwise false.
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
		slog.Debug("config has changed", "newConfig", fmt.Sprintf("%+v", newConfig))
		return true, nil
	} else {
		slog.Debug("config is unchanged")
	}

	return false, nil
}

// ErrEmptyConfigFile is returned when a reload finds an empty config file
var ErrEmptyConfigFile = errors.New("config file is empty, keeping the current configuration")

// ReloadAndCompareConfig works like LoadAndCompareConfig, but rejects an
// empty config file. An empty file is a valid configuration that uses every
// default, but during a reload it is almost always a file that is being
// written or was cleared by mistake, and applying it would move the server to
// the default port.
func (cc *ConfigManager) ReloadAndCompareConfig(ctx context.Context) (bool, error) {
	content, err := os.ReadFile(cc.configPath)
	if err != nil {
		return false, err
	}
	if strings.TrimSpace(string(content)) == "" {
		return false, ErrEmptyConfigFile
	}

	return cc.LoadAndCompareConfig(ctx)
}

// GetCurrentConfig returns the current loaded configuration
func (cc *ConfigManager) GetCurrentConfig() *config.Config {
	cc.mutex.Lock()
	defer cc.mutex.Unlock()

	return cc.currentConfig
}
