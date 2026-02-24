package main

import (
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/hay-kot/httpkit/errchain"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/validate"
)

// mwSecurityHeaders sets defense-in-depth security headers on every response.
// These are applied at the errchain middleware level (per-route), complementing
// the global chi-level headers set in mid.SecurityHeaders().
func (a *app) mwSecurityHeaders(next errchain.Handler) errchain.Handler {
	return errchain.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; connect-src 'self' ws: wss:")

		// Only set HSTS when the request arrived over TLS (direct or via proxy).
		if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		return next.ServeHTTP(w, r)
	})
}

// csrfConfig holds the configuration for the CSRF protection middleware.
type csrfConfig struct {
	// AllowedOrigins is the list of origins that are allowed for state-changing requests.
	// An empty list means only same-origin requests are allowed (Origin must match Host).
	AllowedOrigins []string
}

// mwCSRFProtection validates the Origin and Referer headers on state-changing
// requests (POST, PUT, PATCH, DELETE) to prevent cross-site request forgery.
//
// Requests that carry a Bearer token in the Authorization header are exempt
// because bearer-token authentication is inherently immune to CSRF -- the
// browser will never automatically attach such a token.
func (a *app) mwCSRFProtection(cfg csrfConfig) errchain.Middleware {
	allowed := make(map[string]struct{}, len(cfg.AllowedOrigins))
	for _, o := range cfg.AllowedOrigins {
		allowed[strings.ToLower(strings.TrimRight(o, "/"))] = struct{}{}
	}

	return func(next errchain.Handler) errchain.Handler {
		return errchain.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
			// Safe methods are exempt.
			method := r.Method
			if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
				return next.ServeHTTP(w, r)
			}

			// Bearer-token authenticated requests are CSRF-immune.
			if authHeader := r.Header.Get("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
				return next.ServeHTTP(w, r)
			}

			// Validate Origin header.
			origin := r.Header.Get("Origin")
			if origin != "" {
				if !csrfOriginAllowed(origin, r.Host, allowed) {
					return validate.NewRequestError(
						errors.New("CSRF check failed: origin not allowed"),
						http.StatusForbidden,
					)
				}
				return next.ServeHTTP(w, r)
			}

			// Fall back to Referer when Origin is absent.
			referer := r.Header.Get("Referer")
			if referer != "" {
				if !csrfRefererAllowed(referer, r.Host, allowed) {
					return validate.NewRequestError(
						errors.New("CSRF check failed: referer not allowed"),
						http.StatusForbidden,
					)
				}
				return next.ServeHTTP(w, r)
			}

			// Neither Origin nor Referer present -- reject.
			return validate.NewRequestError(
				errors.New("CSRF check failed: missing Origin or Referer header"),
				http.StatusForbidden,
			)
		})
	}
}

// csrfOriginAllowed checks whether the given origin value is acceptable.
func csrfOriginAllowed(origin, host string, allowed map[string]struct{}) bool {
	normalized := strings.ToLower(strings.TrimRight(origin, "/"))

	// Check explicit allow-list first.
	if _, ok := allowed[normalized]; ok {
		return true
	}

	// Allow same-origin: extract the host portion of the origin URL and compare.
	originHost := extractHost(normalized)
	return strings.EqualFold(originHost, host)
}

// csrfRefererAllowed checks whether the given referer header is from an acceptable origin.
func csrfRefererAllowed(referer, host string, allowed map[string]struct{}) bool {
	normalized := strings.ToLower(referer)

	// Extract scheme + host from the referer.
	refererOrigin := extractOrigin(normalized)

	// Check explicit allow-list.
	if _, ok := allowed[strings.TrimRight(refererOrigin, "/")]; ok {
		return true
	}

	// Allow same-origin.
	refererHost := extractHost(refererOrigin)
	return strings.EqualFold(refererHost, host)
}

// extractHost returns the host (with optional port) from a URL-like string.
// e.g. "https://example.com:8080/path" -> "example.com:8080"
func extractHost(raw string) string {
	// Strip scheme.
	s := raw
	if idx := strings.Index(s, "://"); idx != -1 {
		s = s[idx+3:]
	}
	// Strip path.
	if idx := strings.IndexByte(s, '/'); idx != -1 {
		s = s[:idx]
	}
	return s
}

// extractOrigin returns the scheme + host[:port] portion of a URL.
func extractOrigin(raw string) string {
	idx := strings.Index(raw, "://")
	if idx == -1 {
		return raw
	}
	rest := raw[idx+3:]
	slashIdx := strings.IndexByte(rest, '/')
	if slashIdx == -1 {
		return raw
	}
	return raw[:idx+3+slashIdx]
}

// requestSizeConfig holds the limits for the request size limiter.
type requestSizeConfig struct {
	// MaxBodyBytes is the default maximum request body size in bytes.
	MaxBodyBytes int64
	// MaxUploadBytes is the maximum request body size for upload endpoints in bytes.
	MaxUploadBytes int64
	// UploadPaths lists the path prefixes treated as upload endpoints.
	UploadPaths []string
}

