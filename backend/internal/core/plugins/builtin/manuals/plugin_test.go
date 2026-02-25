package manuals

import (
	"context"
	"testing"
)

func TestNew(t *testing.T) {
	p := New()
	if p == nil {
		t.Fatal("New() returned nil")
	}
	if p.manuals == nil {
		t.Error("manuals map not initialized")
	}
	if p.confidenceThreshold != 0.7 {
		t.Errorf("expected default confidence threshold 0.7, got %f", p.confidenceThreshold)
	}
	if p.maxSuggestions != 5 {
		t.Errorf("expected default max suggestions 5, got %d", p.maxSuggestions)
	}
}

func TestPluginInfo(t *testing.T) {
	p := New()
	info := p.Info()

	if info.Name != "manuals" {
		t.Errorf("expected name 'manuals', got %q", info.Name)
	}
	if !info.BuiltIn {
		t.Error("expected BuiltIn to be true")
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

func TestBuildFuzzyQueries_ExactName(t *testing.T) {
	queries := BuildFuzzyQueries("DeWalt DCD771C2 Drill/Driver", "", "")

	if len(queries) == 0 {
		t.Fatal("BuildFuzzyQueries returned no queries")
	}

	// First query should be the exact name
	if queries[0].Query != "DeWalt DCD771C2 Drill/Driver" {
		t.Errorf("first query should be exact name, got %q", queries[0].Query)
	}
	if queries[0].Weight != 1.0 {
		t.Errorf("first query should have weight 1.0, got %f", queries[0].Weight)
	}
}

func TestBuildFuzzyQueries_BrandAndModel(t *testing.T) {
	queries := BuildFuzzyQueries("DeWalt DCD771C2 Drill/Driver", "DeWalt", "DCD771C2")

	// Should include brand + model query
	found := false
	for _, q := range queries {
		if q.Query == "DeWalt DCD771C2" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'DeWalt DCD771C2' query from explicit brand + model")
	}
}

func TestBuildFuzzyQueries_WithYear(t *testing.T) {
	queries := BuildFuzzyQueries("2018 Toyota Tacoma", "", "")

	if len(queries) < 2 {
		t.Fatalf("expected at least 2 queries, got %d", len(queries))
	}

	// Should extract year and produce rearranged query
	foundRearranged := false
	for _, q := range queries {
		if q.Query == "Toyota Tacoma 2018" {
			foundRearranged = true
			break
		}
	}
	if !foundRearranged {
		t.Log("queries generated:")
		for _, q := range queries {
			t.Logf("  %q (weight: %.2f)", q.Query, q.Weight)
		}
		t.Error("expected rearranged query 'Toyota Tacoma 2018'")
	}
}

func TestBuildFuzzyQueries_EmptyInput(t *testing.T) {
	queries := BuildFuzzyQueries("", "", "")
	if len(queries) != 0 {
		t.Errorf("expected 0 queries for empty input, got %d", len(queries))
	}
}

func TestBuildFuzzyQueries_SortedByWeight(t *testing.T) {
	queries := BuildFuzzyQueries("Samsung UN55TU7000 55-Inch TV", "Samsung", "UN55TU7000")

	for i := 1; i < len(queries); i++ {
		if queries[i].Weight > queries[i-1].Weight {
			t.Errorf("queries not sorted by weight: [%d].Weight=%f > [%d].Weight=%f",
				i, queries[i].Weight, i-1, queries[i-1].Weight)
		}
	}
}

func TestStringSimilarity_Identical(t *testing.T) {
	score := StringSimilarity("hello world", "hello world")
	if score != 1.0 {
		t.Errorf("expected 1.0 for identical strings, got %f", score)
	}
}

func TestStringSimilarity_Empty(t *testing.T) {
	score := StringSimilarity("", "hello")
	if score != 0.0 {
		t.Errorf("expected 0.0 for one empty string, got %f", score)
	}
}

func TestStringSimilarity_CaseInsensitive(t *testing.T) {
	score := StringSimilarity("Hello World", "hello world")
	if score != 1.0 {
		t.Errorf("expected 1.0 for case-insensitive match, got %f", score)
	}
}

func TestStringSimilarity_Similar(t *testing.T) {
	score := StringSimilarity("DeWalt DCD771C2", "DeWalt DCD771")
	if score < 0.4 {
		t.Errorf("expected similarity > 0.4 for similar strings, got %f", score)
	}
}

func TestStringSimilarity_Different(t *testing.T) {
	score := StringSimilarity("Milwaukee M18", "Toyota Tacoma")
	if score > 0.3 {
		t.Errorf("expected similarity < 0.3 for unrelated strings, got %f", score)
	}
}

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		a, b     string
		expected int
	}{
		{"", "", 0},
		{"abc", "", 3},
		{"", "abc", 3},
		{"abc", "abc", 0},
		{"abc", "abd", 1},
		{"kitten", "sitting", 3},
	}

	for _, tt := range tests {
		dist := levenshteinDistance(tt.a, tt.b)
		if dist != tt.expected {
			t.Errorf("levenshteinDistance(%q, %q) = %d, want %d", tt.a, tt.b, dist, tt.expected)
		}
	}
}

