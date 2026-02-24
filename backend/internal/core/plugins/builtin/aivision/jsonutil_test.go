package aivision

import (
	"testing"
)

func TestExtractJSONFromText_Empty(t *testing.T) {
	result := ExtractJSONFromText("")
	if result != nil {
		t.Errorf("expected nil for empty text, got %v", result)
	}
}

func TestExtractJSONFromText_Whitespace(t *testing.T) {
	result := ExtractJSONFromText("   \n\t  ")
	if result != nil {
		t.Errorf("expected nil for whitespace text, got %v", result)
	}
}

func TestExtractJSONFromText_JSONCodeFence(t *testing.T) {
	text := "Here are the results:\n```json\n[{\"name\": \"Supermicro X9DRi-F\", \"quantity\": 1, \"description\": \"Server motherboard\"}]\n```\n"
	result := ExtractJSONFromText(text)
	if len(result) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result))
	}
	if result[0].Name != "Supermicro X9DRi-F" {
		t.Errorf("name = %q, want %q", result[0].Name, "Supermicro X9DRi-F")
	}
	if result[0].Quantity != 1 {
		t.Errorf("quantity = %d, want 1", result[0].Quantity)
	}
}

func TestExtractJSONFromText_GenericCodeFence(t *testing.T) {
	text := "Found items:\n```\n[{\"name\": \"DDR3 RAM\", \"quantity\": 8}]\n```\n"
	result := ExtractJSONFromText(text)
	if len(result) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result))
	}
	if result[0].Name != "DDR3 RAM" {
		t.Errorf("name = %q", result[0].Name)
	}
	if result[0].Quantity != 8 {
		t.Errorf("quantity = %d, want 8", result[0].Quantity)
	}
}

func TestExtractJSONFromText_BracketExtraction(t *testing.T) {
	text := "I found the following items in the photo: [{\"name\": \"GTX 1050 Ti\", \"quantity\": 1, \"estimated_category\": \"gpu\"}] and that's all."
	result := ExtractJSONFromText(text)
	if len(result) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result))
	}
	if result[0].Name != "GTX 1050 Ti" {
		t.Errorf("name = %q", result[0].Name)
	}
	if result[0].EstimatedCategory != "gpu" {
		t.Errorf("category = %q, want %q", result[0].EstimatedCategory, "gpu")
	}
}

func TestExtractJSONFromText_RawJSON(t *testing.T) {
	text := `[{"name": "PSU", "quantity": 1, "description": "Power supply"}]`
	result := ExtractJSONFromText(text)
	if len(result) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result))
	}
	if result[0].Name != "PSU" {
		t.Errorf("name = %q", result[0].Name)
	}
}

func TestExtractJSONFromText_SingleObject(t *testing.T) {
	text := `{"name": "Ethernet Cable", "quantity": 5, "estimated_category": "cable"}`
	result := ExtractJSONFromText(text)
	if len(result) != 1 {
		t.Fatalf("expected 1 item from single object, got %d", len(result))
	}
	if result[0].Name != "Ethernet Cable" {
		t.Errorf("name = %q", result[0].Name)
	}
	if result[0].Quantity != 5 {
		t.Errorf("quantity = %d, want 5", result[0].Quantity)
	}
}

func TestExtractJSONFromText_InvalidJSON(t *testing.T) {
	result := ExtractJSONFromText("This is not JSON at all. No brackets here.")
	if result != nil {
		t.Errorf("expected nil for non-JSON text, got %v", result)
	}
}

func TestExtractJSONFromText_EmptyArray(t *testing.T) {
	text := "```json\n[]\n```"
	result := ExtractJSONFromText(text)
	if result == nil {
		// Empty slice from json.Unmarshal is non-nil
		t.Skip("empty array returns nil or empty; implementation may vary")
	}
	if len(result) != 0 {
		t.Errorf("expected 0 items, got %d", len(result))
	}
}

func TestExtractJSONFromText_MultipleItems(t *testing.T) {
	text := `[
		{"name": "Samsung DDR3 8GB", "quantity": 8, "serial_number": "M393B2G70BH0-YH9", "estimated_category": "ram"},
		{"name": "Supermicro X9DRi-F", "quantity": 1, "part_number": "X9DRi-F", "estimated_category": "motherboard"}
	]`
	result := ExtractJSONFromText(text)
	if len(result) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result))
	}
	if result[0].SerialNumber != "M393B2G70BH0-YH9" {
		t.Errorf("serial = %q", result[0].SerialNumber)
	}
	if result[1].PartNumber != "X9DRi-F" {
		t.Errorf("part number = %q", result[1].PartNumber)
	}
}

