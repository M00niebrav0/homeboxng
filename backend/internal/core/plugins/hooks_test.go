package plugins

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestHookRegistry_Register(t *testing.T) {
	hr := NewHookRegistry()

	hr.Register("my-plugin", BeforeItemCreate, func(ctx context.Context, hc *HookContext) error {
		return nil
	})

	registered := hr.GetRegistered()
	plugins, ok := registered[BeforeItemCreate]
	if !ok {
		t.Fatal("expected BeforeItemCreate to have registered handlers")
	}
	if len(plugins) != 1 {
		t.Fatalf("expected 1 handler, got %d", len(plugins))
	}
	if plugins[0] != "my-plugin" {
		t.Errorf("plugin name = %q, want %q", plugins[0], "my-plugin")
	}
}

func TestHookRegistry_Execute(t *testing.T) {
	hr := NewHookRegistry()

	var receivedEntityID string
	var receivedHookPoint HookPoint

	hr.Register("checker", AfterItemCreate, func(ctx context.Context, hc *HookContext) error {
		receivedEntityID = hc.EntityID
		receivedHookPoint = hc.HookPoint
		return nil
	})

	hc := NewHookContext(AfterItemCreate, "item-123", "item", "user-1", "group-1")
	hc.Data["itemName"] = "Test Item"
	hc.Data["quantity"] = 5

	err := hr.Execute(context.Background(), AfterItemCreate, hc)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if receivedEntityID != "item-123" {
		t.Errorf("EntityID = %q, want %q", receivedEntityID, "item-123")
	}
	if receivedHookPoint != AfterItemCreate {
		t.Errorf("HookPoint = %q, want %q", receivedHookPoint, AfterItemCreate)
	}
}

func TestHookRegistry_ExecuteOrder(t *testing.T) {
	hr := NewHookRegistry()

	var order []string

	hr.Register("first-plugin", BeforeItemUpdate, func(ctx context.Context, hc *HookContext) error {
		order = append(order, "first")
		return nil
	})
	hr.Register("second-plugin", BeforeItemUpdate, func(ctx context.Context, hc *HookContext) error {
		order = append(order, "second")
		return nil
	})
	hr.Register("third-plugin", BeforeItemUpdate, func(ctx context.Context, hc *HookContext) error {
		order = append(order, "third")
		return nil
	})

	hc := NewHookContext(BeforeItemUpdate, "", "item", "", "")
	err := hr.Execute(context.Background(), BeforeItemUpdate, hc)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if len(order) != 3 {
		t.Fatalf("expected 3 handlers called, got %d", len(order))
	}

	expected := []string{"first", "second", "third"}
	for i, name := range order {
		if name != expected[i] {
			t.Errorf("order[%d] = %q, want %q", i, name, expected[i])
		}
	}
}

func TestHookRegistry_Cancel(t *testing.T) {
	hr := NewHookRegistry()

	var secondCalled bool

	hr.Register("validator", BeforeItemDelete, func(ctx context.Context, hc *HookContext) error {
		hc.Cancel()
		return nil
	})
	hr.Register("logger", BeforeItemDelete, func(ctx context.Context, hc *HookContext) error {
		secondCalled = true
		return nil
	})

	hc := NewHookContext(BeforeItemDelete, "item-1", "item", "", "")
	err := hr.Execute(context.Background(), BeforeItemDelete, hc)
	if err == nil {
		t.Fatal("expected error from cancelled hook")
	}

	if secondCalled {
		t.Error("second handler should not have been called after cancellation")
	}

	// Verify it's a HookCancelledError.
	if !IsHookCancelled(err) {
		t.Errorf("expected HookCancelledError, got: %v", err)
	}

	// Verify the error message contains the plugin name.
	errMsg := err.Error()
	if !strings.Contains(errMsg, "validator") {
		t.Errorf("error should mention the cancelling plugin, got: %s", errMsg)
	}
}

func TestHookRegistry_ErrorPropagation(t *testing.T) {
	hr := NewHookRegistry()

	var secondCalled bool
	expectedErr := errors.New("database connection lost")

	hr.Register("faulty", AfterItemUpdate, func(ctx context.Context, hc *HookContext) error {
		return expectedErr
	})
	hr.Register("observer", AfterItemUpdate, func(ctx context.Context, hc *HookContext) error {
		secondCalled = true
		return nil
	})

	hc := NewHookContext(AfterItemUpdate, "", "item", "", "")
	err := hr.Execute(context.Background(), AfterItemUpdate, hc)
	if err == nil {
		t.Fatal("expected error from failed handler")
	}

	if !errors.Is(err, expectedErr) {
		t.Errorf("error should wrap the original error, got: %v", err)
	}

	if secondCalled {
		t.Error("second handler should not have been called after error")
	}
}

