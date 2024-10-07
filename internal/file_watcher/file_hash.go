package file_watcher

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

func calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("could not open file: %v", err)
	}
	defer file.Close()

	hash := sha256.New()

	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("could not calculate hash: %v", err)
	}

	hashSum := hash.Sum(nil)

	return hex.EncodeToString(hashSum), nil
}
