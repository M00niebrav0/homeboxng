package labelprinter

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"strings"
)

// Brother QL label dimensions in pixels at 300 DPI.
const (
	// DK-1202 (62mm x 100mm) at 300 DPI
	LabelWidthPx  = 696
	LabelHeightPx = 1109

	// Half label dimensions
	HalfHeightPx = LabelHeightPx / 2

	// QR code size
	QRSizePx = 200

	// Accent bar height
	AccentBarPx = 20

	// Margins
	MarginPx = 20

	// Font placeholder size (actual font rendering requires freetype or similar)
	CharWidthPx  = 10
	CharHeightPx = 18
)

// LabelRenderer generates label images for the Brother QL printer.
type LabelRenderer struct {
	presets []Preset
}

// NewLabelRenderer creates a renderer with default presets.
func NewLabelRenderer() *LabelRenderer {
	return &LabelRenderer{
		presets: defaultPresets(),
	}
}

// RenderLabel generates a single full-size label image.
func (r *LabelRenderer) RenderLabel(content LabelContent, orientation Orientation) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, LabelWidthPx, LabelHeightPx))

	// White background
	draw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)

	switch orientation {
	case OrientationLandscape:
		r.renderLandscape(img, content)
	default:
		r.renderPortrait(img, content)
	}

	return img
}

// RenderHalfLabels generates a full-size image with two half-labels.
func (r *LabelRenderer) RenderHalfLabels(content1, content2 LabelContent, orientation Orientation) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, LabelWidthPx, LabelHeightPx))

	// White background
	draw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)

	// Draw divider line at midpoint
	midY := HalfHeightPx
	dividerColor := color.RGBA{R: 180, G: 180, B: 180, A: 255}
	for x := MarginPx; x < LabelWidthPx-MarginPx; x++ {
		img.Set(x, midY, dividerColor)
		img.Set(x, midY+1, dividerColor)
	}

	// Top half
	r.renderHalf(img, content1, orientation, 0)

	// Bottom half
	r.renderHalf(img, content2, orientation, midY+2)

	return img
}

// renderPortrait draws text at top, QR in center, accent bar at bottom.
func (r *LabelRenderer) renderPortrait(img *image.RGBA, content LabelContent) {
	accentColor := parseColor(content.AccentColor)

	// Title text at top
	r.drawText(img, content.Title, MarginPx+10, MarginPx+30, 2)
	if content.Subtitle != "" {
		r.drawText(img, content.Subtitle, MarginPx+10, MarginPx+70, 1)
	}

	// QR code in center
	qrY := (LabelHeightPx - QRSizePx) / 2
	qrX := (LabelWidthPx - QRSizePx) / 2
	r.drawQRPlaceholder(img, content.QRData, qrX, qrY)

	// Accent bar at bottom
	r.drawAccentBar(img, accentColor, LabelHeightPx-AccentBarPx-MarginPx, LabelWidthPx)
}

// renderLandscape draws text on left (60%), QR on right (40%).
func (r *LabelRenderer) renderLandscape(img *image.RGBA, content LabelContent) {
	accentColor := parseColor(content.AccentColor)

	// Text region: left 60%
	textWidth := LabelWidthPx * 6 / 10

	// Title
	r.drawText(img, content.Title, MarginPx+10, MarginPx+40, 2)
	if content.Subtitle != "" {
		r.drawText(img, content.Subtitle, MarginPx+10, MarginPx+80, 1)
	}

	// QR region: right 40%
	qrX := textWidth + (LabelWidthPx-textWidth-QRSizePx)/2
	qrY := (LabelHeightPx - QRSizePx) / 2
	r.drawQRPlaceholder(img, content.QRData, qrX, qrY)

	// Accent bar at bottom
	r.drawAccentBar(img, accentColor, LabelHeightPx-AccentBarPx-MarginPx, LabelWidthPx)
}

