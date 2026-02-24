package plugins

import (
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestAuditLog_Record(t *testing.T) {
	al := NewAuditLog(100)

	entry := AuditEntry{
		Timestamp:  time.Now(),
		PluginName: "test-plugin",
		Action:     "plugin.started",
		Severity:   "info",
		Actor:      "user-1",
	}

	al.Record(entry)

	if al.Len() != 1 {
		t.Fatalf("Len() = %d, want 1", al.Len())
	}

	// Query it back.
	results := al.Query(AuditFilter{PluginName: "test-plugin"})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	got := results[0]
	if got.PluginName != "test-plugin" {
		t.Errorf("PluginName = %q, want %q", got.PluginName, "test-plugin")
	}
	if got.Action != "plugin.started" {
		t.Errorf("Action = %q, want %q", got.Action, "plugin.started")
	}
	if got.Severity != "info" {
		t.Errorf("Severity = %q, want %q", got.Severity, "info")
	}
	if got.Actor != "user-1" {
		t.Errorf("Actor = %q, want %q", got.Actor, "user-1")
	}
}

func TestAuditLog_RecordAutoTimestamp(t *testing.T) {
	al := NewAuditLog(100)

	before := time.Now()
	al.Record(AuditEntry{
		PluginName: "test",
		Action:     "plugin.started",
		Severity:   "info",
	})
	after := time.Now()

	results := al.Export()
	if len(results) != 1 {
		t.Fatal("expected 1 entry")
	}

	ts := results[0].Timestamp
	if ts.Before(before) || ts.After(after) {
		t.Errorf("auto-set Timestamp = %v, expected between %v and %v", ts, before, after)
	}
}

func TestAuditLog_CircularBuffer(t *testing.T) {
	maxEntries := 5
	al := NewAuditLog(maxEntries)

	// Record 8 entries (exceeding the buffer size of 5).
	for i := 0; i < 8; i++ {
		al.Record(AuditEntry{
			Timestamp:  time.Now().Add(time.Duration(i) * time.Second),
			PluginName: "test",
			Action:     "api.access",
			Severity:   "info",
			Details:    map[string]string{"index": string(rune('0' + i))},
		})
	}

	// Count should be capped at maxEntries.
	if al.Len() != maxEntries {
		t.Errorf("Len() = %d, want %d", al.Len(), maxEntries)
	}

	// Export should return entries 3,4,5,6,7 (oldest evicted).
	exported := al.Export()
	if len(exported) != maxEntries {
		t.Fatalf("Export() returned %d entries, want %d", len(exported), maxEntries)
	}
}

func TestAuditLog_QueryByPlugin(t *testing.T) {
	al := NewAuditLog(100)

	al.Record(AuditEntry{PluginName: "plugin-a", Action: "plugin.started", Severity: "info"})
	al.Record(AuditEntry{PluginName: "plugin-b", Action: "plugin.started", Severity: "info"})
	al.Record(AuditEntry{PluginName: "plugin-a", Action: "plugin.stopped", Severity: "info"})

	results := al.Query(AuditFilter{PluginName: "plugin-a"})
	if len(results) != 2 {
		t.Fatalf("expected 2 results for plugin-a, got %d", len(results))
	}
	for _, r := range results {
		if r.PluginName != "plugin-a" {
			t.Errorf("expected all results to be from plugin-a, got %q", r.PluginName)
		}
	}
}

func TestAuditLog_QueryByAction(t *testing.T) {
	al := NewAuditLog(100)

	al.Record(AuditEntry{PluginName: "p", Action: "plugin.started", Severity: "info"})
	al.Record(AuditEntry{PluginName: "p", Action: "permission.granted", Severity: "info"})
	al.Record(AuditEntry{PluginName: "p", Action: "plugin.started", Severity: "info"})

	results := al.Query(AuditFilter{Action: "plugin.started"})
	if len(results) != 2 {
		t.Fatalf("expected 2 results for plugin.started action, got %d", len(results))
	}
}

func TestAuditLog_QueryBySeverity(t *testing.T) {
	al := NewAuditLog(100)

	al.Record(AuditEntry{PluginName: "p", Action: "plugin.started", Severity: "info"})
	al.Record(AuditEntry{PluginName: "p", Action: "plugin.error", Severity: "critical"})
	al.Record(AuditEntry{PluginName: "p", Action: "config.updated", Severity: "warning"})
	al.Record(AuditEntry{PluginName: "p", Action: "plugin.error", Severity: "critical"})

	results := al.Query(AuditFilter{Severity: "critical"})
	if len(results) != 2 {
		t.Fatalf("expected 2 critical entries, got %d", len(results))
	}
	for _, r := range results {
		if r.Severity != "critical" {
			t.Errorf("expected all results to be critical, got %q", r.Severity)
		}
	}
}

func TestAuditLog_QuerySince(t *testing.T) {
	al := NewAuditLog(100)

	baseTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	al.Record(AuditEntry{Timestamp: baseTime, PluginName: "p", Action: "plugin.started", Severity: "info"})
	al.Record(AuditEntry{Timestamp: baseTime.Add(1 * time.Hour), PluginName: "p", Action: "plugin.started", Severity: "info"})
	al.Record(AuditEntry{Timestamp: baseTime.Add(2 * time.Hour), PluginName: "p", Action: "plugin.started", Severity: "info"})

	// Query since 30 minutes into the first hour.
	since := baseTime.Add(30 * time.Minute)
	results := al.Query(AuditFilter{Since: since})
	if len(results) != 2 {
		t.Fatalf("expected 2 entries since %v, got %d", since, len(results))
	}
}

func TestAuditLog_QueryCombinedFilters(t *testing.T) {
	al := NewAuditLog(100)

	al.Record(AuditEntry{PluginName: "plugin-a", Action: "plugin.started", Severity: "info"})
	al.Record(AuditEntry{PluginName: "plugin-a", Action: "plugin.error", Severity: "critical"})
	al.Record(AuditEntry{PluginName: "plugin-b", Action: "plugin.error", Severity: "critical"})

	// Query for plugin-a critical errors.
	results := al.Query(AuditFilter{
		PluginName: "plugin-a",
		Severity:   "critical",
	})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Action != "plugin.error" {
		t.Errorf("Action = %q, want %q", results[0].Action, "plugin.error")
	}
}

func TestAuditLog_QueryNoResults(t *testing.T) {
	al := NewAuditLog(100)

	al.Record(AuditEntry{PluginName: "p", Action: "plugin.started", Severity: "info"})

	results := al.Query(AuditFilter{PluginName: "nonexistent"})
	if len(results) != 0 {
		t.Errorf("expected 0 results for nonexistent plugin, got %d", len(results))
	}
}

func TestAuditLog_Export(t *testing.T) {
	al := NewAuditLog(100)

	al.Record(AuditEntry{PluginName: "a", Action: "plugin.started", Severity: "info", Actor: "admin"})
	al.Record(AuditEntry{PluginName: "b", Action: "plugin.stopped", Severity: "info", Actor: "admin"})

	entries := al.Export()
	if len(entries) != 2 {
		t.Fatalf("Export() returned %d entries, want 2", len(entries))
	}

	// Should be in chronological order.
	if entries[0].PluginName != "a" {
		t.Errorf("first entry PluginName = %q, want %q", entries[0].PluginName, "a")
	}
	if entries[1].PluginName != "b" {
		t.Errorf("second entry PluginName = %q, want %q", entries[1].PluginName, "b")
	}
}

func TestAuditLog_ExportEmpty(t *testing.T) {
	al := NewAuditLog(100)

	entries := al.Export()
	if entries != nil {
		t.Errorf("Export() on empty log should return nil, got %v", entries)
	}
}

func TestAuditHelpers(t *testing.T) {
	al := NewAuditLog(100)

	LogPermissionGrant(al, "my-plugin", "items:read", "admin")
	LogPermissionRevoke(al, "my-plugin", "items:write", "admin")
	LogConfigChange(al, "my-plugin", "api_url", "old", "new", "admin")
	LogPluginStateChange(al, "my-plugin", "disabled", "admin")

	if al.Len() != 4 {
		t.Fatalf("Len() = %d, want 4", al.Len())
	}

	entries := al.Export()

	// Verify each helper created the correct entry.
	if entries[0].Action != "permission.granted" {
		t.Errorf("entry[0] Action = %q, want %q", entries[0].Action, "permission.granted")
	}
	if entries[0].Severity != "info" {
		t.Errorf("entry[0] Severity = %q, want %q", entries[0].Severity, "info")
	}
	if entries[0].Actor != "admin" {
		t.Errorf("entry[0] Actor = %q, want %q", entries[0].Actor, "admin")
	}

	if entries[1].Action != "permission.revoked" {
		t.Errorf("entry[1] Action = %q, want %q", entries[1].Action, "permission.revoked")
	}
	if entries[1].Severity != "warning" {
		t.Errorf("entry[1] Severity = %q, want %q", entries[1].Severity, "warning")
	}

	if entries[2].Action != "config.updated" {
		t.Errorf("entry[2] Action = %q, want %q", entries[2].Action, "config.updated")
	}

	if entries[3].Action != "plugin.disabled" {
		t.Errorf("entry[3] Action = %q, want %q", entries[3].Action, "plugin.disabled")
	}
	if entries[3].Severity != "warning" {
		t.Errorf("entry[3] Severity = %q, want %q", entries[3].Severity, "warning")
	}
}

func TestJSONExporter(t *testing.T) {
	exporter := &JSONExporter{}

	entries := []AuditEntry{
		{
			Timestamp:  time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC),
			PluginName: "my-plugin",
			Action:     "plugin.started",
			Severity:   "info",
			Actor:      "user-1",
		},
		{
			Timestamp:  time.Date(2026, 1, 15, 11, 0, 0, 0, time.UTC),
			PluginName: "my-plugin",
			Action:     "plugin.error",
			Severity:   "critical",
			Details:    map[string]string{"error": "stack trace here"},
		},
	}

	data, err := exporter.Export(entries)
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	// Verify it's valid JSON.
	var parsed []AuditEntry
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to parse exported JSON: %v", err)
	}

	if len(parsed) != 2 {
		t.Fatalf("expected 2 entries in JSON, got %d", len(parsed))
	}

	if parsed[0].PluginName != "my-plugin" {
		t.Errorf("parsed[0].PluginName = %q, want %q", parsed[0].PluginName, "my-plugin")
	}
	if parsed[1].Action != "plugin.error" {
		t.Errorf("parsed[1].Action = %q, want %q", parsed[1].Action, "plugin.error")
	}
}

