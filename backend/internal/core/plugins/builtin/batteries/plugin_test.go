package batteries

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

	if info.Name != "batteries" {
		t.Errorf("expected name 'batteries', got %q", info.Name)
	}
	if info.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %q", info.Version)
	}
	if !info.BuiltIn {
		t.Error("expected BuiltIn to be true")
	}
}

func TestBatteryPlatformsNotEmpty(t *testing.T) {
	if len(batteryPlatforms) == 0 {
		t.Fatal("batteryPlatforms is empty")
	}

	// Check a few known platforms
	slugs := make(map[string]bool)
	for _, bp := range batteryPlatforms {
		slugs[bp.Slug] = true
		if bp.Brand == "" {
			t.Errorf("platform %q has empty brand", bp.Slug)
		}
		if bp.NominalVolts <= 0 {
			t.Errorf("platform %q has invalid voltage %f", bp.Slug, bp.NominalVolts)
		}
		if len(bp.Batteries) == 0 {
			t.Errorf("platform %q has no batteries", bp.Slug)
		}
		if len(bp.Chargers) == 0 {
			t.Errorf("platform %q has no chargers", bp.Slug)
		}
	}

	expected := []string{"milwaukee-m18", "milwaukee-m12", "ryobi-one-plus", "dewalt-20v-max", "makita-lxt"}
	for _, slug := range expected {
		if !slugs[slug] {
			t.Errorf("expected platform %q not found", slug)
		}
	}
}

func TestStorageSystemsNotEmpty(t *testing.T) {
	if len(storageSystems) == 0 {
		t.Fatal("storageSystems is empty")
	}

	slugs := make(map[string]bool)
	for _, ss := range storageSystems {
		slugs[ss.Slug] = true
		if ss.Brand == "" {
			t.Errorf("storage system %q has empty brand", ss.Slug)
		}
		if len(ss.Items) == 0 {
			t.Errorf("storage system %q has no items", ss.Slug)
		}
	}

	expected := []string{"milwaukee-packout", "ryobi-link", "dewalt-toughsystem"}
	for _, slug := range expected {
		if !slugs[slug] {
			t.Errorf("expected storage system %q not found", slug)
		}
	}
}

func TestBatterySKUsUnique(t *testing.T) {
	seen := make(map[string]string) // SKU -> platform
	for _, bp := range batteryPlatforms {
		for _, bat := range bp.Batteries {
			if prev, ok := seen[bat.SKU]; ok {
				t.Errorf("duplicate battery SKU %q in platforms %q and %q", bat.SKU, prev, bp.Slug)
			}
			seen[bat.SKU] = bp.Slug
		}
	}
}

func TestChargerSKUsUnique(t *testing.T) {
	seen := make(map[string]string) // SKU -> platform
	for _, bp := range batteryPlatforms {
		for _, chg := range bp.Chargers {
			if prev, ok := seen[chg.SKU]; ok {
				t.Errorf("duplicate charger SKU %q in platforms %q and %q", chg.SKU, prev, bp.Slug)
			}
			seen[chg.SKU] = bp.Slug
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
	for _, field := range schema {
		if field.Key == "" {
			t.Error("config field has empty key")
		}
		if field.Label == "" {
			t.Errorf("config field %q has empty label", field.Key)
		}
	}
}

func TestRequestedPermissions(t *testing.T) {
	p := New()
	perms := p.RequestedPermissions()
	if len(perms) == 0 {
		t.Fatal("RequestedPermissions() returned empty slice")
	}
}