// renderHalf draws a half-label in the given vertical region.
func (r *LabelRenderer) renderHalf(img *image.RGBA, content LabelContent, orientation Orientation, yOffset int) {
	accentColor := parseColor(content.AccentColor)
	halfH := HalfHeightPx

	switch orientation {
	case OrientationLandscape:
		textWidth := LabelWidthPx * 6 / 10
		r.drawText(img, content.Title, MarginPx+10, yOffset+MarginPx+25, 1)
		if content.Subtitle != "" {
			r.drawText(img, content.Subtitle, MarginPx+10, yOffset+MarginPx+50, 1)
		}

		qrSize := halfH - MarginPx*3
		if qrSize > QRSizePx {
			qrSize = QRSizePx
		}
		qrX := textWidth + (LabelWidthPx-textWidth-qrSize)/2
		qrY := yOffset + (halfH-qrSize)/2
		r.drawQRPlaceholder(img, content.QRData, qrX, qrY)

	default:
		r.drawText(img, content.Title, MarginPx+10, yOffset+MarginPx+20, 1)
		if content.Subtitle != "" {
			r.drawText(img, content.Subtitle, MarginPx+10, yOffset+MarginPx+45, 1)
		}

		qrSize := halfH - MarginPx*4 - 60
		if qrSize > QRSizePx {
			qrSize = QRSizePx
		}
		qrX := (LabelWidthPx - qrSize) / 2
		qrY := yOffset + 60 + MarginPx
		r.drawQRPlaceholder(img, content.QRData, qrX, qrY)
	}

	// Accent bar at bottom of half
	r.drawAccentBar(img, accentColor, yOffset+halfH-AccentBarPx-5, LabelWidthPx)
}

// drawText renders text on the image using simple bitmap rendering.
// scale=1 for normal, scale=2 for large.
func (r *LabelRenderer) drawText(img *image.RGBA, text string, x, y, scale int) {
	textColor := color.Black
	charW := CharWidthPx * scale
	charH := CharHeightPx * scale

	for i, ch := range text {
		cx := x + i*charW
		if cx+charW >= LabelWidthPx-MarginPx {
			break
		}
		// Draw a simple block character (placeholder for real font rendering)
		for dy := 0; dy < charH; dy++ {
			for dx := 0; dx < charW-2; dx++ {
				if ch != ' ' {
					img.Set(cx+dx, y+dy, textColor)
				}
			}
		}
	}
}

// drawQRPlaceholder draws a QR code placeholder (border + data text).
// Real QR generation requires a library like github.com/skip2/go-qrcode.
func (r *LabelRenderer) drawQRPlaceholder(img *image.RGBA, data string, x, y int) {
	borderColor := color.Black

	// Draw border
	for i := 0; i < QRSizePx; i++ {
		img.Set(x+i, y, borderColor)
		img.Set(x+i, y+QRSizePx-1, borderColor)
		img.Set(x, y+i, borderColor)
		img.Set(x+QRSizePx-1, y+i, borderColor)
	}

	// Draw inner pattern (simplified QR appearance)
	for i := 5; i < 25; i++ {
		for j := 5; j < 25; j++ {
			img.Set(x+i, y+j, borderColor)
		}
	}

	// Center text "QR" as indicator
	r.drawText(img, "QR", x+QRSizePx/2-10, y+QRSizePx/2-9, 1)
}

// drawAccentBar draws a colored horizontal bar.
func (r *LabelRenderer) drawAccentBar(img *image.RGBA, c color.RGBA, y, width int) {
	for dy := 0; dy < AccentBarPx; dy++ {
		for dx := MarginPx; dx < width-MarginPx; dx++ {
			img.Set(dx, y+dy, c)
		}
	}
}

// MatchPreset finds a preset matching the given location name (case-insensitive).
func (r *LabelRenderer) MatchPreset(locationName string) *Preset {
	lower := strings.ToLower(locationName)
	for _, preset := range r.presets {
		for _, word := range preset.MatchWords {
			if strings.Contains(lower, strings.ToLower(word)) {
				return &preset
			}
		}
	}
	return nil
}

// EncodePNG encodes an RGBA image as PNG bytes.
func EncodePNG(img *image.RGBA) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("encoding PNG: %w", err)
	}
	return buf.Bytes(), nil
}

// EncodeBase64PNG encodes an RGBA image as a base64 PNG data URI.
func EncodeBase64PNG(img *image.RGBA) (string, error) {
	data, err := EncodePNG(img)
	if err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(data), nil
}

// parseColor converts a named color string to color.RGBA.
func parseColor(name string) color.RGBA {
	switch AccentColor(strings.ToLower(name)) {
	case ColorBlue:
		return color.RGBA{R: 33, G: 150, B: 243, A: 255}
	case ColorRed:
		return color.RGBA{R: 244, G: 67, B: 54, A: 255}
	case ColorGreen:
		return color.RGBA{R: 76, G: 175, B: 80, A: 255}
	case ColorYellow:
		return color.RGBA{R: 255, G: 193, B: 7, A: 255}
	case ColorOrange:
		return color.RGBA{R: 255, G: 152, B: 0, A: 255}
	case ColorPurple:
		return color.RGBA{R: 156, G: 39, B: 176, A: 255}
	case ColorWhite:
		return color.RGBA{R: 255, G: 255, B: 255, A: 255}
	default:
		return color.RGBA{R: 33, G: 33, B: 33, A: 255} // black
	}
}
