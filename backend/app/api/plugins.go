package main

import (
	"github.com/rs/zerolog/log"
	"github.com/sysadminsmedia/homebox/backend/internal/core/plugins"
	"github.com/sysadminsmedia/homebox/backend/internal/core/plugins/builtin/aivision"
	"github.com/sysadminsmedia/homebox/backend/internal/core/plugins/builtin/batteries"
	"github.com/sysadminsmedia/homebox/backend/internal/core/plugins/builtin/analytics"
	"github.com/sysadminsmedia/homebox/backend/internal/core/plugins/builtin/excelexport"
	"github.com/sysadminsmedia/homebox/backend/internal/core/plugins/builtin/eyefi"
	"github.com/sysadminsmedia/homebox/backend/internal/core/plugins/builtin/example"
	"github.com/sysadminsmedia/homebox/backend/internal/core/plugins/builtin/habridge"
	"github.com/sysadminsmedia/homebox/backend/internal/core/plugins/builtin/itasset"
	"github.com/sysadminsmedia/homebox/backend/internal/core/plugins/builtin/labelprinter"
	"github.com/sysadminsmedia/homebox/backend/internal/core/plugins/builtin/lending"
	"github.com/sysadminsmedia/homebox/backend/internal/core/plugins/builtin/maintenance"
	"github.com/sysadminsmedia/homebox/backend/internal/core/plugins/builtin/manuals"
	"github.com/sysadminsmedia/homebox/backend/internal/core/plugins/builtin/paperless"
	"github.com/sysadminsmedia/homebox/backend/internal/core/plugins/builtin/shopping"
	notifyDiscord "github.com/sysadminsmedia/homebox/backend/internal/core/plugins/builtin/notifications/discord"
	notifyEmail "github.com/sysadminsmedia/homebox/backend/internal/core/plugins/builtin/notifications/email"
	notifyGotify "github.com/sysadminsmedia/homebox/backend/internal/core/plugins/builtin/notifications/gotify"
	notifyNtfy "github.com/sysadminsmedia/homebox/backend/internal/core/plugins/builtin/notifications/ntfy"
	notifyWebpush "github.com/sysadminsmedia/homebox/backend/internal/core/plugins/builtin/notifications/webpush"
)

