package itasset

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/hay-kot/httpkit/server"
	"github.com/rs/zerolog"
	"github.com/sysadminsmedia/homebox/backend/internal/core/plugins"
)

// Plugin provides IT asset management for HomeBoxNG.
// Tracks servers, desktops, laptops, and networking equipment with detailed
// hardware component tracking (CPU, RAM, storage, PCIe, GPU, NIC, PSU),
// remote connection info (SSH, RDP, IPMI), and VaultWarden credential references.
type Plugin struct {
	logger zerolog.Logger
	pctx   plugins.PluginContext
}

// ============================================================================
// Templates — predefined IT asset types with expected component categories
// ============================================================================

// AssetTemplate defines a type of IT asset and which hardware component
// categories are relevant for that type.
type AssetTemplate struct {
	Slug        string   `json:"slug"`
	Label       string   `json:"label"`
	Description string   `json:"description"`
	FormFactors []string `json:"formFactors"`
	Components  struct {
		CPU        bool `json:"cpu"`
		Memory     bool `json:"memory"`
		Storage    bool `json:"storage"`
		PCIe       bool `json:"pcie"`
		GPU        bool `json:"gpu"`
		NIC        bool `json:"nic"`
		PSU        bool `json:"psu"`
		Display    bool `json:"display"`
	} `json:"components"`
	SuggestedLabels []string `json:"suggestedLabels"`
}

// HardwareLabel defines the label taxonomy for hardware components.
type HardwareLabel struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

// ConnectionProtocol defines a supported remote access protocol.
type ConnectionProtocol struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Port  int    `json:"defaultPort"`
	Tools []string `json:"tools"`
}

// ComponentFieldSchema defines the custom fields expected for a hardware component type.
type ComponentFieldSchema struct {
	ComponentType string        `json:"componentType"`
	Label         string        `json:"label"`
	Fields        []FieldDef    `json:"fields"`
}

// FieldDef defines a single field in a component schema.
type FieldDef struct {
	Key         string   `json:"key"`
	Label       string   `json:"label"`
	Type        string   `json:"type"` // "text", "number", "boolean", "select"
	Options     []string `json:"options,omitempty"`
	Required    bool     `json:"required"`
	Description string   `json:"description,omitempty"`
}

// ============================================================================
// Static data — asset templates, labels, protocols, component schemas
// ============================================================================

