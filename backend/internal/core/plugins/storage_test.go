package plugins

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestMemoryStore_SetAndGet(t *testing.T) {
	store := NewMemoryStore("test-plugin", 100, 1024)
	ctx := context.Background()

	// Set a value.
	if err := store.Set(ctx, "api_key", "secret123"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Get it back.
	val, err := store.Get(ctx, "api_key")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if val != "secret123" {
		t.Errorf("Get() = %q, want %q", val, "secret123")
	}

	// Overwrite.
	if err := store.Set(ctx, "api_key", "updated"); err != nil {
		t.Fatalf("Set() overwrite error = %v", err)
	}
	val, err = store.Get(ctx, "api_key")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if val != "updated" {
		t.Errorf("Get() after overwrite = %q, want %q", val, "updated")
	}

	// Get nonexistent key returns empty string and no error.
	val, err = store.Get(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Get() nonexistent error = %v", err)
	}
	if val != "" {
		t.Errorf("Get() nonexistent = %q, want empty string", val)
	}
}

func TestMemoryStore_Delete(t *testing.T) {
	store := NewMemoryStore("test", 100, 1024)
	ctx := context.Background()

	store.Set(ctx, "key1", "value1")
	store.Set(ctx, "key2", "value2")

	store.Delete(ctx, "key1")

	val, _ := store.Get(ctx, "key1")
	if val != "" {
		t.Error("expected key1 to be deleted (empty string)")
	}

	val, _ = store.Get(ctx, "key2")
	if val != "value2" {
		t.Error("key2 should still exist after deleting key1")
	}

	// Deleting nonexistent key should not error.
	if err := store.Delete(ctx, "nonexistent"); err != nil {
		t.Errorf("Delete() nonexistent error = %v", err)
	}
}

func TestMemoryStore_List(t *testing.T) {
	store := NewMemoryStore("test", 100, 1024)
	ctx := context.Background()

	store.Set(ctx, "config:api_url", "http://example.com")
	store.Set(ctx, "config:api_key", "secret")
	store.Set(ctx, "config:timeout", "30")
	store.Set(ctx, "cache:item1", "data1")
	store.Set(ctx, "cache:item2", "data2")

	// List by "config:" prefix.
	configKVs, err := store.List(ctx, "config:")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(configKVs) != 3 {
		t.Fatalf("expected 3 config keys, got %d", len(configKVs))
	}

	// List by "cache:" prefix.
	cacheKVs, err := store.List(ctx, "cache:")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(cacheKVs) != 2 {
		t.Fatalf("expected 2 cache keys, got %d", len(cacheKVs))
	}

	// List with empty prefix returns all keys.
	allKVs, err := store.List(ctx, "")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(allKVs) != 5 {
		t.Fatalf("expected 5 total keys, got %d", len(allKVs))
	}

	// List with prefix matching nothing.
	emptyKVs, err := store.List(ctx, "unknown:")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(emptyKVs) != 0 {
		t.Errorf("expected 0 keys for unknown prefix, got %d", len(emptyKVs))
	}
}

func TestMemoryStore_Clear(t *testing.T) {
	store := NewMemoryStore("test", 100, 1024)
	ctx := context.Background()

	store.Set(ctx, "a", "1")
	store.Set(ctx, "b", "2")
	store.Set(ctx, "c", "3")

	store.Clear(ctx)

	allKVs, _ := store.List(ctx, "")
	if len(allKVs) != 0 {
		t.Errorf("expected 0 keys after Clear(), got %d", len(allKVs))
	}

	val, _ := store.Get(ctx, "a")
	if val != "" {
		t.Error("expected key to not exist after Clear()")
	}
}

func TestMemoryStore_Isolation(t *testing.T) {
	storeA := NewMemoryStore("plugin-a", 100, 1024)
	storeB := NewMemoryStore("plugin-b", 100, 1024)
	ctx := context.Background()

	storeA.Set(ctx, "shared_key", "value-from-a")
	storeB.Set(ctx, "shared_key", "value-from-b")

	valA, _ := storeA.Get(ctx, "shared_key")
	if valA != "value-from-a" {
		t.Errorf("storeA.Get() = %q, want %q", valA, "value-from-a")
	}

	valB, _ := storeB.Get(ctx, "shared_key")
	if valB != "value-from-b" {
		t.Errorf("storeB.Get() = %q, want %q", valB, "value-from-b")
	}

	// Clearing one store should not affect the other.
	storeA.Clear(ctx)

	valA, _ = storeA.Get(ctx, "shared_key")
	if valA != "" {
		t.Error("storeA should be empty after Clear()")
	}

	valB, _ = storeB.Get(ctx, "shared_key")
	if valB != "value-from-b" {
		t.Error("storeB should not be affected by storeA.Clear()")
	}
}

func TestMemoryStore_MaxKeys(t *testing.T) {
	store := NewMemoryStore("limited", 3, 1024)
	ctx := context.Background()

	if err := store.Set(ctx, "k1", "v1"); err != nil {
		t.Fatalf("Set(k1) error = %v", err)
	}
	if err := store.Set(ctx, "k2", "v2"); err != nil {
		t.Fatalf("Set(k2) error = %v", err)
	}
	if err := store.Set(ctx, "k3", "v3"); err != nil {
		t.Fatalf("Set(k3) error = %v", err)
	}

	// Fourth key should fail.
	err := store.Set(ctx, "k4", "v4")
	if err == nil {
		t.Error("expected error when exceeding max keys")
	}
	if !strings.Contains(err.Error(), "maximum") {
		t.Errorf("error should mention maximum, got: %v", err)
	}

	// Overwriting an existing key should succeed (not a new key).
	if err := store.Set(ctx, "k1", "v1-updated"); err != nil {
		t.Fatalf("overwriting existing key should succeed, got error: %v", err)
	}

	// Deleting a key and adding a new one should succeed.
	store.Delete(ctx, "k2")
	if err := store.Set(ctx, "k4", "v4"); err != nil {
		t.Fatalf("Set(k4) after delete should succeed, got error: %v", err)
	}
}

func TestMemoryStore_MaxValueSize(t *testing.T) {
	store := NewMemoryStore("limited", 100, 10)
	ctx := context.Background()

	// Short value should succeed.
	if err := store.Set(ctx, "short", "hello"); err != nil {
		t.Fatalf("Set() with short value error = %v", err)
	}

	// Exact limit should succeed.
	if err := store.Set(ctx, "exact", "1234567890"); err != nil {
		t.Fatalf("Set() at exact limit error = %v", err)
	}

	// One byte over should fail.
	err := store.Set(ctx, "toolong", "12345678901")
	if err == nil {
		t.Error("expected error when exceeding max value size")
	}
	if !strings.Contains(err.Error(), "exceeds") {
		t.Errorf("error should mention exceeds, got: %v", err)
	}
}

func TestStoreManager_GetStore(t *testing.T) {
	sm := NewStoreManager(100, 1024)

	store := sm.GetStore("my-plugin")
	if store == nil {
		t.Fatal("expected non-nil store")
	}

	// Getting the same store again should return the same instance.
	store2 := sm.GetStore("my-plugin")
	if store != store2 {
		t.Error("expected GetStore to return the same instance for the same plugin")
	}

	// Different plugin should get a different store.
	store3 := sm.GetStore("other-plugin")
	if store3 == store {
		t.Error("different plugins should get different stores")
	}
}

func TestStoreManager_DeleteStore(t *testing.T) {
	sm := NewStoreManager(100, 1024)
	ctx := context.Background()

	store := sm.GetStore("doomed")
	store.Set(ctx, "key", "value")

	sm.DeleteStore("doomed")

	// Getting the store again should create a new, empty one.
	newStore := sm.GetStore("doomed")
	val, _ := newStore.Get(ctx, "key")
	if val != "" {
		t.Error("new store should not contain old data")
	}
}

func TestStoreManager_DeleteStore_Nonexistent(t *testing.T) {
	sm := NewStoreManager(0, 0)

	// Deleting a nonexistent store should not panic.
	sm.DeleteStore("nonexistent")
}

func TestStoreManager_GetUsage(t *testing.T) {
	sm := NewStoreManager(100, 1024)
	ctx := context.Background()

	store := sm.GetStore("usage-test")
	store.Set(ctx, "k1", "hello")
	store.Set(ctx, "k2", "world")

	usage := sm.GetUsage("usage-test")
	if usage.KeyCount != 2 {
		t.Errorf("KeyCount = %d, want 2", usage.KeyCount)
	}
	if usage.MaxKeys != 100 {
		t.Errorf("MaxKeys = %d, want 100", usage.MaxKeys)
	}

	// Usage for nonexistent plugin.
	noUsage := sm.GetUsage("nonexistent")
	if noUsage.KeyCount != 0 {
		t.Errorf("KeyCount for nonexistent = %d, want 0", noUsage.KeyCount)
	}
}

func TestStoreManager_ConcurrentAccess(t *testing.T) {
	sm := NewStoreManager(1000, 1024)
	ctx := context.Background()

	var wg sync.WaitGroup
	const goroutines = 50

	// Multiple goroutines accessing different stores.
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			name := "plugin-" + string(rune('a'+idx%5))
			store := sm.GetStore(name)
			store.Set(ctx, "counter", "value")
			store.Get(ctx, "counter")
			store.List(ctx, "")
		}(i)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// success
	case <-time.After(5 * time.Second):
		t.Fatal("concurrent store access timed out (possible deadlock)")
	}
}

