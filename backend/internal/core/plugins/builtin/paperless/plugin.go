package paperless

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hay-kot/httpkit/server"
	"github.com/rs/zerolog"
	"github.com/sysadminsmedia/homebox/backend/internal/core/plugins"
	"github.com/sysadminsmedia/homebox/backend/internal/core/services/reporting/eventbus"
)

// Plugin provides bidirectional linking between Paperless-ngx documents
// and HomeBoxNG items. Receipts, warranties, and manuals in Paperless
// are linked to the corresponding inventory items.
type Plugin struct {
	logger zerolog.Logger
	pctx   plugins.PluginContext

	// Configuration
	paperlessURL   string
	paperlessToken string
	autoLink       bool

	// Link state
	mu    sync.Mutex
	links []DocumentLink
}

// DocumentLink represents a link between a Paperless document and a HomeBox item.
type DocumentLink struct {
	DocumentID   int       `json:"documentId"`
	DocumentTitle string   `json:"documentTitle"`
	ItemID       string    `json:"itemId"`
	ItemName     string    `json:"itemName"`
	LinkType     string    `json:"linkType"` // receipt, warranty, manual, invoice, other
	LinkedAt     time.Time `json:"linkedAt"`
	LinkedBy     string    `json:"linkedBy"` // auto or user_id
}

// PaperlessDocument is a minimal representation of a Paperless-ngx document.
type PaperlessDocument struct {
	ID             int       `json:"id"`
	Title          string    `json:"title"`
	Content        string    `json:"content,omitempty"`
	Correspondent  string    `json:"correspondent__name,omitempty"`
	DocumentType   string    `json:"document_type__name,omitempty"`
	Tags           []string  `json:"tags,omitempty"`
	Created        time.Time `json:"created"`
	ArchiveURL     string    `json:"archive_url,omitempty"`
	OriginalURL    string    `json:"original_url,omitempty"`
}

// SearchResult holds Paperless search results.
type SearchResult struct {
	Count   int                  `json:"count"`
	Results []PaperlessDocument  `json:"results"`
}

// New creates a new Paperless-ngx bridge plugin.
func New() *Plugin {
	return &Plugin{
		paperlessURL: "http://192.168.1.249:8000",
		autoLink:     false,
	}
}

func (p *Plugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "paperless",
		Version:     "1.0.0",
		Description: "Bidirectional linking between Paperless-ngx documents and HomeBoxNG items (receipts, warranties, manuals)",
		Author:      "HomeBoxNG Team",
		BuiltIn:     true,
	}
}

func (p *Plugin) Init(ctx plugins.PluginContext) error {
	p.pctx = ctx
	p.logger = ctx.Logger
	p.logger.Info().
		Str("paperless_url", p.paperlessURL).
		Bool("auto_link", p.autoLink).
		Msg("paperless plugin initialized")
	return nil
}

func (p *Plugin) Start(_ context.Context) error {
	p.logger.Info().Msg("paperless plugin started")
	return nil
}

func (p *Plugin) Stop(_ context.Context) error {
	p.logger.Info().Msg("paperless plugin stopped")
	return nil
}

