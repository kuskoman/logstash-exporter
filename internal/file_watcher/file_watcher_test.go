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
	// todo: parallelize tests
	t.Run("executes listener on file modification", func(t *testing.T) {
		listenerCalled := make(chan struct{})
		dname, err := os.MkdirTemp("", "sampledir")

		tempFile := file_utils.CreateTempFileInDir(t, "initial content", dname)
		defer file_utils.RemoveDir(t, dname)

		fw, err := NewFileWatcher(tempFile, mockListenerWithChannel(listenerCalled))
		if err != nil {
			t.Fatalf("failed to create file watcher: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(testTimeout))
		defer cancel()

		go func() {
			err := fw.Watch(ctx)
			if err != nil {
				t.Errorf("failed to watch file: %v", err)
			}
		}()

		// ************ Add a content three times to make sure its written ***********
		//f, err := os.OpenFile(tempFile, os.O_APPEND | os.O_WRONLY, 0644)
		//defer f.Close()
		//if err != nil {
		//	t.Fatal(err)
		//}
		//f.Sync()

		//if _, err := f.Write([]byte("appended some data\n")); err != nil {
		//	f.Close() // ignore error; Write error takes precedence
		//	t.Fatal(err)
		//}
		//f.Sync()

		//f.Sync()
		//time.Sleep(50 * time.Millisecond) // give system time to sync write change before delete
		//if _, err := f.Write([]byte("appended some data\n")); err != nil {
		//	f.Close() // ignore error; Write error takes precedence
		//	t.Fatal(err)
		//}

		//f.Sync()
		//time.Sleep(50 * time.Millisecond) // give system time to sync write change before delete
		//if _, err := f.Write([]byte("appended some data\n")); err != nil {
		//	f.Close() // ignore error; Write error takes precedence
		//	t.Fatal(err)
		//}
		//f.Sync()
		// ***************************************************************************
		file_utils.ModifyFile(t, tempFile, "new content")

		select {
		case <-listenerCalled:
		case <-time.After(testTimeout):
			t.Fatal("expected listener to be called, but it wasn't")
		}

		cancel()
	})

	t.Run("does not execute listener if content hash is unchanged", func(t *testing.T) {
		listenerCalled := make(chan struct{})
		tempFile := file_utils.CreateTempFile(t, "same content")
		defer file_utils.RemoveFile(t, tempFile)

		fw, err := NewFileWatcher(tempFile, mockListenerWithChannel(listenerCalled))
		if err != nil {
			t.Fatalf("failed to create file watcher: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(testTimeout))
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
		listener1Called := make(chan struct{})
		listener2Called := make(chan struct{})
		tempFile := file_utils.CreateTempFile(t, "initial content")
		defer file_utils.RemoveFile(t, tempFile)

		fw, err := NewFileWatcher(tempFile, mockListenerWithChannel(listener1Called), mockListenerWithChannel(listener2Called))
		if err != nil {
			t.Fatalf("failed to create file watcher: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(testTimeout))
		defer cancel()

		go func() {
			err := fw.Watch(ctx)
			if err != nil {
				t.Errorf("failed to watch file: %v", err)
			}
		}()

		file_utils.ModifyFile(t, tempFile, "new content")
		//if err := os.WriteFile(tempFile, []byte("new content"), 0644); err != nil {
		//	t.Fatalf("failed to modify file: %v", err)
		//}

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
}

func mockListenerWithChannel(called chan struct{}) func() error {
	return func() error {
		called <- struct{}{}
		return nil
	}
}
