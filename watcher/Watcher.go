package watcher

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"ddbt/utils"
)

type Watcher struct {
	mutex sync.Mutex

	watcher     *fsnotify.Watcher
	directories map[string]struct{}
	stop        chan struct{}

	EventsReady chan struct{}

	events         *Events
	notifyListener utils.DebouncedFunction
}

func NewWatcher() (*Watcher, error) {
	fswatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		watcher:     fswatcher,
		directories: make(map[string]struct{}),
		stop:        make(chan struct{}),
		events:      nil,
		EventsReady: make(chan struct{}),
	}

	// We debounce this to give the system time to process mass file updates
	w.notifyListener = utils.Debounce(func() {
		w.EventsReady <- struct{}{}
	}, 50*time.Millisecond)

	go w.listenForChangeEvents()

	return w, nil
}

func (w *Watcher) RecursivelyWatch(folder string) error {
	folder = filepath.Clean(folder)

	w.mutex.Lock()

	// Track the fact we're watching this directory
	if _, found := w.directories[folder]; found {
		w.mutex.Unlock()
		return nil
	}
	w.directories[folder] = struct{}{}
	w.mutex.Unlock() // unlock here to prevent reentrant locks during recursion

	if err := w.watcher.Add(folder); err != nil {
		return err
	}

	return filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return w.RecursivelyWatch(path)
		}

		return nil
	})
}

func (w *Watcher) listenForChangeEvents() {
	for {
		select {
		case <-w.stop:
			_ = w.watcher.Close()
			return

		case event := <-w.watcher.Events:
			switch {
			case event.Op&fsnotify.Create == fsnotify.Create:
				w.handleCreateEvent(event.Name)
			case event.Op&fsnotify.Write == fsnotify.Write:
				w.handleWriteEvent(event.Name)
			case event.Op&fsnotify.Remove == fsnotify.Remove:
				w.handleDeleteEvent(event.Name)
			}

		case err := <-w.watcher.Errors:
			fmt.Println("ERROR", err)
		}
	}
}

func (w *Watcher) handleCreateEvent(path string) {
	if info, err := os.Stat(path); err != nil {
		fmt.Printf("⚠️ Unable to stat %s: %s\n", path, err)
	} else if info.IsDir() {
		if err := w.RecursivelyWatch(path); err != nil {
			fmt.Printf("⚠️ Unable to start watching %s: %s\n", path, err)
		}
	} else {
		w.recordEventInBatch(path, CREATED, info)
	}
}

func (w *Watcher) handleDeleteEvent(path string) {
	// If it's a directory we're watching, stop watching it
	w.mutex.Lock()
	delete(w.directories, path)
	w.mutex.Unlock()

	w.recordEventInBatch(path, DELETED, nil)
}

func (w *Watcher) handleWriteEvent(path string) {
	if info, err := os.Stat(path); err != nil {
		fmt.Printf("⚠️ Unable to stat %s: %s\n", path, err)
	} else if !info.IsDir() {
		w.recordEventInBatch(path, MODIFIED, info)
	}
}

func (w *Watcher) recordEventInBatch(path string, event EventType, info os.FileInfo) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if w.events == nil {
		w.events = newEventBatch()
		w.notifyListener()
	}

	w.events.addEvent(path, event, info)
}

func (w *Watcher) GetEventsBatch() *Events {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	events := w.events
	w.events = nil

	return events
}

func (w *Watcher) Close() {
	w.stop <- struct{}{}
	close(w.EventsReady)
	close(w.stop)
}
