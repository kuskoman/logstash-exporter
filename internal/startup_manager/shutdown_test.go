package startup_manager

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/kuskoman/logstash-exporter/internal/flags"
)

// blockingMockServer is a mock AppServer that blocks on Shutdown until the context is done
type blockingMockServer struct {
	shutdownCalled chan struct{}
}

func (s *blockingMockServer) ListenAndServe() error {
	return nil
}

func (s *blockingMockServer) Shutdown(ctx context.Context) error {
	// Signal that Shutdown was called
	s.shutdownCalled <- struct{}{}
	// Block until the context is done
	<-ctx.Done()
	// Return the context error (typically DeadlineExceeded)
	return ctx.Err()
}

// Test for edge cases in the shutdownServer method
func TestShutdownServerEdgeCases(t *testing.T) {
	// Do not use t.Parallel() to avoid Prometheus registration issues
	
	t.Run("should_handle_http_errserverclosed_gracefully", func(t *testing.T) {
		// Setup
		configPath := createValidConfigFile(t)
		flagsCfg := &flags.FlagsConfig{
			HotReload: true,
		}

		sm, err := NewStartupManager(configPath, flagsCfg)
		if err != nil {
			t.Fatalf("failed to create startup manager: %v", err)
		}

		// Create a mock server that returns ErrServerClosed
		mockSrv := newMockAppServer(nil, http.ErrServerClosed)
		
		// Set server
		sm.server = mockSrv

		// Execute
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err = sm.shutdownServer(ctx)
		
		// Verify
		if err != nil {
			t.Errorf("expected nil error for ErrServerClosed, got %v", err)
		}

		// Verify that Shutdown was called on the server
		select {
		case <-mockSrv.shutdownCalled:
			// Success
		default:
			t.Errorf("expected server.Shutdown to be called")
		}
	})

	t.Run("should_handle_custom_error", func(t *testing.T) {
		// Setup
		configPath := createValidConfigFile(t)
		flagsCfg := &flags.FlagsConfig{
			HotReload: true,
		}

		sm, err := NewStartupManager(configPath, flagsCfg)
		if err != nil {
			t.Fatalf("failed to create startup manager: %v", err)
		}

		// Create a custom error
		customError := errors.New("custom server error")
		
		// Create a mock server that returns a custom error
		mockSrv := newMockAppServer(nil, customError)
		
		// Set server
		sm.server = mockSrv

		// Execute
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err = sm.shutdownServer(ctx)
		
		// Verify
		if !errors.Is(err, customError) {
			t.Errorf("expected custom error, got %v", err)
		}

		// Verify that Shutdown was called on the server
		select {
		case <-mockSrv.shutdownCalled:
			// Success
		default:
			t.Errorf("expected server.Shutdown to be called")
		}
	})

	t.Run("should_handle_context_deadline", func(t *testing.T) {
		// Setup
		configPath := createValidConfigFile(t)
		flagsCfg := &flags.FlagsConfig{
			HotReload: true,
		}

		sm, err := NewStartupManager(configPath, flagsCfg)
		if err != nil {
			t.Fatalf("failed to create startup manager: %v", err)
		}

		// Create a blocking mock server
		// We create a custom implementation
		mockSrv := &blockingMockServer{
			shutdownCalled: make(chan struct{}, 1),
		}
		
		// Set server
		sm.server = mockSrv

		// Execute with a short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err = sm.shutdownServer(ctx)
		
		// Verify
		if err == nil {
			t.Errorf("expected context deadline error, got nil")
		}
		
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("expected context.DeadlineExceeded, got %v", err)
		}
	})
}