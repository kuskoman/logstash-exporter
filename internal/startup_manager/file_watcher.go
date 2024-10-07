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
	reloadManager reload.Manager
}

// NewFileWatcher initializes a file watcher for hot-reload
func NewFileWatcher(configLocation string, reloadManager reload.Manager) (*FileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	fileName := filepath.Base(configLocation)
	return &FileWatcher{
		watcher:       watcher,
		fileName:      fileName,
		reloadManager: reloadManager,
	}, nil
}

// Watch sets up file watching for config changes
func (fw *FileWatcher) Watch(ctx context.Context, configComparator *ConfigComparator) error {
	fw.reloadManager.On(reload.NotifierFunc(func(ctx context.Context) (string, error) {
		for {
			select {
			case event, ok := <-fw.watcher.Events:
				if !ok {
					return EventError, nil
				}

				if !strings.Contains(event.Name, fw.fileName) {
					return NoEvent, nil
				}

				modified, err := configComparator.LoadAndCompareConfig(ctx)
				if err != nil {
					slog.Error("config reload error", "err", err)
					return EventError, err
				}

				if modified {
					slog.Info("config modified", "config file", event.Name)
					return FileModified, nil
				}

				return NoEvent, nil
			case err := <-fw.watcher.Errors:
				slog.Error("file watcher error", "err", err)
				return EventError, err
			}
		}
	}))

	return nil
}

// Stop stops the file watcher
func (fw *FileWatcher) Stop() error {
	return fw.watcher.Close()
}
