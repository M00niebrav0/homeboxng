package labelprinter

// Orientation specifies label print orientation.
type Orientation string

const (
	OrientationPortrait  Orientation = "portrait"
	OrientationLandscape Orientation = "landscape"
)

// LabelSize specifies full or half-size printing.
type LabelSize string

const (
	LabelSizeFull LabelSize = "full"
	LabelSizeHalf LabelSize = "half"
)

// AccentColor specifies the accent bar color on labels.
type AccentColor string

const (
	ColorBlack  AccentColor = "black"
	ColorBlue   AccentColor = "blue"
	ColorRed    AccentColor = "red"
	ColorGreen  AccentColor = "green"
	ColorYellow AccentColor = "yellow"
	ColorOrange AccentColor = "orange"
	ColorPurple AccentColor = "purple"
	ColorWhite  AccentColor = "white"
)

// LabelContent holds the data to render on a label.
type LabelContent struct {
	Title       string `json:"title"`
	Subtitle    string `json:"subtitle,omitempty"`
	QRData      string `json:"qrData"`
	AccentColor string `json:"accentColor,omitempty"`
	LocationID  string `json:"locationId,omitempty"`
}

// PrintRequest is an API request to print labels.
type PrintRequest struct {
	Labels      []LabelContent `json:"labels"`
	Orientation Orientation    `json:"orientation"`
	Size        LabelSize      `json:"size"`
	Copies      int            `json:"copies"`
}

// PrintResult reports the outcome of a print job.
type PrintResult struct {
	Success    bool   `json:"success"`
	JobID      string `json:"jobId,omitempty"`
	Error      string `json:"error,omitempty"`
	LabelCount int    `json:"labelCount"`
}

// Preset defines automatic label formatting for matching storage types.
type Preset struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	MatchWords  []string    `json:"matchWords"`
	Orientation Orientation `json:"orientation"`
	Size        LabelSize   `json:"size"`
	AccentColor AccentColor `json:"accentColor"`
}

// PrintHistory records a printed label for tracking.
type PrintHistory struct {
	LocationID string `json:"locationId"`
	Title      string `json:"title"`
	PrintedAt  string `json:"printedAt"`
	Preset     string `json:"preset,omitempty"`
}

// defaultPresets returns the built-in label presets.
func defaultPresets() []Preset {
	return []Preset{
		{
			ID:          "alex-drawer",
			Name:        "IKEA Alex Drawer",
			MatchWords:  []string{"alex drawer", "alex"},
			Orientation: OrientationLandscape,
			Size:        LabelSizeHalf,
			AccentColor: ColorWhite,
		},
		{
			ID:          "yellow-bin",
			Name:        "Yellow Top Bin",
			MatchWords:  []string{"yellow bin", "yellow top", "yellow box"},
			Orientation: OrientationLandscape,
			Size:        LabelSizeFull,
			AccentColor: ColorYellow,
		},
		{
			ID:          "server-rack",
			Name:        "Server Rack",
			MatchWords:  []string{"server rack", "rack", "sr-rack"},
			Orientation: OrientationPortrait,
			Size:        LabelSizeFull,
			AccentColor: ColorBlack,
		},
		{
			ID:          "shelf",
			Name:        "Generic Shelf",
			MatchWords:  []string{"shelf", "bookshelf", "shelving"},
			Orientation: OrientationPortrait,
			Size:        LabelSizeFull,
			AccentColor: ColorBlue,
		},
	}
}
