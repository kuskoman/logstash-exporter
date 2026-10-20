package file_watcher

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kuskoman/logstash-exporter/internal/file_utils"
)

const testTimeout = 1 * time.Second

func TestFileWatcher(t *testing.T) {
	t.Run("should_execute_listener_on_file_modification", func(t *testing.T) {
		listenerCalled := make(chan struct{})
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

		if _, err := os.Stat(tempFile); os.IsNotExist(err) {
			t.Fatalf("temp file does not exist: %v", err)
		}

		fw, err := NewFileWatcher(tempFile, mockListenerWithChannel(listenerCalled))
		if err != nil {
			t.Fatalf("failed to create file watcher: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		readyCh, err := fw.Watch(ctx)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		<-readyCh

		file_utils.AppendToFilex3(t, tempFile, "new content")

		select {
		case <-listenerCalled:
			// Success
		case <-time.After(testTimeout):
			t.Errorf("expected listener to be called, but it wasn't")
		}
	})

	t.Run("should_not_execute_listener_if_content_hash_is_unchanged", func(t *testing.T) {
		listenerCalled := make(chan struct{})

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

		if _, err := os.Stat(tempFile); os.IsNotExist(err) {
			t.Fatalf("temp file does not exist: %v", err)
		}

		fw, err := NewFileWatcher(tempFile, mockListenerWithChannel(listenerCalled))
		if err != nil {
			t.Fatalf("failed to create file watcher: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		readyCh, err := fw.Watch(ctx)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		<-readyCh

		if err := os.WriteFile(tempFile, []byte("same content"), 0644); err != nil {
			t.Fatalf("failed to modify file: %v", err)
		}

		select {
		case <-listenerCalled:
			t.Errorf("expected listener not to be called, but it was")
		case <-time.After(testTimeout):
			// Success
		}
	})

	t.Run("should_handle_multiple_listeners", func(t *testing.T) {
		listener1Called := make(chan struct{})
		listener2Called := make(chan struct{})

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

		readyCh, err := fw.Watch(ctx)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		<-readyCh

		file_utils.AppendToFilex3(t, tempFile, "new content")

		select {
		case <-listener1Called:
			// Success
		case <-time.After(testTimeout):
			t.Errorf("expected listener 1 to be called, but it wasn't")
		}

		select {
		case <-listener2Called:
			// Success
		case <-time.After(testTimeout):
			t.Errorf("expected listener 2 to be called, but it wasn't")
		}
	})
}

func TestFileWatcherDebounce(t *testing.T) {
	t.Run("should_read_final_content_when_file_is_truncated_before_write", func(t *testing.T) {
		tempFile := file_utils.CreateTempFileInDir(t, "initial content", t.TempDir())

		contents := make(chan string, 10)
		fw, err := NewFileWatcher(tempFile, func() error {
			data, err := os.ReadFile(tempFile)
			if err != nil {
				return err
			}
			contents <- string(data)
			return nil
		})
		if err != nil {
			t.Fatalf("failed to create file watcher: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		readyCh, err := fw.Watch(ctx)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		<-readyCh

		// Simulate a non-atomic save: the file is empty for a moment
		// before the new content is written.
		file, err := os.OpenFile(tempFile, os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			t.Fatalf("failed to truncate file: %v", err)
		}
		time.Sleep(fw.debounceTime / 5)
		if _, err := file.WriteString("final content"); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}
		if err := file.Close(); err != nil {
			t.Fatalf("failed to close file: %v", err)
		}

		select {
		case content := <-contents:
			if content != "final content" {
				t.Errorf("expected listener to see %q, got %q", "final content", content)
			}
		case <-time.After(testTimeout):
			t.Fatal("expected listener to be called, but it wasn't")
		}

		select {
		case content := <-contents:
			t.Errorf("expected listener to be called once, but it was called again with %q", content)
		case <-time.After(3 * fw.debounceTime):
		}
	})

	t.Run("should_call_listener_once_for_a_burst_of_writes", func(t *testing.T) {
		tempFile := file_utils.CreateTempFileInDir(t, "initial content", t.TempDir())

		calls := make(chan struct{}, 10)
		fw, err := NewFileWatcher(tempFile, func() error {
			calls <- struct{}{}
			return nil
		})
		if err != nil {
			t.Fatalf("failed to create file watcher: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		readyCh, err := fw.Watch(ctx)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		<-readyCh

		for i := 0; i < 5; i++ {
			file_utils.AppendToFilex3(t, tempFile, "more content")
			time.Sleep(fw.debounceTime / 5)
		}

		select {
		case <-calls:
		case <-time.After(testTimeout):
			t.Fatal("expected listener to be called, but it wasn't")
		}

		select {
		case <-calls:
			t.Error("expected listener to be called once for the burst, but it was called again")
		case <-time.After(3 * fw.debounceTime):
		}
	})
}

func TestFileWatcherReplacedFile(t *testing.T) {
	t.Run("should_execute_listener_when_file_is_replaced_by_rename", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "config.yml")
		writeTestFile(t, configPath, "initial content")

		listenerCalled := make(chan struct{}, 10)
		startWatcher(t, configPath, listenerCalled)

		tempPath := filepath.Join(dir, ".config.yml.swp")
		writeTestFile(t, tempPath, "new content")
		if err := os.Rename(tempPath, configPath); err != nil {
			t.Fatalf("failed to rename file: %v", err)
		}

		expectCall(t, listenerCalled)
	})

	t.Run("should_execute_listener_when_symlinked_directory_is_swapped", func(t *testing.T) {
		// Layout of a Kubernetes ConfigMap volume:
		// config.yml -> ..data/config.yml, ..data -> ..<timestamp>
		dir := t.TempDir()
		writeTestFile(t, filepath.Join(dir, "..v1", "config.yml"), "initial content")
		mustSymlink(t, "..v1", filepath.Join(dir, "..data"))
		mustSymlink(t, filepath.Join("..data", "config.yml"), filepath.Join(dir, "config.yml"))

		listenerCalled := make(chan struct{}, 10)
		startWatcher(t, filepath.Join(dir, "config.yml"), listenerCalled)

		writeTestFile(t, filepath.Join(dir, "..v2", "config.yml"), "new content")
		mustSymlink(t, "..v2", filepath.Join(dir, "..data_tmp"))
		if err := os.Rename(filepath.Join(dir, "..data_tmp"), filepath.Join(dir, "..data")); err != nil {
			t.Fatalf("failed to swap symlink: %v", err)
		}

		expectCall(t, listenerCalled)
	})

	t.Run("should_not_execute_listener_when_other_file_changes", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "config.yml")
		writeTestFile(t, configPath, "initial content")

		listenerCalled := make(chan struct{}, 10)
		startWatcher(t, configPath, listenerCalled)

		writeTestFile(t, filepath.Join(dir, "other.yml"), "other content")

		select {
		case <-listenerCalled:
			t.Error("expected listener not to be called when the watched content is unchanged")
		case <-time.After(3 * 100 * time.Millisecond):
		}
	})
}

func startWatcher(t *testing.T, path string, listenerCalled chan struct{}) {
	t.Helper()

	fw, err := NewFileWatcher(path, func() error {
		listenerCalled <- struct{}{}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to create file watcher: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	readyCh, err := fw.Watch(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	<-readyCh
}

func expectCall(t *testing.T, called chan struct{}) {
	t.Helper()

	select {
	case <-called:
	case <-time.After(testTimeout):
		t.Fatal("expected listener to be called, but it wasn't")
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
}

func mustSymlink(t *testing.T, target, link string) {
	t.Helper()

	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}
}

func mockListenerWithChannel(called chan struct{}) func() error {
	return func() error {
		called <- struct{}{}
		return nil
	}
}
