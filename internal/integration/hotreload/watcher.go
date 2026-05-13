package hotreload

import (
	"os"
	"sync"
	"time"

	"github.com/zhaoxiaoyang741/HomeStock/pkg/logger"
)

// Watcher polls the config file for modifications and triggers a reload.
//
// Inspired by picoclaw's setupConfigWatcherPolling:
//   - polls every 2 seconds
//   - compares mtime and file size
//   - waits 500 ms after detection for writes to settle
type Watcher struct {
	configPath string
	interval   time.Duration
	settleTime time.Duration
	reloadFn   func() error

	stopChan chan struct{}
	wg       sync.WaitGroup
}

// NewWatcher creates a Watcher that calls reloadFn when the config file changes.
func NewWatcher(configPath string, reloadFn func() error) *Watcher {
	return &Watcher{
		configPath: configPath,
		interval:   2 * time.Second,
		settleTime: 500 * time.Millisecond,
		reloadFn:   reloadFn,
		stopChan:   make(chan struct{}),
	}
}

// Start begins polling the config file in a background goroutine.
func (w *Watcher) Start() {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()

		lastModTime := getFileModTime(w.configPath)
		lastSize := getFileSize(w.configPath)

		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				currentModTime := getFileModTime(w.configPath)
				currentSize := getFileSize(w.configPath)

				if currentModTime.After(lastModTime) || currentSize != lastSize {
					// Wait for writes to settle to avoid reading a partial write.
					time.Sleep(w.settleTime)

					logger.InfoCF("hotreload", "config file change detected, reloading...", nil)
					if err := w.reloadFn(); err != nil {
						logger.ErrorCF("hotreload", "reload failed", map[string]any{
							"error": err.Error(),
						})
					}

					lastModTime = currentModTime
					lastSize = currentSize
				}

			case <-w.stopChan:
				return
			}
		}
	}()
}

// Stop stops the file watcher.
func (w *Watcher) Stop() {
	close(w.stopChan)
	w.wg.Wait()
}

func getFileModTime(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

func getFileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}
