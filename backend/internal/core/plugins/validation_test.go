package plugins

import (
	"strings"
	"testing"
)

func TestValidatePluginName_Valid(t *testing.T) {
	validNames := []string{
		"my-plugin",
		"ai-vision",
		"a1b",
		"abc",
		"plugin-name-with-hyphens",
		"123",
		"a-b",
		"test-plugin-v2",
		"x1y",
		"My-Plugin",
		"ABC",
	}

	for _, name := range validNames {
		t.Run(name, func(t *testing.T) {
			if err := ValidatePluginName(name); err != nil {
				t.Errorf("ValidatePluginName(%q) = error %v, want nil", name, err)
			}
		})
	}
}

func TestValidatePluginName_Invalid(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		errContains string
	}{
		{
			name:        "too short (2 chars)",
			input:       "ab",
			errContains: "at least 3",
		},
		{
			name:        "too short (1 char)",
			input:       "a",
			errContains: "at least 3",
		},
		{
			name:        "empty string",
			input:       "",
			errContains: "at least 3",
		},
		{
			name:        "starts with hyphen",
			input:       "-start",
			errContains: "hyphen",
		},
		{
			name:        "ends with hyphen",
			input:       "end-",
			errContains: "hyphen",
		},
		{
			name:        "too long (51 chars)",
			input:       strings.Repeat("a", 51),
			errContains: "at most 50",
		},
		{
			name:        "underscore",
			input:       "my_plugin",
			errContains: "alphanumeric",
		},
		{
			name:        "special characters",
			input:       "my@plugin",
			errContains: "alphanumeric",
		},
		{
			name:        "dot in name",
			input:       "my.plugin",
			errContains: "alphanumeric",
		},
		{
			name:        "contains space",
			input:       "my plugin",
			errContains: "alphanumeric",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePluginName(tt.input)
			if err == nil {
				t.Errorf("ValidatePluginName(%q) = nil, want error", tt.input)
				return
			}
			if !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("ValidatePluginName(%q) error = %q, want to contain %q", tt.input, err.Error(), tt.errContains)
			}
		})
	}
}

func TestValidatePluginName_BoundaryLength(t *testing.T) {
	// Exactly 3 characters should be valid.
	if err := ValidatePluginName("abc"); err != nil {
		t.Errorf("3-char name should be valid, got error: %v", err)
	}

	// Exactly 50 characters should be valid.
	name50 := "a" + strings.Repeat("b", 48) + "c"
	if err := ValidatePluginName(name50); err != nil {
		t.Errorf("50-char name should be valid, got error: %v", err)
	}
}

func TestValidateConfigValue_String(t *testing.T) {
	field := ConfigField{Key: "test", Type: "string"}

	tests := []struct {
		value string
		valid bool
	}{
		{"hello world", true},
		{"", true},
		{"any string is valid", true},
		{"123", true},
		{"special chars: !@#$%", true},
	}

	for _, tt := range tests {
		err := ValidateConfigValue(field, tt.value)
		if tt.valid && err != nil {
			t.Errorf("ValidateConfigValue(%q, string) = error %v, want nil", tt.value, err)
		}
		if !tt.valid && err == nil {
			t.Errorf("ValidateConfigValue(%q, string) = nil, want error", tt.value)
		}
	}
}

func TestValidateConfigValue_Number(t *testing.T) {
	field := ConfigField{Key: "num", Type: "number"}

	tests := []struct {
		value string
		valid bool
	}{
		{"123", true},
		{"12.5", true},
		{"0", true},
		{"-42", true},
		{"-3.14", true},
		{"abc", false},
		{"12.5.6", false},
		// Empty value for a non-required field is valid.
		{"", true},
		{"one", false},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			err := ValidateConfigValue(field, tt.value)
			if tt.valid && err != nil {
				t.Errorf("ValidateConfigValue(%q, number) = error %v, want nil", tt.value, err)
			}
			if !tt.valid && err == nil {
				t.Errorf("ValidateConfigValue(%q, number) = nil, want error", tt.value)
			}
		})
	}
}

func TestValidateConfigValue_Boolean(t *testing.T) {
	field := ConfigField{Key: "flag", Type: "boolean"}

	tests := []struct {
		value string
		valid bool
	}{
		{"true", true},
		{"false", true},
		{"True", true},
		{"FALSE", true},
		{"yes", false},
		{"no", false},
		{"1", false},
		{"0", false},
		// Empty value for a non-required field is valid.
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			err := ValidateConfigValue(field, tt.value)
			if tt.valid && err != nil {
				t.Errorf("ValidateConfigValue(%q, boolean) = error %v, want nil", tt.value, err)
			}
			if !tt.valid && err == nil {
				t.Errorf("ValidateConfigValue(%q, boolean) = nil, want error", tt.value)
			}
		})
	}
}

