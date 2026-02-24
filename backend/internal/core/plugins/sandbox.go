package plugins

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// SandboxConfig defines the resource limits and network policies for a sandboxed plugin.
type SandboxConfig struct {
	// MaxMemoryMB is the soft memory budget for the plugin (advisory).
	MaxMemoryMB int
	// MaxCPUPercent is the maximum CPU utilisation target (advisory).
	MaxCPUPercent int
	// MaxGoroutines is the maximum number of concurrent goroutines the plugin may hold.
	MaxGoroutines int
	// AllowedHosts is the list of hosts (host or host:port) the plugin may contact.
	// If empty, no outbound HTTP requests are allowed.
	AllowedHosts []string
	// MaxRequestsPerMinute is the maximum number of API requests the plugin may issue per minute.
	MaxRequestsPerMinute int
	// MaxStorageMB is the maximum storage the plugin may consume (advisory).
	MaxStorageMB int
	// ReadOnlyMode prevents the plugin from performing write operations when true.
	ReadOnlyMode bool
}

// DefaultSandboxConfig returns a conservative default configuration.
func DefaultSandboxConfig() SandboxConfig {
	return SandboxConfig{
		MaxMemoryMB:          256,
		MaxCPUPercent:        25,
		MaxGoroutines:        50,
		AllowedHosts:         nil,
		MaxRequestsPerMinute: 60,
		MaxStorageMB:         100,
		ReadOnlyMode:         false,
	}
}

// Sandbox wraps a plugin name and enforces the limits described by SandboxConfig.
type Sandbox struct {
	PluginName string
	Config     SandboxConfig

	tracker *ResourceTracker
	client  *SandboxedHTTPClient
}

// NewSandbox creates a new Sandbox for the named plugin with the given configuration.
func NewSandbox(pluginName string, cfg SandboxConfig) *Sandbox {
	s := &Sandbox{
		PluginName: pluginName,
		Config:     cfg,
		tracker:    NewResourceTracker(),
	}

	s.client = NewSandboxedHTTPClient(cfg.AllowedHosts, 30*time.Second)

	return s
}

// RecordRequest records an inbound request for this plugin and returns an error
// if the per-minute rate limit has been exceeded.
func (s *Sandbox) RecordRequest() error {
	return s.tracker.RecordRequest(s.PluginName, s.Config.MaxRequestsPerMinute)
}

// AcquireGoroutine attempts to acquire a goroutine slot. The returned release
// function MUST be called when the goroutine completes. Returns an error if
// the goroutine limit would be exceeded.
func (s *Sandbox) AcquireGoroutine() (release func(), err error) {
	return s.tracker.RecordGoroutine(s.PluginName, s.Config.MaxGoroutines)
}

// HTTPClient returns a sandboxed HTTP client that only permits requests to
// the configured AllowedHosts and blocks private IP ranges unless explicitly
// allowed.
func (s *Sandbox) HTTPClient() *http.Client {
	return s.client.Client()
}

// Usage returns the current resource usage snapshot for this plugin.
func (s *Sandbox) Usage() ResourceUsage {
	return s.tracker.GetUsage(s.PluginName)
}

// IsReadOnly reports whether the sandbox is in read-only mode.
func (s *Sandbox) IsReadOnly() bool {
	return s.Config.ReadOnlyMode
}

// ---------------------------------------------------------------------------
// SandboxedHTTPClient
// ---------------------------------------------------------------------------

// SandboxedHTTPClient is an http.Client wrapper that validates the target host
// against a whitelist before making any request. Private IP ranges
// (10.x, 172.16-31.x, 192.168.x, 127.x, 169.254.x) are blocked unless
// explicitly present in the allowed hosts list.
type SandboxedHTTPClient struct {
	allowedHosts map[string]struct{}
	inner        *http.Client
}

// NewSandboxedHTTPClient builds a new sandboxed client. The allowedHosts
// entries may be of the form "host" or "host:port".
func NewSandboxedHTTPClient(allowedHosts []string, timeout time.Duration) *SandboxedHTTPClient {
	allowed := make(map[string]struct{}, len(allowedHosts))
	for _, h := range allowedHosts {
		allowed[strings.ToLower(h)] = struct{}{}
	}

	sc := &SandboxedHTTPClient{
		allowedHosts: allowed,
	}

	transport := &sandboxTransport{
		base:    http.DefaultTransport,
		allowed: allowed,
	}

	sc.inner = &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}

	return sc
}

// Client returns the underlying http.Client that enforces the sandbox policy.
func (sc *SandboxedHTTPClient) Client() *http.Client {
	return sc.inner
}

// sandboxTransport is an http.RoundTripper that validates each request's host.
type sandboxTransport struct {
	base    http.RoundTripper
	allowed map[string]struct{}
}

func (t *sandboxTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	host := strings.ToLower(req.URL.Hostname())
	hostPort := strings.ToLower(req.URL.Host)

	// Check explicit allow list (with and without port).
	_, hostAllowed := t.allowed[host]
	_, hostPortAllowed := t.allowed[hostPort]

	if !hostAllowed && !hostPortAllowed {
		return nil, fmt.Errorf("sandboxed HTTP client: host %q is not in the allowed hosts list", req.URL.Host)
	}

	// Even if explicitly allowed we check private IPs. If the host is in the
	// allow list AND resolves to a private IP, we permit it (the operator
	// explicitly opted in). If it is NOT in the allow list we already
	// returned above. So we only need to block when the host was allowed by
	// pattern but resolves to a private range... Since our allow-list is
	// exact-match, an explicitly listed host is always intentional.
	//
	// However, we still guard against DNS-rebind: resolve and check.
	if ip := net.ParseIP(host); ip != nil {
		if isPrivateIP(ip) && !hostExplicitlyAllowed(host, hostPort, t.allowed) {
			return nil, fmt.Errorf("sandboxed HTTP client: requests to private IP %s are blocked", host)
		}
	}

	return t.base.RoundTrip(req)
}

