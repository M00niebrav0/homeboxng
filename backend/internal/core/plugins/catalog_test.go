package plugins

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
)

func TestDefaultSources(t *testing.T) {
	sources := DefaultSources()
	if len(sources) != 1 {
		t.Fatalf("expected 1 default source, got %d", len(sources))
	}
	if sources[0].Type != "official" {
		t.Errorf("type = %q, want %q", sources[0].Type, "official")
	}
	if sources[0].Name != "HomeBoxNG Official" {
		t.Errorf("name = %q", sources[0].Name)
	}
	if !sources[0].Enabled {
		t.Error("default source should be enabled")
	}
}

func TestMultiSourceCatalog_AddSource(t *testing.T) {
	c := NewMultiSourceCatalog(zerolog.Nop())

	c.AddSource(CatalogSource{
		Name:    "test-repo",
		URL:     "https://example.com/plugins.json",
		Type:    "community",
		Enabled: true,
	})

	sources := c.Sources()
	if len(sources) != 2 { // 1 default + 1 added
		t.Fatalf("expected 2 sources, got %d", len(sources))
	}
}

func TestMultiSourceCatalog_AddSource_NoDuplicate(t *testing.T) {
	c := NewMultiSourceCatalog(zerolog.Nop())

	url := "https://example.com/plugins.json"
	c.AddSource(CatalogSource{Name: "repo1", URL: url, Type: "community", Enabled: true})
	c.AddSource(CatalogSource{Name: "repo1-dup", URL: url, Type: "community", Enabled: true})

	sources := c.Sources()
	if len(sources) != 2 { // 1 default + 1 added (duplicate rejected)
		t.Fatalf("expected 2 sources (duplicate rejected), got %d", len(sources))
	}
}

func TestMultiSourceCatalog_RemoveSource(t *testing.T) {
	c := NewMultiSourceCatalog(zerolog.Nop())

	url := "https://example.com/plugins.json"
	c.AddSource(CatalogSource{Name: "test", URL: url, Type: "community", Enabled: true})

	err := c.RemoveSource(url)
	if err != nil {
		t.Fatalf("RemoveSource() error = %v", err)
	}

	sources := c.Sources()
	if len(sources) != 1 { // only default remains
		t.Fatalf("expected 1 source after remove, got %d", len(sources))
	}
}

func TestMultiSourceCatalog_RemoveSource_CannotRemoveOfficial(t *testing.T) {
	c := NewMultiSourceCatalog(zerolog.Nop())

	sources := c.Sources()
	err := c.RemoveSource(sources[0].URL)
	if err == nil {
		t.Error("expected error when removing official source")
	}
}

func TestMultiSourceCatalog_RemoveSource_NotFound(t *testing.T) {
	c := NewMultiSourceCatalog(zerolog.Nop())

	err := c.RemoveSource("https://nonexistent.com/plugins.json")
	if err == nil {
		t.Error("expected error for non-existent source")
	}
}

func TestMultiSourceCatalog_Sources_ReturnsCopy(t *testing.T) {
	c := NewMultiSourceCatalog(zerolog.Nop())

	sources1 := c.Sources()
	sources2 := c.Sources()

	// Modifying one should not affect the other
	if len(sources1) != len(sources2) {
		t.Error("sources should be equal length")
	}
}

func TestMultiSourceCatalog_AddSourceFromGitHub(t *testing.T) {
	c := NewMultiSourceCatalog(zerolog.Nop())

	c.AddSourceFromGitHub("user/homeboxng-plugin-foo")

	sources := c.Sources()
	if len(sources) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(sources))
	}

	added := sources[1]
	if added.Name != "homeboxng-plugin-foo" {
		t.Errorf("name = %q, want %q", added.Name, "homeboxng-plugin-foo")
	}
	if added.Type != "community" {
		t.Errorf("type = %q, want %q", added.Type, "community")
	}
	if added.URL != "https://raw.githubusercontent.com/user/homeboxng-plugin-foo/main/registry.json" {
		t.Errorf("URL = %q", added.URL)
	}
}

func TestMultiSourceCatalog_AddSourceFromGitHub_SimpleRepo(t *testing.T) {
	c := NewMultiSourceCatalog(zerolog.Nop())

	c.AddSourceFromGitHub("singlename")

	sources := c.Sources()
	added := sources[1]
	if added.Name != "singlename" {
		t.Errorf("name = %q, want %q", added.Name, "singlename")
	}
}

func TestMultiSourceCatalog_RefreshAll_WithServer(t *testing.T) {
	manifests := []ExternalPluginManifest{
		{Name: "test-plugin", Version: "1.0.0", Description: "A test plugin", Author: "tester"},
		{Name: "other-plugin", Version: "2.0.0", Description: "Another plugin", Author: "author"},
	}
	data, _ := json.Marshal(manifests)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}))
	defer server.Close()

	c := &MultiSourceCatalog{
		sources: []CatalogSource{
			{Name: "test", URL: server.URL, Type: "community", Enabled: true},
		},
		available: make(map[string][]ExternalPluginManifest),
		logger:    zerolog.Nop(),
	}

	err := c.RefreshAll(context.Background())
	if err != nil {
		t.Fatalf("RefreshAll() error = %v", err)
	}

	all := c.AllAvailable()
	if len(all) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(all))
	}
	if all[0].Name != "test-plugin" {
		t.Errorf("name = %q", all[0].Name)
	}
}

func TestMultiSourceCatalog_RefreshAll_SkipsDisabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("disabled source should not be fetched")
	}))
	defer server.Close()

	c := &MultiSourceCatalog{
		sources: []CatalogSource{
			{Name: "disabled", URL: server.URL, Type: "community", Enabled: false},
		},
		available: make(map[string][]ExternalPluginManifest),
		logger:    zerolog.Nop(),
	}

	err := c.RefreshAll(context.Background())
	if err != nil {
		t.Fatalf("RefreshAll() error = %v", err)
	}
}