func TestHookRegistry_MultiplePlugins(t *testing.T) {
	hr := NewHookRegistry()

	var called []string

	// Multiple plugins registering for the same hook point.
	hr.Register("plugin-a", AfterItemCreate, func(ctx context.Context, hc *HookContext) error {
		called = append(called, "a")
		return nil
	})
	hr.Register("plugin-b", AfterItemCreate, func(ctx context.Context, hc *HookContext) error {
		called = append(called, "b")
		return nil
	})
	hr.Register("plugin-c", AfterItemCreate, func(ctx context.Context, hc *HookContext) error {
		called = append(called, "c")
		return nil
	})

	// Also register plugin-a for a different hook point.
	hr.Register("plugin-a", AfterItemDelete, func(ctx context.Context, hc *HookContext) error {
		called = append(called, "a-delete")
		return nil
	})

	hc := NewHookContext(AfterItemCreate, "", "item", "", "")
	err := hr.Execute(context.Background(), AfterItemCreate, hc)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if len(called) != 3 {
		t.Fatalf("expected 3 handlers called, got %d", len(called))
	}

	// Verify the correct handlers were called.
	expected := []string{"a", "b", "c"}
	for i, name := range called {
		if name != expected[i] {
			t.Errorf("called[%d] = %q, want %q", i, name, expected[i])
		}
	}
}

func TestHookRegistry_ExecuteNoHandlers(t *testing.T) {
	hr := NewHookRegistry()

	// Executing a hook with no handlers should succeed silently.
	hc := NewHookContext(AfterItemCreate, "", "item", "", "")
	err := hr.Execute(context.Background(), AfterItemCreate, hc)
	if err != nil {
		t.Errorf("Execute() with no handlers should not error, got: %v", err)
	}
}

func TestHookRegistry_ConcurrentExecution(t *testing.T) {
	hr := NewHookRegistry()

	var mu sync.Mutex
	count := 0

	// Register from multiple goroutines.
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			name := "plugin-" + string(rune('a'+idx%10))
			hr.Register(name, BeforeItemCreate, func(ctx context.Context, hc *HookContext) error {
				mu.Lock()
				count++
				mu.Unlock()
				return nil
			})
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
		t.Fatal("concurrent registration timed out (possible deadlock)")
	}

	// Execute to verify all handlers work.
	hc := NewHookContext(BeforeItemCreate, "", "item", "", "")
	err := hr.Execute(context.Background(), BeforeItemCreate, hc)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	mu.Lock()
	if count != 20 {
		t.Errorf("expected 20 handlers called, got %d", count)
	}
	mu.Unlock()
}

func TestHookRegistry_GetRegistered(t *testing.T) {
	hr := NewHookRegistry()

	hr.Register("alpha", BeforeItemCreate, func(ctx context.Context, hc *HookContext) error { return nil })
	hr.Register("beta", BeforeItemCreate, func(ctx context.Context, hc *HookContext) error { return nil })
	hr.Register("alpha", AfterItemCreate, func(ctx context.Context, hc *HookContext) error { return nil })
	hr.Register("gamma", AfterItemDelete, func(ctx context.Context, hc *HookContext) error { return nil })

	registered := hr.GetRegistered()

	// BeforeItemCreate should have 2 plugins.
	if len(registered[BeforeItemCreate]) != 2 {
		t.Errorf("BeforeItemCreate: expected 2 plugins, got %d", len(registered[BeforeItemCreate]))
	}

	// AfterItemCreate should have 1 plugin.
	if len(registered[AfterItemCreate]) != 1 {
		t.Errorf("AfterItemCreate: expected 1 plugin, got %d", len(registered[AfterItemCreate]))
	}

	// AfterItemDelete should have 1 plugin.
	if len(registered[AfterItemDelete]) != 1 {
		t.Errorf("AfterItemDelete: expected 1 plugin, got %d", len(registered[AfterItemDelete]))
	}

	// Verify specific plugin names.
	beforeCreate := registered[BeforeItemCreate]
	if beforeCreate[0] != "alpha" || beforeCreate[1] != "beta" {
		t.Errorf("BeforeItemCreate plugins = %v, want [alpha, beta]", beforeCreate)
	}
}