var assetTemplates = []AssetTemplate{
	{
		Slug: "server-rackmount", Label: "Server (Rackmount)",
		Description: "Rackmount server (1U-5U+)",
		FormFactors: []string{"1U", "2U", "3U", "4U", "5U+"},
		SuggestedLabels: []string{"it-server", "hw-rackmount"},
	},
	{
		Slug: "server-tower", Label: "Server (Tower)",
		Description: "Tower/pedestal server",
		FormFactors: []string{"Mini Tower", "Mid Tower", "Full Tower"},
		SuggestedLabels: []string{"it-server"},
	},
	{
		Slug: "server-blade", Label: "Server (Blade)",
		Description: "Blade server in a chassis",
		FormFactors: []string{"Half-Height", "Full-Height"},
		SuggestedLabels: []string{"it-server", "hw-blade"},
	},
	{
		Slug: "desktop-pc", Label: "Desktop PC",
		Description: "Desktop workstation or gaming PC",
		FormFactors: []string{"SFF", "Mini-ITX", "Micro-ATX", "Mid-Tower", "Full-Tower"},
		SuggestedLabels: []string{"it-desktop"},
	},
	{
		Slug: "laptop", Label: "Laptop",
		Description: "Laptop or notebook computer",
		FormFactors: []string{"Ultrabook", "Standard", "Workstation", "Gaming", "Rugged", "2-in-1"},
		SuggestedLabels: []string{"it-laptop"},
	},
	{
		Slug: "nas", Label: "NAS",
		Description: "Network Attached Storage device",
		FormFactors: []string{"Desktop (2-bay)", "Desktop (4-bay)", "Desktop (6-bay)", "Desktop (8-bay)", "Rackmount"},
		SuggestedLabels: []string{"it-nas"},
	},
	{
		Slug: "network-switch", Label: "Network Switch",
		Description: "Ethernet switch",
		FormFactors: []string{"Desktop (5-port)", "Desktop (8-port)", "Desktop (16-port)", "Rackmount (24-port)", "Rackmount (48-port)"},
		SuggestedLabels: []string{"it-switch"},
	},
	{
		Slug: "router-firewall", Label: "Router / Firewall",
		Description: "Network router or firewall appliance",
		FormFactors: []string{"Appliance", "Rackmount", "Desktop", "Virtual"},
		SuggestedLabels: []string{"it-router"},
	},
	{
		Slug: "ups", Label: "UPS",
		Description: "Uninterruptible Power Supply",
		FormFactors: []string{"Tower", "Rackmount (1U)", "Rackmount (2U)", "Inline/Strip"},
		SuggestedLabels: []string{"it-ups"},
	},
	{
		Slug: "kvm", Label: "KVM",
		Description: "KVM switch or IP-KVM",
		FormFactors: []string{"Desktop", "Rackmount", "IP-KVM"},
		SuggestedLabels: []string{"it-kvm"},
	},
	{
		Slug: "monitor", Label: "Monitor / Display",
		Description: "Monitor, display, or TV used with IT equipment",
		FormFactors: []string{"Standard", "Ultrawide", "Portable", "Rack Console"},
		SuggestedLabels: []string{"it-monitor"},
	},
	{
		Slug: "printer", Label: "Printer / Scanner",
		Description: "Printer, scanner, or MFP",
		FormFactors: []string{"Desktop Inkjet", "Desktop Laser", "Desktop MFP", "Large Format", "Label Printer"},
		SuggestedLabels: []string{"it-printer"},
	},
}

var hardwareLabels = []HardwareLabel{
	{Key: "hw-cpu", Label: "CPU / Processor", Description: "Central processing unit", Icon: "mdi-chip"},
	{Key: "hw-memory", Label: "RAM / DIMM", Description: "Memory module", Icon: "mdi-memory"},
	{Key: "hw-storage", Label: "Disk / Drive", Description: "HDD, SSD, NVMe storage", Icon: "mdi-harddisk"},
	{Key: "hw-pcie", Label: "PCIe Card", Description: "PCIe expansion card", Icon: "mdi-expansion-card"},
	{Key: "hw-gpu", Label: "Graphics Card", Description: "GPU / video card", Icon: "mdi-gpu"},
	{Key: "hw-nic-port", Label: "Network Port", Description: "Network interface", Icon: "mdi-ethernet"},
	{Key: "hw-psu", Label: "Power Supply", Description: "PSU", Icon: "mdi-flash"},
	{Key: "hw-motherboard", Label: "Motherboard", Description: "Main board", Icon: "mdi-developer-board"},
	{Key: "hw-chassis", Label: "Chassis", Description: "Enclosure / case", Icon: "mdi-server"},
	{Key: "hw-cable", Label: "Cable", Description: "Network, power, or data cable", Icon: "mdi-cable-data"},
}

