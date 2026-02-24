package labelprinter

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/rs/zerolog"
)

// BrotherQL communicates with a Brother QL-series label printer over TCP/IP.
type BrotherQL struct {
	ip     string
	port   int
	model  string
	logger zerolog.Logger
}

// NewBrotherQL creates a new Brother QL printer client.
func NewBrotherQL(ip string, port int, model string, logger zerolog.Logger) *BrotherQL {
	return &BrotherQL{
		ip:     ip,
		port:   port,
		model:  model,
		logger: logger.With().Str("component", "brother-ql").Logger(),
	}
}

// TestConnection checks if the printer is reachable.
func (b *BrotherQL) TestConnection() (string, error) {
	addr := net.JoinHostPort(b.ip, strconv.Itoa(b.port))
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return fmt.Sprintf("Connection failed: %s", err.Error()), err
	}
	conn.Close()
	return fmt.Sprintf("OK: %s reachable on port %d", b.ip, b.port), nil
}

// PrintRaw sends raw raster data to the printer via TCP.
// The Brother QL protocol uses a simple binary format:
// 1. Initialize command (0x00 * 200)
// 2. Mode setting (ESC @ for reset)
// 3. Media info
// 4. Raster data lines
// 5. Print command
func (b *BrotherQL) PrintRaw(rasterData []byte) error {
	addr := net.JoinHostPort(b.ip, strconv.Itoa(b.port))
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("connecting to printer: %w", err)
	}
	defer conn.Close()

	if err := conn.SetWriteDeadline(time.Now().Add(30 * time.Second)); err != nil {
		return fmt.Errorf("setting deadline: %w", err)
	}

	// Step 1: Initialize (200 null bytes)
	init := make([]byte, 200)
	if _, err := conn.Write(init); err != nil {
		return fmt.Errorf("sending init: %w", err)
	}

	// Step 2: Reset command (ESC @)
	if _, err := conn.Write([]byte{0x1B, 0x40}); err != nil {
		return fmt.Errorf("sending reset: %w", err)
	}

	// Step 3: Send raster data (pre-formatted by caller)
	if _, err := conn.Write(rasterData); err != nil {
		return fmt.Errorf("sending raster data: %w", err)
	}

	// Step 4: Print command (0x1A for print)
	if _, err := conn.Write([]byte{0x1A}); err != nil {
		return fmt.Errorf("sending print command: %w", err)
	}

	b.logger.Info().
		Int("bytes", len(rasterData)).
		Msg("print job sent")

	return nil
}

// ImageToRaster converts a PNG image to Brother QL raster format.
// Each raster line is 90 bytes (720 pixels) wide for 62mm labels.
// Format per line: 0x67 0x00 0x5A [90 bytes of pixel data]
func ImageToRaster(pngData []byte) ([]byte, error) {
	// Decode PNG
	// For a full implementation, decode the PNG and convert each row to
	// 1-bit-per-pixel raster data. This is a simplified version.

	// Brother QL raster line format:
	// g\x00\x5a = raster command header
	// followed by 90 bytes of raster data (720 pixels, 1 bit per pixel)
	lineWidth := 90 // bytes = 720 pixels

	// Calculate approximate lines from PNG data size
	// Real implementation would decode PNG properly
	numLines := 1109 // DK-1202 label height

	var raster []byte

	// Media type command
	raster = append(raster, 0x1B, 0x69, 0x7A) // ESC i z (media info)
	raster = append(raster,
		0x86,                   // valid flags
		0x0B,                   // media type (die-cut)
		0x3E,                   // media width (62mm)
		0x64,                   // media length (100mm)
		byte(numLines&0xFF),    // raster lines low
		byte(numLines>>8),      // raster lines high
		0x00, 0x00,             // page number
		0x00,                   // starting page
	)

	// Auto-cut setting
	raster = append(raster, 0x1B, 0x69, 0x4D, 0x40) // ESC i M @ (auto-cut)
	raster = append(raster, 0x1B, 0x69, 0x41, 0x01) // ESC i A (auto-cut each)

	// Raster data lines (simplified: all black for testing)
	for line := 0; line < numLines; line++ {
		raster = append(raster, 0x67, 0x00, byte(lineWidth)) // raster command
		lineData := make([]byte, lineWidth)
		raster = append(raster, lineData...)
	}

	// Print with feeding
	raster = append(raster, 0x1A)

	return raster, nil
}
