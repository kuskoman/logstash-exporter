package file_watcher

import (
	"os"
	"testing"
)

func TestCalculateFileHash(t *testing.T) {
	t.Parallel()

	t.Run("calculates hash for valid file", func(t *testing.T) {
		t.Parallel()

		content := "hello world"
		path := createTempFile(t, content)
		defer removeFile(t, path)

		expectedHash, err := calculateFileHash(path)
		if err != nil {
			t.Fatalf("could not calculate expected hash: %v", err)
		}

		hash, err := calculateFileHash(path)
		if err != nil {
			t.Fatalf("got an error: %v", err)
		}
		if hash != expectedHash {
			t.Errorf("expected hash to be %v, got %v", expectedHash, hash)
		}
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		t.Parallel()

		_, err := calculateFileHash("non_existent_file.txt")
		if err == nil {
			t.Fatal("expected error, got none")
		}
	})

	t.Run("calculates hash for empty file", func(t *testing.T) {
		t.Parallel()

		path := createTempFile(t, "")
		defer removeFile(t, path)

		expectedHash, err := calculateFileHash(path)
		if err != nil {
			t.Fatalf("could not calculate expected hash: %v", err)
		}

		hash, err := calculateFileHash(path)
		if err != nil {
			t.Fatalf("got an error: %v", err)
		}
		if hash != expectedHash {
			t.Errorf("expected hash to be %v, got %v", expectedHash, hash)
		}
	})

	t.Run("hashes are different for different file contents", func(t *testing.T) {
		t.Parallel()

		path1 := createTempFile(t, "file one content")
		path2 := createTempFile(t, "file two content")
		defer removeFile(t, path1)
		defer removeFile(t, path2)

		assertHashesNotEqual(t, path1, path2)
	})

	t.Run("hashes are identical for same file contents", func(t *testing.T) {
		t.Parallel()

		content := "same content in both files"
		path1 := createTempFile(t, content)
		path2 := createTempFile(t, content)
		defer removeFile(t, path1)
		defer removeFile(t, path2)

		assertHashesEqual(t, path1, path2)
	})
}

func createTempFile(t *testing.T, content string) string {
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

func removeFile(t *testing.T, path string) {
	t.Helper()

	if err := os.Remove(path); err != nil {
		t.Errorf("failed to remove temp file: %v", err)
	}
}

func assertHashesEqual(t *testing.T, path1, path2 string) {
	t.Helper()

	hash1, err := calculateFileHash(path1)
	if err != nil {
		t.Fatalf("failed to calculate hash for file %s: %v", path1, err)
	}

	hash2, err := calculateFileHash(path2)
	if err != nil {
		t.Fatalf("failed to calculate hash for file %s: %v", path2, err)
	}

	if hash1 != hash2 {
		t.Errorf("expected hashes to be equal, got %s and %s", hash1, hash2)
	}
}

func assertHashesNotEqual(t *testing.T, path1, path2 string) {
	t.Helper()

	hash1, err := calculateFileHash(path1)
	if err != nil {
		t.Fatalf("failed to calculate hash for file %s: %v", path1, err)
	}

	hash2, err := calculateFileHash(path2)
	if err != nil {
		t.Fatalf("failed to calculate hash for file %s: %v", path2, err)
	}

	if hash1 == hash2 {
		t.Errorf("expected hashes to be different, but got the same: %s", hash1)
	}
}
