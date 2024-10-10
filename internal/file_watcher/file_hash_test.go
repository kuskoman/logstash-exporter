package file_watcher

import (
	"testing"

	"github.com/kuskoman/logstash-exporter/internal/file_utils"
)

func TestCalculateFileHash(t *testing.T) {
	t.Parallel()

	t.Run("calculates hash for valid file", func(t *testing.T) {
		t.Parallel()

		content := "hello world"
		path := file_utils.CreateTempFile(t, content)
		defer file_utils.RemoveFile(t, path)

		expectedHash, err := CalculateFileHash(path)
		if err != nil {
			t.Fatalf("could not calculate expected hash: %v", err)
		}

		hash, err := CalculateFileHash(path)
		if err != nil {
			t.Fatalf("got an error: %v", err)
		}
		if hash != expectedHash {
			t.Errorf("expected hash to be %v, got %v", expectedHash, hash)
		}
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		t.Parallel()

		_, err := CalculateFileHash("non_existent_file.txt")
		if err == nil {
			t.Fatal("expected error, got none")
		}
	})

	t.Run("calculates hash for empty file", func(t *testing.T) {
		t.Parallel()

		path := file_utils.CreateTempFile(t, "")
		defer file_utils.RemoveFile(t, path)

		expectedHash, err := CalculateFileHash(path)
		if err != nil {
			t.Fatalf("could not calculate expected hash: %v", err)
		}

		hash, err := CalculateFileHash(path)
		if err != nil {
			t.Fatalf("got an error: %v", err)
		}
		if hash != expectedHash {
			t.Errorf("expected hash to be %v, got %v", expectedHash, hash)
		}
	})

	t.Run("hashes are different for different file contents", func(t *testing.T) {
		t.Parallel()

		path1 := file_utils.CreateTempFile(t, "file one content")
		path2 := file_utils.CreateTempFile(t, "file two content")
		defer file_utils.RemoveFile(t, path1)
		defer file_utils.RemoveFile(t, path2)

		assertHashesNotEqual(t, path1, path2)
	})

	t.Run("hashes are identical for same file contents", func(t *testing.T) {
		t.Parallel()

		content := "same content in both files"
		path1 := file_utils.CreateTempFile(t, content)
		path2 := file_utils.CreateTempFile(t, content)
		defer file_utils.RemoveFile(t, path1)
		defer file_utils.RemoveFile(t, path2)

		assertHashesEqual(t, path1, path2)
	})
}

func assertHashesEqual(t *testing.T, path1, path2 string) {
	t.Helper()

	hash1, err := CalculateFileHash(path1)
	if err != nil {
		t.Fatalf("failed to calculate hash for file %s: %v", path1, err)
	}

	hash2, err := CalculateFileHash(path2)
	if err != nil {
		t.Fatalf("failed to calculate hash for file %s: %v", path2, err)
	}

	if hash1 != hash2 {
		t.Errorf("expected hashes to be equal, got %s and %s", hash1, hash2)
	}
}

func assertHashesNotEqual(t *testing.T, path1, path2 string) {
	t.Helper()

	hash1, err := CalculateFileHash(path1)
	if err != nil {
		t.Fatalf("failed to calculate hash for file %s: %v", path1, err)
	}

	hash2, err := CalculateFileHash(path2)
	if err != nil {
		t.Fatalf("failed to calculate hash for file %s: %v", path2, err)
	}

	if hash1 == hash2 {
		t.Errorf("expected hashes to be different, but got the same: %s", hash1)
	}
}
