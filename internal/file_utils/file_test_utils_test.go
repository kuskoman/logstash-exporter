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

func TestCreateTempFileInDir(t *testing.T) {
	t.Run("creates temporary file with given content in directory", func(t *testing.T) {
		content := "hello world"
		dname, err := os.MkdirTemp("", "sampledir")
		if err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}

		path := CreateTempFileInDir(t, content, dname)
		defer RemoveDir(t, dname)

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

func TestAppendToFilex3(t *testing.T) {
	t.Run("appends content to a file three times", func(t *testing.T) {
		// Setup
		content := "hello world"
		new_content := "!"
		expected := "hello world!!!"
		path := CreateTempFile(t, content)
		defer RemoveFile(t, path)

		// Read the file content before modification and verify it matches
		readContent, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read temp file: %v", err)
		}

		if string(readContent) != content {
			t.Errorf("expected file content to be '%s', got '%s'", content, string(readContent))
		}

		// Execute
		AppendToFilex3(t, path, new_content)

		// Read the file content after modification and verify it matches
		readContent, err = os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read temp file: %v", err)
		}

		if string(readContent) != expected {
			t.Errorf("expected file content to be '%s', got '%s'", expected, string(readContent))
		}
	})

	t.Run("handles multi-character content correctly", func(t *testing.T) {
		// Setup
		initialContent := "base content"
		appendContent := "APPEND"
		expected := "base contentAPPENDAPPENDAPPEND"
		path := CreateTempFile(t, initialContent)
		defer RemoveFile(t, path)

		// Execute
		AppendToFilex3(t, path, appendContent)

		// Verify
		resultContent, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read temp file: %v", err)
		}

		if string(resultContent) != expected {
			t.Errorf("expected file content to be '%s', got '%s'", expected, string(resultContent))
		}
	})

	t.Run("works with empty initial content", func(t *testing.T) {
		// Setup
		initialContent := ""
		appendContent := "data"
		expected := "datadatadata"
		path := CreateTempFile(t, initialContent)
		defer RemoveFile(t, path)

		// Execute
		AppendToFilex3(t, path, appendContent)

		// Verify
		resultContent, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read temp file: %v", err)
		}

		if string(resultContent) != expected {
			t.Errorf("expected file content to be '%s', got '%s'", expected, string(resultContent))
		}
	})

	t.Run("works with empty append content", func(t *testing.T) {
		// Setup - should not change the file
		initialContent := "unchanged"
		appendContent := ""
		expected := "unchanged"
		path := CreateTempFile(t, initialContent)
		defer RemoveFile(t, path)

		// Execute
		AppendToFilex3(t, path, appendContent)

		// Verify
		resultContent, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read temp file: %v", err)
		}

		if string(resultContent) != expected {
			t.Errorf("expected file content to be '%s', got '%s'", expected, string(resultContent))
		}
	})
}

func TestRemoveDir(t *testing.T) {
	t.Run("removes the dir at the given path", func(t *testing.T) {
		dname, err := os.MkdirTemp("", "sampledir")
		if err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}

		// Ensure the dir exists before removing
		if _, err := os.Stat(dname); os.IsNotExist(err) {
			t.Fatalf("expected file to exist, but it does not: %v", err)
		}

		RemoveDir(t, dname)

		// Ensure the dir does not exist after removing
		if _, err := os.Stat(dname); !os.IsNotExist(err) {
			t.Errorf("expected file to be removed, but it still exists")
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