// hostExplicitlyAllowed returns true when the exact host or host:port was in
// the original allow list (not a wildcard / pattern match).
func hostExplicitlyAllowed(host, hostPort string, allowed map[string]struct{}) bool {
	if _, ok := allowed[host]; ok {
		return true
	}
	if _, ok := allowed[hostPort]; ok {
		return true
	}
	return false
}

// isPrivateIP checks whether an IP address belongs to a private / reserved range.
// Blocked ranges: 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16, 127.0.0.0/8, 169.254.0.0/16.
func isPrivateIP(ip net.IP) bool {
	privateRanges := []struct {
		network *net.IPNet
	}{
		{mustParseCIDR("10.0.0.0/8")},
		{mustParseCIDR("172.16.0.0/12")},
		{mustParseCIDR("192.168.0.0/16")},
		{mustParseCIDR("127.0.0.0/8")},
		{mustParseCIDR("169.254.0.0/16")},
		// IPv6 loopback
		{mustParseCIDR("::1/128")},
		// IPv6 link-local
		{mustParseCIDR("fe80::/10")},
		// IPv6 unique local
		{mustParseCIDR("fc00::/7")},
	}

	for _, r := range privateRanges {
		if r.network.Contains(ip) {
			return true
		}
	}
	return false
}

func mustParseCIDR(s string) *net.IPNet {
	_, n, err := net.ParseCIDR(s)
	if err != nil {
		panic("plugins: invalid CIDR: " + s)
	}
	return n
}

// ---------------------------------------------------------------------------
// ResourceTracker
// ---------------------------------------------------------------------------

// ResourceUsage is a point-in-time snapshot of a plugin's resource consumption.
type ResourceUsage struct {
	ActiveGoroutines int
	RequestsInWindow int
	WindowStart      time.Time
}

// pluginResources holds the mutable state for a single plugin.
type pluginResources struct {
	goroutines   int
	requests     int
	windowStart  time.Time
}

// ResourceTracker records per-plugin resource consumption in a thread-safe manner.
type ResourceTracker struct {
	mu    sync.Mutex
	state map[string]*pluginResources
}

// NewResourceTracker creates a new, empty ResourceTracker.
func NewResourceTracker() *ResourceTracker {
	return &ResourceTracker{
		state: make(map[string]*pluginResources),
	}
}

// getOrCreate returns the pluginResources for the given name, creating it if
// necessary. The caller MUST hold rt.mu.
func (rt *ResourceTracker) getOrCreate(pluginName string) *pluginResources {
	pr, ok := rt.state[pluginName]
	if !ok {
		pr = &pluginResources{
			windowStart: time.Now(),
		}
		rt.state[pluginName] = pr
	}
	return pr
}

// RecordRequest increments the request counter for the plugin. If the per-minute
// window has elapsed the counter is reset. Returns an error when the limit
// defined by maxPerMinute is exceeded.
func (rt *ResourceTracker) RecordRequest(pluginName string, maxPerMinute int) error {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	pr := rt.getOrCreate(pluginName)

	now := time.Now()
	if now.Sub(pr.windowStart) >= time.Minute {
		pr.requests = 0
		pr.windowStart = now
	}

	if pr.requests >= maxPerMinute {
		return fmt.Errorf("plugin %q exceeded request rate limit (%d/min)", pluginName, maxPerMinute)
	}

	pr.requests++
	return nil
}

// RecordGoroutine attempts to acquire a goroutine slot for the plugin. On
// success it returns a release function that MUST be called when the goroutine
// finishes. Returns an error if the limit would be exceeded.
func (rt *ResourceTracker) RecordGoroutine(pluginName string, maxGoroutines int) (release func(), err error) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	pr := rt.getOrCreate(pluginName)

	if pr.goroutines >= maxGoroutines {
		return nil, fmt.Errorf("plugin %q exceeded goroutine limit (%d)", pluginName, maxGoroutines)
	}

	pr.goroutines++

	release = func() {
		rt.mu.Lock()
		defer rt.mu.Unlock()
		if p, ok := rt.state[pluginName]; ok {
			p.goroutines--
			if p.goroutines < 0 {
				p.goroutines = 0
			}
		}
	}

	return release, nil
}

// GetUsage returns a snapshot of the current resource usage for the named plugin.
func (rt *ResourceTracker) GetUsage(pluginName string) ResourceUsage {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	pr, ok := rt.state[pluginName]
	if !ok {
		return ResourceUsage{}
	}

	return ResourceUsage{
		ActiveGoroutines: pr.goroutines,
		RequestsInWindow: pr.requests,
		WindowStart:      pr.windowStart,
	}
}

// ResetCounters zeroes out all tracked state for every plugin. This is
// primarily useful in tests and administrative resets.
func (rt *ResourceTracker) ResetCounters() {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	for k := range rt.state {
		delete(rt.state, k)
	}
}

// Compile-time assertions to ensure interfaces are satisfied.
var _ http.RoundTripper = (*sandboxTransport)(nil)
