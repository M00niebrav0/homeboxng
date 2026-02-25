package manuals

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/go-chi/chi/v5"
	"github.com/hay-kot/httpkit/server"
	"github.com/rs/zerolog"
	"github.com/sysadminsmedia/homebox/backend/internal/core/plugins"
	"github.com/sysadminsmedia/homebox/backend/internal/core/services/reporting/eventbus"
)

// Plugin provides manual lookup, fuzzy matching, and user-uploaded manual
// management for HomeBoxNG inventory items. Supports linking manuals from
// ManualsLib, manufacturer websites, iFixit, or direct user uploads.
type Plugin struct {
	logger zerolog.Logger
	pctx   plugins.PluginContext

	// In-memory cache of manual links keyed by item ID.
	mu      sync.Mutex
	manuals map[string][]ManualLink

	// Configuration
	manualsLibEnabled   bool
	autoSuggest         bool
	searchManufacturers bool
	confidenceThreshold float64
	maxSuggestions      int
}

// ManualLink represents a linked manual (either from ManualsLib or user-uploaded).
type ManualLink struct {
	ID           string    `json:"id"`
	ItemID       string    `json:"itemId"`
	Title        string    `json:"title"`                  // "Owner's Manual", "Service Manual", etc.
	Category     string    `json:"category"`               // manual type classification
	Source       string    `json:"source"`                 // "manualslib", "manufacturer", "user_upload", "ifixit"
	URL          string    `json:"url"`                    // External URL or internal attachment path
	AttachmentID string    `json:"attachmentId,omitempty"` // If user-uploaded, reference to attachment
	Manufacturer string    `json:"manufacturer"`
	ModelNumber  string    `json:"modelNumber"`
	Confidence   float64   `json:"confidence"` // 0.0-1.0 match confidence
	Pages        int       `json:"pages"`
	FileSize     int64     `json:"fileSize"`
	MimeType     string    `json:"mimeType"`
	CreatedAt    time.Time `json:"createdAt"`
}

// SearchResult represents a single manual search result from ManualsLib or other sources.
type SearchResult struct {
	Title      string  `json:"title"`
	URL        string  `json:"url"`
	Brand      string  `json:"brand"`
	Category   string  `json:"category"` // manual type
	Pages      int     `json:"pages"`
	Confidence float64 `json:"confidence"`
}

// LinkRequest is the request body for linking a manual to an item.
type LinkRequest struct {
	Title        string `json:"title"`
	Category     string `json:"category"`
	Source       string `json:"source"`
	URL          string `json:"url"`
	Manufacturer string `json:"manufacturer"`
	ModelNumber  string `json:"modelNumber"`
	Pages        int    `json:"pages"`
}

// UploadRequest is the metadata sent alongside a manual file upload.
type UploadRequest struct {
	Title        string `json:"title"`
	Category     string `json:"category"`
	Manufacturer string `json:"manufacturer"`
	ModelNumber  string `json:"modelNumber"`
}

// SuggestRequest is the request body for AI-suggested manual lookup.
type SuggestRequest struct {
	ItemName     string `json:"itemName"`
	Manufacturer string `json:"manufacturer"`
	ModelNumber  string `json:"modelNumber"`
}

// FuzzyQuery holds a generated search query with its weight.
type FuzzyQuery struct {
	Query  string  `json:"query"`
	Weight float64 `json:"weight"` // 1.0 = exact, lower = more fuzzy
}

// manualCategories enumerates the valid manual category slugs.
var manualCategories = []map[string]string{
	{"slug": "user_manual", "label": "User Manual", "description": "Standard owner/operator manual"},
	{"slug": "service_manual", "label": "Service Manual", "description": "Repair and service procedures"},
	{"slug": "parts_manual", "label": "Parts Manual", "description": "Parts list and exploded diagrams"},
	{"slug": "installation_guide", "label": "Installation Guide", "description": "Setup and installation instructions"},
	{"slug": "quick_start", "label": "Quick Start Guide", "description": "Quick start / getting started guide"},
	{"slug": "wiring_diagram", "label": "Wiring Diagram", "description": "Electrical wiring and schematics"},
	{"slug": "engine_manual", "label": "Engine Manual", "description": "Engine-specific service manual"},
	{"slug": "safety_manual", "label": "Safety Manual", "description": "Safety data sheet or safety guide"},
	{"slug": "custom", "label": "Custom", "description": "User-defined manual type"},
}

