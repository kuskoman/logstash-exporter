package file_utils

import (
	"os"
	"testing"
)

// CreateTempFile creates a temporary file with the given content and returns the path to it.
func CreateTempFile(t *testing.T, content string) string {
	t.Helper()

	tempFile, err := os.CreateTemp("", "testfile")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	if _, err := tempFile.WriteString(content); err != nil {
		tempFile.Close()
		t.Fatalf("failed to write to temp file: %v", err)
	}

	if err := tempFile.Close(); err != nil {
		t.Fatalf("failed to close temp file: %v", err)
	}

	return tempFile.Name()
}

// RemoveFile removes the file at the given path.
func RemoveFile(t *testing.T, path string) {
	t.Helper()

	if err := os.Remove(path); err != nil {
		t.Errorf("failed to remove temp file: %v", err)
	}
}