// registerBuiltinPlugins registers all built-in plugins that ship with HomeBoxNG.
// These plugins cannot be uninstalled but can be disabled via the Plugin Manager UI.
//
// Built-in plugins are Go implementations that compile directly into the binary.
// Third-party plugins connect as external services via the Plugin API.
func registerBuiltinPlugins(app *app) {
	// Register example plugin (developer reference)
	if err := app.pluginRegistry.Register(example.New()); err != nil {
		log.Error().Err(err).Msg("failed to register example plugin")
	}

	// =========================================================================
	// Notification Plugins
	//
	// Each platform is a separate plugin that users enable/disable independently.
	// Users choose which platforms they want notifications on:
	// - Email (SMTP)
	// - Discord (webhook)
	// - Web Push (Chrome, Firefox, Edge, Safari)
	// - Gotify (self-hosted, Android/Windows/Linux/iOS via UnifiedPush)
	// - ntfy (self-hosted or ntfy.sh, all platforms)

	notificationPlugins := []plugins.NotificationPlugin{
		notifyEmail.New(),
		notifyDiscord.New(),
		notifyWebpush.New(),
		notifyGotify.New(),
		notifyNtfy.New(),
	}

	for _, np := range notificationPlugins {
		if err := app.pluginRegistry.Register(np); err != nil {
			log.Error().Err(err).Str("plugin", np.Info().Name).Msg("failed to register notification plugin")
			continue
		}
		// Also register with the notification dispatcher for routing
		if app.notificationDispatcher != nil {
			app.notificationDispatcher.RegisterPlatform(np)
		}
	}

	log.Info().Int("notification_platforms", len(notificationPlugins)).Msg("notification plugins registered")

	// =========================================================================
	// AI Vision Plugin
	//
	// Two-step pipeline: Ollama/Qwen3-VL for vision -> Gemini for verification
	if err := app.pluginRegistry.Register(aivision.New()); err != nil {
		log.Error().Err(err).Msg("failed to register ai-vision plugin")
	}

	// =========================================================================
	// Label Printer Plugin
	//
	// Brother QL thermal labels with QR codes, smart presets, half-label support
	if err := app.pluginRegistry.Register(labelprinter.New()); err != nil {
		log.Error().Err(err).Msg("failed to register label-printer plugin")
	}

	// =========================================================================
	// Eye-Fi Plugin
	//
	// WiFi SD card receiver with SOAP protocol for wireless camera uploads
	if err := app.pluginRegistry.Register(eyefi.New()); err != nil {
		log.Error().Err(err).Msg("failed to register eyefi plugin")
	}

	// =========================================================================
	// Excel Export Plugin
	//
	// CSV/Excel export and import for items, locations, and labels
	if err := app.pluginRegistry.Register(excelexport.New()); err != nil {
		log.Error().Err(err).Msg("failed to register excel-export plugin")
	}

	// =========================================================================
	// Paperless-ngx Bridge Plugin
	//
	// Bidirectional document-item linking with Paperless-ngx
	if err := app.pluginRegistry.Register(paperless.New()); err != nil {
		log.Error().Err(err).Msg("failed to register paperless plugin")
	}

	// =========================================================================
	// Analytics Plugin
	//
	// Dashboard metrics, activity feeds, value tracking, category breakdowns
	if err := app.pluginRegistry.Register(analytics.New()); err != nil {
		log.Error().Err(err).Msg("failed to register analytics plugin")
	}

	// =========================================================================
	// Home Assistant Bridge Plugin
	//
	// MQTT Discovery for HA entities, bidirectional sync, QR scan events
	if err := app.pluginRegistry.Register(habridge.New()); err != nil {
		log.Error().Err(err).Msg("failed to register ha-bridge plugin")
	}

	// =========================================================================
	// Lending Plugin - Item checkout and return tracking
	if err := app.pluginRegistry.Register(lending.New()); err != nil {
		log.Error().Err(err).Msg("failed to register lending plugin")
	}

	// =========================================================================
	// Maintenance Plugin - Scheduled maintenance and repair logging
	if err := app.pluginRegistry.Register(maintenance.New()); err != nil {
		log.Error().Err(err).Msg("failed to register maintenance plugin")
	}

	// =========================================================================
	// Shopping Plugin - Shopping list with reorder triggers
	if err := app.pluginRegistry.Register(shopping.New()); err != nil {
		log.Error().Err(err).Msg("failed to register shopping plugin")
	}

	// =========================================================================
	// Batteries & Storage Plugin - Power tool battery, charger, and modular
	// storage system tracking (Milwaukee PACKOUT, Ryobi LINK, DeWalt TOUGHSYSTEM, etc.)
	if err := app.pluginRegistry.Register(batteries.New()); err != nil {
		log.Error().Err(err).Msg("failed to register batteries plugin")
	}

	// =========================================================================
	// IT Asset Plugin - Server, desktop, laptop, and network equipment tracking
	// with detailed hardware components (CPU, RAM, storage, GPU, NIC, PSU),
	// remote connection protocols, and VaultWarden credential references.
	if err := app.pluginRegistry.Register(itasset.New()); err != nil {
		log.Error().Err(err).Msg("failed to register it-assets plugin")
	}

	// =========================================================================
	// Manuals Plugin - Manual lookup, fuzzy matching, ManualsLib search,
	// and user-uploaded manual management for inventory items.
	if err := app.pluginRegistry.Register(manuals.New()); err != nil {
		log.Error().Err(err).Msg("failed to register manuals plugin")
	}

	// Planned built-in plugins (registered as they are implemented):
	// - discord-bot:   Full Discord bot integration
	// - verification:  Insurance photo verification
	// - voice:         Voice control API
	// - nfc:           NFC tag support
	// - geolocation:   Multi-property tracking
	// - 3d-print:      3D print suggestions

	log.Info().Int("total", len(app.pluginRegistry.List())).Msg("built-in plugins registered")
}