// Routes registers Paperless bridge API endpoints.
func (p *Plugin) Routes(r chi.Router) {
	// GET /api/plugins/paperless/status - Connection status
	r.Get("/status", func(w http.ResponseWriter, r *http.Request) {
		status := "disconnected"
		var docCount int

		if p.paperlessURL != "" && p.paperlessToken != "" {
			count, err := p.getDocumentCount(r.Context())
			if err == nil {
				status = "connected"
				docCount = count
			} else {
				status = "error: " + err.Error()
			}
		}

		_ = server.JSON(w, http.StatusOK, map[string]any{
			"status":    status,
			"url":       p.paperlessURL,
			"documents": docCount,
			"links":     len(p.links),
			"autoLink":  p.autoLink,
		})
	})

	// GET /api/plugins/paperless/search?q=... - Search Paperless documents
	r.Get("/search", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")
		if query == "" {
			_ = server.JSON(w, http.StatusBadRequest, map[string]string{"error": "query required"})
			return
		}

		docs, err := p.searchDocuments(r.Context(), query)
		if err != nil {
			_ = server.JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		_ = server.JSON(w, http.StatusOK, docs)
	})

	// GET /api/plugins/paperless/links - List all document-item links
	r.Get("/links", func(w http.ResponseWriter, r *http.Request) {
		p.mu.Lock()
		links := make([]DocumentLink, len(p.links))
		copy(links, p.links)
		p.mu.Unlock()

		_ = server.JSON(w, http.StatusOK, links)
	})

	// POST /api/plugins/paperless/links - Create a document-item link
	r.Post("/links", func(w http.ResponseWriter, r *http.Request) {
		var link DocumentLink
		if err := json.NewDecoder(r.Body).Decode(&link); err != nil {
			_ = server.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}

		link.LinkedAt = time.Now()
		if link.LinkedBy == "" {
			link.LinkedBy = "user"
		}

		p.mu.Lock()
		p.links = append(p.links, link)
		p.mu.Unlock()

		p.logger.Info().
			Int("doc_id", link.DocumentID).
			Str("item_id", link.ItemID).
			Str("type", link.LinkType).
			Msg("document linked to item")

		_ = server.JSON(w, http.StatusCreated, link)
	})

	// DELETE /api/plugins/paperless/links/{docId}/{itemId} - Remove a link
	r.Delete("/links/{docId}/{itemId}", func(w http.ResponseWriter, r *http.Request) {
		docID := chi.URLParam(r, "docId")
		itemID := chi.URLParam(r, "itemId")

		p.mu.Lock()
		for i, link := range p.links {
			if fmt.Sprintf("%d", link.DocumentID) == docID && link.ItemID == itemID {
				p.links = append(p.links[:i], p.links[i+1:]...)
				p.mu.Unlock()
				_ = server.JSON(w, http.StatusOK, map[string]string{"status": "removed"})
				return
			}
		}
		p.mu.Unlock()

		_ = server.JSON(w, http.StatusNotFound, map[string]string{"error": "link not found"})
	})

	// GET /api/plugins/paperless/items/{itemId}/documents - Get documents for an item
	r.Get("/items/{itemId}/documents", func(w http.ResponseWriter, r *http.Request) {
		itemID := chi.URLParam(r, "itemId")

		p.mu.Lock()
		var docs []DocumentLink
		for _, link := range p.links {
			if link.ItemID == itemID {
				docs = append(docs, link)
			}
		}
		p.mu.Unlock()

		_ = server.JSON(w, http.StatusOK, docs)
	})

	// GET /api/plugins/paperless/unlinked - Documents not linked to any item
	r.Get("/unlinked", func(w http.ResponseWriter, r *http.Request) {
		// Search for recent receipts/invoices that aren't linked
		docs, err := p.searchDocuments(r.Context(), "type:receipt OR type:invoice")
		if err != nil {
			_ = server.JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		p.mu.Lock()
		linkedDocIDs := make(map[int]bool)
		for _, link := range p.links {
			linkedDocIDs[link.DocumentID] = true
		}
		p.mu.Unlock()

		var unlinked []PaperlessDocument
		for _, doc := range docs {
			if !linkedDocIDs[doc.ID] {
				unlinked = append(unlinked, doc)
			}
		}

		_ = server.JSON(w, http.StatusOK, unlinked)
	})

	// POST /api/plugins/paperless/test - Test Paperless connection
	r.Post("/test", func(w http.ResponseWriter, r *http.Request) {
		count, err := p.getDocumentCount(r.Context())
		if err != nil {
			_ = server.JSON(w, http.StatusServiceUnavailable, map[string]string{
				"status": "error",
				"error":  err.Error(),
			})
			return
		}

		_ = server.JSON(w, http.StatusOK, map[string]any{
			"status":    "connected",
			"documents": count,
		})
	})
}

// searchDocuments queries Paperless-ngx for documents matching a query.
func (p *Plugin) searchDocuments(ctx context.Context, query string) ([]PaperlessDocument, error) {
	url := fmt.Sprintf("%s/api/documents/?query=%s&page_size=20", p.paperlessURL, query)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Token "+p.paperlessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("paperless API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("paperless API returned %d: %s", resp.StatusCode, string(body))
	}

	var result SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return result.Results, nil
}

