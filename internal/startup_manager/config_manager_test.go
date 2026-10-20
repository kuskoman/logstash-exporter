package startup_manager

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/kuskoman/logstash-exporter/internal/file_utils"
)

const (
	initialServerPort = 8080
	updatedServerPort = 9090
)

func TestConfigComparator_LoadAndCompareConfig(t *testing.T) {
	// Do not run in parallel to avoid file system issues and ensure test isolation

	ctx := context.Background()

	t.Run("loads initial config", func(t *testing.T) {
		// Do not run in parallel

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
		// Do not run in parallel

		validConfigContent := `
server:
  port: 8080
`
		configPath := file_utils.CreateTempFile(t, validConfigContent)
		defer file_utils.RemoveFile(t, configPath)

		cc := NewConfigManager(configPath)
		_, err := cc.LoadAndCompareConfig(ctx) // Initial load
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

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
		// Do not run in parallel

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
		_, err := cc.LoadAndCompareConfig(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

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
		// Do not run in parallel

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

func TestConfigManager_ReloadAndCompareConfig(t *testing.T) {
	ctx := context.Background()

	for _, content := range []string{"", "  \n\t\n"} {
		t.Run(fmt.Sprintf("rejects empty config %q and keeps the current one", content), func(t *testing.T) {
			configPath := file_utils.CreateTempFile(t, "server:\n  port: 8080\n")
			defer file_utils.RemoveFile(t, configPath)

			cc := NewConfigManager(configPath)
			if _, err := cc.LoadAndCompareConfig(ctx); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
				t.Fatalf("failed to write config: %v", err)
			}

			changed, err := cc.ReloadAndCompareConfig(ctx)
			if !errors.Is(err, ErrEmptyConfigFile) {
				t.Errorf("expected ErrEmptyConfigFile, got %v", err)
			}
			if changed {
				t.Error("expected config not to be marked as changed")
			}
			if port := cc.GetCurrentConfig().Server.Port; port != initialServerPort {
				t.Errorf("expected port to stay %d, got %d", initialServerPort, port)
			}
		})
	}

	t.Run("applies a changed config", func(t *testing.T) {
		configPath := file_utils.CreateTempFile(t, "server:\n  port: 8080\n")
		defer file_utils.RemoveFile(t, configPath)

		cc := NewConfigManager(configPath)
		if _, err := cc.LoadAndCompareConfig(ctx); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if err := os.WriteFile(configPath, []byte("server:\n  port: 9090\n"), 0644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		changed, err := cc.ReloadAndCompareConfig(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !changed {
			t.Error("expected config to be marked as changed")
		}
		if port := cc.GetCurrentConfig().Server.Port; port != updatedServerPort {
			t.Errorf("expected port to be %d, got %d", updatedServerPort, port)
		}
	})

	t.Run("returns an error when the file is missing", func(t *testing.T) {
		cc := NewConfigManager(filepath.Join(t.TempDir(), "missing.yml"))
		if _, err := cc.ReloadAndCompareConfig(ctx); err == nil {
			t.Error("expected an error for a missing file")
		}
	})
}

func TestConfigComparator_GetCurrentConfig(t *testing.T) {
	// Do not run in parallel to avoid file system issues and ensure test isolation

	ctx := context.Background()

	t.Run("returns nil if no config is loaded", func(t *testing.T) {
		// Do not run in parallel

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
		// Do not run in parallel

		validConfigContent := `
server:
  port: 8080
`
		configPath := file_utils.CreateTempFile(t, validConfigContent)
		defer file_utils.RemoveFile(t, configPath)

		cc := NewConfigManager(configPath)
		_, err := cc.LoadAndCompareConfig(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cc.GetCurrentConfig() == nil {
			t.Error("expected current config to be non-nil, but it was nil")
		}

		if cc.GetCurrentConfig().Server.Port != initialServerPort {
			t.Errorf("expected port to be %d, got %d", initialServerPort, cc.GetCurrentConfig().Server.Port)
		}
	})
}
