package plugins

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

const (
	// DefaultMaxKeysPerPlugin is the default maximum number of keys a plugin can store.
	DefaultMaxKeysPerPlugin = 1000
	// DefaultMaxValueSize is the default maximum size of a single value in bytes.
	DefaultMaxValueSize = 64 * 1024 // 64 KB
)

// PluginStore provides isolated key-value storage for a single plugin.
type PluginStore interface {
	// Get retrieves a value by key. Returns empty string and nil error if the key does not exist.
	Get(ctx context.Context, key string) (string, error)

	// Set stores a value under the given key, creating or overwriting as needed.
	Set(ctx context.Context, key string, value string) error

	// Delete removes a key. No error is returned if the key does not exist.
	Delete(ctx context.Context, key string) error

	// List returns all key-value pairs whose keys start with the given prefix.
	// An empty prefix returns all keys.
	List(ctx context.Context, prefix string) (map[string]string, error)

	// Clear removes all keys belonging to this plugin.
	Clear(ctx context.Context) error
}

// StoreUsage contains statistics about a plugin's storage consumption.
type StoreUsage struct {
	// KeyCount is the number of keys currently stored.
	KeyCount int `json:"keyCount"`
	// TotalSizeBytes is the total size of all stored values in bytes.
	TotalSizeBytes int64 `json:"totalSizeBytes"`
	// MaxKeys is the maximum number of keys allowed.
	MaxKeys int `json:"maxKeys"`
}

// MemoryStore is an in-memory implementation of PluginStore.
// Each plugin's keys are isolated via a namespace prefix.
// It enforces configurable limits on key count and value size.
type MemoryStore struct {
	mu        sync.RWMutex
	data      map[string]string
	namespace string
	maxKeys   int
	maxValue  int
}

// NewMemoryStore creates a MemoryStore for the given plugin name.
// The namespace prefix is "plugin:{name}:" to ensure isolation.
func NewMemoryStore(pluginName string, maxKeys int, maxValueSize int) *MemoryStore {
	if maxKeys <= 0 {
		maxKeys = DefaultMaxKeysPerPlugin
	}
	if maxValueSize <= 0 {
		maxValueSize = DefaultMaxValueSize
	}
	return &MemoryStore{
		data:      make(map[string]string),
		namespace: "plugin:" + pluginName + ":",
		maxKeys:   maxKeys,
		maxValue:  maxValueSize,
	}
}

// nsKey returns the full namespaced key.
func (ms *MemoryStore) nsKey(key string) string {
	return ms.namespace + key
}

// stripNS removes the namespace prefix from a full key, returning the user-visible key.
func (ms *MemoryStore) stripNS(fullKey string) string {
	return strings.TrimPrefix(fullKey, ms.namespace)
}

// Get retrieves a value by key.
func (ms *MemoryStore) Get(_ context.Context, key string) (string, error) {
	if key == "" {
		return "", fmt.Errorf("key must not be empty")
	}

	ms.mu.RLock()
	defer ms.mu.RUnlock()

	val, ok := ms.data[ms.nsKey(key)]
	if !ok {
		return "", nil
	}
	return val, nil
}

// Set stores a value. It enforces max keys and max value size limits.
func (ms *MemoryStore) Set(_ context.Context, key string, value string) error {
	if key == "" {
		return fmt.Errorf("key must not be empty")
	}
	if len(value) > ms.maxValue {
		return fmt.Errorf("value size %d bytes exceeds maximum %d bytes", len(value), ms.maxValue)
	}

	fullKey := ms.nsKey(key)

	ms.mu.Lock()
	defer ms.mu.Unlock()

	// If this is a new key, check the limit.
	if _, exists := ms.data[fullKey]; !exists {
		count := ms.keyCount()
		if count >= ms.maxKeys {
			return fmt.Errorf("maximum key count %d reached for plugin store", ms.maxKeys)
		}
	}

	ms.data[fullKey] = value
	return nil
}

// Delete removes a key from the store.
func (ms *MemoryStore) Delete(_ context.Context, key string) error {
	if key == "" {
		return fmt.Errorf("key must not be empty")
	}

	ms.mu.Lock()
	defer ms.mu.Unlock()

	delete(ms.data, ms.nsKey(key))
	return nil
}

// List returns all key-value pairs whose keys match the given prefix.
// The returned keys have the namespace stripped so callers see only their own keys.
func (ms *MemoryStore) List(_ context.Context, prefix string) (map[string]string, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	fullPrefix := ms.namespace + prefix
	result := make(map[string]string)

	for k, v := range ms.data {
		if strings.HasPrefix(k, fullPrefix) {
			result[ms.stripNS(k)] = v
		}
	}

	return result, nil
}

// Clear removes all keys belonging to this plugin's namespace.
func (ms *MemoryStore) Clear(_ context.Context) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	for k := range ms.data {
		if strings.HasPrefix(k, ms.namespace) {
			delete(ms.data, k)
		}
	}
	return nil
}

// Usage returns the current storage usage statistics for this store.
func (ms *MemoryStore) Usage() StoreUsage {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	var totalSize int64
	count := 0
	for k, v := range ms.data {
		if strings.HasPrefix(k, ms.namespace) {
			count++
			totalSize += int64(len(v))
		}
	}

	return StoreUsage{
		KeyCount:       count,
		TotalSizeBytes: totalSize,
		MaxKeys:        ms.maxKeys,
	}
}

// keyCount returns the number of keys in this plugin's namespace.
// Must be called while holding at least a read lock.
func (ms *MemoryStore) keyCount() int {
	count := 0
	for k := range ms.data {
		if strings.HasPrefix(k, ms.namespace) {
			count++
		}
	}
	return count
}

// StoreManager creates and tracks per-plugin PluginStore instances.
type StoreManager struct {
	mu       sync.RWMutex
	stores   map[string]*MemoryStore
	maxKeys  int
	maxValue int
}

// NewStoreManager creates a StoreManager with the given per-plugin limits.
// Zero or negative values use defaults.
func NewStoreManager(maxKeysPerPlugin int, maxValueSize int) *StoreManager {
	if maxKeysPerPlugin <= 0 {
		maxKeysPerPlugin = DefaultMaxKeysPerPlugin
	}
	if maxValueSize <= 0 {
		maxValueSize = DefaultMaxValueSize
	}
	return &StoreManager{
		stores:   make(map[string]*MemoryStore),
		maxKeys:  maxKeysPerPlugin,
		maxValue: maxValueSize,
	}
}

// GetStore returns the PluginStore for the named plugin, creating one if needed.
func (sm *StoreManager) GetStore(pluginName string) PluginStore {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	store, ok := sm.stores[pluginName]
	if !ok {
		store = NewMemoryStore(pluginName, sm.maxKeys, sm.maxValue)
		sm.stores[pluginName] = store
	}
	return store
}

// GetUsage returns storage usage statistics for a specific plugin.
// If the plugin has no store, returns zero usage with the configured max keys.
func (sm *StoreManager) GetUsage(pluginName string) StoreUsage {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	store, ok := sm.stores[pluginName]
	if !ok {
		return StoreUsage{
			KeyCount:       0,
			TotalSizeBytes: 0,
			MaxKeys:        sm.maxKeys,
		}
	}
	return store.Usage()
}

// DeleteStore removes a plugin's store and all its data.
func (sm *StoreManager) DeleteStore(pluginName string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.stores, pluginName)
}
