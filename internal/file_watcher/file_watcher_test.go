package file_watcher

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/kuskoman/logstash-exporter/internal/file_utils"
)

const testTimeout = 1 * time.Second

func TestFileWatcher(t *testing.T) {
	// These tests involve filesystem operations which can be flaky when run in parallel
	// Do not run tests in parallel for better reliability
	
	t.Run("should_execute_listener_on_file_modification", func(t *testing.T) {
		// Setup
		listenerCalled := make(chan struct{})
		dname, err := os.MkdirTemp("", "sampledir")
		if err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
		// Important to clean up at the end
		defer func() {
			if err := os.RemoveAll(dname); err != nil {
				t.Logf("failed to remove temp dir: %v", err)
			}
		}()

		tempFile := file_utils.CreateTempFileInDir(t, "initial content", dname)
		
		// Ensure the file exists
		if _, err := os.Stat(tempFile); os.IsNotExist(err) {
			t.Fatalf("temp file does not exist: %v", err)
		}
		
		// Create the file watcher
		fw, err := NewFileWatcher(tempFile, mockListenerWithChannel(listenerCalled))
		if err != nil {
			t.Fatalf("failed to create file watcher: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		// Execute
		readyCh, err := fw.Watch(ctx)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		
		// Wait for the watcher to be ready
		<-readyCh
		
		file_utils.AppendToFilex3(t, tempFile, "new content")

		// Verify
		select {
		case <-listenerCalled:
			// Success - listener was called
		case <-time.After(testTimeout):
			t.Errorf("expected listener to be called, but it wasn't")
		}
	})

	t.Run("should_not_execute_listener_if_content_hash_is_unchanged", func(t *testing.T) {
		// Setup
		listenerCalled := make(chan struct{})
		
		// Create a temp directory for this test
		dname, err := os.MkdirTemp("", "sampledir")
		if err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
		defer func() {
			if err := os.RemoveAll(dname); err != nil {
				t.Logf("failed to remove temp dir: %v", err)
			}
		}()
		
		tempFile := file_utils.CreateTempFileInDir(t, "same content", dname)
		
		// Ensure the file exists
		if _, err := os.Stat(tempFile); os.IsNotExist(err) {
			t.Fatalf("temp file does not exist: %v", err)
		}

		fw, err := NewFileWatcher(tempFile, mockListenerWithChannel(listenerCalled))
		if err != nil {
			t.Fatalf("failed to create file watcher: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		// Execute
		readyCh, err := fw.Watch(ctx)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		
		// Wait for the watcher to be ready
		<-readyCh
		
		// Write the same content
		if err := os.WriteFile(tempFile, []byte("same content"), 0644); err != nil {
			t.Fatalf("failed to modify file: %v", err)
		}

		// Verify
		select {
		case <-listenerCalled:
			t.Errorf("expected listener not to be called, but it was")
		case <-time.After(testTimeout):
			// Success - listener was not called
		}
	})

	t.Run("should_handle_multiple_listeners", func(t *testing.T) {
		// Setup
		listener1Called := make(chan struct{})
		listener2Called := make(chan struct{})
		
		// Create a temp directory for this test
		dname, err := os.MkdirTemp("", "sampledir")
		if err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
		defer func() {
			if err := os.RemoveAll(dname); err != nil {
				t.Logf("failed to remove temp dir: %v", err)
			}
		}()
		
		tempFile := file_utils.CreateTempFileInDir(t, "initial content", dname)
		
		// Ensure the file exists
		if _, err := os.Stat(tempFile); os.IsNotExist(err) {
			t.Fatalf("temp file does not exist: %v", err)
		}

		fw, err := NewFileWatcher(
			tempFile,
			mockListenerWithChannel(listener1Called),
			mockListenerWithChannel(listener2Called),
		)
		if err != nil {
			t.Fatalf("failed to create file watcher: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		// Execute
		readyCh, err := fw.Watch(ctx)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		
		// Wait for the watcher to be ready
		<-readyCh

		file_utils.AppendToFilex3(t, tempFile, "new content")

		// Verify both listeners were called
		select {
		case <-listener1Called:
			// Success - listener 1 was called
		case <-time.After(testTimeout):
			t.Errorf("expected listener 1 to be called, but it wasn't")
		}

		select {
		case <-listener2Called:
			// Success - listener 2 was called
		case <-time.After(testTimeout):
			t.Errorf("expected listener 2 to be called, but it wasn't")
		}
	})
}

// mockListenerWithChannel creates a listener function that signals the provided channel when called
func mockListenerWithChannel(called chan struct{}) func() error {
	return func() error {
		called <- struct{}{}
		return nil
	}
}
