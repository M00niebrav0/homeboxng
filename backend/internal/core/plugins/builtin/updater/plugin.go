package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hay-kot/httpkit/server"
	"github.com/rs/zerolog"
	"github.com/sysadminsmedia/homebox/backend/internal/core/plugins"
)

// Plugin provides automatic and manual update management for HomeBoxNG.
// Checks GitHub releases, performs backups before updates, and handles
// the git pull + rebuild + restart cycle.
type Plugin struct {
	logger zerolog.Logger
	pctx   plugins.PluginContext

	mu sync.Mutex

	// Configuration
	autoUpdateEnabled bool
	backupBeforeUpdate bool
	checkIntervalHours int
	githubRepo         string
	branch             string
	dataDir            string

	// State
	currentVersion  string
	latestVersion   string
	lastCheck       time.Time
	updateAvailable bool
	updateLog       []UpdateLogEntry
}

// UpdateLogEntry records an update attempt.
type UpdateLogEntry struct {
	Timestamp   time.Time `json:"timestamp"`
	FromVersion string    `json:"fromVersion"`
	ToVersion   string    `json:"toVersion"`
	Status      string    `json:"status"` // "success", "failed", "skipped"
	BackupPath  string    `json:"backupPath,omitempty"`
	Error       string    `json:"error,omitempty"`
	Duration    string    `json:"duration,omitempty"`
}

// VersionInfo is returned by the status endpoint.
type VersionInfo struct {
	CurrentVersion  string    `json:"currentVersion"`
	CurrentCommit   string    `json:"currentCommit"`
	LatestVersion   string    `json:"latestVersion"`
	LatestCommit    string    `json:"latestCommit"`
	UpdateAvailable bool      `json:"updateAvailable"`
	LastCheck       time.Time `json:"lastCheck"`
	AutoUpdate      bool      `json:"autoUpdate"`
	BackupEnabled   bool      `json:"backupEnabled"`
	Branch          string    `json:"branch"`
}

// UpdateRequest is sent by the user to trigger a manual update.
type UpdateRequest struct {
	BackupFirst bool `json:"backupFirst"`
}

// New creates a new updater plugin.
func New() *Plugin {
	return &Plugin{
		autoUpdateEnabled:  false, // OFF by default
		backupBeforeUpdate: true,  // ON by default
		checkIntervalHours: 24,
		githubRepo:         "M00niebrav0/homeboxng",
		branch:             "dev/homeboxng-init",
		dataDir:            "/data",
		updateLog:          make([]UpdateLogEntry, 0),
	}
}

func (p *Plugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "updater",
		Version:     "1.0.0",
		Description: "Automatic update management with backup-before-update support",
		Author:      "HomeBoxNG Team",
		BuiltIn:     true,
	}
}

func (p *Plugin) Init(ctx plugins.PluginContext) error {
	p.pctx = ctx
	p.logger = ctx.Logger

	// Detect current version from git
	p.currentVersion = p.detectCurrentVersion()

	p.logger.Info().
		Str("current_version", p.currentVersion).
		Bool("auto_update", p.autoUpdateEnabled).
		Bool("backup_before_update", p.backupBeforeUpdate).
		Msg("updater plugin initialized")
	return nil
}

func (p *Plugin) Start(ctx context.Context) error {
	p.logger.Info().Msg("updater plugin started")

	// Start background check loop if auto-update is enabled
	if p.autoUpdateEnabled {
		go p.backgroundCheckLoop(ctx)
	}
	return nil
}

func (p *Plugin) Stop(_ context.Context) error {
	p.logger.Info().Msg("updater plugin stopped")
	return nil
}

// backgroundCheckLoop periodically checks for updates and applies them if auto-update is on.
func (p *Plugin) backgroundCheckLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(p.checkIntervalHours) * time.Hour)
	defer ticker.Stop()

	// Initial check after 5 minutes (let the system settle)
	select {
	case <-time.After(5 * time.Minute):
	case <-ctx.Done():
		return
	}

	p.checkForUpdates()
	if p.autoUpdateEnabled && p.updateAvailable {
		p.performUpdate(p.backupBeforeUpdate)
	}

	for {
		select {
		case <-ticker.C:
			p.checkForUpdates()
			if p.autoUpdateEnabled && p.updateAvailable {
				p.performUpdate(p.backupBeforeUpdate)
			}
		case <-ctx.Done():
			return
		}
	}
}

