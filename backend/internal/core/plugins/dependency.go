package plugins

import (
	"fmt"
	"strconv"
	"strings"
)

// Dependency declares that a plugin depends on another plugin.
type Dependency struct {
	// Name is the name of the required plugin.
	Name string `json:"name"`
	// MinVersion is the minimum required version (semver, e.g., "1.0.0").
	// Empty string means any version is acceptable.
	MinVersion string `json:"minVersion,omitempty"`
	// Optional indicates that the plugin can function without this dependency,
	// possibly in a degraded mode.
	Optional bool `json:"optional"`
}

// DependencyPlugin is implemented by plugins that declare dependencies on other plugins.
type DependencyPlugin interface {
	// Dependencies returns the list of plugins this plugin depends on.
	Dependencies() []Dependency
}

// DependencyError describes a problem with a plugin's dependency resolution.
type DependencyError struct {
	// PluginName is the plugin that has the dependency issue.
	PluginName string `json:"pluginName"`
	// DependencyName is the dependency that could not be resolved.
	DependencyName string `json:"dependencyName"`
	// Required is the minimum version required.
	Required string `json:"required"`
	// Available is the version that was found, empty if missing.
	Available string `json:"available"`
	// Type is the kind of error: "missing", "version_mismatch", or "circular".
	Type string `json:"type"`
}

// Error implements the error interface.
func (e DependencyError) Error() string {
	switch e.Type {
	case "missing":
		return fmt.Sprintf("plugin %q requires %q which is not available", e.PluginName, e.DependencyName)
	case "version_mismatch":
		return fmt.Sprintf("plugin %q requires %q >= %s but found %s", e.PluginName, e.DependencyName, e.Required, e.Available)
	case "circular":
		return fmt.Sprintf("circular dependency detected involving %q and %q", e.PluginName, e.DependencyName)
	default:
		return fmt.Sprintf("dependency error for plugin %q: %s (type: %s)", e.PluginName, e.DependencyName, e.Type)
	}
}

// DependencyResolver resolves plugin load order based on declared dependencies.
type DependencyResolver struct{}

// NewDependencyResolver creates a new DependencyResolver.
func NewDependencyResolver() *DependencyResolver {
	return &DependencyResolver{}
}

// Resolve performs a topological sort on the provided plugins based on their
// declared dependencies. Returns the plugins in a safe initialization order
// (dependencies before dependents) or an error if resolution fails.
//
// Plugins that do not implement DependencyPlugin are treated as having no
// dependencies and can appear in any position in the result.
func (dr *DependencyResolver) Resolve(plugins []Plugin) ([]Plugin, error) {
	// Build a name-to-plugin index and extract version info.
	byName := make(map[string]Plugin, len(plugins))
	versions := make(map[string]string, len(plugins))
	for _, p := range plugins {
		info := p.Info()
		byName[info.Name] = p
		versions[info.Name] = info.Version
	}

	// Build adjacency list: edges go from dependency -> dependent.
	// Also collect all dependency declarations for validation.
	adj := make(map[string][]string)           // dependency -> list of dependents
	inDegree := make(map[string]int, len(plugins))

	for _, p := range plugins {
		name := p.Info().Name
		if _, exists := inDegree[name]; !exists {
			inDegree[name] = 0
		}
	}

	for _, p := range plugins {
		dp, ok := p.(DependencyPlugin)
		if !ok {
			continue
		}

		name := p.Info().Name
		for _, dep := range dp.Dependencies() {
			// Check if the dependency exists.
			if _, exists := byName[dep.Name]; !exists {
				if !dep.Optional {
					return nil, DependencyError{
						PluginName:     name,
						DependencyName: dep.Name,
						Required:       dep.MinVersion,
						Available:      "",
						Type:           "missing",
					}
				}
				// Skip optional missing dependencies.
				continue
			}

			// Check version constraint.
			if dep.MinVersion != "" {
				available := versions[dep.Name]
				if available != "" && CompareVersions(available, dep.MinVersion) < 0 {
					if !dep.Optional {
						return nil, DependencyError{
							PluginName:     name,
							DependencyName: dep.Name,
							Required:       dep.MinVersion,
							Available:      available,
							Type:           "version_mismatch",
						}
					}
					continue
				}
			}

			// Add edge: dep.Name must come before name.
			adj[dep.Name] = append(adj[dep.Name], name)
			inDegree[name]++
		}
	}

	// Kahn's algorithm for topological sort.
	queue := make([]string, 0)
	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}

	var sorted []string
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		sorted = append(sorted, current)

		for _, dependent := range adj[current] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	// If we didn't sort all plugins, there's a cycle.
	if len(sorted) != len(plugins) {
		// Find plugins involved in the cycle for a better error message.
		for name, degree := range inDegree {
			if degree > 0 {
				// Find one of its dependencies that is also unsorted.
				if dp, ok := byName[name].(DependencyPlugin); ok {
					for _, dep := range dp.Dependencies() {
						if inDegree[dep.Name] > 0 {
							return nil, DependencyError{
								PluginName:     name,
								DependencyName: dep.Name,
								Type:           "circular",
							}
						}
					}
				}
				// Fallback: just report the plugin itself.
				return nil, DependencyError{
					PluginName:     name,
					DependencyName: name,
					Type:           "circular",
				}
			}
		}
	}

	// Build the result in the sorted order.
	result := make([]Plugin, 0, len(sorted))
	for _, name := range sorted {
		if p, ok := byName[name]; ok {
			result = append(result, p)
		}
	}

	return result, nil
}

// Check validates a single plugin's dependencies against the available plugins.
// Returns a list of DependencyErrors (empty if all dependencies are satisfied).
func (dr *DependencyResolver) Check(plugin Plugin, available map[string]string) []DependencyError {
	dp, ok := plugin.(DependencyPlugin)
	if !ok {
		return nil
	}

	name := plugin.Info().Name
	var errors []DependencyError

	for _, dep := range dp.Dependencies() {
		ver, exists := available[dep.Name]
		if !exists {
			errors = append(errors, DependencyError{
				PluginName:     name,
				DependencyName: dep.Name,
				Required:       dep.MinVersion,
				Available:      "",
				Type:           "missing",
			})
			continue
		}

		if dep.MinVersion != "" && ver != "" {
			if CompareVersions(ver, dep.MinVersion) < 0 {
				errors = append(errors, DependencyError{
					PluginName:     name,
					DependencyName: dep.Name,
					Required:       dep.MinVersion,
					Available:      ver,
					Type:           "version_mismatch",
				})
			}
		}
	}

	return errors
}

// CompareVersions performs a simple semver comparison.
// It splits each version string on "." and compares each numeric component.
// Returns:
//
//	-1 if a < b
//	 0 if a == b
//	+1 if a > b
//
// Non-numeric parts are compared as 0.
func CompareVersions(a, b string) int {
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	// Pad the shorter version with zeros.
	maxLen := len(aParts)
	if len(bParts) > maxLen {
		maxLen = len(bParts)
	}

	for i := 0; i < maxLen; i++ {
		var aNum, bNum int
		if i < len(aParts) {
			aNum, _ = strconv.Atoi(aParts[i])
		}
		if i < len(bParts) {
			bNum, _ = strconv.Atoi(bParts[i])
		}

		if aNum < bNum {
			return -1
		}
		if aNum > bNum {
			return 1
		}
	}

	return 0
}
