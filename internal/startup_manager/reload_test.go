package startup_manager

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kuskoman/logstash-exporter/internal/flags"
)

// Additional tests for the Reload method with a mock interface
func TestReloadWithMock(t *testing.T) {
	// Do not use t.Parallel() at the top level to avoid Prometheus registration issues
	
	t.Run("should_handle_config_load_error", func(t *testing.T) {
		// Do not use t.Parallel() in subtests to avoid Prometheus registration issues
		// Setup
		mockCfgManager := newMockConfigManager(nil, false, errors.New("config load error"))
		mockSrv := newMockAppServer(nil, nil)

		sm := newTestableStartupManager(
			mockCfgManager,
			true,  // watchEnabled
			true,  // initialized
			mockSrv,
			nil,   // No collector
			nil,   // No watcher
		)

		// Execute
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err := sm.Reload(ctx)
		
		// Verify error was propagated
		if err == nil || err.Error() != "config load error" {
			t.Errorf("expected config load error, got %v", err)
		}

		// Verify LoadAndCompareConfig was called
		select {
		case <-mockCfgManager.loadAndCompareConfigCalled:
			// Success
		default:
			t.Errorf("expected LoadAndCompareConfig to be called")
		}
	})

	t.Run("should_handle_nil_config", func(t *testing.T) {
		// Setup
		mockCfgManager := newMockConfigManager(nil, true, nil)
		mockSrv := newMockAppServer(nil, nil)

		sm := newTestableStartupManager(
			mockCfgManager,
			true,  // watchEnabled
			true,  // initialized
			mockSrv,
			nil,   // No collector
			nil,   // No watcher
		)

		// Execute
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err := sm.Reload(ctx)
		
		// Verify
		if err == nil {
			t.Errorf("expected error for nil config, got nil")
		}
		
		expectedErrMsg := "config is nil"
		if err.Error() != expectedErrMsg {
			t.Errorf("expected error message '%s', got '%s'", expectedErrMsg, err.Error())
		}
	})

	t.Run("should_handle_server_shutdown_error", func(t *testing.T) {
		// Setup
		cfg := createTestConfig()
		mockCfgManager := newMockConfigManager(cfg, true, nil)
		
		// Create a mock server that returns an error on shutdown
		shutdownError := errors.New("server shutdown error")
		mockSrv := newMockAppServer(nil, shutdownError)

		sm := newTestableStartupManager(
			mockCfgManager,
			true,  // watchEnabled
			true,  // initialized
			mockSrv,
			nil,   // No collector
			nil,   // No watcher
		)

		// Execute
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err := sm.Reload(ctx)
		
		// Verify
		if !errors.Is(err, shutdownError) {
			t.Errorf("expected shutdown error, got %v", err)
		}

		// Verify that Shutdown was called on the server
		select {
		case <-mockSrv.shutdownCalled:
			// Success
		default:
			t.Errorf("expected server.Shutdown to be called")
		}
	})
}