// Routes registers updater API endpoints.
func (p *Plugin) Routes(r chi.Router) {
	// GET /api/plugins/updater/status — Current version and update status
	r.Get("/status", p.handleStatus)

	// POST /api/plugins/updater/check — Check for updates now
	r.Post("/check", p.handleCheck)

	// POST /api/plugins/updater/update — Trigger manual update
	r.Post("/update", p.handleUpdate)

	// POST /api/plugins/updater/backup — Create a manual backup
	r.Post("/backup", p.handleBackup)

	// GET /api/plugins/updater/log — Get update history
	r.Get("/log", p.handleLog)

	// GET /api/plugins/updater/changelog — Get commits between current and latest
	r.Get("/changelog", p.handleChangelog)
}

func (p *Plugin) handleStatus(w http.ResponseWriter, _ *http.Request) {
	p.mu.Lock()
	defer p.mu.Unlock()

	info := VersionInfo{
		CurrentVersion:  p.currentVersion,
		CurrentCommit:   p.getShortCommit(),
		LatestVersion:   p.latestVersion,
		LatestCommit:    "", // filled during check
		UpdateAvailable: p.updateAvailable,
		LastCheck:       p.lastCheck,
		AutoUpdate:      p.autoUpdateEnabled,
		BackupEnabled:   p.backupBeforeUpdate,
		Branch:          p.branch,
	}

	_ = server.JSON(w, http.StatusOK, info)
}

func (p *Plugin) handleCheck(w http.ResponseWriter, _ *http.Request) {
	p.checkForUpdates()

	p.mu.Lock()
	defer p.mu.Unlock()

	_ = server.JSON(w, http.StatusOK, map[string]any{
		"updateAvailable": p.updateAvailable,
		"currentVersion":  p.currentVersion,
		"latestVersion":   p.latestVersion,
		"lastCheck":       p.lastCheck,
	})
}

func (p *Plugin) handleUpdate(w http.ResponseWriter, r *http.Request) {
	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Default to backup
		req.BackupFirst = true
	}

	entry := p.performUpdate(req.BackupFirst)

	status := http.StatusOK
	if entry.Status == "failed" {
		status = http.StatusInternalServerError
	}

	_ = server.JSON(w, status, entry)
}

