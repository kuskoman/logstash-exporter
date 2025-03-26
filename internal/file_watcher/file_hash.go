package file_watcher

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
)

// openFile opens the file at the given path and returns the file pointer.
func openFile(filePath string) (*os.File, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not open file: %v", err)
	}
	return file, nil
}

// computeHash reads the file data and computes the SHA-256 hash.
func computeHash(file io.Reader) ([]byte, error) {
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return nil, fmt.Errorf("could not calculate hash: %v", err)
	}
	return hash.Sum(nil), nil
}

// encodeHashToString encodes the hash sum into a hexadecimal string.
func encodeHashToString(hashSum []byte) string {
	return hex.EncodeToString(hashSum)
}

// CalculateFileHash calculates the SHA-256 hash of a file and returns it as a string.
func CalculateFileHash(filePath string) (string, error) {
	file, err := openFile(filePath)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := file.Close(); err != nil {
			slog.Error("failed to close file", "error", err)
		}
	}()

	hashSum, err := computeHash(file)
	if err != nil {
		return "", err
	}

	return encodeHashToString(hashSum), nil
}
