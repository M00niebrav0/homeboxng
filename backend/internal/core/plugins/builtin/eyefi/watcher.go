package eyefi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// FileWatcher polls directories for new JPEG files.
type FileWatcher struct {
	mu            sync.Mutex
	sources       []*IngestSource
	mode          ImportMode
	pollInterval  time.Duration
	processedHash map[string]bool // SHA256 -> processed
	onNewFiles    func(source string, files []string)
	logger        zerolog.Logger
	cancel        context.CancelFunc
}

// NewFileWatcher creates a file watcher with the given poll interval.
func NewFileWatcher(pollInterval time.Duration, logger zerolog.Logger) *FileWatcher {
	return &FileWatcher{
		mode:          ImportModeManual,
		pollInterval:  pollInterval,
		processedHash: make(map[string]bool),
		logger:        logger.With().Str("component", "file-watcher").Logger(),
	}
}

// AddSource adds an ingestion source to monitor.
func (w *FileWatcher) AddSource(source *IngestSource) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.sources = append(w.sources, source)
}

// SetMode changes the import mode (auto/manual/off).
func (w *FileWatcher) SetMode(mode ImportMode) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.mode = mode
}

// GetMode returns the current import mode.
func (w *FileWatcher) GetMode() ImportMode {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.mode
}

// OnNewFiles sets the callback for newly discovered files.
func (w *FileWatcher) OnNewFiles(fn func(source string, files []string)) {
	w.onNewFiles = fn
}

// Start begins polling all enabled sources.
func (w *FileWatcher) Start(ctx context.Context) {
	ctx, w.cancel = context.WithCancel(ctx)

	go func() {
		ticker := time.NewTicker(w.pollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				w.scan()
			}
		}
	}()

	w.logger.Info().
		Dur("interval", w.pollInterval).
		Msg("file watcher started")
}

// Stop halts the file watcher.
func (w *FileWatcher) Stop() {
	if w.cancel != nil {
		w.cancel()
	}
}

// scan checks all enabled sources for new JPEG files.
func (w *FileWatcher) scan() {
	w.mu.Lock()
	sources := make([]*IngestSource, len(w.sources))
	copy(sources, w.sources)
	mode := w.mode
	w.mu.Unlock()

	if mode == ImportModeOff {
		return
	}

	for _, src := range sources {
		if !src.Enabled || src.Type != "local" {
			continue
		}

		newFiles := w.scanDirectory(src.Path, src.Name)
		if len(newFiles) > 0 && mode == ImportModeAuto && w.onNewFiles != nil {
			w.onNewFiles(src.Name, newFiles)
		}
	}
}

// scanDirectory finds new JPEG files in a directory.
func (w *FileWatcher) scanDirectory(dir, sourceName string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		w.logger.Warn().Err(err).Str("dir", dir).Msg("failed to read directory")
		return nil
	}

	var newFiles []string

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".jpg" && ext != ".jpeg" {
			continue
		}

		fullPath := filepath.Join(dir, entry.Name())

		// Check hash for deduplication
		hash, err := fileHash(fullPath)
		if err != nil {
			continue
		}

		w.mu.Lock()
		seen := w.processedHash[hash]
		if !seen {
			w.processedHash[hash] = true
		}
		w.mu.Unlock()

		if !seen {
			newFiles = append(newFiles, fullPath)
		}
	}

	if len(newFiles) > 0 {
		w.logger.Info().
			Str("source", sourceName).
			Int("count", len(newFiles)).
			Msg("new files detected")
	}

	return newFiles
}

// TestSource performs a real write/read/delete test on an ingestion source.
func TestSource(src *IngestSource) (string, error) {
	switch src.Type {
	case "local":
		return testLocalSource(src.Path)
	default:
		return "", nil
	}
}

// testLocalSource verifies a local directory is accessible.
func testLocalSource(dir string) (string, error) {
	// Check directory exists
	info, err := os.Stat(dir)
	if err != nil {
		return "Directory not found: " + err.Error(), err
	}
	if !info.IsDir() {
		return "Path is not a directory", os.ErrInvalid
	}

	// Write test file
	testFile := filepath.Join(dir, ".homeboxng-test")
	testData := []byte("homeboxng-connection-test-" + time.Now().Format(time.RFC3339))

	if err := os.WriteFile(testFile, testData, 0644); err != nil {
		return "Write failed: " + err.Error(), err
	}

	// Read back
	readData, err := os.ReadFile(testFile)
	if err != nil {
		return "Read failed: " + err.Error(), err
	}

	if string(readData) != string(testData) {
		return "Read/write mismatch", os.ErrInvalid
	}

	// Delete
	if err := os.Remove(testFile); err != nil {
		return "Delete failed: " + err.Error(), err
	}

	return "OK: write/read/delete verified", nil
}

// fileHash computes SHA256 of a file for deduplication.
func fileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// GetStats returns current watcher statistics.
func (w *FileWatcher) GetStats() []WatcherStats {
	w.mu.Lock()
	defer w.mu.Unlock()

	var stats []WatcherStats
	for _, src := range w.sources {
		if !src.Enabled {
			continue
		}

		pending := 0
		if src.Type == "local" {
			entries, err := os.ReadDir(src.Path)
			if err == nil {
				for _, e := range entries {
					ext := strings.ToLower(filepath.Ext(e.Name()))
					if !e.IsDir() && (ext == ".jpg" || ext == ".jpeg") {
						pending++
					}
				}
			}
		}

		stats = append(stats, WatcherStats{
			SourceName:     src.Name,
			PendingFiles:   pending,
			ProcessedFiles: len(w.processedHash),
			LastScanAt:     time.Now(),
			Mode:           w.mode,
		})
	}

	return stats
}