var connectionProtocols = []ConnectionProtocol{
	{ID: "ssh", Label: "SSH", Port: 22, Tools: []string{"PuTTY", "TeraTerm", "OpenSSH", "mRemoteNG", "WinSCP"}},
	{ID: "rdp", Label: "RDP", Port: 3389, Tools: []string{"mRemoteNG", "Remote Desktop", "Remmina", "FreeRDP"}},
	{ID: "vnc", Label: "VNC", Port: 5900, Tools: []string{"TightVNC", "RealVNC", "mRemoteNG", "Remmina"}},
	{ID: "telnet", Label: "Telnet", Port: 23, Tools: []string{"PuTTY", "TeraTerm", "mRemoteNG"}},
	{ID: "http", Label: "HTTP", Port: 80, Tools: []string{"browser"}},
	{ID: "https", Label: "HTTPS", Port: 443, Tools: []string{"browser"}},
	{ID: "ipmi", Label: "IPMI / BMC", Port: 623, Tools: []string{"browser", "ipmitool"}},
	{ID: "ilo", Label: "iLO (HP)", Port: 443, Tools: []string{"browser"}},
	{ID: "idrac", Label: "iDRAC (Dell)", Port: 443, Tools: []string{"browser"}},
	{ID: "kvm-ip", Label: "KVM over IP", Port: 443, Tools: []string{"browser", "Java KVM client"}},
	{ID: "serial", Label: "Serial Console", Port: 0, Tools: []string{"PuTTY", "TeraTerm", "minicom"}},
	{ID: "snmp", Label: "SNMP", Port: 161, Tools: []string{"snmpwalk", "LibreNMS", "Zabbix"}},
	{ID: "winrm", Label: "WinRM", Port: 5985, Tools: []string{"PowerShell", "Ansible"}},
}

