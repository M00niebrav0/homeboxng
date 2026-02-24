package plugins

import (
	"testing"
)

func TestRingBuffer_Basic(t *testing.T) {
	rb := newRingBuffer(5)

	rb.Add(LogEntry{Level: "info", Message: "one"})
	rb.Add(LogEntry{Level: "info", Message: "two"})
	rb.Add(LogEntry{Level: "info", Message: "three"})

	entries := rb.Entries()
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	if entries[0].Message != "one" {
		t.Errorf("entries[0] = %q, want %q", entries[0].Message, "one")
	}
	if entries[2].Message != "three" {
		t.Errorf("entries[2] = %q, want %q", entries[2].Message, "three")
	}
}

func TestRingBuffer_Wraparound(t *testing.T) {
	rb := newRingBuffer(3)

	rb.Add(LogEntry{Message: "a"})
	rb.Add(LogEntry{Message: "b"})
	rb.Add(LogEntry{Message: "c"})
	rb.Add(LogEntry{Message: "d"}) // overwrites "a"
	rb.Add(LogEntry{Message: "e"}) // overwrites "b"

	entries := rb.Entries()
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries after wraparound, got %d", len(entries))
	}

	// Should contain c, d, e (oldest first)
	expected := []string{"c", "d", "e"}
	for i, msg := range expected {
		if entries[i].Message != msg {
			t.Errorf("entries[%d] = %q, want %q", i, entries[i].Message, msg)
		}
	}
}

func TestRingBuffer_Empty(t *testing.T) {
	rb := newRingBuffer(10)

	entries := rb.Entries()
	if entries != nil {
		t.Errorf("expected nil entries for empty buffer, got %v", entries)
	}
}

func TestRingBuffer_Clear(t *testing.T) {
	rb := newRingBuffer(5)
	rb.Add(LogEntry{Message: "test"})
	rb.Clear()

	entries := rb.Entries()
	if entries != nil {
		t.Error("expected nil entries after Clear()")
	}
}

func TestRingBuffer_SingleElement(t *testing.T) {
	rb := newRingBuffer(1)

	rb.Add(LogEntry{Message: "first"})
	entries := rb.Entries()
	if len(entries) != 1 || entries[0].Message != "first" {
		t.Error("single element buffer should hold one entry")
	}

	rb.Add(LogEntry{Message: "second"})
	entries = rb.Entries()
	if len(entries) != 1 || entries[0].Message != "second" {
		t.Error("single element buffer should overwrite on add")
	}
}

func TestPluginLogCollector_Log(t *testing.T) {
	c := NewPluginLogCollector(100)

	c.Log("plugin-a", "info", "hello")
	c.Log("plugin-a", "error", "oops")
	c.Log("plugin-b", "debug", "debug msg")

	logsA := c.GetLogs("plugin-a")
	if len(logsA) != 2 {
		t.Fatalf("expected 2 logs for plugin-a, got %d", len(logsA))
	}
	if logsA[0].Level != "info" || logsA[0].Message != "hello" {
		t.Errorf("unexpected log entry: %+v", logsA[0])
	}

	logsB := c.GetLogs("plugin-b")
	if len(logsB) != 1 {
		t.Fatalf("expected 1 log for plugin-b, got %d", len(logsB))
	}
}

func TestPluginLogCollector_GetLogs_Unknown(t *testing.T) {
	c := NewPluginLogCollector(100)

	logs := c.GetLogs("nonexistent")
	if logs != nil {
		t.Error("expected nil logs for unknown plugin")
	}
}

func TestPluginLogCollector_ClearLogs(t *testing.T) {
	c := NewPluginLogCollector(100)
	c.Log("test", "info", "msg")
	c.ClearLogs("test")

	logs := c.GetLogs("test")
	if logs != nil {
		t.Error("expected nil logs after clear")
	}
}

func TestPluginLogCollector_AllLogs(t *testing.T) {
	c := NewPluginLogCollector(100)
	c.Log("a", "info", "one")
	c.Log("b", "info", "two")

	all := c.AllLogs()
	if len(all) != 2 {
		t.Fatalf("expected 2 plugins in AllLogs, got %d", len(all))
	}
}

func TestPluginLogCollector_LogStats(t *testing.T) {
	c := NewPluginLogCollector(100)
	c.Log("a", "info", "msg1")
	c.Log("a", "info", "msg2")
	c.Log("b", "info", "msg3")

	stats := c.LogStats()
	if stats["a"] != 2 {
		t.Errorf("plugin-a count = %d, want 2", stats["a"])
	}
	if stats["b"] != 1 {
		t.Errorf("plugin-b count = %d, want 1", stats["b"])
	}
}

func TestFormatEntry(t *testing.T) {
	entry := LogEntry{
		Level:   "error",
		Message: "something broke",
		Plugin:  "test",
	}

	formatted := FormatEntry(entry)
	if formatted == "" {
		t.Error("expected non-empty formatted string")
	}

	// Should contain the level, plugin name, and message
	if !containsAll(formatted, "error", "test", "something broke") {
		t.Errorf("formatted output missing expected parts: %s", formatted)
	}
}

func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		found := false
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
