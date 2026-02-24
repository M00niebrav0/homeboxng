package plugins

import (
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// LogEntry represents a single log line from a plugin.
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Plugin    string    `json:"plugin"`
}

// PluginLogCollector captures log entries per plugin in a ring buffer.
// Each plugin gets its own log buffer so users can view per-plugin logs
// in the Plugin Manager UI (similar to HACS log viewer in Home Assistant).
type PluginLogCollector struct {
	mu      sync.RWMutex
	buffers map[string]*ringBuffer
	maxSize int
}

// ringBuffer is a fixed-size circular buffer for log entries.
type ringBuffer struct {
	entries []LogEntry
	head    int
	count   int
	size    int
}

func newRingBuffer(size int) *ringBuffer {
	return &ringBuffer{
		entries: make([]LogEntry, size),
		size:    size,
	}
}

func (rb *ringBuffer) Add(entry LogEntry) {
	rb.entries[rb.head] = entry
	rb.head = (rb.head + 1) % rb.size
	if rb.count < rb.size {
		rb.count++
	}
}

func (rb *ringBuffer) Entries() []LogEntry {
	if rb.count == 0 {
		return nil
	}

	result := make([]LogEntry, rb.count)
	if rb.count < rb.size {
		copy(result, rb.entries[:rb.count])
	} else {
		// Wrap around: oldest entries start after head
		start := rb.head
		for i := 0; i < rb.count; i++ {
			result[i] = rb.entries[(start+i)%rb.size]
		}
	}
	return result
}

func (rb *ringBuffer) Clear() {
	rb.head = 0
	rb.count = 0
}

// NewPluginLogCollector creates a log collector with a per-plugin buffer size.
func NewPluginLogCollector(maxEntriesPerPlugin int) *PluginLogCollector {
	return &PluginLogCollector{
		buffers: make(map[string]*ringBuffer),
		maxSize: maxEntriesPerPlugin,
	}
}

// Log adds a log entry for a plugin.
func (c *PluginLogCollector) Log(pluginName, level, message string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	buf, ok := c.buffers[pluginName]
	if !ok {
		buf = newRingBuffer(c.maxSize)
		c.buffers[pluginName] = buf
	}

	buf.Add(LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Plugin:    pluginName,
	})
}

// GetLogs returns all log entries for a plugin (oldest first).
func (c *PluginLogCollector) GetLogs(pluginName string) []LogEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()

	buf, ok := c.buffers[pluginName]
	if !ok {
		return nil
	}
	return buf.Entries()
}

// ClearLogs removes all log entries for a plugin.
func (c *PluginLogCollector) ClearLogs(pluginName string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if buf, ok := c.buffers[pluginName]; ok {
		buf.Clear()
	}
}

// PluginLogWriter creates a zerolog.Logger for a plugin that captures output
// into the log collector while also writing to the parent logger.
func (c *PluginLogCollector) PluginLogWriter(pluginName string, parent zerolog.Logger) zerolog.Logger {
	hook := &pluginLogHook{
		collector:  c,
		pluginName: pluginName,
	}

	return parent.Hook(hook)
}

// pluginLogHook is a zerolog hook that captures log entries into the collector.
type pluginLogHook struct {
	collector  *PluginLogCollector
	pluginName string
}

func (h *pluginLogHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	levelStr := level.String()
	h.collector.Log(h.pluginName, levelStr, msg)
}

// AllLogs returns logs for all plugins (for admin overview).
func (c *PluginLogCollector) AllLogs() map[string][]LogEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string][]LogEntry)
	for name, buf := range c.buffers {
		entries := buf.Entries()
		if len(entries) > 0 {
			result[name] = entries
		}
	}
	return result
}

// LogStats returns per-plugin log statistics.
func (c *PluginLogCollector) LogStats() map[string]int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := make(map[string]int)
	for name, buf := range c.buffers {
		stats[name] = buf.count
	}
	return stats
}

// FormatEntry formats a log entry as a string (for Discord/terminal output).
func FormatEntry(e LogEntry) string {
	return fmt.Sprintf("[%s] %s [%s] %s",
		e.Timestamp.Format("2006-01-02 15:04:05"),
		e.Level,
		e.Plugin,
		e.Message,
	)
}
