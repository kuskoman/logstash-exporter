package startup_manager

import (
	"context"
	"errors"
	"log/slog"
)

// handleConfigChange is called when the configuration file changes.
// It creates a context with a timeout and calls Reload.
func (sm *StartupManager) handleConfigChange() error {
	ctx, cancel := context.WithTimeout(context.Background(), ServerShutdownTimeout)
	defer cancel()

	err := sm.Reload(ctx)
	if err != nil {
		return err
	}

	return nil
}

// Reload reloads the configuration and restarts the server if the configuration has changed.
func (sm *StartupManager) Reload(ctx context.Context) error {
	changed, err := sm.configManager.LoadAndCompareConfig(ctx)
	if err != nil {
		return err
	}

	if changed {
		cfg := sm.configManager.GetCurrentConfig()
		if cfg == nil {
			return errors.New("config is nil")
		}

		slog.Info("config has changed, reloading server")

		sm.shutdownPrometheus()
		err := sm.shutdownServer(ctx)
		if err != nil {
			return err
		}

		sm.startPrometheus(cfg)
		sm.startServer(cfg)

		slog.Info("application reloaded")
	} else {
		slog.Debug("skipping reload, config is unchanged")
	}

	return nil
}

// handleServerErrors handles errors from the server.
// If the server is closed and hot reload is enabled, it logs a message and continues.
// Otherwise, it propagates the error to the caller.
func (sm *StartupManager) handleServerErrors(applicationErrorChan chan error) {
	for err := range sm.serverErrorChan {
		slog.Debug("server error occurred", "error", err)

		if errors.Is(err, ErrServerClosed) {
			if sm.watchEnabled {
				slog.Info("server closed for hot reload")
				continue
			} else {
				slog.Error("server closed while hot reload is disabled")
				applicationErrorChan <- err
			}
		} else if err != nil {
			applicationErrorChan <- err
		}
	}
}