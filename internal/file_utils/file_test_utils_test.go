package file_utils

import (
	"os"
	"testing"
)

func TestCreateTempFile(t *testing.T) {
	t.Run("creates temporary file with given content", func(t *testing.T) {
		content := "hello world"
		path := CreateTempFile(t, content)
		defer RemoveFile(t, path)

		// Check if the file exists
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Fatalf("expected file to exist, but it does not: %v", err)
		}

		// Read the file content and verify it matches
		readContent, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read temp file: %v", err)
		}

		if string(readContent) != content {
			t.Errorf("expected file content to be '%s', got '%s'", content, string(readContent))
		}
	})
}

func TestRemoveFile(t *testing.T) {
	t.Run("removes the file at the given path", func(t *testing.T) {
		content := "file to be deleted"
		path := CreateTempFile(t, content)

		// Ensure the file exists before removing
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Fatalf("expected file to exist, but it does not: %v", err)
		}

		RemoveFile(t, path)

		// Ensure the file does not exist after removing
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("expected file to be removed, but it still exists")
		}
	})
}
