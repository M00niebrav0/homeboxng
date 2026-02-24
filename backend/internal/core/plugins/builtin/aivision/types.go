// Package aivision provides AI-powered photo identification for HomeBoxNG.
// It uses local LLMs (Ollama/Qwen3-VL) for vision and cloud models (Gemini)
// for text-only verification in a two-step pipeline.
package aivision

import "time"

// ImageAnalysis holds the result of analyzing a single image.
type ImageAnalysis struct {
	ImageIndex     int              `json:"imageIndex"`
	RawItems       []IdentifiedItem `json:"rawItems"`
	Success        bool             `json:"success"`
	Error          string           `json:"error,omitempty"`
	ResponseLength int              `json:"responseLength"`
}

// IdentifiedItem represents a single item identified from a photo.
type IdentifiedItem struct {
	Name              string   `json:"name"`
	Quantity          int      `json:"quantity"`
	SerialNumber      string   `json:"serial_number,omitempty"`
	PartNumber        string   `json:"part_number,omitempty"`
	Description       string   `json:"description"`
	EstimatedCategory string   `json:"estimated_category"`
	LabelText         string   `json:"label_text,omitempty"`
	Confidence        int      `json:"confidence,omitempty"`
	SuggestedLocation string   `json:"suggested_location,omitempty"`
	SuggestedLabels   []string `json:"suggested_labels,omitempty"`
	Notes             string   `json:"notes,omitempty"`
}

// PipelineResult holds the complete result from the two-step vision pipeline.
type PipelineResult struct {
	RawItems          []IdentifiedItem `json:"rawItems"`
	VerifiedItems     []IdentifiedItem `json:"verifiedItems"`
	VisionModel       string           `json:"visionModel"`
	VerificationModel string           `json:"verificationModel,omitempty"`
	ImageCount        int              `json:"imageCount"`
	Error             string           `json:"error,omitempty"`
	ProcessedAt       time.Time        `json:"processedAt"`
}

// PendingImage holds an image waiting to be processed.
type PendingImage struct {
	ImageBytes []byte  `json:"-"`
	MimeType   string  `json:"mimeType"`
	Filename   string  `json:"filename"`
	MessageID  string  `json:"messageId"`
	DiscordTS  float64 `json:"discordTs"`
	ExifTS     float64 `json:"exifTs,omitempty"`
}

// BestTimestamp returns the most accurate timestamp for this image.
func (p *PendingImage) BestTimestamp() float64 {
	if p.ExifTS > 0 {
		return p.ExifTS
	}
	return p.DiscordTS
}

// VisionSession tracks a group of images being accumulated for processing.
type VisionSession struct {
	SessionID   string          `json:"sessionId"`
	UserID      string          `json:"userId"`
	ChannelID   string          `json:"channelId"`
	Images      []*PendingImage `json:"images"`
	UserContext string          `json:"userContext"`
	CreatedAt   time.Time       `json:"createdAt"`
	LastImageAt time.Time       `json:"lastImageAt"`
	State       string          `json:"state"` // "collecting", "processing", "done"
}

// Category constants for identified items.
const (
	CategoryMotherboard = "motherboard"
	CategoryCPU         = "cpu"
	CategoryRAM         = "ram"
	CategoryGPU         = "gpu"
	CategoryStorage     = "storage"
	CategoryNIC         = "nic"
	CategoryPSU         = "psu"
	CategoryCase        = "case"
	CategoryCable       = "cable"
	CategoryCooling     = "cooling"
	CategoryPeripheral  = "peripheral"
	CategoryOther       = "other"
)