var componentSchemas = []ComponentFieldSchema{
	{
		ComponentType: "hw-cpu", Label: "CPU",
		Fields: []FieldDef{
			{Key: "socket", Label: "Socket", Type: "text", Description: "e.g., CPU0, LGA1700"},
			{Key: "manufacturer", Label: "Manufacturer", Type: "text"},
			{Key: "model", Label: "Model", Type: "text", Required: true},
			{Key: "microarchitecture", Label: "Architecture", Type: "text", Description: "e.g., Zen 4, Raptor Lake"},
			{Key: "cores", Label: "Cores", Type: "number"},
			{Key: "threads", Label: "Threads", Type: "number"},
			{Key: "base_clock_mhz", Label: "Base Clock (MHz)", Type: "number"},
			{Key: "boost_clock_mhz", Label: "Boost Clock (MHz)", Type: "number"},
			{Key: "tdp_watts", Label: "TDP (Watts)", Type: "number"},
		},
	},
	{
		ComponentType: "hw-memory", Label: "Memory",
		Fields: []FieldDef{
			{Key: "slot", Label: "Slot", Type: "text", Description: "e.g., DIMM_A1"},
			{Key: "type", Label: "Type", Type: "select", Options: []string{"DDR3", "DDR4", "DDR5", "DDR3L", "DDR4L", "LPDDR4", "LPDDR5"}},
			{Key: "ecc", Label: "ECC", Type: "boolean"},
			{Key: "registered", Label: "Registered (RDIMM)", Type: "boolean"},
			{Key: "capacity_gb", Label: "Capacity (GB)", Type: "number", Required: true},
			{Key: "speed_mhz", Label: "Speed (MHz)", Type: "number"},
			{Key: "manufacturer", Label: "Manufacturer", Type: "text"},
			{Key: "part_number", Label: "Part Number", Type: "text"},
		},
	},
	{
		ComponentType: "hw-storage", Label: "Storage",
		Fields: []FieldDef{
			{Key: "bay", Label: "Bay/Slot", Type: "text"},
			{Key: "disk_type", Label: "Disk Type", Type: "select", Options: []string{"HDD", "SSD", "NVMe", "M.2-SATA", "M.2-NVMe", "U.2", "mSATA"}, Required: true},
			{Key: "form_factor", Label: "Form Factor", Type: "select", Options: []string{"3.5\"", "2.5\"", "M.2-2280", "M.2-2242", "M.2-2230", "U.2", "AIC"}},
			{Key: "interface", Label: "Interface", Type: "select", Options: []string{"SATA II", "SATA III", "SAS 6Gbps", "SAS 12Gbps", "NVMe PCIe 3.0", "NVMe PCIe 4.0", "NVMe PCIe 5.0", "USB 3.0"}},
			{Key: "capacity_gb", Label: "Capacity (GB)", Type: "number", Required: true},
			{Key: "manufacturer", Label: "Manufacturer", Type: "text"},
			{Key: "model", Label: "Model", Type: "text"},
			{Key: "firmware", Label: "Firmware", Type: "text"},
			{Key: "health_status", Label: "Health", Type: "select", Options: []string{"healthy", "warning", "failing", "unknown"}},
			{Key: "smart_hours", Label: "Power-On Hours", Type: "number"},
			{Key: "smart_reallocated", Label: "Reallocated Sectors", Type: "number"},
			{Key: "smart_temperature", Label: "Temperature (C)", Type: "number"},
		},
	},
	{
		ComponentType: "hw-gpu", Label: "GPU",
		Fields: []FieldDef{
			{Key: "slot", Label: "PCIe Slot", Type: "number"},
			{Key: "manufacturer", Label: "Manufacturer", Type: "text"},
			{Key: "model", Label: "Model", Type: "text", Required: true},
			{Key: "vram_gb", Label: "VRAM (GB)", Type: "number"},
			{Key: "vram_type", Label: "VRAM Type", Type: "select", Options: []string{"GDDR5", "GDDR5X", "GDDR6", "GDDR6X", "HBM2", "HBM2e", "HBM3"}},
			{Key: "base_clock_mhz", Label: "Base Clock (MHz)", Type: "number"},
			{Key: "boost_clock_mhz", Label: "Boost Clock (MHz)", Type: "number"},
			{Key: "tdp_watts", Label: "TDP (Watts)", Type: "number"},
			{Key: "driver_version", Label: "Driver Version", Type: "text"},
			{Key: "passthrough_to", Label: "Passthrough To", Type: "text", Description: "VM/container using this GPU"},
			{Key: "passthrough_method", Label: "Passthrough Method", Type: "select", Options: []string{"none", "PCI passthrough", "cgroup2", "vGPU", "SR-IOV"}},
		},
	},
	{
		ComponentType: "hw-pcie", Label: "PCIe Card",
		Fields: []FieldDef{
			{Key: "slot", Label: "Slot Number", Type: "number"},
			{Key: "pcie_gen", Label: "PCIe Gen", Type: "select", Options: []string{"1", "2", "3", "4", "5"}},
			{Key: "pcie_lanes", Label: "Lanes", Type: "select", Options: []string{"1", "4", "8", "16"}},
			{Key: "card_type", Label: "Card Type", Type: "select", Options: []string{"NIC", "HBA", "RAID", "GPU", "Sound", "Capture", "NVMe Adapter", "USB Expansion", "Thunderbolt", "Other"}, Required: true},
			{Key: "manufacturer", Label: "Manufacturer", Type: "text"},
			{Key: "model", Label: "Model", Type: "text"},
			{Key: "firmware", Label: "Firmware", Type: "text"},
		},
	},
	{
		ComponentType: "hw-nic-port", Label: "Network Port",
		Fields: []FieldDef{
			{Key: "port_number", Label: "Port Number", Type: "number"},
			{Key: "interface_name", Label: "Interface Name", Type: "text", Description: "e.g., eth0, vmbr0"},
			{Key: "speed", Label: "Speed", Type: "select", Options: []string{"100Mbps", "1GbE", "2.5GbE", "5GbE", "10GbE", "25GbE", "40GbE", "100GbE"}, Required: true},
			{Key: "mac_address", Label: "MAC Address", Type: "text"},
			{Key: "is_onboard", Label: "Onboard", Type: "boolean"},
			{Key: "link_status", Label: "Link Status", Type: "select", Options: []string{"up", "down", "unknown"}},
			{Key: "connected_to", Label: "Connected To", Type: "text", Description: "Device/port this connects to"},
			{Key: "sfp_type", Label: "SFP Type", Type: "select", Options: []string{"", "SFP", "SFP+", "QSFP", "QSFP28", "QSFP56", "RJ45", "DAC"}},
		},
	},
	{
		ComponentType: "hw-psu", Label: "Power Supply",
		Fields: []FieldDef{
			{Key: "psu_number", Label: "PSU Number", Type: "number"},
			{Key: "wattage", Label: "Wattage", Type: "number", Required: true},
			{Key: "efficiency", Label: "Efficiency", Type: "select", Options: []string{"80+", "80+ Bronze", "80+ Silver", "80+ Gold", "80+ Platinum", "80+ Titanium"}},
			{Key: "form_factor", Label: "Form Factor", Type: "select", Options: []string{"ATX", "SFX", "SFX-L", "TFX", "Flex ATX", "1U Redundant", "2U Redundant", "Server"}},
			{Key: "modular", Label: "Modular", Type: "select", Options: []string{"Non-modular", "Semi-modular", "Fully Modular"}},
			{Key: "manufacturer", Label: "Manufacturer", Type: "text"},
			{Key: "model", Label: "Model", Type: "text"},
		},
	},
}