func TestMemoryStore_EmptyKeyError(t *testing.T) {
	store := NewMemoryStore("test", 100, 1024)
	ctx := context.Background()

	// Get with empty key should error.
	_, err := store.Get(ctx, "")
	if err == nil {
		t.Error("expected error for empty key on Get")
	}

	// Set with empty key should error.
	err = store.Set(ctx, "", "value")
	if err == nil {
		t.Error("expected error for empty key on Set")
	}

	// Delete with empty key should error.
	err = store.Delete(ctx, "")
	if err == nil {
		t.Error("expected error for empty key on Delete")
	}
}

func TestMemoryStore_DefaultLimits(t *testing.T) {
	// Creating a store with zero values should use defaults.
	store := NewMemoryStore("test", 0, 0)
	ctx := context.Background()

	// Should be able to set at least one key.
	if err := store.Set(ctx, "key", "value"); err != nil {
		t.Fatalf("Set() with default limits error = %v", err)
	}

	val, _ := store.Get(ctx, "key")
	if val != "value" {
		t.Errorf("Get() = %q, want %q", val, "value")
	}
}

func TestMemoryStore_Usage(t *testing.T) {
	store := NewMemoryStore("test", 100, 1024)
	ctx := context.Background()

	store.Set(ctx, "k1", "hello") // 5 bytes
	store.Set(ctx, "k2", "world") // 5 bytes

	usage := store.Usage()
	if usage.KeyCount != 2 {
		t.Errorf("KeyCount = %d, want 2", usage.KeyCount)
	}
	if usage.TotalSizeBytes != 10 {
		t.Errorf("TotalSizeBytes = %d, want 10", usage.TotalSizeBytes)
	}
	if usage.MaxKeys != 100 {
		t.Errorf("MaxKeys = %d, want 100", usage.MaxKeys)
	}
}
