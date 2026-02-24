package aivision

import (
	"strings"
	"testing"
)

func TestBuildStep1Prompt_Basic(t *testing.T) {
	prompt := BuildStep1Prompt(0, 1, nil, "")

	if !strings.Contains(prompt, "expert hardware identification") {
		t.Error("expected hardware identification instruction")
	}
	if !strings.Contains(prompt, "image 1 of 1") {
		t.Error("expected image count")
	}
	if !strings.Contains(prompt, "JSON array") {
		t.Error("expected JSON output instruction")
	}
}

func TestBuildStep1Prompt_WithContextItems(t *testing.T) {
	contextItems := []string{"Supermicro X9DRi-F", "Samsung DDR3 RAM"}
	prompt := BuildStep1Prompt(1, 3, contextItems, "")

	if !strings.Contains(prompt, "Previously identified items") {
		t.Error("expected context items section")
	}
	if !strings.Contains(prompt, "Supermicro X9DRi-F") {
		t.Error("expected first context item")
	}
	if !strings.Contains(prompt, "Samsung DDR3 RAM") {
		t.Error("expected second context item")
	}
	if !strings.Contains(prompt, "image 2 of 3") {
		t.Errorf("expected 'image 2 of 3' (0-indexed input), got prompt without it")
	}
}

func TestBuildStep1Prompt_WithUserContext(t *testing.T) {
	prompt := BuildStep1Prompt(0, 1, nil, "These are server rack parts")

	if !strings.Contains(prompt, "User context: These are server rack parts") {
		t.Error("expected user context in prompt")
	}
}

func TestBuildStep1Prompt_NoContextSections(t *testing.T) {
	prompt := BuildStep1Prompt(0, 1, nil, "")

	if strings.Contains(prompt, "Previously identified") {
		t.Error("should not contain context section when no items")
	}
	if strings.Contains(prompt, "User context:") {
		t.Error("should not contain user context section when empty")
	}
}

func TestBuildStep1Prompt_HardwareKeywords(t *testing.T) {
	prompt := BuildStep1Prompt(0, 1, nil, "")

	keywords := []string{
		"Silk-screen", "Serial number", "MAC addresses",
		"IPMI", "SAS", "RAM module", "Model numbers",
	}
	for _, kw := range keywords {
		if !strings.Contains(prompt, kw) {
			t.Errorf("expected keyword %q in prompt", kw)
		}
	}
}

func TestBuildStep2Prompt_Basic(t *testing.T) {
	rawJSON := `[{"name": "Test Item"}]`
	prompt := BuildStep2Prompt(rawJSON, nil, nil, "")

	if !strings.Contains(prompt, "verification and inventory classification") {
		t.Error("expected verification instruction")
	}
	if !strings.Contains(prompt, rawJSON) {
		t.Error("expected raw items JSON in prompt")
	}
	if !strings.Contains(prompt, "confidence score") {
		t.Error("expected confidence instruction")
	}
}

func TestBuildStep2Prompt_WithLocations(t *testing.T) {
	locations := []string{"Server Rack U4", "Yellow Bin #3", "Alex Drawer 1"}
	prompt := BuildStep2Prompt("[]", locations, nil, "")

	if !strings.Contains(prompt, "Available storage locations") {
		t.Error("expected locations section")
	}
	for _, loc := range locations {
		if !strings.Contains(prompt, loc) {
			t.Errorf("expected location %q in prompt", loc)
		}
	}
}

func TestBuildStep2Prompt_WithLabels(t *testing.T) {
	labels := []string{"Electronics", "Server", "Networking"}
	prompt := BuildStep2Prompt("[]", nil, labels, "")

	if !strings.Contains(prompt, "Available labels/tags") {
		t.Error("expected labels section")
	}
	for _, label := range labels {
		if !strings.Contains(prompt, label) {
			t.Errorf("expected label %q in prompt", label)
		}
	}
}

func TestBuildStep2Prompt_WithUserContext(t *testing.T) {
	prompt := BuildStep2Prompt("[]", nil, nil, "Inventory for home lab")

	if !strings.Contains(prompt, "User context: Inventory for home lab") {
		t.Error("expected user context")
	}
}

func TestBuildStep2Prompt_NoOptionalSections(t *testing.T) {
	prompt := BuildStep2Prompt("[]", nil, nil, "")

	if strings.Contains(prompt, "Available storage locations") {
		t.Error("should not contain locations section when empty")
	}
	if strings.Contains(prompt, "Available labels/tags") {
		t.Error("should not contain labels section when empty")
	}
	if strings.Contains(prompt, "User context:") {
		t.Error("should not contain user context when empty")
	}
}

func TestBuildStep2Prompt_VerificationCriteria(t *testing.T) {
	prompt := BuildStep2Prompt("[]", nil, nil, "")

	criteria := []string{
		"server vs. consumer", "ECC RAM",
		"RAM specs", "CPU-socket",
		"confidence score", "correction notes",
	}
	for _, c := range criteria {
		if !strings.Contains(prompt, c) {
			t.Errorf("expected verification criterion %q in prompt", c)
		}
	}
}
