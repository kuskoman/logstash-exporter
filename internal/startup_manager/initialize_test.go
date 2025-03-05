package startup_manager

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/kuskoman/logstash-exporter/internal/flags"
)

// Test for the Initialize method - simplified version
func TestInitializeMethod(t *testing.T) {
	// Only run this test in isolation or when explicitly needed
	if testing.Short() {
		t.Skip("Skipping test with real components in short mode")
	}
	
	// Do not use t.Parallel() to avoid Prometheus registration issues
	
	t.Run("initialize_without_hot_reload", func(t *testing.T) {
		// Setup
		configPath := createValidConfigFile(t)
		flagsCfg := &flags.FlagsConfig{
			HotReload: false,
		}

		sm, err := NewStartupManager(configPath, flagsCfg)
		if err != nil {
			t.Fatalf("failed to create startup manager: %v", err)
		}

		// We need to trigger shutdown to avoid blocking
		go func() {
			time.Sleep(50 * time.Millisecond)
			sm.serverErrorChan <- http.ErrServerClosed
		}()

		// Execute in background
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		// Initialize should return with our error
		err = sm.Initialize(ctx)
		if !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("expected http.ErrServerClosed, got %v", err)
		}

		// Check that initialization was successful
		if !sm.isInitialized {
			t.Errorf("expected StartupManager to be initialized")
		}

		// Clean up
		err = sm.Shutdown(ctx)
		if err != nil {
			t.Logf("shutdown error: %v", err)
		}
	})
}