// New creates a new manuals plugin instance.
func New() *Plugin {
	return &Plugin{
		manuals:             make(map[string][]ManualLink),
		manualsLibEnabled:   true,
		autoSuggest:         true,
		searchManufacturers: true,
		confidenceThreshold: 0.7,
		maxSuggestions:      5,
	}
}

func (p *Plugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "manuals",
		Version:     "1.0.0",
		Description: "Manual lookup, fuzzy matching, and user-uploaded manual management for inventory items",
		Author:      "HomeBoxNG Team",
		BuiltIn:     true,
	}
}

func (p *Plugin) Init(ctx plugins.PluginContext) error {
	p.pctx = ctx
	p.logger = ctx.Logger
	p.logger.Info().
		Bool("manualslib_enabled", p.manualsLibEnabled).
		Bool("auto_suggest", p.autoSuggest).
		Float64("confidence_threshold", p.confidenceThreshold).
		Msg("manuals plugin initialized")
	return nil
}

func (p *Plugin) Start(_ context.Context) error {
	p.logger.Info().Msg("manuals plugin started")
	return nil
}

func (p *Plugin) Stop(_ context.Context) error {
	p.logger.Info().Msg("manuals plugin stopped")
	return nil
}

// Routes registers manuals API endpoints.
func (p *Plugin) Routes(r chi.Router) {
	// GET /api/plugins/manuals/search?q=...&brand=... - Search ManualsLib for manuals
	r.Get("/search", p.handleSearch)

	// GET /api/plugins/manuals/item/{itemId} - Get all manuals linked to an item
	r.Get("/item/{itemId}", p.handleGetItemManuals)

	// POST /api/plugins/manuals/item/{itemId}/link - Link a manual to an item
	r.Post("/item/{itemId}/link", p.handleLinkManual)

	// POST /api/plugins/manuals/item/{itemId}/upload - Upload a manual file to an item
	r.Post("/item/{itemId}/upload", p.handleUploadManual)

	// DELETE /api/plugins/manuals/item/{itemId}/manual/{manualId} - Remove a manual link
	r.Delete("/item/{itemId}/manual/{manualId}", p.handleDeleteManual)

	// POST /api/plugins/manuals/suggest/{itemId} - Get AI-suggested manuals for an item
	r.Post("/suggest/{itemId}", p.handleSuggest)

	// GET /api/plugins/manuals/categories - List manual categories
	r.Get("/categories", p.handleCategories)
}

// handleSearch searches ManualsLib (or other configured sources) for manuals.
func (p *Plugin) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	brand := r.URL.Query().Get("brand")

	if query == "" {
		_ = server.JSON(w, http.StatusBadRequest, map[string]string{"error": "query parameter 'q' is required"})
		return
	}

	if !p.manualsLibEnabled {
		_ = server.JSON(w, http.StatusOK, map[string]any{
			"results": []SearchResult{},
			"message": "ManualsLib search is disabled in plugin configuration",
		})
		return
	}

	// Build the search URL for reference (actual scraping is a TODO).
	searchQuery := query
	if brand != "" {
		searchQuery = brand + " " + query
	}
	manualsLibURL := fmt.Sprintf("https://www.manualslib.com/manual/search/?q=%s", url.QueryEscape(searchQuery))

	// TODO: Implement actual ManualsLib web scraping or API integration.
	// For now, return an empty result set with the constructed search URL
	// so the frontend can provide a "Search on ManualsLib" link.
	_ = server.JSON(w, http.StatusOK, map[string]any{
		"results":    []SearchResult{},
		"query":      searchQuery,
		"searchUrl":  manualsLibURL,
		"message":    "Web search not yet implemented. Use the searchUrl to search ManualsLib directly.",
		"totalFound": 0,
	})
}

// handleGetItemManuals returns all manuals linked to a specific item.
func (p *Plugin) handleGetItemManuals(w http.ResponseWriter, r *http.Request) {
	itemID := chi.URLParam(r, "itemId")
	if itemID == "" {
		_ = server.JSON(w, http.StatusBadRequest, map[string]string{"error": "itemId is required"})
		return
	}

	p.mu.Lock()
	links := make([]ManualLink, len(p.manuals[itemID]))
	copy(links, p.manuals[itemID])
	p.mu.Unlock()

	if links == nil {
		links = []ManualLink{}
	}

	_ = server.JSON(w, http.StatusOK, links)
}

