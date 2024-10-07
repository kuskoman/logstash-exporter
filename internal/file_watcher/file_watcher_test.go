package file_watcher

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"
)

const testTimeout = 1 * time.Second

func TestFileWatcher(t *testing.T) {
	t.Parallel()

	t.Run("executes listener on file modification", func(t *testing.T) {
		t.Parallel()

		listenerCalled := make(chan struct{})
		tempFile := createTempFile(t, "initial content")
		defer removeFile(t, tempFile)

		fw, err := NewFileWatcher(tempFile, mockListenerWithChannel(listenerCalled))
		if err != nil {
			t.Fatalf("failed to create file watcher: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			_ = fw.Watch(ctx)
		}()

		if err := os.WriteFile(tempFile, []byte("new content"), 0644); err != nil {
			t.Fatalf("failed to modify file: %v", err)
		}

		select {
		case <-listenerCalled:
		case <-time.After(testTimeout):
			t.Fatal("expected listener to be called, but it wasn't")
		}

		cancel()
	})

	t.Run("does not execute listener if content hash is unchanged", func(t *testing.T) {
		t.Parallel()

		listenerCalled := make(chan struct{})
		tempFile := createTempFile(t, "same content")
		defer removeFile(t, tempFile)

		fw, err := NewFileWatcher(tempFile, mockListenerWithChannel(listenerCalled))
		if err != nil {
			t.Fatalf("failed to create file watcher: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			_ = fw.Watch(ctx)
		}()

		if err := os.WriteFile(tempFile, []byte("same content"), 0644); err != nil {
			t.Fatalf("failed to modify file: %v", err)
		}

		select {
		case <-listenerCalled:
			t.Fatal("listener should not have been called")
		case <-time.After(testTimeout):
		}

		cancel()
	})

	t.Run("handles multiple listeners", func(t *testing.T) {
		t.Parallel()

		listener1Called := make(chan struct{})
		listener2Called := make(chan struct{})
		tempFile := createTempFile(t, "initial content")
		defer removeFile(t, tempFile)

		fw, err := NewFileWatcher(tempFile, mockListenerWithChannel(listener1Called), mockListenerWithChannel(listener2Called))
		if err != nil {
			t.Fatalf("failed to create file watcher: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			_ = fw.Watch(ctx)
		}()

		if err := os.WriteFile(tempFile, []byte("new content"), 0644); err != nil {
			t.Fatalf("failed to modify file: %v", err)
		}

		select {
		case <-listener1Called:
		case <-time.After(testTimeout):
			t.Fatal("expected listener 1 to be called, but it wasn't")
		}

		select {
		case <-listener2Called:
		case <-time.After(testTimeout):
			t.Fatal("expected listener 2 to be called, but it wasn't")
		}

		cancel()
	})

	t.Run("returns error if listener fails", func(t *testing.T) {
		t.Parallel()

		tempFile := createTempFile(t, "initial content")
		defer removeFile(t, tempFile)

		fw, err := NewFileWatcher(tempFile, failingListener())
		if err != nil {
			t.Fatalf("failed to create file watcher: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var wg sync.WaitGroup
		wg.Add(1)
		var watcherErr error
		go func() {
			defer wg.Done()
			watcherErr = fw.Watch(ctx)
		}()

		if err := os.WriteFile(tempFile, []byte("new content"), 0644); err != nil {
			t.Fatalf("failed to modify file: %v", err)
		}

		wg.Wait()

		if watcherErr == nil {
			t.Error("expected error from listener, got nil")
		}

		cancel()
	})
}

func mockListenerWithChannel(called chan struct{}) func() error {
	return func() error {
		called <- struct{}{}
		return nil
	}
}

func failingListener() func() error {
	return func() error {
		return errors.New("listener failed")
	}
}