func init() {
	// Set component flags on templates
	for i := range assetTemplates {
		t := &assetTemplates[i]
		switch t.Slug {
		case "server-rackmount", "server-tower", "desktop-pc":
			t.Components.CPU = true
			t.Components.Memory = true
			t.Components.Storage = true
			t.Components.PCIe = true
			t.Components.GPU = true
			t.Components.NIC = true
			t.Components.PSU = true
		case "server-blade":
			t.Components.CPU = true
			t.Components.Memory = true
			t.Components.Storage = true
			t.Components.PCIe = true
			t.Components.GPU = true
			t.Components.NIC = true
		case "laptop":
			t.Components.CPU = true
			t.Components.Memory = true
			t.Components.Storage = true
			t.Components.GPU = true
			t.Components.NIC = true
			t.Components.Display = true
		case "nas":
			t.Components.CPU = true
			t.Components.Memory = true
			t.Components.Storage = true
			t.Components.NIC = true
			t.Components.PSU = true
		case "network-switch", "router-firewall":
			t.Components.NIC = true
			t.Components.PSU = true
		case "kvm":
			t.Components.NIC = true
			t.Components.Display = true
		case "monitor":
			t.Components.Display = true
		}
	}
}

// New creates a new IT asset management plugin.
func New() *Plugin {
	return &Plugin{}
}

func (p *Plugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "it-assets",
		Version:     "1.0.0",
		Description: "IT asset management with hardware component tracking, connection info, and VaultWarden integration",
		Author:      "HomeBoxNG Team",
		BuiltIn:     true,
	}
}

func (p *Plugin) Init(ctx plugins.PluginContext) error {
	p.pctx = ctx
	p.logger = ctx.Logger
	p.logger.Info().Msg("IT asset plugin initialized")
	return nil
}

func (p *Plugin) Start(_ context.Context) error {
	p.logger.Info().Msg("IT asset plugin started")
	return nil
}

func (p *Plugin) Stop(_ context.Context) error {
	p.logger.Info().Msg("IT asset plugin stopped")
	return nil
}