// handleLinkManual links a manual (from a URL or search result) to an item.
func (p *Plugin) handleLinkManual(w http.ResponseWriter, r *http.Request) {
	itemID := chi.URLParam(r, "itemId")
	if itemID == "" {
		_ = server.JSON(w, http.StatusBadRequest, map[string]string{"error": "itemId is required"})
		return
	}

	var req LinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		_ = server.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.URL == "" {
		_ = server.JSON(w, http.StatusBadRequest, map[string]string{"error": "url is required"})
		return
	}
	if req.Title == "" {
		req.Title = "Manual"
	}
	if req.Category == "" {
		req.Category = "user_manual"
	}
	if req.Source == "" {
		req.Source = "manualslib"
	}

	link := ManualLink{
		ID:           generateID(),
		ItemID:       itemID,
		Title:        req.Title,
		Category:     req.Category,
		Source:       req.Source,
		URL:          req.URL,
		Manufacturer: req.Manufacturer,
		ModelNumber:  req.ModelNumber,
		Confidence:   1.0, // User-linked manuals have full confidence
		Pages:        req.Pages,
		MimeType:     "text/html", // URL links default to HTML
		CreatedAt:    time.Now(),
	}

	p.mu.Lock()
	p.manuals[itemID] = append(p.manuals[itemID], link)
	p.mu.Unlock()

	p.logger.Info().
		Str("item_id", itemID).
		Str("manual_id", link.ID).
		Str("source", link.Source).
		Str("title", link.Title).
		Msg("manual linked to item")

	_ = server.JSON(w, http.StatusCreated, link)
}

// handleUploadManual handles manual file uploads (PDF/JPG/PNG) for an item.
func (p *Plugin) handleUploadManual(w http.ResponseWriter, r *http.Request) {
	itemID := chi.URLParam(r, "itemId")
	if itemID == "" {
		_ = server.JSON(w, http.StatusBadRequest, map[string]string{"error": "itemId is required"})
		return
	}

	// Parse multipart form (max 50MB)
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		_ = server.JSON(w, http.StatusBadRequest, map[string]string{"error": "failed to parse multipart form: " + err.Error()})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		_ = server.JSON(w, http.StatusBadRequest, map[string]string{"error": "file field is required"})
		return
	}
	defer file.Close()

	// Validate MIME type
	mimeType := header.Header.Get("Content-Type")
	allowedTypes := map[string]bool{
		"application/pdf": true,
		"image/jpeg":      true,
		"image/png":       true,
		"image/tiff":      true,
	}
	if !allowedTypes[mimeType] {
		_ = server.JSON(w, http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("unsupported file type: %s (allowed: PDF, JPEG, PNG, TIFF)", mimeType),
		})
		return
	}

	// Read metadata from form fields
	title := r.FormValue("title")
	if title == "" {
		title = header.Filename
	}
	category := r.FormValue("category")
	if category == "" {
		category = "user_manual"
	}

	// TODO: Store the file using the HomeBoxNG attachment/storage system.
	// For now, create the manual link record with metadata only.
	attachmentID := generateID()

	link := ManualLink{
		ID:           generateID(),
		ItemID:       itemID,
		Title:        title,
		Category:     category,
		Source:       "user_upload",
		URL:          fmt.Sprintf("/api/plugins/manuals/attachments/%s", attachmentID),
		AttachmentID: attachmentID,
		Manufacturer: r.FormValue("manufacturer"),
		ModelNumber:  r.FormValue("modelNumber"),
		Confidence:   1.0,
		FileSize:     header.Size,
		MimeType:     mimeType,
		CreatedAt:    time.Now(),
	}

	p.mu.Lock()
	p.manuals[itemID] = append(p.manuals[itemID], link)
	p.mu.Unlock()

	p.logger.Info().
		Str("item_id", itemID).
		Str("manual_id", link.ID).
		Str("filename", header.Filename).
		Int64("size", header.Size).
		Str("mime", mimeType).
		Msg("manual uploaded for item")

	_ = server.JSON(w, http.StatusCreated, link)
}