func TestValidateConfigValue_Select(t *testing.T) {
	field := ConfigField{
		Key:     "level",
		Type:    "select",
		Options: []string{"low", "medium", "high"},
	}

	tests := []struct {
		value string
		valid bool
	}{
		{"low", true},
		{"medium", true},
		{"high", true},
		{"ultra", false},
		{"Low", false},
		// Empty value for a non-required field is valid.
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			err := ValidateConfigValue(field, tt.value)
			if tt.valid && err != nil {
				t.Errorf("ValidateConfigValue(%q, select) = error %v, want nil", tt.value, err)
			}
			if !tt.valid && err == nil {
				t.Errorf("ValidateConfigValue(%q, select) = nil, want error", tt.value)
			}
		})
	}
}

func TestValidateConfigValue_Secret(t *testing.T) {
	field := ConfigField{Key: "secret", Type: "secret"}

	// Secrets should behave like strings (any value is valid).
	if err := ValidateConfigValue(field, "my-secret-key-123"); err != nil {
		t.Errorf("ValidateConfigValue(secret) = error %v, want nil", err)
	}
}

func TestValidateConfigValue_UnknownType(t *testing.T) {
	field := ConfigField{Key: "custom", Type: "custom"}

	err := ValidateConfigValue(field, "value")
	if err == nil {
		t.Error("expected error for unknown config field type")
	}
}

func TestValidateConfigValue_Required(t *testing.T) {
	field := ConfigField{Key: "required_field", Type: "string", Required: true}

	// Empty value for a required field should fail.
	err := ValidateConfigValue(field, "")
	if err == nil {
		t.Error("expected error for empty required field")
	}

	// Non-empty value should pass.
	err = ValidateConfigValue(field, "value")
	if err != nil {
		t.Errorf("ValidateConfigValue(required, non-empty) = error %v, want nil", err)
	}
}

func TestSanitizeConfigValue(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "trims whitespace",
			input: "  hello world  ",
			want:  "hello world",
		},
		{
			name:  "strips control characters",
			input: "hello\x00world\x01test",
			want:  "helloworldtest",
		},
		{
			name:  "preserves newlines and tabs",
			input: "line1\nline2\ttab",
			want:  "line1\nline2\ttab",
		},
		{
			name:  "truncates at 4096",
			input: strings.Repeat("a", 5000),
			want:  strings.Repeat("a", 4096),
		},
		{
			name:  "empty string stays empty",
			input: "",
			want:  "",
		},
		{
			name:  "only whitespace",
			input: "   ",
			want:  "",
		},
		{
			name:  "mixed control chars and whitespace",
			input: "  \x00hello\x01  ",
			want:  "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeConfigValue(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeConfigValue() = %q (len %d), want %q (len %d)",
					got, len(got), tt.want, len(tt.want))
			}
		})
	}
}

func TestValidateWebhookURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		valid       bool
		errContains string
	}{
		{
			name:  "valid HTTPS URL",
			url:   "https://example.com/webhook",
			valid: true,
		},
		{
			name:  "valid HTTPS with port",
			url:   "https://hooks.slack.com:443/services/T00/B00/xxx",
			valid: true,
		},
		{
			name:        "HTTP not allowed",
			url:         "http://example.com/webhook",
			valid:       false,
			errContains: "HTTPS",
		},
		{
			name:        "FTP not allowed",
			url:         "ftp://example.com/file",
			valid:       false,
			errContains: "HTTPS",
		},
		{
			name:        "private IP 192.168.x.x",
			url:         "https://192.168.1.1/hook",
			valid:       false,
			errContains: "private",
		},
		{
			name:        "private IP 10.x.x.x",
			url:         "https://10.0.0.1/hook",
			valid:       false,
			errContains: "private",
		},
		{
			name:        "private IP 172.16.x.x",
			url:         "https://172.16.0.1/hook",
			valid:       false,
			errContains: "private",
		},
		{
			name:        "loopback 127.0.0.1",
			url:         "https://127.0.0.1/hook",
			valid:       false,
			errContains: "private",
		},
		{
			name:  "localhost with HTTP allowed for dev",
			url:   "http://localhost/hook",
			valid: true,
		},
		{
			name:        "link-local 169.254.x.x",
			url:         "https://169.254.1.1/hook",
			valid:       false,
			errContains: "private",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWebhookURL(tt.url)
			if tt.valid && err != nil {
				t.Errorf("ValidateWebhookURL(%q) = error %v, want nil", tt.url, err)
			}
			if !tt.valid {
				if err == nil {
					t.Errorf("ValidateWebhookURL(%q) = nil, want error", tt.url)
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %q, want to contain %q", err.Error(), tt.errContains)
				}
			}
		})
	}
}