func (p *Plugin) handleBackup(w http.ResponseWriter, _ *http.Request) {
	backupPath, err := p.createBackup()
	if err != nil {
		_ = server.JSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	_ = server.JSON(w, http.StatusOK, map[string]string{
		"status":     "backup_created",
		"backupPath": backupPath,
	})
}

func (p *Plugin) handleLog(w http.ResponseWriter, _ *http.Request) {
	p.mu.Lock()
	defer p.mu.Unlock()

	_ = server.JSON(w, http.StatusOK, p.updateLog)
}

func (p *Plugin) handleChangelog(w http.ResponseWriter, _ *http.Request) {
	// Get commits between current HEAD and remote
	out, err := p.runGit("log", "--oneline", "HEAD..origin/"+p.branch)
	if err != nil {
		// Fetch first, then try again
		_, _ = p.runGit("fetch", "origin", p.branch)
		out, err = p.runGit("log", "--oneline", "HEAD..origin/"+p.branch)
		if err != nil {
			_ = server.JSON(w, http.StatusOK, map[string]any{
				"commits": []string{},
				"error":   "could not fetch changelog: " + err.Error(),
			})
			return
		}
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) == 1 && lines[0] == "" {
		lines = []string{}
	}

	_ = server.JSON(w, http.StatusOK, map[string]any{
		"commits":   lines,
		"count":     len(lines),
		"fromCommit": p.getShortCommit(),
	})
}

// ============================================================================
// Core operations
// ============================================================================

// checkForUpdates fetches from origin and compares HEAD to remote.
func (p *Plugin) checkForUpdates() {
	p.logger.Info().Msg("checking for updates...")

	// Fetch remote
	if _, err := p.runGit("fetch", "origin", p.branch); err != nil {
		p.logger.Error().Err(err).Msg("failed to fetch from origin")
		return
	}

	// Compare HEAD with origin
	localHead, err := p.runGit("rev-parse", "HEAD")
	if err != nil {
		p.logger.Error().Err(err).Msg("failed to get local HEAD")
		return
	}

	remoteHead, err := p.runGit("rev-parse", "origin/"+p.branch)
	if err != nil {
		p.logger.Error().Err(err).Msg("failed to get remote HEAD")
		return
	}

	localHead = strings.TrimSpace(localHead)
	remoteHead = strings.TrimSpace(remoteHead)

	p.mu.Lock()
	p.lastCheck = time.Now()
	p.updateAvailable = localHead != remoteHead
	if p.updateAvailable {
		p.latestVersion = remoteHead[:8]
	} else {
		p.latestVersion = p.currentVersion
	}
	p.mu.Unlock()

	if p.updateAvailable {
		p.logger.Info().
			Str("local", localHead[:8]).
			Str("remote", remoteHead[:8]).
			Msg("update available")
	} else {
		p.logger.Info().Msg("already up to date")
	}
}

// performUpdate runs the full update cycle: backup -> git pull -> rebuild -> restart.
func (p *Plugin) performUpdate(backupFirst bool) UpdateLogEntry {
	start := time.Now()
	entry := UpdateLogEntry{
		Timestamp:   start,
		FromVersion: p.currentVersion,
	}

	p.logger.Info().Bool("backup_first", backupFirst).Msg("starting update")

	// Step 1: Backup if requested
	if backupFirst {
		backupPath, err := p.createBackup()
		if err != nil {
			entry.Status = "failed"
			entry.Error = "backup failed: " + err.Error()
			entry.Duration = time.Since(start).String()
			p.addLogEntry(entry)
			return entry
		}
		entry.BackupPath = backupPath
		p.logger.Info().Str("path", backupPath).Msg("backup created")
	}

	// Step 2: Git pull
	out, err := p.runGit("pull", "origin", p.branch)
	if err != nil {
		entry.Status = "failed"
		entry.Error = "git pull failed: " + err.Error() + " output: " + out
		entry.Duration = time.Since(start).String()
		p.addLogEntry(entry)
		return entry
	}
	p.logger.Info().Msg("git pull completed")

	// Step 3: Rebuild Docker image
	if err := p.rebuildImage(); err != nil {
		entry.Status = "failed"
		entry.Error = "docker build failed: " + err.Error()
		entry.Duration = time.Since(start).String()
		p.addLogEntry(entry)
		return entry
	}
	p.logger.Info().Msg("docker image rebuilt")

	// Step 4: Update version info
	newVersion := p.detectCurrentVersion()
	entry.ToVersion = newVersion
	entry.Status = "success"
	entry.Duration = time.Since(start).String()

	p.mu.Lock()
	p.currentVersion = newVersion
	p.updateAvailable = false
	p.mu.Unlock()

	p.addLogEntry(entry)

	p.logger.Info().
		Str("from", entry.FromVersion).
		Str("to", entry.ToVersion).
		Str("duration", entry.Duration).
		Msg("update completed successfully — container restart required")

	// Step 5: Recreate the container (docker compose up -d)
	// This will restart the container with the new image
	go func() {
		time.Sleep(2 * time.Second) // brief delay for response to return
		p.restartContainer()
	}()

	return entry
}

// createBackup creates a tarball of the data directory.
func (p *Plugin) createBackup() (string, error) {
	timestamp := time.Now().Format("20060102-150405")
	backupDir := filepath.Dir(p.dataDir)
	backupFile := filepath.Join(backupDir, fmt.Sprintf("homeboxng-backup-%s.tar.gz", timestamp))

	cmd := exec.Command("tar", "-czf", backupFile, "-C", filepath.Dir(p.dataDir), filepath.Base(p.dataDir))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("tar failed: %w — %s", err, string(out))
	}

	p.logger.Info().Str("path", backupFile).Msg("backup created")
	return backupFile, nil
}

// rebuildImage runs docker build for the homeboxng image.
func (p *Plugin) rebuildImage() error {
	repoDir := p.findRepoDir()
	if repoDir == "" {
		return fmt.Errorf("could not find homeboxng repo directory")
	}

	cmd := exec.Command("docker", "build", "-t", "homeboxng:latest", repoDir)
	cmd.Dir = repoDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("build failed: %w — %s", err, string(out))
	}
	return nil
}

// restartContainer recreates the container using docker compose.
func (p *Plugin) restartContainer() {
	repoDir := p.findRepoDir()
	if repoDir == "" {
		p.logger.Error().Msg("could not find repo dir for restart")
		return
	}

	composeFile := filepath.Join(repoDir, "docker-compose.prod.yml")
	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		composeFile = filepath.Join(repoDir, "docker-compose.yml")
	}

	cmd := exec.Command("docker", "compose", "-f", composeFile, "up", "-d", "--force-recreate")
	cmd.Dir = repoDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		p.logger.Error().Err(err).Str("output", string(out)).Msg("container restart failed")
		return
	}
	p.logger.Info().Msg("container restarted with new image")
}

// ============================================================================
// Helpers
// ============================================================================

// detectCurrentVersion gets the short git hash of the current deployment.
func (p *Plugin) detectCurrentVersion() string {
	out, err := p.runGit("describe", "--tags", "--always")
	if err != nil {
		// Fallback to short hash
		out, err = p.runGit("rev-parse", "--short", "HEAD")
		if err != nil {
			return "unknown"
		}
	}
	return strings.TrimSpace(out)
}