// handleDeleteManual removes a manual link from an item.
func (p *Plugin) handleDeleteManual(w http.ResponseWriter, r *http.Request) {
	itemID := chi.URLParam(r, "itemId")
	manualID := chi.URLParam(r, "manualId")

	if itemID == "" || manualID == "" {
		_ = server.JSON(w, http.StatusBadRequest, map[string]string{"error": "itemId and manualId are required"})
		return
	}

	p.mu.Lock()
	links := p.manuals[itemID]
	for i, link := range links {
		if link.ID == manualID {
			p.manuals[itemID] = append(links[:i], links[i+1:]...)
			p.mu.Unlock()

			p.logger.Info().
				Str("item_id", itemID).
				Str("manual_id", manualID).
				Msg("manual removed from item")

			_ = server.JSON(w, http.StatusOK, map[string]string{"status": "removed"})
			return
		}
	}
	p.mu.Unlock()

	_ = server.JSON(w, http.StatusNotFound, map[string]string{"error": "manual not found"})
}

// handleSuggest uses fuzzy matching to suggest manuals for an item.
func (p *Plugin) handleSuggest(w http.ResponseWriter, r *http.Request) {
	itemID := chi.URLParam(r, "itemId")
	if itemID == "" {
		_ = server.JSON(w, http.StatusBadRequest, map[string]string{"error": "itemId is required"})
		return
	}

	var req SuggestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		_ = server.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.ItemName == "" {
		_ = server.JSON(w, http.StatusBadRequest, map[string]string{"error": "itemName is required"})
		return
	}

	// Generate fuzzy search queries from item metadata
	queries := BuildFuzzyQueries(req.ItemName, req.Manufacturer, req.ModelNumber)

	// Build ManualsLib search URLs for each query
	suggestions := make([]map[string]any, 0, len(queries))
	for _, q := range queries {
		if len(suggestions) >= p.maxSuggestions {
			break
		}
		suggestions = append(suggestions, map[string]any{
			"query":     q.Query,
			"weight":    q.Weight,
			"searchUrl": fmt.Sprintf("https://www.manualslib.com/manual/search/?q=%s", url.QueryEscape(q.Query)),
		})
	}

	// TODO: When web scraping is implemented, actually run these queries
	// against ManualsLib and return real SearchResult objects scored
	// by the fuzzy match confidence.

	_ = server.JSON(w, http.StatusOK, map[string]any{
		"itemId":      itemID,
		"itemName":    req.ItemName,
		"queries":     suggestions,
		"results":     []SearchResult{},
		"message":     "Fuzzy search queries generated. Web search not yet implemented.",
		"totalFound":  0,
	})
}

// handleCategories returns the list of available manual categories.
func (p *Plugin) handleCategories(w http.ResponseWriter, _ *http.Request) {
	_ = server.JSON(w, http.StatusOK, manualCategories)
}

// ---------------------------------------------------------------------------
// Fuzzy Matching Implementation
// ---------------------------------------------------------------------------

// BuildFuzzyQueries takes item metadata and generates multiple search queries
// ranked by specificity. It extracts brand, model number, and year from the
// item name and produces exact, simplified, and brand+model queries.
func BuildFuzzyQueries(itemName, manufacturer, modelNumber string) []FuzzyQuery {
	queries := make([]FuzzyQuery, 0, 6)

	// Normalize input
	itemName = strings.TrimSpace(itemName)
	manufacturer = strings.TrimSpace(manufacturer)
	modelNumber = strings.TrimSpace(modelNumber)

	if itemName == "" {
		return queries
	}

	// Extract components from the item name
	extractedBrand, extractedModel, extractedYear, genericWords := parseItemName(itemName)

	// Use explicit values if provided, otherwise fall back to extracted
	brand := manufacturer
	if brand == "" {
		brand = extractedBrand
	}
	model := modelNumber
	if model == "" {
		model = extractedModel
	}

	// Query 1: Exact item name (highest weight)
	queries = append(queries, FuzzyQuery{
		Query:  itemName,
		Weight: 1.0,
	})

	// Query 2: Brand + model number (very specific)
	if brand != "" && model != "" {
		q := brand + " " + model
		if extractedYear != "" {
			q += " " + extractedYear
		}
		if !queryExists(queries, q) {
			queries = append(queries, FuzzyQuery{
				Query:  q,
				Weight: 0.95,
			})
		}
	}

	// Query 3: Model number alone (model numbers are highly unique identifiers)
	if model != "" {
		if !queryExists(queries, model) {
			queries = append(queries, FuzzyQuery{
				Query:  model,
				Weight: 0.85,
			})
		}
	}

	// Query 4: Brand + generic words (e.g., "Toyota Tacoma" without year)
	if brand != "" && len(genericWords) > 0 {
		q := brand + " " + strings.Join(genericWords, " ")
		if !queryExists(queries, q) {
			queries = append(queries, FuzzyQuery{
				Query:  q,
				Weight: 0.75,
			})
		}
	}

	// Query 5: Rearranged (brand + generic + year) for "2018 Toyota Tacoma" -> "Toyota Tacoma 2018"
	if brand != "" && extractedYear != "" && len(genericWords) > 0 {
		q := brand + " " + strings.Join(genericWords, " ") + " " + extractedYear
		if !queryExists(queries, q) {
			queries = append(queries, FuzzyQuery{
				Query:  q,
				Weight: 0.7,
			})
		}
	}

	// Query 6: Brand only (very broad fallback)
	if brand != "" {
		if !queryExists(queries, brand) {
			queries = append(queries, FuzzyQuery{
				Query:  brand,
				Weight: 0.4,
			})
		}
	}

	// Sort by weight descending
	sort.Slice(queries, func(i, j int) bool {
		return queries[i].Weight > queries[j].Weight
	})

	return queries
}

