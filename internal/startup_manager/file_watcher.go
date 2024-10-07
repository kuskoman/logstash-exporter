package startup_manager

import (
	"context"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/slok/reload"
)

const (
	EventError   = "error"
	FileModified = "modified"
	NoEvent      = "no-event"
)

// FileWatcher watches the config file for changes and triggers reloads
type FileWatcher struct {
	watcher       *fsnotify.Watcher
	fileName      string
	filePath      string
	reloadManager reload.Manager
}

// NewFileWatcher initializes a file watcher for hot-reload
func NewFileWatcher(configLocation string, reloadManager reload.Manager) (*FileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	// Watch the directory of the config file
	err = watcher.Add(filepath.Dir(configLocation))
	if err != nil {
		return nil, err
	}

	fileName := filepath.Base(configLocation)
	return &FileWatcher{
		watcher:       watcher,
		fileName:      fileName,
		filePath:      configLocation,
		reloadManager: reloadManager,
	}, nil
}

// Watch sets up file watching for config changes
func (fw *FileWatcher) Watch(ctx context.Context, configComparator *ConfigComparator) error {
	slog.Info("watching config file", "file", fw.filePath)

	fw.reloadManager.On(reload.NotifierFunc(func(ctx context.Context) (string, error) {
		for {
			select {
			case event, ok := <-fw.watcher.Events:
				if !ok {
					slog.Error("file watcher event channel closed")
					return EventError, nil
				}

				slog.Debug("file watcher event", "event", event)

				// Check if the event involves the watched file and is a Write or Rename event
				if fw.isRelevantFileEvent(event) {
					modified, err := configComparator.LoadAndCompareConfig(ctx)
					if err != nil {
						slog.Error("config reload error", "err", err)
						return EventError, err
					}

					if modified {
						slog.Info("config modified", "file", fw.filePath)
						return FileModified, nil
					}
				}

			case err := <-fw.watcher.Errors:
				slog.Error("file watcher error", "err", err)
				return EventError, err
			}
		}
	}))

	go func() {
		for {
			select {
			case <-ctx.Done():
				slog.Info("stopping file watcher")
				err := fw.Stop()
				if err != nil {
					slog.Error("failed to stop file watcher", "err", err)
				}
				return
			case <-fw.watcher.Events:
				err := fw.reloadManager.Run(ctx)
				if err != nil {
					slog.Error("failed to run reload manager", "err", err)
				}
			}
		}
	}()

	return nil
}

// isRelevantFileEvent checks if the event corresponds to a modification of the watched file
func (fw *FileWatcher) isRelevantFileEvent(event fsnotify.Event) bool {
	if !strings.Contains(event.Name, fw.fileName) {
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