func TestTokenOverlap(t *testing.T) {
	// Identical tokens
	score := tokenOverlap("hello world", "hello world")
	if score != 1.0 {
		t.Errorf("expected 1.0 for identical tokens, got %f", score)
	}

	// No overlap
	score = tokenOverlap("hello world", "foo bar")
	if score != 0.0 {
		t.Errorf("expected 0.0 for no overlap, got %f", score)
	}

	// Partial overlap
	score = tokenOverlap("hello world foo", "hello bar")
	if score <= 0.0 || score >= 1.0 {
		t.Errorf("expected partial overlap score, got %f", score)
	}
}

func TestIsModelNumber(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"DCD771C2", true},
		{"UN55TU7000", true},
		{"M18", true},
		{"Hello", false},
		{"12345", false},
		{"abc", false},
	}

	for _, tt := range tests {
		result := isModelNumber(tt.input)
		if result != tt.expected {
			t.Errorf("isModelNumber(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestConfigSchema(t *testing.T) {
	p := New()
	schema := p.ConfigSchema()
	if len(schema) == 0 {
		t.Fatal("ConfigSchema() returned empty slice")
	}

	keys := make(map[string]bool)
	for _, field := range schema {
		keys[field.Key] = true
	}

	expected := []string{"manualslib_enabled", "auto_suggest", "confidence_threshold", "max_suggestions"}
	for _, key := range expected {
		if !keys[key] {
			t.Errorf("expected config key %q not found", key)
		}
	}
}

func TestConfigure(t *testing.T) {
	p := New()

	err := p.Configure(map[string]string{
		"manualslib_enabled":   "false",
		"auto_suggest":         "false",
		"confidence_threshold": "0.5",
		"max_suggestions":      "10",
	})
	if err != nil {
		t.Fatalf("Configure() error: %v", err)
	}

	if p.manualsLibEnabled {
		t.Error("expected manualsLibEnabled=false")
	}
	if p.autoSuggest {
		t.Error("expected autoSuggest=false")
	}
	if p.confidenceThreshold != 0.5 {
		t.Errorf("expected confidenceThreshold=0.5, got %f", p.confidenceThreshold)
	}
	if p.maxSuggestions != 10 {
		t.Errorf("expected maxSuggestions=10, got %d", p.maxSuggestions)
	}
}

func TestRequestedPermissions(t *testing.T) {
	p := New()
	perms := p.RequestedPermissions()
	if len(perms) == 0 {
		t.Fatal("RequestedPermissions() returned empty slice")
	}
}

func TestManualCategories(t *testing.T) {
	if len(manualCategories) == 0 {
		t.Fatal("manualCategories is empty")
	}

	for _, cat := range manualCategories {
		slug, ok := cat["slug"]
		if !ok || slug == "" {
			t.Error("manual category missing slug")
		}
		label, ok := cat["label"]
		if !ok || label == "" {
			t.Errorf("manual category %q missing label", slug)
		}
	}
}

func TestParseItemName(t *testing.T) {
	brand, model, year, _ := parseItemName("2018 Toyota Tacoma SR5")
	if brand != "Toyota" {
		t.Errorf("expected brand 'Toyota', got %q", brand)
	}
	if year != "2018" {
		t.Errorf("expected year '2018', got %q", year)
	}
	// Model might or might not be detected depending on heuristics
	_ = model

	brand, model, _, _ = parseItemName("DeWalt DCD771C2 Drill/Driver")
	if brand != "DeWalt" {
		t.Errorf("expected brand 'DeWalt', got %q", brand)
	}
	if model != "DCD771C2" {
		t.Errorf("expected model 'DCD771C2', got %q", model)
	}
}