// parseItemName extracts brand, model number, year, and generic words from an
// item name string. It handles common patterns such as:
//   - "2018 Toyota Tacoma" -> year=2018, brand=Toyota, generic=[Tacoma]
//   - "DeWalt DCD771C2 Drill/Driver" -> brand=DeWalt, model=DCD771C2, generic=[Drill/Driver]
//   - "Samsung UN55TU7000 55-Inch TV" -> brand=Samsung, model=UN55TU7000, generic=[55-Inch TV]
func parseItemName(name string) (brand, model, year string, genericWords []string) {
	// Regex patterns
	yearRe := regexp.MustCompile(`\b(19|20)\d{2}\b`)
	// Model numbers: alphanumeric strings with at least one letter and one digit,
	// minimum 3 characters. Matches patterns like DCD771C2, UN55TU7000, XJ-13, RT-AC68U.
	modelRe := regexp.MustCompile(`\b[A-Za-z0-9][-A-Za-z0-9]{2,}[A-Za-z0-9]\b`)

	tokens := tokenize(name)

	// Extract year
	if match := yearRe.FindString(name); match != "" {
		year = match
	}

	// Find potential model numbers (tokens with mixed letters + digits)
	for _, tok := range tokens {
		if tok == year {
			continue
		}
		if isModelNumber(tok) {
			model = tok
			break
		}
	}

	// If no model found via heuristic, try the regex
	if model == "" {
		for _, match := range modelRe.FindAllString(name, -1) {
			if match == year {
				continue
			}
			if isModelNumber(match) {
				model = match
				break
			}
		}
	}

	// The first non-year, non-model token is assumed to be the brand
	for _, tok := range tokens {
		if tok == year || tok == model {
			continue
		}
		if isGenericNoise(tok) {
			continue
		}
		brand = tok
		break
	}

	// Everything else that is not year, model, or brand is a generic word
	for _, tok := range tokens {
		if tok == year || tok == model || strings.EqualFold(tok, brand) {
			continue
		}
		if isGenericNoise(tok) {
			continue
		}
		genericWords = append(genericWords, tok)
	}

	return brand, model, year, genericWords
}

// tokenize splits a string into tokens, splitting on whitespace and some
// punctuation while preserving hyphenated and slashed compound words.
func tokenize(s string) []string {
	// Split on whitespace and commas
	fields := strings.FieldsFunc(s, func(r rune) bool {
		return unicode.IsSpace(r) || r == ',' || r == ';' || r == '(' || r == ')'
	})
	return fields
}

// isModelNumber returns true if the token looks like a model number
// (contains at least one letter and at least one digit).
func isModelNumber(s string) bool {
	hasLetter := false
	hasDigit := false
	for _, r := range s {
		if unicode.IsLetter(r) {
			hasLetter = true
		}
		if unicode.IsDigit(r) {
			hasDigit = true
		}
		if hasLetter && hasDigit {
			return true
		}
	}
	return false
}

// isGenericNoise returns true for very common filler words that add no value
// to a search query.
var genericNoise = map[string]bool{
	"the": true, "a": true, "an": true, "and": true, "or": true,
	"for": true, "with": true, "in": true, "of": true, "to": true,
	"-": true, "/": true, "&": true, "inch": true, "set": true, "kit": true,
}

