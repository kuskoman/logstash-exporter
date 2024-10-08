package startup_manager

import (
	"context"
	"testing"

	"github.com/kuskoman/logstash-exporter/internal/file_utils"
)

const (
	initialServerPort = 8080
	updatedServerPort = 9090
)

func TestConfigComparator_LoadAndCompareConfig(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("loads initial config", func(t *testing.T) {
		t.Parallel()

		validConfigContent := `
server:
  port: 8080
`
		configPath := file_utils.CreateTempFile(t, validConfigContent)
		defer file_utils.RemoveFile(t, configPath)

		cc := NewConfigManager(configPath)

		changed, err := cc.LoadAndCompareConfig(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !changed {
			t.Error("expected config to be marked as changed, but it wasn't")
		}

		if cc.GetCurrentConfig().Server.Port != initialServerPort {
			t.Errorf("expected port to be %d, got %d", initialServerPort, cc.GetCurrentConfig().Server.Port)
		}
	})

	t.Run("config remains unchanged", func(t *testing.T) {
		t.Parallel()

		validConfigContent := `
server:
  port: 8080
`
		configPath := file_utils.CreateTempFile(t, validConfigContent)
		defer file_utils.RemoveFile(t, configPath)

		cc := NewConfigManager(configPath)
		cc.LoadAndCompareConfig(ctx) // Initial load

		changed, err := cc.LoadAndCompareConfig(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if changed {
			t.Error("expected config to remain unchanged, but it was marked as changed")
		}

		if cc.GetCurrentConfig().Server.Port != initialServerPort {
			t.Errorf("expected port to be %d, got %d", initialServerPort, cc.GetCurrentConfig().Server.Port)
		}
	})

	t.Run("config changes and reloads", func(t *testing.T) {
		t.Parallel()

		validConfigContent := `
server:
  port: 8080
`
		newConfigContent := `
server:
  port: 9090
`
		validConfigPath := file_utils.CreateTempFile(t, validConfigContent)
		newConfigPath := file_utils.CreateTempFile(t, newConfigContent)
		defer file_utils.RemoveFile(t, validConfigPath)
		defer file_utils.RemoveFile(t, newConfigPath)

		cc := NewConfigManager(validConfigPath)
		cc.LoadAndCompareConfig(ctx)

		cc.configPath = newConfigPath
		changed, err := cc.LoadAndCompareConfig(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !changed {
			t.Error("expected config to be marked as changed, but it wasn't")
		}

		if cc.GetCurrentConfig().Server.Port != updatedServerPort {
			t.Errorf("expected port to be %d, got %d", updatedServerPort, cc.GetCurrentConfig().Server.Port)
		}
	})

	t.Run("returns error on invalid config", func(t *testing.T) {
		t.Parallel()

		invalidConfigContent := `invalid yaml content`
		configPath := file_utils.CreateTempFile(t, invalidConfigContent)
		defer file_utils.RemoveFile(t, configPath)

		cc := NewConfigManager(configPath)

		changed, err := cc.LoadAndCompareConfig(ctx)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if changed {
			t.Error("expected config to not be marked as changed on error")
		}
	})
}

func TestConfigComparator_GetCurrentConfig(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("returns nil if no config is loaded", func(t *testing.T) {
		t.Parallel()

		validConfigContent := `
server:
  port: 8080
`
		configPath := file_utils.CreateTempFile(t, validConfigContent)
		defer file_utils.RemoveFile(t, configPath)

		cc := NewConfigManager(configPath)
		if cc.GetCurrentConfig() != nil {
			t.Error("expected current config to be nil, but it wasn't")
		}
	})

	t.Run("returns current config after load", func(t *testing.T) {
		t.Parallel()

		validConfigContent := `
server:
  port: 8080
`
		configPath := file_utils.CreateTempFile(t, validConfigContent)
		defer file_utils.RemoveFile(t, configPath)

		cc := NewConfigManager(configPath)
		cc.LoadAndCompareConfig(ctx)

		if cc.GetCurrentConfig() == nil {
			t.Error("expected current config to be non-nil, but it was nil")
		}

		if cc.GetCurrentConfig().Server.Port != initialServerPort {
			t.Errorf("expected port to be %d, got %d", initialServerPort, cc.GetCurrentConfig().Server.Port)
		}
	})
}
