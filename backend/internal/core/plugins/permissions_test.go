package plugins

import (
	"testing"
)

func TestPermissionManager_GrantAndCheck(t *testing.T) {
	pm := NewPermissionManager()

	pm.Grant("test-plugin", PermReadItems)

	if !pm.HasPermission("test-plugin", PermReadItems) {
		t.Error("expected permission to be granted")
	}

	if pm.HasPermission("test-plugin", PermWriteItems) {
		t.Error("expected ungranted permission to return false")
	}

	if pm.HasPermission("other-plugin", PermReadItems) {
		t.Error("expected different plugin to not have permission")
	}
}

func TestPermissionManager_Check(t *testing.T) {
	pm := NewPermissionManager()
	pm.Grant("test", PermReadItems)

	if err := pm.Check("test", PermReadItems); err != nil {
		t.Errorf("Check() error = %v for granted permission", err)
	}

	if err := pm.Check("test", PermWriteItems); err == nil {
		t.Error("expected error for ungranted permission")
	}
}

func TestPermissionManager_Revoke(t *testing.T) {
	pm := NewPermissionManager()
	pm.Grant("test", PermReadItems)
	pm.Grant("test", PermWriteItems)

	pm.Revoke("test", PermReadItems)

	if pm.HasPermission("test", PermReadItems) {
		t.Error("expected revoked permission to return false")
	}
	if !pm.HasPermission("test", PermWriteItems) {
		t.Error("expected non-revoked permission to still be granted")
	}
}

func TestPermissionManager_RevokeAll(t *testing.T) {
	pm := NewPermissionManager()
	pm.Grant("test", PermReadItems)
	pm.Grant("test", PermWriteItems)
	pm.Grant("test", PermNetwork)

	pm.RevokeAll("test")

	grants := pm.GetGrants("test")
	if len(grants) != 0 {
		t.Errorf("expected 0 grants after RevokeAll, got %d", len(grants))
	}
}

func TestPermissionManager_GrantAll(t *testing.T) {
	pm := NewPermissionManager()

	requests := []PermissionRequest{
		{Permission: PermReadItems, Reason: "needs to read items", Required: true},
		{Permission: PermNetwork, Reason: "calls external API", Required: false},
	}

	pm.GrantAll("test", requests)

	if !pm.HasPermission("test", PermReadItems) {
		t.Error("expected PermReadItems to be granted")
	}
	if !pm.HasPermission("test", PermNetwork) {
		t.Error("expected PermNetwork to be granted")
	}
}

func TestPermissionManager_GetGrants(t *testing.T) {
	pm := NewPermissionManager()
	pm.Grant("test", PermReadItems)
	pm.Grant("test", PermWriteItems)

	grants := pm.GetGrants("test")
	if len(grants) != 2 {
		t.Fatalf("expected 2 grants, got %d", len(grants))
	}

	// Verify both permissions present (order not guaranteed from map)
	found := map[Permission]bool{}
	for _, g := range grants {
		found[g] = true
	}
	if !found[PermReadItems] || !found[PermWriteItems] {
		t.Error("missing expected permissions in grants")
	}
}

func TestPermissionManager_GetAllGrants(t *testing.T) {
	pm := NewPermissionManager()
	pm.Grant("plugin-a", PermReadItems)
	pm.Grant("plugin-b", PermNetwork)

	all := pm.GetAllGrants()
	if len(all) != 2 {
		t.Fatalf("expected 2 plugins in grants, got %d", len(all))
	}
	if len(all["plugin-a"]) != 1 || all["plugin-a"][0] != PermReadItems {
		t.Error("plugin-a should have PermReadItems")
	}
	if len(all["plugin-b"]) != 1 || all["plugin-b"][0] != PermNetwork {
		t.Error("plugin-b should have PermNetwork")
	}
}

func TestPermissionManager_GetSummary(t *testing.T) {
	pm := NewPermissionManager()
	pm.Grant("test", PermReadItems)

	requested := []PermissionRequest{
		{Permission: PermReadItems, Reason: "read", Required: true},
		{Permission: PermWriteItems, Reason: "write", Required: false},
	}

	summary := pm.GetSummary("test", requested)

	if summary.PluginName != "test" {
		t.Errorf("PluginName = %q, want %q", summary.PluginName, "test")
	}
	if len(summary.Granted) != 1 || summary.Granted[0] != PermReadItems {
		t.Error("expected PermReadItems in granted")
	}
	if len(summary.Denied) != 1 || summary.Denied[0] != PermWriteItems {
		t.Error("expected PermWriteItems in denied")
	}
}

func TestAllPermissions_Count(t *testing.T) {
	perms := AllPermissions()
	if len(perms) != 24 {
		t.Errorf("expected 24 permissions, got %d", len(perms))
	}

	// Verify all categories present
	categories := map[string]bool{}
	for _, p := range perms {
		categories[p.Category] = true
	}
	expected := []string{"Data", "Features", "System", "External"}
	for _, cat := range expected {
		if !categories[cat] {
			t.Errorf("missing category %q", cat)
		}
	}
}

func TestPermissionManager_EmptyPlugin(t *testing.T) {
	pm := NewPermissionManager()

	// No grants for this plugin
	if pm.HasPermission("empty", PermReadItems) {
		t.Error("expected false for plugin with no grants")
	}

	grants := pm.GetGrants("empty")
	if len(grants) != 0 {
		t.Error("expected empty grants for unknown plugin")
	}

	// Revoking from empty should not panic
	pm.Revoke("empty", PermReadItems)
	pm.RevokeAll("empty")
}