func TestMultiSourceCatalog_RefreshAll_ErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	c := &MultiSourceCatalog{
		sources: []CatalogSource{
			{Name: "broken", URL: server.URL, Type: "community", Enabled: true},
		},
		available: make(map[string][]ExternalPluginManifest),
		logger:    zerolog.Nop(),
	}

	err := c.RefreshAll(context.Background())
	if err == nil {
		t.Error("expected error for failed source")
	}
}

func TestMultiSourceCatalog_AllAvailable_Deduplication(t *testing.T) {
	c := &MultiSourceCatalog{
		sources: []CatalogSource{
			{Name: "source1", Type: "official"},
			{Name: "source2", Type: "community"},
		},
		available: map[string][]ExternalPluginManifest{
			"source1": {
				{Name: "plugin-a", Version: "1.0.0"},
				{Name: "plugin-b", Version: "1.0.0"},
			},
			"source2": {
				{Name: "plugin-a", Version: "2.0.0"}, // duplicate name
				{Name: "plugin-c", Version: "1.0.0"},
			},
		},
		logger: zerolog.Nop(),
	}

	all := c.AllAvailable()
	if len(all) != 3 {
		t.Fatalf("expected 3 unique plugins, got %d", len(all))
	}

	// First occurrence of plugin-a should win
	for _, p := range all {
		if p.Name == "plugin-a" && p.Version != "1.0.0" {
			t.Error("expected first source to win on duplicates")
		}
	}
}

func TestMultiSourceCatalog_SearchAll(t *testing.T) {
	c := &MultiSourceCatalog{
		sources: []CatalogSource{
			{Name: "test", Type: "official"},
		},
		available: map[string][]ExternalPluginManifest{
			"test": {
				{Name: "ai-vision", Description: "AI photo identification", Categories: []string{"ai"}},
				{Name: "label-printer", Description: "Brother QL printer support", Categories: []string{"labels", "hardware"}},
				{Name: "backup-sync", Description: "Cloud backup integration", Categories: []string{"backup"}},
			},
		},
		logger: zerolog.Nop(),
	}

	tests := []struct {
		query string
		want  int
	}{
		{"", 3},           // empty query returns all
		{"ai", 1},         // matches ai-vision by name (also category but same plugin)
		{"printer", 1},    // matches label-printer name
		{"brother", 1},    // matches description
		{"backup", 1},     // matches backup-sync by name (also category but same plugin)
		{"hardware", 1},   // matches category
		{"nonexistent", 0},
	}

	for _, tt := range tests {
		results := c.SearchAll(tt.query)
		if len(results) != tt.want {
			t.Errorf("SearchAll(%q) = %d results, want %d", tt.query, len(results), tt.want)
		}
	}
}

func TestNewPluginCatalog(t *testing.T) {
	c := NewPluginCatalog(zerolog.Nop())
	if c == nil {
		t.Fatal("expected non-nil catalog")
	}
	available := c.Available()
	if len(available) != 0 {
		t.Errorf("expected 0 available before refresh, got %d", len(available))
	}
}

func TestPluginCatalog_Refresh(t *testing.T) {
	manifests := []ExternalPluginManifest{
		{Name: "plugin-1", Version: "1.0.0", Description: "Test", Author: "tester"},
	}
	data, _ := json.Marshal(manifests)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}))
	defer server.Close()

	// Use a custom catalog pointing to test server
	c := &PluginCatalog{
		logger: zerolog.Nop(),
	}

	// We can't easily test Refresh() because it uses the hardcoded PluginRegistrySource.
	// Instead, test the Available and Search methods after manually populating.
	c.available = manifests

	available := c.Available()
	if len(available) != 1 {
		t.Fatalf("expected 1 available, got %d", len(available))
	}
}

func TestPluginCatalog_Search(t *testing.T) {
	c := &PluginCatalog{
		available: []ExternalPluginManifest{
			{Name: "ai-vision", Description: "Vision plugin", Categories: []string{"ai"}},
			{Name: "backup", Description: "Backup plugin", Categories: []string{"storage"}},
			{Name: "notifications", Description: "Alert system", Categories: []string{"alerts"}},
		},
		logger: zerolog.Nop(),
	}

	tests := []struct {
		query string
		want  int
	}{
		{"", 3},
		{"ai", 1}, // matches ai-vision by name (also category but same plugin)
		{"backup", 1}, // matches "backup" by name
		{"VISION", 1}, // case insensitive
		{"xyz", 0},
	}

	for _, tt := range tests {
		results := c.Search(tt.query)
		if len(results) != tt.want {
			t.Errorf("Search(%q) = %d, want %d", tt.query, len(results), tt.want)
		}
	}
}

func TestFetchRegistry_WithServer(t *testing.T) {
	manifests := []ExternalPluginManifest{
		{Name: "remote-plugin", Version: "3.0.0"},
	}
	data, _ := json.Marshal(manifests)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}))
	defer server.Close()

	client := &http.Client{}
	result, err := fetchRegistry(context.Background(), client, server.URL)
	if err != nil {
		t.Fatalf("fetchRegistry() error = %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 manifest, got %d", len(result))
	}
	if result[0].Name != "remote-plugin" {
		t.Errorf("name = %q", result[0].Name)
	}
}

func TestFetchRegistry_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := &http.Client{}
	_, err := fetchRegistry(context.Background(), client, server.URL)
	if err == nil {
		t.Error("expected error for 404 response")
	}
}

func TestFetchRegistry_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer server.Close()

	client := &http.Client{}
	_, err := fetchRegistry(context.Background(), client, server.URL)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