// A more realistic test for Reload that actually modifies the config file
func TestReloadWithFileChange(t *testing.T) {
	// Skip test in short mode
	if testing.Short() {
		t.Skip("Skipping test with real components in short mode")
	}
	
	// Do not use t.Parallel() at the top level to avoid Prometheus registration issues
	
	t.Run("should_detect_config_changes", func(t *testing.T) {
		// Setup - create a temporary dir for configs
		dir := t.TempDir()
		configPath := filepath.Join(dir, "config.yml")

		// Create initial config
		initialConfig := `
logstash:
  instances:
    - host: http://localhost:9600
      name: initial
  httpTimeout: 2s
server:
  host: 0.0.0.0
  port: 9198
logging:
  level: info
  format: text
`
		err := os.WriteFile(configPath, []byte(initialConfig), 0644)
		if err != nil {
			t.Fatalf("failed to write initial config: %v", err)
		}

		// Create startup manager with the initial config
		flagsCfg := &flags.FlagsConfig{
			HotReload: true,
		}

		// Create a real startup manager
		sm, err := NewStartupManager(configPath, flagsCfg)
		if err != nil {
			t.Fatalf("failed to create startup manager: %v", err)
		}

		// Set isInitialized to true for testing Reload
		sm.isInitialized = true

		// Prepare updated config
		updatedConfig := `
logstash:
  instances:
    - host: http://localhost:9600
      name: updated
    - host: http://localhost:9601
      name: second
  httpTimeout: 3s
server:
  host: 0.0.0.0
  port: 9198
logging:
  level: debug
  format: text
`
		// Write the updated config to the file
		err = os.WriteFile(configPath, []byte(updatedConfig), 0644)
		if err != nil {
			t.Fatalf("failed to write updated config: %v", err)
		}

		// Execute
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err = sm.Reload(ctx)
		if err != nil {
			t.Fatalf("reload failed: %v", err)
		}

		// Verify the config was updated
		currentConfig := sm.configManager.GetCurrentConfig()
		if currentConfig == nil {
			t.Fatalf("expected config to be non-nil")
		}

		// Check specific config changes
		if len(currentConfig.Logstash.Instances) != 2 {
			t.Errorf("expected 2 instances, got %d", len(currentConfig.Logstash.Instances))
		}

		if currentConfig.Logstash.Instances[0].Name != "updated" {
			t.Errorf("expected instance name to be 'updated', got '%s'", 
				currentConfig.Logstash.Instances[0].Name)
		}

		if currentConfig.Logging.Level != "debug" {
			t.Errorf("expected log level to be 'debug', got '%s'", 
				currentConfig.Logging.Level)
		}
	})
}

func TestHandleConfigChange(t *testing.T) {
	// Do not use t.Parallel() at the top level to avoid Prometheus registration issues
	
	t.Run("should_create_context_with_timeout", func(t *testing.T) {
		// Verify that WithTimeout is called with ServerShutdownTimeout
		// by checking if they're both 10 seconds
		if ServerShutdownTimeout != 10*time.Second {
			t.Errorf("expected shutdown timeout to be 10s, got %v", ServerShutdownTimeout)
		}
		
		// Setup - create testable manager
		mockCfgManager := newMockConfigManager(nil, false, nil)
		
		sm := newTestableStartupManager(
			mockCfgManager,
			true,  // watchEnabled
			true,  // initialized
			nil,   // No server
			nil,   // No collector
			nil,   // No watcher
		)
		
		// Replace reload with custom function that checks context
		reloadCalled := make(chan context.Context, 1)
		sm.mockReload = func(ctx context.Context) error {
			reloadCalled <- ctx
			return nil
		}
		
		// Execute
		err := sm.handleConfigChange()
		
		// Verify
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		
		select {
		case ctx := <-reloadCalled:
			// Check that the context has a deadline
			deadline, hasDeadline := ctx.Deadline()
			if !hasDeadline {
				t.Errorf("expected context to have a deadline")
			}
			
			// Verify the deadline is reasonable (within a few seconds from now)
			maxExpectedDeadline := time.Now().Add(ServerShutdownTimeout + 100*time.Millisecond)
			if deadline.After(maxExpectedDeadline) {
				t.Errorf("deadline is too far in the future: %v", deadline)
			}
		case <-time.After(100 * time.Millisecond):
			t.Errorf("Reload was not called")
		}
	})

	t.Run("should_call_reload_and_propagate_error", func(t *testing.T) {
		// Setup - create testable manager
		mockCfgManager := newMockConfigManager(nil, false, nil)
		
		sm := newTestableStartupManager(
			mockCfgManager,
			true,  // watchEnabled
			true,  // initialized
			nil,   // No server
			nil,   // No collector
			nil,   // No watcher
		)
		
		// Create a custom reload method that fails with a specific error
		reloadError := errors.New("reload error")
		sm.mockReload = func(ctx context.Context) error {
			// Verify context has deadline
			if _, hasDeadline := ctx.Deadline(); !hasDeadline {
				t.Errorf("context passed to Reload has no deadline")
			}
			
			// Return our custom error to test propagation
			return reloadError
		}
		
		// Execute
		err := sm.handleConfigChange()
		
		// Verify
		if !errors.Is(err, reloadError) {
			t.Errorf("expected reload error %v to be propagated, got %v", reloadError, err)
		}
	})
}