package file_watcher

import (
	"context"
	"log/slog"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

// FileWatcher watches the config file for changes and triggers reloads
type FileWatcher struct {
	watcher             *fsnotify.Watcher
	fileName            string
	filePath            string
	previousContentHash string
	listeners           []func() error
}

// NewFileWatcher initializes a file watcher, watching the config file for changes
func NewFileWatcher(configLocation string, listeners ...func() error) (*FileWatcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	err = fsWatcher.Add(filepath.Dir(configLocation))
	if err != nil {
		return nil, err
	}

	fileName := filepath.Base(configLocation)

	contentHash, err := CalculateFileHash(configLocation)
	if err != nil {
		return nil, err
	}

	fileWatcher := &FileWatcher{
		watcher:             fsWatcher,
		fileName:            fileName,
		filePath:            configLocation,
		listeners:           listeners,
		previousContentHash: contentHash,
	}

	return fileWatcher, nil
}

// Watch sets up file watching and returns a channel that is closed when watching is ready
func (fw *FileWatcher) Watch(ctx context.Context) (<-chan struct{}, error) {
	slog.Info("watching file", "file", fw.filePath)

	// Return a value to notify when we're ready to watch
	readyCh := make(chan struct{})

	go func() {
		// Signal that we're ready to watch
		close(readyCh)

		for {
			select {
			case <-ctx.Done():
				slog.Info("stopping file watcher")
				err := fw.Stop()
				if err != nil {
					slog.Error("failed to stop file watcher", "err", err)
				}
				return
			case event := <-fw.watcher.Events:
				if !fw.isRelevantFileEvent(event) {
					continue
				}

				contentHash, err := CalculateFileHash(fw.filePath)
				if err != nil {
					slog.Error("failed to calculate file hash", "err", err)
					continue
				}

				if contentHash != fw.previousContentHash {
					slog.Info("file modified", "file", fw.filePath)
					err := fw.executeListeners()
					if err != nil {
						slog.Error("failed to execute listeners", "err", err)
					}

					fw.previousContentHash = contentHash
				} else {
					slog.Debug("file modified, but content hash is unchanged", "file", fw.filePath)
				}
			}
		}
	}()

	return readyCh, nil
}

func (fw *FileWatcher) executeListeners() error {
	for _, listener := range fw.listeners {
		err := listener()
		if err != nil {
			return err
		}
	}

	return nil
}

// isRelevantFileEvent checks if the event corresponds to a modification of the watched file
func (fw *FileWatcher) isRelevantFileEvent(event fsnotify.Event) bool {
	eventFileName := filepath.Base(event.Name)
	if eventFileName != fw.fileName {
		return false
	}

	// Only act on Write or Rename events, which typically indicate file content changes
	if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Rename == fsnotify.Rename {
		slog.Debug("relevant file event detected", "event", event, "file", fw.fileName)
		return true
	}

	slog.Debug("ignoring irrelevant file event", "event", event, "file", fw.fileName)
	return false
}

// Stop stops the file watcher
func (fw *FileWatcher) Stop() error {
	slog.Info("stopping file watcher")
	return fw.watcher.Close()
}
