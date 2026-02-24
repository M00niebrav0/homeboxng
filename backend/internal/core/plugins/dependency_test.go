package plugins

import (
	"context"
	"strings"
	"testing"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want int
	}{
		{
			name: "equal versions",
			a:    "1.0.0",
			b:    "1.0.0",
			want: 0,
		},
		{
			name: "a less than b (patch)",
			a:    "1.0.0",
			b:    "1.0.1",
			want: -1,
		},
		{
			name: "a greater than b (major)",
			a:    "2.0.0",
			b:    "1.9.9",
			want: 1,
		},
		{
			name: "a less than b (minor)",
			a:    "1.2.0",
			b:    "1.3.0",
			want: -1,
		},
		{
			name: "a greater than b (minor)",
			a:    "1.5.0",
			b:    "1.4.9",
			want: 1,
		},
		{
			name: "a greater than b (major vs minor)",
			a:    "2.0.0",
			b:    "1.99.99",
			want: 1,
		},
		{
			name: "large version numbers",
			a:    "10.20.30",
			b:    "10.20.30",
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CompareVersions(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("CompareVersions(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

// depPlugin is a minimal Plugin+DependencyPlugin for testing dependency resolution.
type depPlugin struct {
	info PluginInfo
	deps []Dependency
}

func (p *depPlugin) Info() PluginInfo                    { return p.info }
func (p *depPlugin) Init(_ PluginContext) error          { return nil }
func (p *depPlugin) Start(_ context.Context) error       { return nil }
func (p *depPlugin) Stop(_ context.Context) error        { return nil }
func (p *depPlugin) Dependencies() []Dependency          { return p.deps }

// noDepsPlugin is a Plugin with no dependencies (does not implement DependencyPlugin).
type noDepsPlugin struct {
	info PluginInfo
}

func (p *noDepsPlugin) Info() PluginInfo                    { return p.info }
func (p *noDepsPlugin) Init(_ PluginContext) error          { return nil }
func (p *noDepsPlugin) Start(_ context.Context) error       { return nil }
func (p *noDepsPlugin) Stop(_ context.Context) error        { return nil }

func TestDependencyResolver_NoDeps(t *testing.T) {
	dr := NewDependencyResolver()

	plugins := []Plugin{
		&noDepsPlugin{info: PluginInfo{Name: "alpha", Version: "1.0.0"}},
		&noDepsPlugin{info: PluginInfo{Name: "beta", Version: "1.0.0"}},
		&noDepsPlugin{info: PluginInfo{Name: "gamma", Version: "1.0.0"}},
	}

	order, err := dr.Resolve(plugins)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if len(order) != 3 {
		t.Fatalf("expected 3 plugins in order, got %d", len(order))
	}

	// All plugins should be present (order may vary since no deps).
	found := make(map[string]bool)
	for _, p := range order {
		found[p.Info().Name] = true
	}
	for _, p := range plugins {
		if !found[p.Info().Name] {
			t.Errorf("missing plugin %q in resolved order", p.Info().Name)
		}
	}
}

func TestDependencyResolver_SimpleDep(t *testing.T) {
	dr := NewDependencyResolver()

	plugins := []Plugin{
		&depPlugin{
			info: PluginInfo{Name: "plugin-a", Version: "1.0.0"},
			deps: []Dependency{
				{Name: "plugin-b", MinVersion: "1.0.0"},
			},
		},
		&noDepsPlugin{
			info: PluginInfo{Name: "plugin-b", Version: "1.0.0"},
		},
	}

	order, err := dr.Resolve(plugins)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	// B must come before A.
	names := pluginNames(order)
	idxB := indexOfStr(names, "plugin-b")
	idxA := indexOfStr(names, "plugin-a")

	if idxB < 0 || idxA < 0 {
		t.Fatalf("order = %v, expected both plugin-a and plugin-b", names)
	}
	if idxB >= idxA {
		t.Errorf("plugin-b (idx %d) must come before plugin-a (idx %d)", idxB, idxA)
	}
}

func TestDependencyResolver_ChainDep(t *testing.T) {
	dr := NewDependencyResolver()

	// A depends on B, B depends on C.
	plugins := []Plugin{
		&depPlugin{
			info: PluginInfo{Name: "plugin-a", Version: "1.0.0"},
			deps: []Dependency{{Name: "plugin-b"}},
		},
		&depPlugin{
			info: PluginInfo{Name: "plugin-b", Version: "1.0.0"},
			deps: []Dependency{{Name: "plugin-c"}},
		},
		&noDepsPlugin{
			info: PluginInfo{Name: "plugin-c", Version: "1.0.0"},
		},
	}

	order, err := dr.Resolve(plugins)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	names := pluginNames(order)
	idxC := indexOfStr(names, "plugin-c")
	idxB := indexOfStr(names, "plugin-b")
	idxA := indexOfStr(names, "plugin-a")

	if idxC >= idxB {
		t.Errorf("plugin-c (idx %d) must come before plugin-b (idx %d)", idxC, idxB)
	}
	if idxB >= idxA {
		t.Errorf("plugin-b (idx %d) must come before plugin-a (idx %d)", idxB, idxA)
	}
}

func TestDependencyResolver_CircularDep(t *testing.T) {
	dr := NewDependencyResolver()

	// A depends on B, B depends on A.
	plugins := []Plugin{
		&depPlugin{
			info: PluginInfo{Name: "plugin-a", Version: "1.0.0"},
			deps: []Dependency{{Name: "plugin-b"}},
		},
		&depPlugin{
			info: PluginInfo{Name: "plugin-b", Version: "1.0.0"},
			deps: []Dependency{{Name: "plugin-a"}},
		},
	}

	_, err := dr.Resolve(plugins)
	if err == nil {
		t.Fatal("expected error for circular dependency")
	}
	if !strings.Contains(err.Error(), "circular") {
		t.Errorf("error should mention circular dependency, got: %v", err)
	}
}

func TestDependencyResolver_MissingDep(t *testing.T) {
	dr := NewDependencyResolver()

	plugins := []Plugin{
		&depPlugin{
			info: PluginInfo{Name: "plugin-a", Version: "1.0.0"},
			deps: []Dependency{{Name: "nonexistent-plugin"}},
		},
	}

	_, err := dr.Resolve(plugins)
	if err == nil {
		t.Fatal("expected error for missing dependency")
	}
	if !strings.Contains(err.Error(), "requires") {
		t.Errorf("error should mention requires, got: %v", err)
	}
}

func TestDependencyResolver_OptionalDep(t *testing.T) {
	dr := NewDependencyResolver()

	plugins := []Plugin{
		&depPlugin{
			info: PluginInfo{Name: "plugin-a", Version: "1.0.0"},
			deps: []Dependency{{Name: "optional-plugin", Optional: true}},
		},
	}

	order, err := dr.Resolve(plugins)
	if err != nil {
		t.Fatalf("Resolve() error = %v (optional dependency missing should not fail)", err)
	}

	if len(order) != 1 {
		t.Fatalf("expected 1 plugin in order, got %d", len(order))
	}
	if order[0].Info().Name != "plugin-a" {
		t.Errorf("order[0] = %q, want %q", order[0].Info().Name, "plugin-a")
	}
}

func TestDependencyResolver_VersionMismatch(t *testing.T) {
	dr := NewDependencyResolver()

	plugins := []Plugin{
		&depPlugin{
			info: PluginInfo{Name: "plugin-a", Version: "1.0.0"},
			deps: []Dependency{{Name: "plugin-b", MinVersion: "2.0.0"}},
		},
		&noDepsPlugin{
			info: PluginInfo{Name: "plugin-b", Version: "1.5.0"}, // Too low
		},
	}

	_, err := dr.Resolve(plugins)
	if err == nil {
		t.Fatal("expected error for version mismatch")
	}
	if !strings.Contains(err.Error(), "requires") {
		t.Errorf("error should mention version requirement, got: %v", err)
	}
}

func TestDependencyResolver_VersionSatisfied(t *testing.T) {
	dr := NewDependencyResolver()

	plugins := []Plugin{
		&depPlugin{
			info: PluginInfo{Name: "plugin-a", Version: "1.0.0"},
			deps: []Dependency{{Name: "plugin-b", MinVersion: "1.5.0"}},
		},
		&noDepsPlugin{
			info: PluginInfo{Name: "plugin-b", Version: "2.0.0"}, // Higher than min
		},
	}

	order, err := dr.Resolve(plugins)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if len(order) != 2 {
		t.Fatalf("expected 2 plugins in order, got %d", len(order))
	}
}

func TestDependencyResolver_DiamondDep(t *testing.T) {
	dr := NewDependencyResolver()

	// Diamond: A and B both depend on C, D depends on both A and B.
	plugins := []Plugin{
		&depPlugin{
			info: PluginInfo{Name: "d", Version: "1.0.0"},
			deps: []Dependency{{Name: "a"}, {Name: "b"}},
		},
		&depPlugin{
			info: PluginInfo{Name: "a", Version: "1.0.0"},
			deps: []Dependency{{Name: "c"}},
		},
		&depPlugin{
			info: PluginInfo{Name: "b", Version: "1.0.0"},
			deps: []Dependency{{Name: "c"}},
		},
		&noDepsPlugin{
			info: PluginInfo{Name: "c", Version: "1.0.0"},
		},
	}

	order, err := dr.Resolve(plugins)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	names := pluginNames(order)
	idxC := indexOfStr(names, "c")
	idxA := indexOfStr(names, "a")
	idxB := indexOfStr(names, "b")
	idxD := indexOfStr(names, "d")

	if idxC >= idxA || idxC >= idxB {
		t.Errorf("c should come before a and b: order = %v", names)
	}
	if idxA >= idxD || idxB >= idxD {
		t.Errorf("a and b should come before d: order = %v", names)
	}
}

// indexOfStr returns the index of name in the order slice, or -1 if not found.
func indexOfStr(order []string, name string) int {
	for i, n := range order {
		if n == name {
			return i
		}
	}
	return -1
}

// pluginNames extracts the names from a list of plugins.
func pluginNames(plugins []Plugin) []string {
	names := make([]string, len(plugins))
	for i, p := range plugins {
		names[i] = p.Info().Name
	}
	return names
}
