package labelprinter

import (
	"image/color"
	"strings"
	"testing"
)

func TestRenderLabel_Portrait_Dimensions(t *testing.T) {
	r := NewLabelRenderer()
	content := LabelContent{
		Title:       "Test Location",
		Subtitle:    "Shelf A",
		QRData:      "https://example.com/loc/1",
		AccentColor: "blue",
	}

	img := r.RenderLabel(content, OrientationPortrait)

	bounds := img.Bounds()
	if bounds.Dx() != LabelWidthPx || bounds.Dy() != LabelHeightPx {
		t.Errorf("label size = %dx%d, want %dx%d", bounds.Dx(), bounds.Dy(), LabelWidthPx, LabelHeightPx)
	}
}

func TestRenderLabel_Landscape_Dimensions(t *testing.T) {
	r := NewLabelRenderer()
	content := LabelContent{
		Title:       "Test Location",
		QRData:      "https://example.com/loc/2",
		AccentColor: "black",
	}

	img := r.RenderLabel(content, OrientationLandscape)

	bounds := img.Bounds()
	if bounds.Dx() != LabelWidthPx || bounds.Dy() != LabelHeightPx {
		t.Errorf("label size = %dx%d, want %dx%d", bounds.Dx(), bounds.Dy(), LabelWidthPx, LabelHeightPx)
	}
}

func TestRenderLabel_WhiteBackground(t *testing.T) {
	r := NewLabelRenderer()
	img := r.RenderLabel(LabelContent{Title: "Test"}, OrientationPortrait)

	// Check corner pixel is white
	c := img.At(0, 0)
	rr, g, b, _ := c.RGBA()
	if rr != 0xFFFF || g != 0xFFFF || b != 0xFFFF {
		t.Error("expected white background at corner")
	}
}

func TestRenderHalfLabels_Dimensions(t *testing.T) {
	r := NewLabelRenderer()
	c1 := LabelContent{Title: "Top", AccentColor: "red"}
	c2 := LabelContent{Title: "Bottom", AccentColor: "blue"}

	img := r.RenderHalfLabels(c1, c2, OrientationPortrait)

	bounds := img.Bounds()
	if bounds.Dx() != LabelWidthPx || bounds.Dy() != LabelHeightPx {
		t.Errorf("half label size = %dx%d, want %dx%d", bounds.Dx(), bounds.Dy(), LabelWidthPx, LabelHeightPx)
	}
}

func TestRenderHalfLabels_HasDivider(t *testing.T) {
	r := NewLabelRenderer()
	c1 := LabelContent{Title: "Top"}
	c2 := LabelContent{Title: "Bottom"}

	img := r.RenderHalfLabels(c1, c2, OrientationPortrait)

	// Check midpoint has non-white pixel (divider line)
	midY := HalfHeightPx
	midX := LabelWidthPx / 2
	c := img.At(midX, midY)
	rr, g, b, _ := c.RGBA()
	if rr == 0xFFFF && g == 0xFFFF && b == 0xFFFF {
		t.Error("expected divider at midpoint to be non-white")
	}
}

func TestRenderLabel_HasAccentBar(t *testing.T) {
	r := NewLabelRenderer()
	content := LabelContent{
		Title:       "Test",
		AccentColor: "red",
	}

	img := r.RenderLabel(content, OrientationPortrait)

	// Accent bar should be near the bottom
	barY := LabelHeightPx - AccentBarPx - MarginPx + AccentBarPx/2
	barX := LabelWidthPx / 2

	c := img.At(barX, barY)
	rr, g, b, _ := c.RGBA()

	// Red accent: R should be high, G and B should be low
	if rr < 0x8000 {
		t.Errorf("expected red accent bar, got RGBA(%d, %d, %d, _)", rr>>8, g>>8, b>>8)
	}
}

func TestMatchPreset_AlexDrawer(t *testing.T) {
	r := NewLabelRenderer()

	tests := []struct {
		name     string
		location string
		want     bool
	}{
		{"exact", "alex drawer", true},
		{"case insensitive", "ALEX DRAWER 3", true},
		{"partial", "Alex Drawer Unit #1", true},
		{"no match", "Kitchen Cabinet #7", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			preset := r.MatchPreset(tt.location)
			got := preset != nil
			if got != tt.want {
				t.Errorf("MatchPreset(%q) matched = %v, want %v", tt.location, got, tt.want)
			}
		})
	}
}

func TestMatchPreset_ServerRack(t *testing.T) {
	r := NewLabelRenderer()

	preset := r.MatchPreset("Server Rack U12")
	if preset == nil {
		t.Fatal("expected to match server rack preset")
	}
	if preset.ID != "server-rack" {
		t.Errorf("preset ID = %q, want %q", preset.ID, "server-rack")
	}
	if preset.Orientation != OrientationPortrait {
		t.Errorf("orientation = %q, want %q", preset.Orientation, OrientationPortrait)
	}
}

