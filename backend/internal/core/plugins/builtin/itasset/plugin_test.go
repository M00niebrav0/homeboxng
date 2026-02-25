package itasset

import (
	"context"
	"testing"
)

func TestNew(t *testing.T) {
	p := New()
	if p == nil {
		t.Fatal("New() returned nil")
	}
}

func TestPluginInfo(t *testing.T) {
	p := New()
	info := p.Info()

	if info.Name != "it-assets" {
		t.Errorf("expected name 'it-assets', got %q", info.Name)
	}
	if info.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %q", info.Version)
	}
	if !info.BuiltIn {
		t.Error("expected BuiltIn to be true")
	}
}

func TestAssetTemplatesNotEmpty(t *testing.T) {
	if len(assetTemplates) == 0 {
		t.Fatal("assetTemplates is empty")
	}

	slugs := make(map[string]bool)
	for _, tmpl := range assetTemplates {
		if slugs[tmpl.Slug] {
			t.Errorf("duplicate template slug: %q", tmpl.Slug)
		}
		slugs[tmpl.Slug] = true

		if tmpl.Label == "" {
			t.Errorf("template %q has empty label", tmpl.Slug)
		}
		if len(tmpl.FormFactors) == 0 {
			t.Errorf("template %q has no form factors", tmpl.Slug)
		}
	}

	expected := []string{"server-rackmount", "desktop-pc", "laptop", "nas", "network-switch"}
	for _, slug := range expected {
		if !slugs[slug] {
			t.Errorf("expected template %q not found", slug)
		}
	}
}

func TestAssetTemplateComponentFlags(t *testing.T) {
	// After init(), component flags should be set
	for _, tmpl := range assetTemplates {
		switch tmpl.Slug {
		case "server-rackmount", "server-tower", "desktop-pc":
			if !tmpl.Components.CPU || !tmpl.Components.Memory || !tmpl.Components.Storage ||
				!tmpl.Components.PCIe || !tmpl.Components.GPU || !tmpl.Components.NIC || !tmpl.Components.PSU {
				t.Errorf("template %q should have all main component flags set", tmpl.Slug)
			}
		case "laptop":
			if !tmpl.Components.CPU || !tmpl.Components.Memory || !tmpl.Components.Storage ||
				!tmpl.Components.Display {
				t.Errorf("template %q missing expected component flags", tmpl.Slug)
			}
		case "monitor":
			if !tmpl.Components.Display {
				t.Errorf("monitor template should have Display=true")
			}
		}
	}
}

func TestHardwareLabelsNotEmpty(t *testing.T) {
	if len(hardwareLabels) == 0 {
		t.Fatal("hardwareLabels is empty")
	}

	for _, label := range hardwareLabels {
		if label.Key == "" {
			t.Error("hardware label has empty key")
		}
		if label.Label == "" {
			t.Errorf("hardware label %q has empty label", label.Key)
		}
	}
}

func TestConnectionProtocolsNotEmpty(t *testing.T) {
	if len(connectionProtocols) == 0 {
		t.Fatal("connectionProtocols is empty")
	}

	ids := make(map[string]bool)
	for _, proto := range connectionProtocols {
		if ids[proto.ID] {
			t.Errorf("duplicate protocol ID: %q", proto.ID)
		}
		ids[proto.ID] = true

		if proto.Label == "" {
			t.Errorf("protocol %q has empty label", proto.ID)
		}
		if len(proto.Tools) == 0 {
			t.Errorf("protocol %q has no tools", proto.ID)
		}
	}

	expected := []string{"ssh", "rdp", "ipmi", "https", "snmp"}
	for _, id := range expected {
		if !ids[id] {
			t.Errorf("expected protocol %q not found", id)
		}
	}
}

func TestComponentSchemasNotEmpty(t *testing.T) {
	if len(componentSchemas) == 0 {
		t.Fatal("componentSchemas is empty")
	}

	for _, schema := range componentSchemas {
		if schema.ComponentType == "" {
			t.Error("component schema has empty type")
		}
		if len(schema.Fields) == 0 {
			t.Errorf("component schema %q has no fields", schema.ComponentType)
		}
		// Each schema should have at least one required field
		hasRequired := false
		for _, field := range schema.Fields {
			if field.Required {
				hasRequired = true
				break
			}
		}
		if !hasRequired {
			// Not all schemas need required fields, but most should have at least a model/capacity
			t.Logf("note: schema %q has no required fields", schema.ComponentType)
		}
	}
}

func TestPluginStartStop(t *testing.T) {
	p := New()
	ctx := context.Background()
	if err := p.Start(ctx); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	if err := p.Stop(ctx); err != nil {
		t.Fatalf("Stop() error: %v", err)
	}
}

func TestConfigSchema(t *testing.T) {
	p := New()
	schema := p.ConfigSchema()
	if len(schema) == 0 {
		t.Fatal("ConfigSchema() returned empty slice")
	}
	// Should have vaultwarden_url config
	found := false
	for _, field := range schema {
		if field.Key == "vaultwarden_url" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'vaultwarden_url' in config schema")
	}
}

func TestRequestedPermissions(t *testing.T) {
	p := New()
	perms := p.RequestedPermissions()
	if len(perms) == 0 {
		t.Fatal("RequestedPermissions() returned empty slice")
	}
}