func isGenericNoise(s string) bool {
	return genericNoise[strings.ToLower(s)]
}

// queryExists checks if a query string is already in the list (case-insensitive).
func queryExists(queries []FuzzyQuery, q string) bool {
	for _, existing := range queries {
		if strings.EqualFold(existing.Query, q) {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// String Similarity (Levenshtein + Token Overlap)
// ---------------------------------------------------------------------------

// StringSimilarity computes a combined similarity score (0.0-1.0) between two
// strings using both Levenshtein distance and token overlap. The final score
// is a weighted combination: 40% normalized Levenshtein, 60% token overlap.
// This weighting favors token-level matches because manual titles often contain
// the same words in different order.
func StringSimilarity(a, b string) float64 {
	a = strings.ToLower(strings.TrimSpace(a))
	b = strings.ToLower(strings.TrimSpace(b))

	if a == b {
		return 1.0
	}
	if a == "" || b == "" {
		return 0.0
	}

	levSim := levenshteinSimilarity(a, b)
	tokenSim := tokenOverlap(a, b)

	// Weighted combination
	return 0.4*levSim + 0.6*tokenSim
}

// levenshteinDistance computes the edit distance between two strings using
// the Wagner-Fischer dynamic programming algorithm with O(min(m,n)) space.
func levenshteinDistance(a, b string) int {
	ra := []rune(a)
	rb := []rune(b)
	la := len(ra)
	lb := len(rb)

	// Ensure a is the shorter string for O(min) space
	if la > lb {
		ra, rb = rb, ra
		la, lb = lb, la
	}

	prev := make([]int, la+1)
	curr := make([]int, la+1)

	for i := 0; i <= la; i++ {
		prev[i] = i
	}

	for j := 1; j <= lb; j++ {
		curr[0] = j
		for i := 1; i <= la; i++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			curr[i] = min3(
				curr[i-1]+1,   // insertion
				prev[i]+1,     // deletion
				prev[i-1]+cost, // substitution
			)
		}
		prev, curr = curr, prev
	}

	return prev[la]
}

// levenshteinSimilarity returns a normalized similarity score (0.0-1.0) based
// on the Levenshtein distance.
func levenshteinSimilarity(a, b string) float64 {
	dist := levenshteinDistance(a, b)
	maxLen := math.Max(float64(len([]rune(a))), float64(len([]rune(b))))
	if maxLen == 0 {
		return 1.0
	}
	return 1.0 - float64(dist)/maxLen
}

// tokenOverlap computes the Jaccard-like overlap of tokens between two strings.
// Tokens are lowercased and deduplicated. The score is |intersection| / |union|.
func tokenOverlap(a, b string) float64 {
	tokensA := uniqueTokens(tokenize(a))
	tokensB := uniqueTokens(tokenize(b))

	if len(tokensA) == 0 && len(tokensB) == 0 {
		return 1.0
	}
	if len(tokensA) == 0 || len(tokensB) == 0 {
		return 0.0
	}

	setB := make(map[string]bool, len(tokensB))
	for _, t := range tokensB {
		setB[t] = true
	}

	intersection := 0
	for _, t := range tokensA {
		if setB[t] {
			intersection++
		}
	}

	union := len(tokensA)
	for _, t := range tokensB {
		found := false
		for _, ta := range tokensA {
			if ta == t {
				found = true
				break
			}
		}
		if !found {
			union++
		}
	}

	if union == 0 {
		return 0.0
	}
	return float64(intersection) / float64(union)
}

// uniqueTokens returns deduplicated, lowercased tokens.
func uniqueTokens(tokens []string) []string {
	seen := make(map[string]bool, len(tokens))
	result := make([]string, 0, len(tokens))
	for _, t := range tokens {
		lower := strings.ToLower(t)
		if !seen[lower] {
			seen[lower] = true
			result = append(result, lower)
		}
	}
	return result
}

// min3 returns the minimum of three integers.
func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// generateID produces a simple unique ID string based on the current
// nanosecond timestamp. For production use, this should be replaced with
// a proper UUID generator.
func generateID() string {
	return fmt.Sprintf("m_%d", time.Now().UnixNano())
}

// ---------------------------------------------------------------------------
// Plugin Interface Implementations (Config, Events, Permissions)
// ---------------------------------------------------------------------------

// ConfigSchema returns the manuals plugin configuration options.
func (p *Plugin) ConfigSchema() []plugins.ConfigField {
	return []plugins.ConfigField{
		{
			Key:         "manualslib_enabled",
			Label:       "Enable ManualsLib Search",
			Description: "Search ManualsLib.com for manuals matching inventory items",
			Type:        "boolean",
			Default:     "true",
			EnvVar:      "HBOX_MANUALS_MANUALSLIB_ENABLED",
			Required:    false,
		},
		{
			Key:         "auto_suggest",
			Label:       "Auto-Suggest Manuals",
			Description: "Automatically suggest manuals when new items are added",
			Type:        "boolean",
			Default:     "true",
			EnvVar:      "HBOX_MANUALS_AUTO_SUGGEST",
			Required:    false,
		},
		{
			Key:         "search_manufacturers",
			Label:       "Search Manufacturer Sites",
			Description: "Also search manufacturer websites for manuals",
			Type:        "boolean",
			Default:     "true",
			EnvVar:      "HBOX_MANUALS_SEARCH_MANUFACTURERS",
			Required:    false,
		},
		{
			Key:         "confidence_threshold",
			Label:       "Confidence Threshold",
			Description: "Minimum confidence score (0.0-1.0) for auto-suggestions",
			Type:        "number",
			Default:     "0.7",
			EnvVar:      "HBOX_MANUALS_CONFIDENCE_THRESHOLD",
			Required:    false,
		},
		{
			Key:         "max_suggestions",
			Label:       "Max Suggestions",
			Description: "Maximum number of manual suggestions per item",
			Type:        "number",
			Default:     "5",
			EnvVar:      "HBOX_MANUALS_MAX_SUGGESTIONS",
			Required:    false,
		},
	}
}

// Configure applies configuration values.
func (p *Plugin) Configure(values map[string]string) error {
	if v, ok := values["manualslib_enabled"]; ok {
		p.manualsLibEnabled = v == "true" || v == "1"
	}
	if v, ok := values["auto_suggest"]; ok {
		p.autoSuggest = v == "true" || v == "1"
	}
	if v, ok := values["search_manufacturers"]; ok {
		p.searchManufacturers = v == "true" || v == "1"
	}
	if v, ok := values["confidence_threshold"]; ok && v != "" {
		var threshold float64
		if _, err := fmt.Sscanf(v, "%f", &threshold); err == nil && threshold >= 0 && threshold <= 1 {
			p.confidenceThreshold = threshold
		}
	}
	if v, ok := values["max_suggestions"]; ok && v != "" {
		var max int
		if _, err := fmt.Sscanf(v, "%d", &max); err == nil && max > 0 {
			p.maxSuggestions = max
		}
	}
	return nil
}

// SubscribeEvents listens for item creation events to auto-suggest manuals.
func (p *Plugin) SubscribeEvents(bus *eventbus.EventBus) {
	bus.Subscribe(eventbus.EventItemMutation, func(data any) {
		if !p.autoSuggest {
			return
		}
		p.logger.Debug().Msg("manuals received item mutation - could auto-suggest manuals")
	})
}

// RequestedPermissions declares what the manuals plugin needs.
func (p *Plugin) RequestedPermissions() []plugins.PermissionRequest {
	return []plugins.PermissionRequest{
		{Permission: plugins.PermReadItems, Reason: "Read item names, manufacturers, and model numbers for manual matching", Required: true},
		{Permission: plugins.PermAPIRoutes, Reason: "Manual search, link, upload, and suggestion endpoints", Required: true},
		{Permission: plugins.PermConfig, Reason: "Store ManualsLib and search configuration", Required: true},
		{Permission: plugins.PermEvents, Reason: "Auto-suggest manuals when new items are created", Required: false},
		{Permission: plugins.PermNetwork, Reason: "Search ManualsLib and manufacturer websites", Required: false},
		{Permission: plugins.PermWriteAttachments, Reason: "Store uploaded manual files as attachments", Required: false},
	}
}

// Compile-time interface checks.
var (
	_ plugins.Plugin                = (*Plugin)(nil)
	_ plugins.RoutePlugin           = (*Plugin)(nil)
	_ plugins.EventPlugin           = (*Plugin)(nil)
	_ plugins.ConfigPlugin          = (*Plugin)(nil)
	_ plugins.PluginWithPermissions = (*Plugin)(nil)
)
