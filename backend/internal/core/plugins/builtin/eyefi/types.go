package eyefi

import "time"

// CardInfo holds the detected Eye-Fi card configuration.
type CardInfo struct {
	MAC       string `json:"mac"`
	UploadKey string `json:"uploadKey"`
	SSID      string `json:"ssid,omitempty"`
}

// UploadEvent records a single photo upload from the card.
type UploadEvent struct {
	Filename  string    `json:"filename"`
	Size      int64     `json:"size"`
	SourceMAC string    `json:"sourceMac"`
	Timestamp time.Time `json:"timestamp"`
}

// ServerStatus reports the Eye-Fi SOAP server state.
type ServerStatus struct {
	Running      bool          `json:"running"`
	Port         int           `json:"port"`
	CardMAC      string        `json:"cardMac,omitempty"`
	LastUpload   *time.Time    `json:"lastUpload,omitempty"`
	TotalUploads int           `json:"totalUploads"`
	UploadsToday int           `json:"uploadsToday"`
	Uptime       time.Duration `json:"uptime"`
}

// ImportMode controls how incoming photos are processed.
type ImportMode string

const (
	ImportModeAuto   ImportMode = "auto"   // Immediately process through vision pipeline
	ImportModeManual ImportMode = "manual" // Accumulate, process on demand
	ImportModeOff    ImportMode = "off"    // Store only, no processing
)

// IngestSource represents a configured photo ingestion source.
type IngestSource struct {
	Name           string     `json:"name"`
	Type           string     `json:"type"` // local, eyefi, ftp, smb
	Path           string     `json:"path"`
	Enabled        bool       `json:"enabled"`
	LastTestedAt   *time.Time `json:"lastTestedAt,omitempty"`
	LastTestResult string     `json:"lastTestResult,omitempty"`
}

// WatcherStats tracks file watcher statistics.
type WatcherStats struct {
	SourceName     string    `json:"sourceName"`
	PendingFiles   int       `json:"pendingFiles"`
	ProcessedFiles int       `json:"processedFiles"`
	LastScanAt     time.Time `json:"lastScanAt"`
	Mode           ImportMode `json:"mode"`
}