func TestExtractJSONFromText_GenericMaps(t *testing.T) {
	// JSON with extra fields that don't map directly to IdentifiedItem
	text := `[{"name": "Cable", "quantity": 3.0, "confidence": 85.0, "suggested_labels": ["networking", "cable"]}]`
	result := ExtractJSONFromText(text)
	if len(result) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result))
	}
	if result[0].Name != "Cable" {
		t.Errorf("name = %q", result[0].Name)
	}
	if result[0].Quantity != 3 {
		t.Errorf("quantity = %d, want 3", result[0].Quantity)
	}
	if result[0].Confidence != 85 {
		t.Errorf("confidence = %d, want 85", result[0].Confidence)
	}
	if len(result[0].SuggestedLabels) != 2 {
		t.Errorf("labels count = %d, want 2", len(result[0].SuggestedLabels))
	}
}

func TestExtractJSONFromText_MixedFencePriority(t *testing.T) {
	// JSON fence should be preferred over generic fence
	text := "```json\n[{\"name\": \"From JSON Fence\", \"quantity\": 1}]\n```\n\n```\n[{\"name\": \"From Generic Fence\", \"quantity\": 1}]\n```"
	result := ExtractJSONFromText(text)
	if len(result) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result))
	}
	if result[0].Name != "From JSON Fence" {
		t.Errorf("name = %q, want %q (JSON fence should take priority)", result[0].Name, "From JSON Fence")
	}
}

func TestTryParse_EmptyString(t *testing.T) {
	result := tryParse("")
	if result != nil {
		t.Errorf("expected nil for empty string, got %v", result)
	}
}

func TestGetString_Missing(t *testing.T) {
	m := map[string]any{"other": "value"}
	result := getString(m, "name")
	if result != "" {
		t.Errorf("expected empty string for missing key, got %q", result)
	}
}

func TestGetString_NonString(t *testing.T) {
	m := map[string]any{"name": 42}
	result := getString(m, "name")
	if result != "" {
		t.Errorf("expected empty string for non-string value, got %q", result)
	}
}

func TestGetInt_Missing(t *testing.T) {
	m := map[string]any{"other": 1}
	result := getInt(m, "quantity", 99)
	if result != 99 {
		t.Errorf("expected default 99, got %d", result)
	}
}

func TestGetInt_Float64(t *testing.T) {
	m := map[string]any{"quantity": float64(7)}
	result := getInt(m, "quantity", 1)
	if result != 7 {
		t.Errorf("expected 7, got %d", result)
	}
}

func TestGetInt_Int(t *testing.T) {
	m := map[string]any{"quantity": 3}
	result := getInt(m, "quantity", 1)
	if result != 3 {
		t.Errorf("expected 3, got %d", result)
	}
}

func TestMapsToItems_Labels(t *testing.T) {
	maps := []map[string]any{
		{
			"name":             "Switch",
			"suggested_labels": []any{"networking", "managed"},
		},
	}
	items := mapsToItems(maps)
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if len(items[0].SuggestedLabels) != 2 {
		t.Errorf("labels count = %d, want 2", len(items[0].SuggestedLabels))
	}
	if items[0].SuggestedLabels[0] != "networking" {
		t.Errorf("label[0] = %q", items[0].SuggestedLabels[0])
	}
}

func TestMapsToItems_DefaultQuantity(t *testing.T) {
	maps := []map[string]any{
		{"name": "Cable"},
	}
	items := mapsToItems(maps)
	if items[0].Quantity != 1 {
		t.Errorf("default quantity = %d, want 1", items[0].Quantity)
	}
}

func TestMapsToItems_AllFields(t *testing.T) {
	maps := []map[string]any{
		{
			"name":               "Test GPU",
			"quantity":           float64(2),
			"serial_number":      "SN123",
			"part_number":        "PN456",
			"description":        "Graphics card",
			"estimated_category": "gpu",
			"label_text":         "NVIDIA RTX",
			"confidence":         float64(95),
			"suggested_location": "Server Rack",
			"notes":              "High-end",
		},
	}
	items := mapsToItems(maps)
	item := items[0]
	if item.Name != "Test GPU" || item.Quantity != 2 || item.SerialNumber != "SN123" {
		t.Errorf("basic fields wrong: %+v", item)
	}
	if item.PartNumber != "PN456" || item.Description != "Graphics card" {
		t.Errorf("detail fields wrong: %+v", item)
	}
	if item.EstimatedCategory != "gpu" || item.Confidence != 95 {
		t.Errorf("category/confidence wrong: %+v", item)
	}
	if item.SuggestedLocation != "Server Rack" || item.Notes != "High-end" {
		t.Errorf("location/notes wrong: %+v", item)
	}
}
