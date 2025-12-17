package library

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/stephencjuliano/media-server/internal/config"
	"github.com/stephencjuliano/media-server/internal/db"
)

// Watcher monitors media sources for file changes
type Watcher struct {
	db      *db.DB
	cfg     *config.Config
	scanner *Scanner
	watcher *fsnotify.Watcher
	done    chan struct{}
}

// NewWatcher creates a new file watcher
func NewWatcher(database *db.DB, cfg *config.Config, scanner *Scanner) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &Watcher{
		db:      database,
		cfg:     cfg,
		scanner: scanner,
		watcher: fsWatcher,
		done:    make(chan struct{}),
	}, nil
}

// Start begins watching all media sources
func (w *Watcher) Start() error {
	sources, err := w.db.GetAllMediaSources()
	if err != nil {
		return err
	}

	for _, source := range sources {
		if !source.Enabled {
			continue
		}
		if err := w.addPath(source.Path); err != nil {
			log.Printf("Error watching %s: %v", source.Path, err)
		}
	}

	go w.eventLoop()
	return nil
}

// Stop stops the watcher
func (w *Watcher) Stop() {
	close(w.done)
	w.watcher.Close()
}

func (w *Watcher) addPath(path string) error {
	// Add the root path
	if err := w.watcher.Add(path); err != nil {
		return err
	}

	// Recursively add subdirectories
	return filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return w.watcher.Add(path)
		}
		return nil
	})
}

func (w *Watcher) eventLoop() {
	for {
		select {
		case <-w.done:
			return

		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			w.handleEvent(event)

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v", err)
		}
	}
}

func (w *Watcher) handleEvent(event fsnotify.Event) {
	ext := strings.ToLower(filepath.Ext(event.Name))
	if !videoExtensions[ext] {
		return
	}

	switch {
	case event.Op&fsnotify.Create == fsnotify.Create:
		log.Printf("New file detected: %s", event.Name)
		// Find which source this file belongs to
		sources, _ := w.db.GetAllMediaSources()
		for _, source := range sources {
			if strings.HasPrefix(event.Name, source.Path) {
				go w.scanner.processFile(event.Name, source)
				break
			}
		}

	case event.Op&fsnotify.Remove == fsnotify.Remove:
		log.Printf("File removed: %s", event.Name)
		// TODO: Remove from database

	case event.Op&fsnotify.Rename == fsnotify.Rename:
		log.Printf("File renamed: %s", event.Name)
		// TODO: Update database
	}
}