// Routes registers IT asset management API endpoints.
func (p *Plugin) Routes(r chi.Router) {
	// GET /api/plugins/it-assets/templates — List all IT asset templates
	r.Get("/templates", func(w http.ResponseWriter, r *http.Request) {
		_ = server.JSON(w, http.StatusOK, assetTemplates)
	})

	// GET /api/plugins/it-assets/templates/{slug} — Get a specific template
	r.Get("/templates/{slug}", func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")
		for _, t := range assetTemplates {
			if t.Slug == slug {
				_ = server.JSON(w, http.StatusOK, t)
				return
			}
		}
		_ = server.JSON(w, http.StatusNotFound, map[string]string{"error": "template not found"})
	})

	// GET /api/plugins/it-assets/labels — List hardware label taxonomy
	r.Get("/labels", func(w http.ResponseWriter, r *http.Request) {
		_ = server.JSON(w, http.StatusOK, hardwareLabels)
	})

	// GET /api/plugins/it-assets/protocols — List supported connection protocols
	r.Get("/protocols", func(w http.ResponseWriter, r *http.Request) {
		_ = server.JSON(w, http.StatusOK, connectionProtocols)
	})

	// GET /api/plugins/it-assets/component-schemas — List component field schemas
	r.Get("/component-schemas", func(w http.ResponseWriter, r *http.Request) {
		_ = server.JSON(w, http.StatusOK, componentSchemas)
	})

	// GET /api/plugins/it-assets/component-schemas/{type} — Get schema for a component type
	r.Get("/component-schemas/{type}", func(w http.ResponseWriter, r *http.Request) {
		compType := chi.URLParam(r, "type")
		for _, cs := range componentSchemas {
			if cs.ComponentType == compType {
				_ = server.JSON(w, http.StatusOK, cs)
				return
			}
		}
		_ = server.JSON(w, http.StatusNotFound, map[string]string{"error": "component schema not found"})
	})

	// GET /api/plugins/it-assets/search?q=...&label=... — Search IT assets
	r.Get("/search", func(w http.ResponseWriter, r *http.Request) {
		query := strings.ToLower(r.URL.Query().Get("q"))
		label := r.URL.Query().Get("label")

		type searchResult struct {
			Query string `json:"query"`
			Label string `json:"label"`
			Note  string `json:"note"`
		}

		// Placeholder — in production this would query the item repository
		result := searchResult{
			Query: query,
			Label: label,
			Note:  "Search integration pending — will query items by hw-* labels and IT profile data",
		}
		_ = server.JSON(w, http.StatusOK, result)
	})
}

// ConfigSchema returns IT asset plugin configuration options.
func (p *Plugin) ConfigSchema() []plugins.ConfigField {
	return []plugins.ConfigField{
		{
			Key:         "vaultwarden_url",
			Label:       "VaultWarden URL",
			Description: "URL of your VaultWarden instance for credential references",
			Type:        "string",
			Default:     "",
			EnvVar:      "HBOX_VAULTWARDEN_URL",
			Required:    false,
		},
		{
			Key:         "auto_create_labels",
			Label:       "Auto-Create Hardware Labels",
			Description: "Automatically create hw-* and it-* labels on first use",
			Type:        "boolean",
			Default:     "true",
			Required:    false,
		},
		{
			Key:         "smart_monitoring",
			Label:       "SMART Monitoring",
			Description: "Enable periodic disk health checks via SSH (requires SSH access to tracked hosts)",
			Type:        "boolean",
			Default:     "false",
			Required:    false,
		},
	}
}

// Configure applies configuration values.
func (p *Plugin) Configure(values map[string]string) error {
	return nil
}

// RequestedPermissions declares what the IT asset plugin needs.
func (p *Plugin) RequestedPermissions() []plugins.PermissionRequest {
	return []plugins.PermissionRequest{
		{Permission: plugins.PermReadItems, Reason: "Read items to find IT assets and components", Required: true},
		{Permission: plugins.PermWriteItems, Reason: "Create sub-items for hardware components", Required: false},
		{Permission: plugins.PermAPIRoutes, Reason: "IT asset management API endpoints", Required: true},
		{Permission: plugins.PermConfig, Reason: "VaultWarden URL and monitoring settings", Required: false},
	}
}

// Compile-time interface checks.
var (
	_ plugins.Plugin                = (*Plugin)(nil)
	_ plugins.RoutePlugin           = (*Plugin)(nil)
	_ plugins.ConfigPlugin          = (*Plugin)(nil)
	_ plugins.PluginWithPermissions = (*Plugin)(nil)
)