func (p *Plugin) getShortCommit() string {
	out, err := p.runGit("rev-parse", "--short", "HEAD")
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(out)
}

func (p *Plugin) findRepoDir() string {
	// Check common locations
	candidates := []string{
		"/opt/stack/homeboxng",
		"/opt/homeboxng",
		"/app",
	}
	for _, dir := range candidates {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
	}

	// Try git rev-parse
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").CombinedOutput()
	if err == nil {
		return strings.TrimSpace(string(out))
	}

	return ""
}

func (p *Plugin) runGit(args ...string) (string, error) {
	repoDir := p.findRepoDir()
	if repoDir == "" {
		return "", fmt.Errorf("repo directory not found")
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = repoDir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func (p *Plugin) addLogEntry(entry UpdateLogEntry) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.updateLog = append(p.updateLog, entry)
	// Keep last 50 entries
	if len(p.updateLog) > 50 {
		p.updateLog = p.updateLog[len(p.updateLog)-50:]
	}
}

// ============================================================================
// Plugin interface implementations
// ============================================================================

// ConfigSchema returns updater configuration options.
func (p *Plugin) ConfigSchema() []plugins.ConfigField {
	return []plugins.ConfigField{
		{
			Key:         "auto_update",
			Label:       "Automatic Updates",
			Description: "Automatically check for and install updates (requires container restart)",
			Type:        "boolean",
			Default:     "false",
			EnvVar:      "HBOX_AUTO_UPDATE",
			Required:    false,
		},
		{
			Key:         "backup_before_update",
			Label:       "Backup Before Update",
			Description: "Create a backup of HomeBoxNG data before applying updates",
			Type:        "boolean",
			Default:     "true",
			EnvVar:      "HBOX_BACKUP_BEFORE_UPDATE",
			Required:    false,
		},
		{
			Key:         "check_interval_hours",
			Label:       "Check Interval (hours)",
			Description: "How often to check for updates (in hours, only when auto-update is enabled)",
			Type:        "number",
			Default:     "24",
			EnvVar:      "HBOX_UPDATE_CHECK_INTERVAL",
			Required:    false,
		},
		{
			Key:         "github_repo",
			Label:       "GitHub Repository",
			Description: "GitHub repository to check for updates (owner/repo)",
			Type:        "string",
			Default:     "M00niebrav0/homeboxng",
			EnvVar:      "HBOX_UPDATE_REPO",
			Required:    false,
		},
		{
			Key:         "branch",
			Label:       "Branch",
			Description: "Git branch to track for updates",
			Type:        "string",
			Default:     "dev/homeboxng-init",
			EnvVar:      "HBOX_UPDATE_BRANCH",
			Required:    false,
		},
		{
			Key:         "data_dir",
			Label:       "Data Directory",
			Description: "Path to HomeBoxNG data directory for backups",
			Type:        "string",
			Default:     "/data",
			EnvVar:      "HBOX_DATA_DIR",
			Required:    false,
		},
	}
}

// Configure applies configuration values.
func (p *Plugin) Configure(values map[string]string) error {
	if v, ok := values["auto_update"]; ok {
		p.autoUpdateEnabled = v == "true" || v == "1"
	}
	if v, ok := values["backup_before_update"]; ok {
		p.backupBeforeUpdate = v == "true" || v == "1"
	}
	if v, ok := values["check_interval_hours"]; ok && v != "" {
		var hours int
		if _, err := fmt.Sscanf(v, "%d", &hours); err == nil && hours > 0 {
			p.checkIntervalHours = hours
		}
	}
	if v, ok := values["github_repo"]; ok && v != "" {
		p.githubRepo = v
	}
	if v, ok := values["branch"]; ok && v != "" {
		p.branch = v
	}
	if v, ok := values["data_dir"]; ok && v != "" {
		p.dataDir = v
	}
	return nil
}

// RequestedPermissions declares what the updater plugin needs.
func (p *Plugin) RequestedPermissions() []plugins.PermissionRequest {
	return []plugins.PermissionRequest{
		{Permission: plugins.PermAPIRoutes, Reason: "Update management API endpoints", Required: true},
		{Permission: plugins.PermConfig, Reason: "Auto-update and backup settings", Required: true},
		{Permission: plugins.PermStorage, Reason: "Create data backups before updates", Required: false},
		{Permission: plugins.PermNetwork, Reason: "Fetch updates from GitHub", Required: true},
	}
}

// Compile-time interface checks.
var (
	_ plugins.Plugin                = (*Plugin)(nil)
	_ plugins.RoutePlugin           = (*Plugin)(nil)
	_ plugins.ConfigPlugin          = (*Plugin)(nil)
	_ plugins.PluginWithPermissions = (*Plugin)(nil)
)
