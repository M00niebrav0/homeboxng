package plugins

import (
	"context"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
	"github.com/sysadminsmedia/homebox/backend/internal/core/services/reporting/eventbus"
)

// testPlugin is a minimal plugin implementation for testing.
type testPlugin struct {
	info        PluginInfo
	initErr     error
	startErr    error
	stopErr     error
	started     bool
	stopped     bool
	initialized bool
}

func newTestPlugin(name string) *testPlugin {
	return &testPlugin{
		info: PluginInfo{
			Name:    name,
			Version: "1.0.0",
			Author:  "test",
			BuiltIn: true,
		},
	}
}

func (p *testPlugin) Info() PluginInfo                    { return p.info }
func (p *testPlugin) Init(_ PluginContext) error          { p.initialized = true; return p.initErr }
func (p *testPlugin) Start(_ context.Context) error       { p.started = true; return p.startErr }
func (p *testPlugin) Stop(_ context.Context) error        { p.stopped = true; return p.stopErr }

// testRoutePlugin adds route support.
type testRoutePlugin struct {
	testPlugin
	routesMounted bool
}

func (p *testRoutePlugin) Routes(r chi.Router) {
	p.routesMounted = true
}

// testEventPlugin adds event support.
type testEventPlugin struct {
	testPlugin
	subscribed bool
}

func (p *testEventPlugin) SubscribeEvents(_ *eventbus.EventBus) {
	p.subscribed = true
}

// testScheduledPlugin adds scheduled task support.
type testScheduledPlugin struct {
	testPlugin
	taskRan bool
}

func (p *testScheduledPlugin) Schedule() []ScheduledTask {
	return []ScheduledTask{
		{
			Name: "test-task",
			Fn:   func(_ context.Context) { p.taskRan = true },
		},
	}
}

// testConfigPlugin adds configuration support.
type testConfigPlugin struct {
	testPlugin
	configured bool
	configVals map[string]string
}

func (p *testConfigPlugin) ConfigSchema() []ConfigField {
	return []ConfigField{
		{Key: "api_url", Label: "API URL", Type: "string", Required: true},
		{Key: "enabled", Label: "Enabled", Type: "boolean", Default: "true"},
	}
}

func (p *testConfigPlugin) Configure(values map[string]string) error {
	p.configured = true
	p.configVals = values
	return nil
}

func newTestRegistry() *Registry {
	logger := zerolog.Nop()
	return NewRegistry(PluginContext{
		Logger: logger,
	})
}

func TestRegistry_Register(t *testing.T) {
	r := newTestRegistry()
	p := newTestPlugin("test-plugin")

	err := r.Register(p)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if !p.initialized {
		t.Error("expected plugin to be initialized")
	}
}

func TestRegistry_RegisterDuplicate(t *testing.T) {
	r := newTestRegistry()
	p1 := newTestPlugin("dup")
	p2 := newTestPlugin("dup")

	if err := r.Register(p1); err != nil {
		t.Fatalf("first Register() error = %v", err)
	}

	err := r.Register(p2)
	if err == nil {
		t.Error("expected error when registering duplicate plugin")
	}
}

func TestRegistry_RegisterInitError(t *testing.T) {
	r := newTestRegistry()
	p := newTestPlugin("fail-init")
	p.initErr = context.DeadlineExceeded

	err := r.Register(p)
	if err == nil {
		t.Error("expected error from Init failure")
	}

	// Plugin should still be in registry but in error state
	list := r.List()
	if len(list) != 1 {
		t.Fatalf("expected 1 plugin in list, got %d", len(list))
	}
	if list[0].State != StateError {
		t.Errorf("expected state Error, got %s", list[0].State)
	}
}

func TestRegistry_List(t *testing.T) {
	r := newTestRegistry()
	r.Register(newTestPlugin("alpha"))
	r.Register(newTestPlugin("beta"))
	r.Register(newTestPlugin("gamma"))

	list := r.List()
	if len(list) != 3 {
		t.Fatalf("expected 3 plugins, got %d", len(list))
	}

	// Verify registration order
	names := []string{list[0].Info.Name, list[1].Info.Name, list[2].Info.Name}
	expected := []string{"alpha", "beta", "gamma"}
	for i, n := range names {
		if n != expected[i] {
			t.Errorf("list[%d] = %q, want %q", i, n, expected[i])
		}
	}
}

func TestRegistry_Get(t *testing.T) {
	r := newTestRegistry()
	r.Register(newTestPlugin("findme"))

	p, ok := r.Get("findme")
	if !ok || p == nil {
		t.Fatal("expected to find plugin 'findme'")
	}
	if p.Info().Name != "findme" {
		t.Errorf("got name %q, want %q", p.Info().Name, "findme")
	}

	_, ok = r.Get("nonexistent")
	if ok {
		t.Error("expected Get to return false for nonexistent plugin")
	}
}

func TestRegistry_StartAllStopAll(t *testing.T) {
	r := newTestRegistry()
	p1 := newTestPlugin("a")
	p2 := newTestPlugin("b")
	r.Register(p1)
	r.Register(p2)

	ctx := context.Background()
	if err := r.StartAll(ctx); err != nil {
		t.Fatalf("StartAll() error = %v", err)
	}

	if !p1.started || !p2.started {
		t.Error("expected both plugins to be started")
	}

	r.StopAll(ctx)

	if !p1.stopped || !p2.stopped {
		t.Error("expected both plugins to be stopped")
	}
}