func TestMatchPreset_YellowBin(t *testing.T) {
	r := NewLabelRenderer()

	for _, name := range []string{"Yellow Bin #3", "yellow top bin", "Yellow Box #1"} {
		preset := r.MatchPreset(name)
		if preset == nil {
			t.Errorf("expected match for %q", name)
			continue
		}
		if preset.ID != "yellow-bin" {
			t.Errorf("for %q: preset ID = %q, want %q", name, preset.ID, "yellow-bin")
		}
	}
}

func TestMatchPreset_NoMatch(t *testing.T) {
	r := NewLabelRenderer()
	preset := r.MatchPreset("Kitchen Cabinet #7")
	if preset != nil {
		t.Errorf("expected no match for Kitchen Cabinet, got %q", preset.ID)
	}
}

func TestParseColor(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantR    uint8
		wantG    uint8
		wantB    uint8
	}{
		{"blue", "blue", 33, 150, 243},
		{"red", "red", 244, 67, 54},
		{"green", "green", 76, 175, 80},
		{"yellow", "yellow", 255, 193, 7},
		{"orange", "orange", 255, 152, 0},
		{"purple", "purple", 156, 39, 176},
		{"white", "white", 255, 255, 255},
		{"black", "black", 33, 33, 33},
		{"unknown", "magenta", 33, 33, 33}, // defaults to black
		{"mixed case", "BLUE", 33, 150, 243},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := parseColor(tt.input)
			if c.R != tt.wantR || c.G != tt.wantG || c.B != tt.wantB {
				t.Errorf("parseColor(%q) = RGB(%d,%d,%d), want RGB(%d,%d,%d)",
					tt.input, c.R, c.G, c.B, tt.wantR, tt.wantG, tt.wantB)
			}
			if c.A != 255 {
				t.Errorf("alpha should always be 255, got %d", c.A)
			}
		})
	}
}

func TestEncodePNG(t *testing.T) {
	r := NewLabelRenderer()
	img := r.RenderLabel(LabelContent{Title: "Test"}, OrientationPortrait)

	data, err := EncodePNG(img)
	if err != nil {
		t.Fatalf("EncodePNG() error = %v", err)
	}

	// Check PNG magic bytes
	if len(data) < 8 {
		t.Fatal("PNG data too short")
	}
	if data[0] != 0x89 || data[1] != 0x50 || data[2] != 0x4E || data[3] != 0x47 {
		t.Error("invalid PNG magic bytes")
	}
}

func TestEncodeBase64PNG(t *testing.T) {
	r := NewLabelRenderer()
	img := r.RenderLabel(LabelContent{Title: "Test"}, OrientationPortrait)

	b64, err := EncodeBase64PNG(img)
	if err != nil {
		t.Fatalf("EncodeBase64PNG() error = %v", err)
	}

	if !strings.HasPrefix(b64, "data:image/png;base64,") {
		t.Error("expected data URI prefix")
	}
}

func TestDefaultPresets(t *testing.T) {
	presets := defaultPresets()
	if len(presets) < 4 {
		t.Fatalf("expected at least 4 default presets, got %d", len(presets))
	}

	ids := map[string]bool{}
	for _, p := range presets {
		ids[p.ID] = true
		if len(p.MatchWords) == 0 {
			t.Errorf("preset %q has no match words", p.ID)
		}
	}

	for _, expected := range []string{"alex-drawer", "yellow-bin", "server-rack", "shelf"} {
		if !ids[expected] {
			t.Errorf("missing expected preset %q", expected)
		}
	}
}

func TestRenderLabel_Portrait_HasText(t *testing.T) {
	r := NewLabelRenderer()
	img := r.RenderLabel(LabelContent{
		Title: "Server Rack",
	}, OrientationPortrait)

	// Text should have black pixels near the top
	hasBlack := false
	for x := MarginPx; x < LabelWidthPx/2; x++ {
		for y := MarginPx; y < MarginPx+50; y++ {
			c := img.At(x, y)
			rr, g, b, _ := c.RGBA()
			if rr == 0 && g == 0 && b == 0 {
				hasBlack = true
				break
			}
		}
		if hasBlack {
			break
		}
	}

	if !hasBlack {
		t.Error("expected text (black pixels) in the title area")
	}
}

func TestAccentColor_Constants(t *testing.T) {
	colors := []AccentColor{
		ColorBlack, ColorBlue, ColorRed, ColorGreen,
		ColorYellow, ColorOrange, ColorPurple, ColorWhite,
	}
	if len(colors) != 8 {
		t.Errorf("expected 8 accent colors, got %d", len(colors))
	}

	// Verify all parse without panic
	for _, c := range colors {
		parsed := parseColor(string(c))
		if parsed == (color.RGBA{}) {
			t.Errorf("parseColor(%q) returned zero value", c)
		}
	}
}
