package plugins

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
)

func TestExternalPlugin_Info(t *testing.T) {
	p := &ExternalPlugin{
		reg: ExternalPluginRegistration{
			Name:    "test-plugin",
			Version: "1.2.3",
			BaseURL: "http://localhost:8080",
		},
		logger: zerolog.Nop(),
	}

	info := p.Info()
	if info.Name != "test-plugin" {
		t.Errorf("Name = %q, want %q", info.Name, "test-plugin")
	}
	if info.Version != "1.2.3" {
		t.Errorf("Version = %q, want %q", info.Version, "1.2.3")
	}
	if info.BuiltIn {
		t.Error("external plugin should not be built-in")
	}
}

func TestExternalPlugin_Init(t *testing.T) {
	p := &ExternalPlugin{
		reg:    ExternalPluginRegistration{Name: "test"},
		logger: zerolog.Nop(),
	}

	err := p.Init(PluginContext{})
	if err != nil {
		t.Errorf("Init() should be no-op for external plugins, got %v", err)
	}
}

func TestExternalPlugin_StartStop(t *testing.T) {
	// Create a test health endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	p := &ExternalPlugin{
		reg: ExternalPluginRegistration{
			Name:           "test",
			BaseURL:        server.URL,
			HealthEndpoint: "/health",
		},
		client: &http.Client{},
		logger: zerolog.Nop(),
	}

	ctx := context.Background()
	err := p.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Stop should not error
	err = p.Stop(ctx)
	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}

func TestExternalPlugin_Stop_NilCancel(t *testing.T) {
	p := &ExternalPlugin{
		reg:    ExternalPluginRegistration{Name: "test"},
		logger: zerolog.Nop(),
	}

	// Stop before Start should not panic
	err := p.Stop(context.Background())
	if err != nil {
		t.Errorf("Stop() without Start() error = %v", err)
	}
}

func TestExternalPluginRegistration_Defaults(t *testing.T) {
	reg := ExternalPluginRegistration{
		Name:    "plugin",
		Version: "1.0.0",
		BaseURL: "http://localhost:8080",
	}

	if reg.HealthEndpoint != "" {
		t.Errorf("HealthEndpoint should default to empty, got %q", reg.HealthEndpoint)
	}
	if reg.WebhookEndpoint != "" {
		t.Errorf("WebhookEndpoint should default to empty, got %q", reg.WebhookEndpoint)
	}
}

func TestExternalPluginManifest_Fields(t *testing.T) {
	m := ExternalPluginManifest{
		Name:        "test-plugin",
		Version:     "2.0.0",
		Description: "A test plugin",
		Author:      "tester",
		Repository:  "https://github.com/user/plugin",
		DockerImage: "ghcr.io/user/plugin:latest",
		MinVersion:  "0.1.0",
		Categories:  []string{"ai", "integration"},
		Stars:       42,
		Downloads:   100,
	}

	if m.Name != "test-plugin" {
		t.Errorf("Name = %q", m.Name)
	}
	if len(m.Categories) != 2 {
		t.Errorf("Categories count = %d", len(m.Categories))
	}
	if m.Stars != 42 {
		t.Errorf("Stars = %d", m.Stars)
	}
}

func TestContainsIgnoreCase(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		substr string
		want   bool
	}{
		{"exact match", "hello", "hello", true},
		{"case insensitive", "Hello World", "hello", true},
		{"uppercase substr", "test string", "STRING", true},
		{"empty substr", "anything", "", true},
		{"not found", "abc", "xyz", false},
		{"partial match", "foobar", "bar", true},
		{"both empty", "", "", true},
		{"empty s non-empty substr", "", "x", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsIgnoreCase(tt.s, tt.substr)
			if got != tt.want {
				t.Errorf("containsIgnoreCase(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}

func TestToLower(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Hello", "hello"},
		{"UPPERCASE", "uppercase"},
		{"already lower", "already lower"},
		{"Mixed123Case", "mixed123case"},
		{"", ""},
	}

	for _, tt := range tests {
		got := toLower(tt.input)
		if got != tt.want {
			t.Errorf("toLower(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestIndexOf(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		want   int
	}{
		{"hello world", "world", 6},
		{"hello", "hello", 0},
		{"hello", "xyz", -1},
		{"abcabc", "bc", 1},
		{"", "", 0},
		{"abc", "", 0},
	}

	for _, tt := range tests {
		got := indexOf(tt.s, tt.substr)
		if got != tt.want {
			t.Errorf("indexOf(%q, %q) = %d, want %d", tt.s, tt.substr, got, tt.want)
		}
	}
}

func TestPluginRegistrySource_Constant(t *testing.T) {
	if PluginRegistrySource == "" {
		t.Error("PluginRegistrySource should not be empty")
	}
	if PluginRegistrySource != "https://raw.githubusercontent.com/M00niebrav0/homeboxng-plugins/main/registry.json" {
		t.Errorf("PluginRegistrySource = %q", PluginRegistrySource)
	}
}
