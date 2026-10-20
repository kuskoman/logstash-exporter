package startup_manager

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
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

func TestInitializeShutsDownWhenContextIsCancelled(t *testing.T) {
	// Do not use t.Parallel() to avoid Prometheus registration issues

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find a free port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatalf("failed to close listener: %v", err)
	}

	configPath := filepath.Join(t.TempDir(), "config.yml")
	configContent := fmt.Sprintf("server:\n  host: 127.0.0.1\n  port: %d\n", port)
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	sm, err := NewStartupManager(configPath, &flags.FlagsConfig{})
	if err != nil {
		t.Fatalf("failed to create startup manager: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() { result <- sm.Initialize(ctx) }()

	address := fmt.Sprintf("http://127.0.0.1:%d/version", port)
	deadline := time.Now().Add(5 * time.Second)
	for {
		resp, err := http.Get(address)
		if err == nil {
			_ = resp.Body.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("server did not start: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	cancel()

	select {
	case err := <-result:
		if err != nil {
			t.Errorf("expected Initialize to return nil after cancellation, got %v", err)
		}
	case <-time.After(ServerShutdownTimeout):
		t.Fatal("Initialize did not return after the context was cancelled")
	}

	if resp, err := http.Get(address); err == nil {
		_ = resp.Body.Close()
		t.Error("expected the server to be stopped")
	}
}