func TestRegistry_StartAll_SkipsErrored(t *testing.T) {
	r := newTestRegistry()
	bad := newTestPlugin("bad")
	bad.initErr = context.Canceled
	good := newTestPlugin("good")

	r.Register(bad) // Will be in error state
	r.Register(good)

	r.StartAll(context.Background())

	if bad.started {
		t.Error("errored plugin should not have been started")
	}
	if !good.started {
		t.Error("good plugin should have been started")
	}
}

func TestRegistry_StopAll_ReverseOrder(t *testing.T) {
	r := newTestRegistry()
	order := []string{}

	// Create plugins that track stop order
	for _, name := range []string{"first", "second", "third"} {
		p := &stopOrderPlugin{
			testPlugin: *newTestPlugin(name),
			stopOrder:  &order,
		}
		r.Register(p)
	}

	r.StartAll(context.Background())
	r.StopAll(context.Background())

	if len(order) != 3 {
		t.Fatalf("expected 3 stops, got %d", len(order))
	}
	// Should stop in reverse order
	expected := []string{"third", "second", "first"}
	for i, n := range order {
		if n != expected[i] {
			t.Errorf("stop order[%d] = %q, want %q", i, n, expected[i])
		}
	}
}

type stopOrderPlugin struct {
	testPlugin
	stopOrder *[]string
}

func (p *stopOrderPlugin) Stop(_ context.Context) error {
	*p.stopOrder = append(*p.stopOrder, p.info.Name)
	return nil
}

func TestRegistry_SubscribeAll(t *testing.T) {
	r := newTestRegistry()
	ep := &testEventPlugin{testPlugin: *newTestPlugin("events")}
	r.Register(ep)
	r.Register(newTestPlugin("no-events"))

	bus := eventbus.New()
	r.SubscribeAll(bus)

	if !ep.subscribed {
		t.Error("expected event plugin to be subscribed")
	}
}

func TestRegistry_GetScheduledTasks(t *testing.T) {
	r := newTestRegistry()
	sp := &testScheduledPlugin{testPlugin: *newTestPlugin("sched")}
	r.Register(sp)
	r.Register(newTestPlugin("no-schedule"))

	tasks := r.GetScheduledTasks()
	if len(tasks) != 1 {
		t.Fatalf("expected 1 scheduled task, got %d", len(tasks))
	}
	if tasks[0].Name != "sched/test-task" {
		t.Errorf("task name = %q, want %q", tasks[0].Name, "sched/test-task")
	}
}

func TestRegistry_ConfigurePlugin(t *testing.T) {
	r := newTestRegistry()
	cp := &testConfigPlugin{testPlugin: *newTestPlugin("configurable")}
	r.Register(cp)

	values := map[string]string{"api_url": "http://localhost:8080"}
	if err := r.ConfigurePlugin("configurable", values); err != nil {
		t.Fatalf("ConfigurePlugin() error = %v", err)
	}

	if !cp.configured {
		t.Error("expected plugin to be configured")
	}
	if cp.configVals["api_url"] != "http://localhost:8080" {
		t.Errorf("config value = %q, want %q", cp.configVals["api_url"], "http://localhost:8080")
	}
}

func TestRegistry_ConfigurePlugin_NotConfigurable(t *testing.T) {
	r := newTestRegistry()
	r.Register(newTestPlugin("basic"))

	err := r.ConfigurePlugin("basic", map[string]string{"key": "val"})
	if err == nil {
		t.Error("expected error configuring non-ConfigPlugin")
	}
}

func TestRegistry_ConfigurePlugin_NotFound(t *testing.T) {
	r := newTestRegistry()

	err := r.ConfigurePlugin("nonexistent", map[string]string{})
	if err == nil {
		t.Error("expected error for nonexistent plugin")
	}
}

func TestRegistry_GetConfigSchema(t *testing.T) {
	r := newTestRegistry()
	cp := &testConfigPlugin{testPlugin: *newTestPlugin("configurable")}
	r.Register(cp)

	schema, err := r.GetConfigSchema("configurable")
	if err != nil {
		t.Fatalf("GetConfigSchema() error = %v", err)
	}
	if len(schema) != 2 {
		t.Fatalf("expected 2 config fields, got %d", len(schema))
	}
	if schema[0].Key != "api_url" {
		t.Errorf("field[0].Key = %q, want %q", schema[0].Key, "api_url")
	}
}

func TestRegistry_Unregister(t *testing.T) {
	r := newTestRegistry()

	// External (non-built-in) plugin
	ext := newTestPlugin("external")
	ext.info.BuiltIn = false
	r.Register(ext)

	if err := r.Unregister("external"); err != nil {
		t.Fatalf("Unregister() error = %v", err)
	}

	_, ok := r.Get("external")
	if ok {
		t.Error("expected plugin to be removed after unregister")
	}

	list := r.List()
	if len(list) != 0 {
		t.Errorf("expected 0 plugins after unregister, got %d", len(list))
	}
}

func TestRegistry_Unregister_BuiltIn(t *testing.T) {
	r := newTestRegistry()
	r.Register(newTestPlugin("builtin"))

	err := r.Unregister("builtin")
	if err == nil {
		t.Error("expected error when unregistering built-in plugin")
	}
}
