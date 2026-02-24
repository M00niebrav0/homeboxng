package plugins

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// ---------------------------------------------------------------------------
// Plugin name validation
// ---------------------------------------------------------------------------

// pluginNameRe matches alphanumeric characters and hyphens, 3-50 chars.
var pluginNameRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]{1,48}[a-zA-Z0-9]$`)

// ValidatePluginName checks that name is a valid plugin identifier.
//
// Rules:
//   - 3-50 characters long
//   - Only alphanumeric characters and hyphens
//   - Must not start or end with a hyphen
func ValidatePluginName(name string) error {
	if len(name) < 3 {
		return errors.New("plugin name must be at least 3 characters")
	}
	if len(name) > 50 {
		return errors.New("plugin name must be at most 50 characters")
	}
	if strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") {
		return errors.New("plugin name must not start or end with a hyphen")
	}
	if !pluginNameRe.MatchString(name) {
		return errors.New("plugin name may only contain alphanumeric characters and hyphens")
	}
	return nil
}

// ---------------------------------------------------------------------------
// Config value validation
// ---------------------------------------------------------------------------

// ValidateConfigValue checks that value is appropriate for the given ConfigField
// definition (from plugin.go). It validates based on the field's Type string.
func ValidateConfigValue(field ConfigField, value string) error {
	if field.Required && strings.TrimSpace(value) == "" {
		return fmt.Errorf("config field %q is required", field.Key)
	}

	// An empty value for a non-required field is always valid.
	if strings.TrimSpace(value) == "" {
		return nil
	}

	switch field.Type {
	case "number":
		if _, err := strconv.ParseFloat(value, 64); err != nil {
			return fmt.Errorf("config field %q must be a valid number", field.Key)
		}
	case "boolean":
		lower := strings.ToLower(strings.TrimSpace(value))
		if lower != "true" && lower != "false" {
			return fmt.Errorf("config field %q must be true or false", field.Key)
		}
	case "select":
		found := false
		for _, opt := range field.Options {
			if opt == value {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("config field %q must be one of: %s", field.Key, strings.Join(field.Options, ", "))
		}
	case "string", "secret", "text", "url", "textarea":
		// No additional validation for free-form text types.
	default:
		return fmt.Errorf("unknown config field type %q for field %q", field.Type, field.Key)
	}

	return nil
}

// ---------------------------------------------------------------------------
// Sanitisation
// ---------------------------------------------------------------------------

const maxConfigValueLength = 4096

// SanitizeConfigValue normalises a configuration value by trimming whitespace,
// stripping control characters, and truncating to 4096 characters.
func SanitizeConfigValue(value string) string {
	// Strip control characters (except common whitespace: \t, \n, \r).
	cleaned := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\t' && r != '\n' && r != '\r' {
			return -1
		}
		return r
	}, value)

	cleaned = strings.TrimSpace(cleaned)

	if len(cleaned) > maxConfigValueLength {
		cleaned = cleaned[:maxConfigValueLength]
	}

	return cleaned
}

// ---------------------------------------------------------------------------
// Permission request validation
// ---------------------------------------------------------------------------

// KnownPermissions is the set of permissions that may be requested by plugins.
// Maps Permission constants from permissions.go to empty struct for O(1) lookup.
var KnownPermissions = map[Permission]struct{}{
	PermReadItems:        {},
	PermWriteItems:       {},
	PermReadLocations:    {},
	PermWriteLocations:   {},
	PermReadTags:         {},
	PermWriteTags:        {},
	PermReadAttachments:  {},
	PermWriteAttachments: {},
	PermReadUsers:        {},
	PermReadGroups:       {},
	PermMaintenance:      {},
	PermTemplates:        {},
	PermNotifiers:        {},
	PermLabels:           {},
	PermImportExport:     {},
	PermEvents:           {},
	PermScheduledTasks:   {},
	PermAPIRoutes:        {},
	PermWebUI:            {},
	PermConfig:           {},
	PermNetwork:          {},
	PermWebhooks:         {},
	PermStorage:          {},
	PermDatabase:         {},
}

// ValidatePermissionRequest checks that the requested permission is in the
// known set and that required fields are populated.
// Uses the PermissionRequest type from permissions.go.
func ValidatePermissionRequest(pluginName string, req PermissionRequest) error {
	if strings.TrimSpace(pluginName) == "" {
		return errors.New("permission request: plugin name is required")
	}
	if strings.TrimSpace(string(req.Permission)) == "" {
		return errors.New("permission request: permission is required")
	}
	if _, ok := KnownPermissions[req.Permission]; !ok {
		return fmt.Errorf("permission request: unknown permission %q", req.Permission)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Webhook URL validation
// ---------------------------------------------------------------------------

// ValidateWebhookURL checks that rawURL is a well-formed URL suitable for use
// as a webhook endpoint.
//
// Rules:
//   - Must be a valid URL
//   - Must use HTTPS (HTTP is only allowed for localhost, for development)
//   - Must not point to a private IP address
func ValidateWebhookURL(rawURL string) error {
	if strings.TrimSpace(rawURL) == "" {
		return errors.New("webhook URL must not be empty")
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("webhook URL is not a valid URL: %w", err)
	}

	if parsed.Scheme == "" || parsed.Host == "" {
		return errors.New("webhook URL must include a scheme and host")
	}

	hostname := parsed.Hostname()

	// Allow plain HTTP only for localhost/loopback (development convenience).
	isLocalhost := hostname == "localhost" || hostname == "127.0.0.1" || hostname == "::1"

	if parsed.Scheme != "https" {
		if parsed.Scheme == "http" && isLocalhost {
			// Acceptable for development.
			return nil
		}
		return errors.New("webhook URL must use HTTPS")
	}

	// Block private IPs (even over HTTPS) to prevent SSRF.
	if ip := net.ParseIP(hostname); ip != nil {
		if isPrivateIP(ip) {
			return fmt.Errorf("webhook URL must not point to a private IP address (%s)", hostname)
		}
	}

	return nil
}