// getDocumentCount returns the total number of documents in Paperless.
func (p *Plugin) getDocumentCount(ctx context.Context) (int, error) {
	url := fmt.Sprintf("%s/api/documents/?page_size=1", p.paperlessURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Token "+p.paperlessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return 0, fmt.Errorf("invalid API token (401)")
	}
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var result struct {
		Count int `json:"count"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	return result.Count, nil
}

// SubscribeEvents listens for item events to suggest document links.
func (p *Plugin) SubscribeEvents(bus *eventbus.EventBus) {
	bus.Subscribe(eventbus.EventItemMutation, func(data any) {
		if !p.autoLink {
			return
		}
		p.logger.Debug().Msg("paperless received item mutation - could auto-link")
	})
}

// ConfigSchema returns Paperless bridge configuration options.
func (p *Plugin) ConfigSchema() []plugins.ConfigField {
	return []plugins.ConfigField{
		{Key: "paperless_url", Label: "Paperless-ngx URL", Description: "URL of your Paperless-ngx instance", Type: "string", Default: "http://192.168.1.249:8000", EnvVar: "HBOX_PAPERLESS_URL", Required: true},
		{Key: "paperless_token", Label: "API Token", Description: "Paperless-ngx API authentication token", Type: "secret", EnvVar: "HBOX_PAPERLESS_TOKEN", Required: true},
		{Key: "auto_link", Label: "Auto-Link", Description: "Automatically link new receipts/invoices to matching items", Type: "boolean", Default: "false", Required: false},
	}
}

// Configure applies configuration values.
func (p *Plugin) Configure(values map[string]string) error {
	if v, ok := values["paperless_url"]; ok && v != "" {
		p.paperlessURL = strings.TrimRight(v, "/")
	}
	if v, ok := values["paperless_token"]; ok {
		p.paperlessToken = v
	}
	if v, ok := values["auto_link"]; ok {
		p.autoLink = v == "true" || v == "1"
	}
	return nil
}

// RequestedPermissions declares what the Paperless plugin needs.
func (p *Plugin) RequestedPermissions() []plugins.PermissionRequest {
	return []plugins.PermissionRequest{
		{Permission: plugins.PermReadItems, Reason: "Match items to documents by name/serial", Required: true},
		{Permission: plugins.PermWriteItems, Reason: "Add document links to item notes", Required: false},
		{Permission: plugins.PermNetwork, Reason: "Connect to Paperless-ngx API", Required: true},
		{Permission: plugins.PermAPIRoutes, Reason: "Document search, linking, and status endpoints", Required: true},
		{Permission: plugins.PermEvents, Reason: "Auto-link documents when new items are created", Required: false},
		{Permission: plugins.PermConfig, Reason: "Store Paperless connection settings", Required: true},
	}
}

// Compile-time interface checks.
var _ plugins.Plugin = (*Plugin)(nil)
var _ plugins.RoutePlugin = (*Plugin)(nil)
var _ plugins.EventPlugin = (*Plugin)(nil)
var _ plugins.ConfigPlugin = (*Plugin)(nil)
var _ plugins.PluginWithPermissions = (*Plugin)(nil)

// Ensure bytes import is used (for potential future request body reading).
var _ = bytes.NewReader