// defaultRequestSizeConfig returns sensible defaults for the request size limiter.
func defaultRequestSizeConfig() requestSizeConfig {
	return requestSizeConfig{
		MaxBodyBytes:   10 * 1024 * 1024,  // 10 MB
		MaxUploadBytes: 50 * 1024 * 1024,  // 50 MB
		UploadPaths:    []string{"/attachments", "/import"},
	}
}

// mwRequestSizeLimit enforces a maximum request body size. Upload endpoints
// (identified by path prefix matching) get a higher limit.
func (a *app) mwRequestSizeLimit(cfg requestSizeConfig) errchain.Middleware {
	return func(next errchain.Handler) errchain.Handler {
		return errchain.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
			if r.Body == nil || r.Body == http.NoBody {
				return next.ServeHTTP(w, r)
			}

			limit := cfg.MaxBodyBytes
			path := r.URL.Path
			for _, prefix := range cfg.UploadPaths {
				if strings.Contains(path, prefix) {
					limit = cfg.MaxUploadBytes
					break
				}
			}

			// Wrap the body with a limiting reader. If the limit is exceeded the
			// read will fail and we return 413.
			r.Body = http.MaxBytesReader(w, r.Body, limit)

			return next.ServeHTTP(w, r)
		})
	}
}

// pluginRateLimiter provides per-client, per-plugin rate limiting for plugin endpoints.
type pluginRateLimiter struct {
	mu          sync.Mutex
	limiters    map[string]*rateLimiterEntry
	rate        int           // requests allowed per window
	window      time.Duration // time window
	stopCleanup chan struct{}
	stopOnce    sync.Once
}

// newPluginRateLimiter creates a new plugin rate limiter with the specified rate and window.
func newPluginRateLimiter(rate int, window time.Duration) *pluginRateLimiter {
	if rate <= 0 {
		rate = 60
	}
	if window <= 0 {
		window = time.Minute
	}

	pl := &pluginRateLimiter{
		limiters:    make(map[string]*rateLimiterEntry),
		rate:        rate,
		window:      window,
		stopCleanup: make(chan struct{}),
	}

	go pl.cleanupLoop()

	return pl
}

// cleanupLoop periodically removes stale entries.
func (pl *pluginRateLimiter) cleanupLoop() {
	interval := pl.window * 2
	if interval < 5*time.Minute {
		interval = 5 * time.Minute
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pl.cleanup()
		case <-pl.stopCleanup:
			return
		}
	}
}

// cleanup removes entries that have not been accessed recently.
func (pl *pluginRateLimiter) cleanup() {
	pl.mu.Lock()
	defer pl.mu.Unlock()

	now := time.Now()
	threshold := pl.window * 2

	for key, entry := range pl.limiters {
		if now.Sub(entry.lastRefill) > threshold {
			delete(pl.limiters, key)
		}
	}
}

// Stop gracefully stops the cleanup goroutine.
func (pl *pluginRateLimiter) Stop() {
	pl.stopOnce.Do(func() {
		close(pl.stopCleanup)
	})
}

// allow checks whether a request identified by the composite key should be allowed.
func (pl *pluginRateLimiter) allow(key string) bool {
	pl.mu.Lock()
	defer pl.mu.Unlock()

	now := time.Now()
	entry, exists := pl.limiters[key]

	if !exists {
		pl.limiters[key] = &rateLimiterEntry{
			tokens:     pl.rate - 1,
			lastRefill: now,
		}
		return true
	}

	elapsed := now.Sub(entry.lastRefill)
	if elapsed >= pl.window {
		entry.tokens = pl.rate - 1
		entry.lastRefill = now
		return true
	}

	if entry.tokens > 0 {
		entry.tokens--
		return true
	}

	return false
}

// mwPluginRateLimit creates middleware that enforces per-client, per-plugin
// rate limiting. The pluginName parameter identifies which plugin this
// middleware protects. The rate limiter key is: clientIP + "|" + pluginName.
func mwPluginRateLimit(limiter *pluginRateLimiter, pluginName string) errchain.Middleware {
	return func(next errchain.Handler) errchain.Handler {
		return errchain.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
			clientIP := pluginClientIP(r)
			key := clientIP + "|" + pluginName

			if !limiter.allow(key) {
				w.Header().Set("Retry-After", "60")
				return validate.NewRequestError(
					errors.New("plugin rate limit exceeded"),
					http.StatusTooManyRequests,
				)
			}

			return next.ServeHTTP(w, r)
		})
	}
}

// pluginClientIP extracts the client IP from the request, following the same
// strategy used by the other rate limiters in the codebase.
func pluginClientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}

	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		parts := strings.Split(ip, ",")
		return strings.TrimSpace(parts[0])
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}

	return r.RemoteAddr
}

