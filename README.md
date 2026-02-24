<div align="center">
  <img src="/docs/src/assets/lilbox.svg" height="200"/>
</div>

<h1 align="center" style="margin-top: -10px;"> HomeBoxNG </h1>
<p align="center"><em>HomeBox, supercharged.</em></p>
<p align="center" style="width: 100%;">
   <a href="#quick-start">Quick Start</a>
   |
   <a href="#whats-new-in-homeboxng">What's New</a>
   |
   <a href="#plugin-system">Plugins</a>
</p>
<p align="center" style="width: 100%;">
    <img src="https://img.shields.io/github/license/M00niebrav0/homeboxng" alt="License"/>
    <img src="https://img.shields.io/github/v/release/M00niebrav0/homeboxng?sort=semver&display_name=release" alt="Release"/>
</p>

## What is HomeBoxNG?

HomeBoxNG is a fork of [HomeBox](https://github.com/sysadminsmedia/homebox) that keeps everything you love about the original -- simple, fast, portable -- and adds power-user features for people who want to go deeper.

**Same foundation. Same principles. More capability.**

### HomeBox Core Principles (preserved)

- **Simple but Expandable** - No complicated setup. Works out of the box. Expand as needed.
- **Blazingly Fast** - Written in Go. Under 50MB idle memory. Minimal resources.
- **Portable** - SQLite + embedded web UI. Easy to deploy, use, and backup.

### HomeBox Core Features (preserved)

- Rich Organization - Locations, tags, custom fields
- Powerful Search - Full-text search across your inventory
- Image Upload - Photo documentation for every item
- Document & Warranty Tracking - Receipts, manuals, warranty dates
- Purchase & Maintenance Tracking - Costs, schedules, service history
- Responsive Design - Desktop, tablet, and mobile

---

## What's New in HomeBoxNG

### AI-Powered Vision Pipeline
Send photos of your items and AI identifies them automatically. Uses local LLMs (Ollama/Qwen3-VL) or cloud models (Gemini, GPT-4o). Multi-photo grouping with EXIF timestamp correlation.

### Plugin System
First-class plugin architecture. Extend HomeBoxNG with custom functionality without forking. Plugins can add API endpoints, event hooks, UI pages, and scheduled tasks. [Learn more](#plugin-system)

### Deep Location Hierarchy
Unlimited nesting depth. Server > Rack > Shelf > Drive Bay > Disk 1. Containers, drawers, jars, bins -- organize however your brain works. OneNote-style drill-down navigation.

### Discord Bot Integration
Full inventory management from Discord. Natural language queries, photo scanning, label printing, shopping lists. "Where is my drill?" just works.

### Smart Label Printing
Brother QL thermal labels with QR codes. Auto-presets by location type (Alex drawers get half-size, server racks get detailed). Scan any label to jump straight to that location.

### Eye-Fi / WiFi SD Card Support
Shoot photos on a DSLR with an Eye-Fi card, they upload to HomeBoxNG automatically. Full SOAP protocol implementation with auto-setup from USB card reader.

### Item Lending & Checkout
Track items lent to friends and family. Due dates, overdue alerts, automatic HomeBox notes. Never lose track of borrowed tools again.

### Insurance Verification
Value-tiered re-verification schedules. High-value items every 6 months, low-value every 2 years. Generates verification reports for insurance claims.

### Multi-Property Tracking
Home, cabin, storage unit, office. Track which property items are at. Move items between properties with one command.

### Excel Import/Export
Full inventory as formatted .xlsx with auto-sized columns, conditional formatting, and hyperlinks back to HomeBox.

### Home Assistant Bridge
MQTT-based real-time sync. HomeBox items become HA sensor entities. QR scan fires HA events for automations.

### Paperless-ngx Integration
Bi-directional document linking. Receipts in Paperless auto-link to matching HomeBox items.

### Voice Control API
"Alexa, ask HomeBox where is my drill?" Natural language query endpoint for any voice assistant.

### NFC Tag Support
Generate NDEF payloads for NTAG215 stickers. Tap your phone on any tagged location to open it instantly.

### 3D Print Suggestions
Search Makerworld and Thingiverse for organizers that fit your items and storage locations.

### Shopping List & Reorder
Low-stock alerts, per-store grouping, price estimates. Auto-add consumables when maintenance is due.

### Analytics Dashboard
Charts, stats, activity feeds. Item counts, value over time, category breakdown, storage utilization.

---

## Plugin System

HomeBoxNG introduces a plugin architecture that lets you extend functionality without modifying core code.

### What Plugins Can Do

| Capability | Example |
|-----------|---------|
| **API Endpoints** | Add custom REST routes under `/api/plugins/{name}/` |
| **Event Hooks** | React to item created/updated/deleted, location changes |
| **Scheduled Tasks** | Run periodic jobs (backups, sync, alerts) |
| **UI Pages** | Add pages to the web interface via plugin slots |
| **Config Sections** | Register plugin-specific settings |

### Plugin Lifecycle

```
Register -> Configure -> Enable -> [Hooks fire on events] -> Disable -> Unregister
```

### Built-in Plugins (ship with HomeBoxNG)

| Plugin | Description |
|--------|-------------|
| `ai-vision` | Photo identification via local/cloud LLMs |
| `discord-bot` | Full Discord bot integration |
| `label-printer` | Brother QL thermal label printing |
| `ha-bridge` | Home Assistant MQTT bridge |
| `paperless` | Paperless-ngx document linking |
| `eyefi` | Eye-Fi WiFi SD card receiver |

### Third-Party Plugins

Plugins are Go packages that implement the `Plugin` interface. Drop them in the plugins directory or register via the API.

---

## Quick Start

```bash
mkdir -p /path/to/data/folder
docker run -d \
  --name homeboxng \
  --restart unless-stopped \
  --publish 3100:7745 \
  --env TZ=America/New_York \
  --volume /path/to/data/folder/:/data \
  ghcr.io/m00niebrav0/homeboxng:latest
```

### Upgrading from HomeBox

HomeBoxNG is a drop-in replacement. Point it at your existing HomeBox data volume and it just works. All existing data, locations, items, and attachments are preserved.

```bash
# Stop HomeBox
docker stop homebox

# Start HomeBoxNG with the same data volume
docker run -d \
  --name homeboxng \
  --restart unless-stopped \
  --publish 3100:7745 \
  --volume /path/to/existing-homebox-data/:/data \
  ghcr.io/m00niebrav0/homeboxng:latest
```

---

## Syncing with Upstream HomeBox

HomeBoxNG regularly merges updates from [sysadminsmedia/homebox](https://github.com/sysadminsmedia/homebox) to stay current with bug fixes and improvements.

```bash
git remote add upstream https://github.com/sysadminsmedia/homebox.git
git fetch upstream
git merge upstream/main
```

---

## Credits

- **Original HomeBox** by [@hay-kot](https://github.com/hay-kot)
- **HomeBox continuation** by [sysadminsmedia](https://github.com/sysadminsmedia/homebox)
- **HomeBox logo** by [@lakotelman](https://github.com/lakotelman)
- **HomeBoxNG** by [@M00niebrav0](https://github.com/M00niebrav0)

## License

AGPL-3.0 -- same as HomeBox. See [LICENSE](LICENSE) for details.
