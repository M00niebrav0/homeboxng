package plugins

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// AuditEntry
// ---------------------------------------------------------------------------

// AuditEntry represents a single auditable event within the plugin system.
type AuditEntry struct {
	// ID is a unique identifier for this entry (UUID v4).
	ID string `json:"id" csv:"id"`
	// Timestamp is when the event occurred.
	Timestamp time.Time `json:"timestamp" csv:"timestamp"`
	// PluginName identifies which plugin produced the event.
	PluginName string `json:"plugin_name" csv:"plugin_name"`
	// Action describes what happened (e.g. "permission.granted", "config.updated").
	Action string `json:"action" csv:"action"`
	// Actor is the user or system principal that triggered the event.
	Actor string `json:"actor" csv:"actor"`
	// Details carries arbitrary key-value metadata about the event.
	Details map[string]string `json:"details,omitempty" csv:"-"`
	// Severity is one of "info", "warning", or "critical".
	Severity string `json:"severity" csv:"severity"`
}

// ---------------------------------------------------------------------------
// AuditFilter
// ---------------------------------------------------------------------------

// AuditFilter describes the criteria for querying audit entries.
type AuditFilter struct {
	PluginName string
	Action     string
	Actor      string
	Since      time.Time
	Severity   string
	Limit      int
}

// ---------------------------------------------------------------------------
// AuditLog — thread-safe circular buffer
// ---------------------------------------------------------------------------

const defaultMaxEntries = 10000

// AuditLog is a thread-safe, bounded circular buffer of AuditEntry records.
type AuditLog struct {
	mu         sync.RWMutex
	entries    []AuditEntry
	head       int // index of the oldest entry
	count      int
	maxEntries int
}

// NewAuditLog creates a new AuditLog. If maxEntries is <= 0 the default of
// 10 000 is used.
func NewAuditLog(maxEntries int) *AuditLog {
	if maxEntries <= 0 {
		maxEntries = defaultMaxEntries
	}
	return &AuditLog{
		entries:    make([]AuditEntry, maxEntries),
		maxEntries: maxEntries,
	}
}

// Record appends an AuditEntry to the log. If the entry has no ID one will be
// generated. If Timestamp is zero it is set to time.Now().
func (al *AuditLog) Record(entry AuditEntry) {
	al.mu.Lock()
	defer al.mu.Unlock()

	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	idx := (al.head + al.count) % al.maxEntries
	if al.count == al.maxEntries {
		// Buffer is full — overwrite oldest, advance head.
		al.entries[al.head] = entry
		al.head = (al.head + 1) % al.maxEntries
	} else {
		al.entries[idx] = entry
		al.count++
	}
}

// Query returns entries matching the supplied filter. The results are in
// chronological order (oldest first). A zero-value filter matches everything.
func (al *AuditLog) Query(filter AuditFilter) []AuditEntry {
	al.mu.RLock()
	defer al.mu.RUnlock()

	var results []AuditEntry

	limit := filter.Limit
	if limit <= 0 {
		limit = al.count
	}

	for i := 0; i < al.count && len(results) < limit; i++ {
		idx := (al.head + i) % al.maxEntries
		e := al.entries[idx]

		if filter.PluginName != "" && e.PluginName != filter.PluginName {
			continue
		}
		if filter.Action != "" && e.Action != filter.Action {
			continue
		}
		if filter.Actor != "" && e.Actor != filter.Actor {
			continue
		}
		if !filter.Since.IsZero() && e.Timestamp.Before(filter.Since) {
			continue
		}
		if filter.Severity != "" && e.Severity != filter.Severity {
			continue
		}

		results = append(results, e)
	}

	return results
}

// Export returns a copy of all entries currently in the log in chronological
// order. This is equivalent to Query with a zero-value filter.
func (al *AuditLog) Export() []AuditEntry {
	return al.Query(AuditFilter{})
}

// Len returns the number of entries currently stored.
func (al *AuditLog) Len() int {
	al.mu.RLock()
	defer al.mu.RUnlock()
	return al.count
}

