package file_watcher

import (
	"context"
	"log/slog"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// FileWatcher watches the config file for changes and triggers reloads
type FileWatcher struct {
	watcher             *fsnotify.Watcher
	fileName            string
	filePath            string
	previousContentHash string
	listeners           []func() error
	mu                  sync.Mutex
	debounceTime        time.Duration
	lastEventTime       time.Time
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
		debounceTime:        50 * time.Millisecond,
	}

	return fileWatcher, nil
}

// Watch sets up file watching and returns a channel that is closed when watching is ready
func (fw *FileWatcher) Watch(ctx context.Context) (<-chan struct{}, error) {
	slog.Info("watching file", "file", fw.filePath)

	readyCh := make(chan struct{})

	go func() {
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

				fw.mu.Lock()
				now := time.Now()
				if now.Sub(fw.lastEventTime) < fw.debounceTime {
					slog.Debug("debouncing file event", "file", fw.filePath)
					fw.mu.Unlock()
					continue
				}
				fw.lastEventTime = now
				fw.mu.Unlock()

				go fw.processFileEvent()
			}
		}
	}()

	return readyCh, nil
}

func (fw *FileWatcher) processFileEvent() {
	contentHash, err := CalculateFileHash(fw.filePath)
	if err != nil {
		slog.Error("failed to calculate file hash", "err", err)
		return
	}

	fw.mu.Lock()
	defer fw.mu.Unlock()

	if contentHash == fw.previousContentHash {
		slog.Debug("file modified, but content hash is unchanged", "file", fw.filePath)
		return
	}

	slog.Info("file modified", "file", fw.filePath)
	slog.Info("content hash changed, executing listeners", "file", fw.filePath)

	fw.previousContentHash = contentHash

	err = fw.executeListeners()
	if err != nil {
		slog.Error("failed to execute listeners", "err", err)
	}
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
