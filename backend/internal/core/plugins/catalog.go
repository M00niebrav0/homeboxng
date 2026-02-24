package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// CatalogSource represents a plugin source repository (like HACS custom repos).
type CatalogSource struct {
	// Name is the display name for this source.
	Name string `json:"name"`

	// URL is the raw JSON URL for the registry file.
	// For GitHub repos: https://raw.githubusercontent.com/{owner}/{repo}/main/registry.json
	URL string `json:"url"`

	// Type is "official" for the HomeBoxNG default, "community" for user-added repos.
	Type string `json:"type"`

	// Enabled controls whether this source is fetched.
	Enabled bool `json:"enabled"`
}

// DefaultSources returns the official plugin sources.
func DefaultSources() []CatalogSource {
	return []CatalogSource{
		{
			Name:    "HomeBoxNG Official",
			URL:     "https://raw.githubusercontent.com/M00niebrav0/homeboxng-plugins/main/registry.json",
			Type:    "official",
			Enabled: true,
		},
	}
}

// MultiSourceCatalog manages multiple plugin sources (official + user-added).
// This works like HACS: users can add GitHub repos as custom plugin sources.
type MultiSourceCatalog struct {
	mu        sync.RWMutex
	sources   []CatalogSource
	available map[string][]ExternalPluginManifest // keyed by source name
	lastFetch time.Time
	logger    zerolog.Logger
}

// NewMultiSourceCatalog creates a catalog with default official sources.
func NewMultiSourceCatalog(logger zerolog.Logger) *MultiSourceCatalog {
	return &MultiSourceCatalog{
		sources:   DefaultSources(),
		available: make(map[string][]ExternalPluginManifest),
		logger:    logger.With().Str("component", "multi-catalog").Logger(),
	}
}

// AddSource adds a custom plugin source (like HACS custom repositories).
func (c *MultiSourceCatalog) AddSource(source CatalogSource) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check for duplicate URLs
	for _, existing := range c.sources {
		if existing.URL == source.URL {
			return
		}
	}

	c.sources = append(c.sources, source)
	c.logger.Info().Str("name", source.Name).Str("url", source.URL).Msg("plugin source added")
}

// RemoveSource removes a custom plugin source by URL.
// Official sources cannot be removed.
func (c *MultiSourceCatalog) RemoveSource(url string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i, s := range c.sources {
		if s.URL == url {
			if s.Type == "official" {
				return fmt.Errorf("cannot remove official plugin source")
			}
			c.sources = append(c.sources[:i], c.sources[i+1:]...)
			delete(c.available, s.Name)
			return nil
		}
	}
	return fmt.Errorf("source not found: %s", url)
}

// Sources returns all configured plugin sources.
func (c *MultiSourceCatalog) Sources() []CatalogSource {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]CatalogSource, len(c.sources))
	copy(result, c.sources)
	return result
}

// RefreshAll fetches plugin lists from all enabled sources.
func (c *MultiSourceCatalog) RefreshAll(ctx context.Context) error {
	c.mu.RLock()
	sources := make([]CatalogSource, len(c.sources))
	copy(sources, c.sources)
	c.mu.RUnlock()

	client := &http.Client{Timeout: 15 * time.Second}
	var errs []string

	for _, source := range sources {
		if !source.Enabled {
			continue
		}

		manifests, err := fetchRegistry(ctx, client, source.URL)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", source.Name, err))
			c.logger.Warn().Err(err).Str("source", source.Name).Msg("failed to fetch plugin source")
			continue
		}

		c.mu.Lock()
		c.available[source.Name] = manifests
		c.mu.Unlock()

		c.logger.Info().Str("source", source.Name).Int("plugins", len(manifests)).Msg("source refreshed")
	}

	c.mu.Lock()
	c.lastFetch = time.Now()
	c.mu.Unlock()

	if len(errs) > 0 {
		return fmt.Errorf("some sources failed: %s", strings.Join(errs, "; "))
	}
	return nil
}

// AllAvailable returns plugins from all sources, merged and deduplicated.
func (c *MultiSourceCatalog) AllAvailable() []ExternalPluginManifest {
	c.mu.RLock()
	defer c.mu.RUnlock()

	seen := make(map[string]bool)
	var result []ExternalPluginManifest

	// Official sources first
	for _, source := range c.sources {
		manifests := c.available[source.Name]
		for _, m := range manifests {
			if !seen[m.Name] {
				seen[m.Name] = true
				result = append(result, m)
			}
		}
	}

	return result
}

// Search finds plugins matching a query across all sources.
func (c *MultiSourceCatalog) SearchAll(query string) []ExternalPluginManifest {
	all := c.AllAvailable()
	if query == "" {
		return all
	}

	q := strings.ToLower(query)
	var results []ExternalPluginManifest
	for _, m := range all {
		if strings.Contains(strings.ToLower(m.Name), q) ||
			strings.Contains(strings.ToLower(m.Description), q) {
			results = append(results, m)
			continue
		}
		for _, cat := range m.Categories {
			if strings.Contains(strings.ToLower(cat), q) {
				results = append(results, m)
				break
			}
		}
	}
	return results
}

// fetchRegistry fetches and parses a plugin registry from a URL.
func fetchRegistry(ctx context.Context, client *http.Client, registryURL string) ([]ExternalPluginManifest, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, registryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching registry: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("reading registry: %w", err)
	}

	var manifests []ExternalPluginManifest
	if err := json.Unmarshal(body, &manifests); err != nil {
		return nil, fmt.Errorf("parsing registry: %w", err)
	}

	return manifests, nil
}

// AddSourceFromGitHub is a convenience method for adding a GitHub repo as a plugin source.
// It constructs the raw URL from the repo path (e.g., "user/homeboxng-plugin-foo").
func (c *MultiSourceCatalog) AddSourceFromGitHub(repoPath string) {
	url := fmt.Sprintf("https://raw.githubusercontent.com/%s/main/registry.json", repoPath)
	parts := strings.Split(repoPath, "/")
	name := repoPath
	if len(parts) >= 2 {
		name = parts[1]
	}

	c.AddSource(CatalogSource{
		Name:    name,
		URL:     url,
		Type:    "community",
		Enabled: true,
	})
}