// ---------------------------------------------------------------------------
// Pre-built audit helpers
// ---------------------------------------------------------------------------

// LogPermissionGrant records that a permission was granted to a plugin.
func LogPermissionGrant(log *AuditLog, plugin, permission, actor string) {
	log.Record(AuditEntry{
		PluginName: plugin,
		Action:     "permission.granted",
		Actor:      actor,
		Details:    map[string]string{"permission": permission},
		Severity:   "info",
	})
}

// LogPermissionRevoke records that a permission was revoked from a plugin.
func LogPermissionRevoke(log *AuditLog, plugin, permission, actor string) {
	log.Record(AuditEntry{
		PluginName: plugin,
		Action:     "permission.revoked",
		Actor:      actor,
		Details:    map[string]string{"permission": permission},
		Severity:   "warning",
	})
}

// LogConfigChange records a configuration value change for a plugin.
func LogConfigChange(log *AuditLog, plugin, key, oldValue, newValue, actor string) {
	log.Record(AuditEntry{
		PluginName: plugin,
		Action:     "config.updated",
		Actor:      actor,
		Details: map[string]string{
			"key":       key,
			"old_value": oldValue,
			"new_value": newValue,
		},
		Severity: "info",
	})
}

// LogPluginStateChange records a plugin lifecycle event (enable, disable, reset, etc.).
func LogPluginStateChange(log *AuditLog, plugin, action, actor string) {
	severity := "info"
	if action == "disabled" || action == "reset" {
		severity = "warning"
	}

	log.Record(AuditEntry{
		PluginName: plugin,
		Action:     "plugin." + action,
		Actor:      actor,
		Details:    nil,
		Severity:   severity,
	})
}

// ---------------------------------------------------------------------------
// AuditExporter interface + implementations
// ---------------------------------------------------------------------------

// AuditExporter defines an interface for serialising audit entries.
type AuditExporter interface {
	// Export serialises the provided entries and returns the raw bytes.
	Export(entries []AuditEntry) ([]byte, error)
	// ContentType returns the MIME type of the exported data.
	ContentType() string
}

// ---------------------------------------------------------------------------
// JSONExporter
// ---------------------------------------------------------------------------

// JSONExporter serialises audit entries as a JSON array.
type JSONExporter struct{}

// Export serialises entries as indented JSON.
func (JSONExporter) Export(entries []AuditEntry) ([]byte, error) {
	return json.MarshalIndent(entries, "", "  ")
}

// ContentType returns the JSON MIME type.
func (JSONExporter) ContentType() string {
	return "application/json"
}

// ---------------------------------------------------------------------------
// CSVExporter
// ---------------------------------------------------------------------------

// CSVExporter serialises audit entries as CSV. The Details map is flattened
// into a single JSON-encoded column.
type CSVExporter struct{}

// Export writes the entries as RFC 4180 CSV.
func (CSVExporter) Export(entries []AuditEntry) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	// Header row.
	header := []string{"id", "timestamp", "plugin_name", "action", "actor", "severity", "details"}
	if err := w.Write(header); err != nil {
		return nil, fmt.Errorf("csv header: %w", err)
	}

	for _, e := range entries {
		detailsJSON := ""
		if len(e.Details) > 0 {
			b, err := json.Marshal(e.Details)
			if err == nil {
				detailsJSON = string(b)
			}
		}

		row := []string{
			e.ID,
			e.Timestamp.UTC().Format(time.RFC3339),
			e.PluginName,
			e.Action,
			e.Actor,
			e.Severity,
			detailsJSON,
		}
		if err := w.Write(row); err != nil {
			return nil, fmt.Errorf("csv row: %w", err)
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return nil, fmt.Errorf("csv flush: %w", err)
	}

	return buf.Bytes(), nil
}

// ContentType returns the CSV MIME type.
func (CSVExporter) ContentType() string {
	return "text/csv"
}

// Compile-time interface satisfaction checks.
var (
	_ AuditExporter = JSONExporter{}
	_ AuditExporter = CSVExporter{}
)
