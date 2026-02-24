package aivision

import (
	"encoding/json"
	"regexp"
	"strings"
)

// 4-stage JSON extraction with fallbacks, matching the Python implementation.
// Handles code fences, raw brackets, and full-text parsing.

var (
	jsonFenceRe   = regexp.MustCompile("(?s)```json\\s*\\n?(.*?)\\n?```")
	genericFenceRe = regexp.MustCompile("(?s)```\\s*\\n?(.*?)\\n?```")
	bracketRe     = regexp.MustCompile(`(?s)\[.*\]`)
)

// ExtractJSONFromText attempts to parse a JSON array from LLM text output.
// Uses a 4-stage fallback strategy:
//  1. Try ```json ... ``` code fence
//  2. Try generic ``` ... ``` fence
//  3. Regex search for outermost [...] bracket pair
//  4. Try parsing entire text as JSON
//  5. Return empty slice if all fail
func ExtractJSONFromText(text string) []IdentifiedItem {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	// Stage 1: JSON code fence
	if matches := jsonFenceRe.FindStringSubmatch(text); len(matches) > 1 {
		if items := tryParse(matches[1]); items != nil {
			return items
		}
	}

	// Stage 2: Generic code fence
	if matches := genericFenceRe.FindStringSubmatch(text); len(matches) > 1 {
		if items := tryParse(matches[1]); items != nil {
			return items
		}
	}

	// Stage 3: Outermost bracket pair
	if match := bracketRe.FindString(text); match != "" {
		if items := tryParse(match); items != nil {
			return items
		}
	}

	// Stage 4: Try entire text
	if items := tryParse(text); items != nil {
		return items
	}

	return nil
}

// tryParse attempts to unmarshal text as a JSON array of IdentifiedItem.
func tryParse(text string) []IdentifiedItem {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	// Try array format first
	var items []IdentifiedItem
	if err := json.Unmarshal([]byte(text), &items); err == nil {
		return items
	}

	// Try single object wrapped in array
	var single IdentifiedItem
	if err := json.Unmarshal([]byte(text), &single); err == nil {
		return []IdentifiedItem{single}
	}

	// Try array of generic maps (flexible parsing)
	var maps []map[string]any
	if err := json.Unmarshal([]byte(text), &maps); err == nil {
		return mapsToItems(maps)
	}

	return nil
}

// mapsToItems converts generic maps to IdentifiedItem structs.
func mapsToItems(maps []map[string]any) []IdentifiedItem {
	items := make([]IdentifiedItem, 0, len(maps))
	for _, m := range maps {
		item := IdentifiedItem{
			Name:              getString(m, "name"),
			Quantity:          getInt(m, "quantity", 1),
			SerialNumber:      getString(m, "serial_number"),
			PartNumber:        getString(m, "part_number"),
			Description:       getString(m, "description"),
			EstimatedCategory: getString(m, "estimated_category"),
			LabelText:         getString(m, "label_text"),
			Confidence:        getInt(m, "confidence", 0),
			SuggestedLocation: getString(m, "suggested_location"),
			Notes:             getString(m, "notes"),
		}
		if labels, ok := m["suggested_labels"]; ok {
			if arr, ok := labels.([]any); ok {
				for _, l := range arr {
					if s, ok := l.(string); ok {
						item.SuggestedLabels = append(item.SuggestedLabels, s)
					}
				}
			}
		}
		items = append(items, item)
	}
	return items
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getInt(m map[string]any, key string, def int) int {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return def
}