func TestJSONExporter_Empty(t *testing.T) {
	exporter := &JSONExporter{}

	data, err := exporter.Export([]AuditEntry{})
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	var parsed []AuditEntry
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to parse empty JSON: %v", err)
	}

	if len(parsed) != 0 {
		t.Errorf("expected 0 entries, got %d", len(parsed))
	}
}

func TestCSVExporter(t *testing.T) {
	exporter := &CSVExporter{}

	entries := []AuditEntry{
		{
			ID:         "id-1",
			Timestamp:  time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC),
			PluginName: "plugin-a",
			Action:     "plugin.started",
			Severity:   "info",
			Actor:      "admin",
		},
		{
			ID:         "id-2",
			Timestamp:  time.Date(2026, 1, 15, 11, 0, 0, 0, time.UTC),
			PluginName: "plugin-b",
			Action:     "plugin.error",
			Severity:   "critical",
			Actor:      "system",
			Details:    map[string]string{"reason": "timeout"},
		},
	}

	data, err := exporter.Export(entries)
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	// Parse the CSV.
	reader := csv.NewReader(strings.NewReader(string(data)))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("failed to parse CSV: %v", err)
	}

	// Header + 2 data rows = 3 total.
	if len(records) != 3 {
		t.Fatalf("expected 3 CSV rows (header + 2 data), got %d", len(records))
	}

	// Verify header matches actual CSV exporter header.
	header := records[0]
	expectedHeader := []string{"id", "timestamp", "plugin_name", "action", "actor", "severity", "details"}
	if len(header) != len(expectedHeader) {
		t.Fatalf("header length = %d, want %d", len(header), len(expectedHeader))
	}
	for i, h := range expectedHeader {
		if header[i] != h {
			t.Errorf("header[%d] = %q, want %q", i, header[i], h)
		}
	}

	// Verify first data row.
	row1 := records[1]
	if row1[2] != "plugin-a" {
		t.Errorf("row1 plugin_name = %q, want %q", row1[2], "plugin-a")
	}
	if row1[3] != "plugin.started" {
		t.Errorf("row1 action = %q, want %q", row1[3], "plugin.started")
	}
	if row1[4] != "admin" {
		t.Errorf("row1 actor = %q, want %q", row1[4], "admin")
	}
}

func TestCSVExporter_Empty(t *testing.T) {
	exporter := &CSVExporter{}

	data, err := exporter.Export([]AuditEntry{})
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	// Should still have a header row.
	reader := csv.NewReader(strings.NewReader(string(data)))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("failed to parse empty CSV: %v", err)
	}

	if len(records) != 1 {
		t.Errorf("expected 1 CSV row (header only), got %d", len(records))
	}
}

func TestAuditLog_DefaultMaxEntries(t *testing.T) {
	// Zero maxEntries should default to 10000.
	al := NewAuditLog(0)

	// Record 10001 entries.
	for i := 0; i < 10001; i++ {
		al.Record(AuditEntry{
			PluginName: "test",
			Action:     "api.access",
			Severity:   "info",
		})
	}

	if al.Len() != 10000 {
		t.Errorf("Len() = %d, want 10000 (default max)", al.Len())
	}
}