func TestHookRegistry_HandlerCount(t *testing.T) {
	hr := NewHookRegistry()

	hr.Register("a", BeforeItemCreate, func(ctx context.Context, hc *HookContext) error { return nil })
	hr.Register("b", BeforeItemCreate, func(ctx context.Context, hc *HookContext) error { return nil })
	hr.Register("a", AfterItemCreate, func(ctx context.Context, hc *HookContext) error { return nil })

	if hr.HandlerCount(BeforeItemCreate) != 2 {
		t.Errorf("HandlerCount(BeforeItemCreate) = %d, want 2", hr.HandlerCount(BeforeItemCreate))
	}
	if hr.HandlerCount(AfterItemCreate) != 1 {
		t.Errorf("HandlerCount(AfterItemCreate) = %d, want 1", hr.HandlerCount(AfterItemCreate))
	}
	if hr.HandlerCount(BeforeItemDelete) != 0 {
		t.Errorf("HandlerCount(BeforeItemDelete) = %d, want 0", hr.HandlerCount(BeforeItemDelete))
	}
}

func TestHookRegistry_Clear(t *testing.T) {
	hr := NewHookRegistry()

	hr.Register("a", BeforeItemCreate, func(ctx context.Context, hc *HookContext) error { return nil })
	hr.Register("b", AfterItemCreate, func(ctx context.Context, hc *HookContext) error { return nil })

	hr.Clear()

	if hr.HandlerCount(BeforeItemCreate) != 0 {
		t.Error("expected 0 handlers after Clear()")
	}
	if hr.HandlerCount(AfterItemCreate) != 0 {
		t.Error("expected 0 handlers after Clear()")
	}
}

func TestHookRegistry_ClearPlugin(t *testing.T) {
	hr := NewHookRegistry()

	hr.Register("keep", BeforeItemCreate, func(ctx context.Context, hc *HookContext) error { return nil })
	hr.Register("remove", BeforeItemCreate, func(ctx context.Context, hc *HookContext) error { return nil })
	hr.Register("remove", AfterItemCreate, func(ctx context.Context, hc *HookContext) error { return nil })

	hr.ClearPlugin("remove")

	if hr.HandlerCount(BeforeItemCreate) != 1 {
		t.Errorf("HandlerCount(BeforeItemCreate) = %d, want 1 after ClearPlugin", hr.HandlerCount(BeforeItemCreate))
	}
	if hr.HandlerCount(AfterItemCreate) != 0 {
		t.Errorf("HandlerCount(AfterItemCreate) = %d, want 0 after ClearPlugin", hr.HandlerCount(AfterItemCreate))
	}
}

func TestHookContext_Cancel(t *testing.T) {
	hc := NewHookContext(BeforeItemDelete, "entity-1", "item", "user-1", "group-1")

	if hc.Cancelled {
		t.Error("new context should not be cancelled")
	}

	hc.Cancel()

	if !hc.Cancelled {
		t.Error("context should be cancelled after Cancel()")
	}
}

func TestHookContext_Data(t *testing.T) {
	hc := NewHookContext(BeforeItemCreate, "item-1", "item", "user-1", "group-1")

	if hc.Data == nil {
		t.Fatal("Data map should be initialized")
	}

	hc.Data["key"] = "value"
	if hc.Data["key"] != "value" {
		t.Errorf("Data[key] = %v, want %q", hc.Data["key"], "value")
	}
}

func TestAllHookPoints(t *testing.T) {
	points := AllHookPoints()
	if len(points) != 12 {
		t.Errorf("AllHookPoints() returned %d points, want 12", len(points))
	}
}

func TestIsHookCancelled(t *testing.T) {
	cancelled := &HookCancelledError{PluginName: "test", HookPoint: BeforeItemCreate}
	if !IsHookCancelled(cancelled) {
		t.Error("IsHookCancelled should return true for HookCancelledError")
	}

	regular := errors.New("regular error")
	if IsHookCancelled(regular) {
		t.Error("IsHookCancelled should return false for regular error")
	}

	if IsHookCancelled(nil) {
		t.Error("IsHookCancelled should return false for nil")
	}
}
