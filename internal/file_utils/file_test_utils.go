package file_utils

import (
	"log/slog"
	"os"
	"testing"
	"time"
)

func HandleTempDirRemoval(t *testing.T, tempDir string) {
	t.Helper()

	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Errorf("failed to remove temp directory: %v", err)
		}
	}()
}

func CreateTempFileInDir(t *testing.T, content, dir string) string {
	t.Helper()

	tempFile, err := os.CreateTemp(dir, "testfile")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	if _, err := tempFile.WriteString(content); err != nil {
		if err := tempFile.Close(); err != nil {
			slog.Error("failed to close temp file", "error", err)
		}
		t.Fatalf("failed to write to temp file: %v", err)
	}

	if err := tempFile.Close(); err != nil {
		t.Fatalf("failed to close temp file: %v", err)
	}

	return tempFile.Name()
}

func AppendToFilex3(t *testing.T, file, content string) {
	t.Helper()
	// ************ Add a content three times to make sure its written ***********
	f, err := os.OpenFile(file, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("failed to open a file: %v", err)
	}

	defer func() {
		if err := f.Close(); err != nil {
			slog.Error("failed to close file", "error", err)
		}
	}()

	if err := f.Sync(); err != nil {
		t.Fatalf("failed to sync file: %v", err)
	}

	if _, err := f.Write([]byte(content)); err != nil {
		t.Fatalf("failed to write to file: %v", err)
	}
	if err := f.Sync(); err != nil {
		t.Fatalf("failed to sync file: %v", err)
	}

	time.Sleep(50 * time.Millisecond) // give system time to sync write change before delete
	if _, err := f.Write([]byte(content)); err != nil {
		t.Fatalf("failed to write to file: %v", err)
	}

	if err := f.Sync(); err != nil {
		t.Fatalf("failed to sync file: %v", err)
	}
	time.Sleep(50 * time.Millisecond) // give system time to sync write change before delete
	if _, err := f.Write([]byte(content)); err != nil {
		t.Fatalf("failed to write to file: %v", err)
	}
	if err := f.Sync(); err != nil {
		t.Fatalf("failed to sync file: %v", err)
	}
}

// CreateTempFile creates a temporary file with the given content and returns the path to it.
func CreateTempFile(t *testing.T, content string) string {
	return CreateTempFileInDir(t, content, "")
}

func RemoveDir(t *testing.T, path string) {
	t.Helper()

	if err := os.RemoveAll(path); err != nil {
		t.Errorf("failed to remove temp file: %v", err)
	}
}

// RemoveFile removes the file at the given path.
func RemoveFile(t *testing.T, path string) {
	t.Helper()

	if err := os.Remove(path); err != nil {
		t.Errorf("failed to remove temp file: %v", err)
	}
}
