package monitor

import (
	"context"
	"log"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
)

// newWatcherFunc is a test hook for creating watchers
var newWatcherFunc = fsnotify.NewWatcher

// FileChangeHandler interface for handling file changes
type FileChangeHandler interface {
	ReloadData() error
}

// FileMonitor watches for changes in the buddy folder
type FileMonitor struct {
	path    string
	handler FileChangeHandler
	watcher *fsnotify.Watcher
}

// NewFileMonitor creates a new file monitor
func NewFileMonitor(path string, handler FileChangeHandler) *FileMonitor {
	return &FileMonitor{
		path:    path,
		handler: handler,
	}
}

// Start starts monitoring the buddy folder
func (fm *FileMonitor) Start(ctx context.Context) error {
	watcher, err := newWatcherFunc()
	if err != nil {
		return err
	}
	fm.watcher = watcher

	// Add all subdirectories to watch
	subdirs := []string{
		fm.path,
		filepath.Join(fm.path, "rules"),
		filepath.Join(fm.path, "knowledge"),
		filepath.Join(fm.path, "database"),
		filepath.Join(fm.path, "todos"),
		filepath.Join(fm.path, "history"),
		filepath.Join(fm.path, "backups"),
	}

	for _, dir := range subdirs {
		if err := watcher.Add(dir); err != nil {
			log.Printf("Failed to watch directory %s: %v", dir, err)
		}
	}

	go fm.watchLoop(ctx)

	return nil
}

// watchLoop watches for file events
func (fm *FileMonitor) watchLoop(ctx context.Context) {
	defer fm.watcher.Close()

	for {
		select {
		case <-ctx.Done():
			return

		case event, ok := <-fm.watcher.Events:
			if !ok {
				return
			}

			// Filter relevant events
			if fm.isRelevantEvent(event) {
				log.Printf("File change detected: %s (%s)", event.Name, event.Op)

				// Reload data
				if err := fm.handler.ReloadData(); err != nil {
					log.Printf("Error reloading data: %v", err)
				}
			}

		case err, ok := <-fm.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("File watcher error: %v", err)
		}
	}
}

// isRelevantEvent checks if the event should trigger a reload
func (fm *FileMonitor) isRelevantEvent(event fsnotify.Event) bool {
	// Skip temporary files
	if strings.HasPrefix(filepath.Base(event.Name), ".") ||
		strings.HasSuffix(event.Name, "~") ||
		strings.HasSuffix(event.Name, ".swp") ||
		strings.HasSuffix(event.Name, ".tmp") {
		return false
	}

	// Only care about markdown and JSON files
	if !strings.HasSuffix(event.Name, ".md") &&
		!strings.HasSuffix(event.Name, ".json") &&
		!strings.HasSuffix(event.Name, ".sql") {
		return false
	}

	// Only care about write and create events
	if event.Op&fsnotify.Write == 0 && event.Op&fsnotify.Create == 0 {
		return false
	}

	return true
}
