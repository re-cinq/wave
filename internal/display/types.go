// Package display provides types and utilities for rendering progress visualization
// in the Wave CLI. It includes terminal capability detection, ANSI color and Unicode
// support detection, and shared types for progress rendering.
package display

// ProgressState represents the current state of a pipeline or step.
type ProgressState string

const (
	StateNotStarted ProgressState = "not_started"
	StateRunning    ProgressState = "running"
	StateCompleted  ProgressState = "completed"
	StateFailed     ProgressState = "failed"
	StateSkipped    ProgressState = "skipped"
	StateCancelled  ProgressState = "cancelled"
)

// AnimationType defines the animation style for progress indicators.
type AnimationType string

const (
	AnimationDots        AnimationType = "dots"
	AnimationLine        AnimationType = "line"
	AnimationBars        AnimationType = "bars"
	AnimationSpinner     AnimationType = "spinner"
	AnimationClock       AnimationType = "clock"
	AnimationBouncingBar AnimationType = "bouncing_bar"
)

// ColorPalette defines ANSI color codes for different UI elements.
type ColorPalette struct {
	Primary    string // Main color for active elements
	Success    string // Color for successful states
	Warning    string // Color for warnings
	Error      string // Color for errors
	Muted      string // Color for inactive/secondary text
	Background string // Background color if supported
	Reset      string // Reset code
}

// TerminalCapabilities describes what the terminal can display.
type TerminalCapabilities struct {
	IsTTY             bool   // Whether output is to a terminal
	Width             int    // Terminal width in columns
	Height            int    // Terminal height in rows
	SupportsANSI      bool   // ANSI escape sequence support
	SupportsColor     bool   // Full 24-bit RGB color support
	Supports256Colors bool   // 256-color palette support
	SupportsUnicode   bool   // Unicode character support
	SupportsAlternate bool   // Alternate screen buffer support
	HasMouseSupport   bool   // Mouse input support
	ColorScheme       string // "light" or "dark"
}

// StepProgress tracks progress information for a single pipeline step.
type StepProgress struct {
	StepID        string
	Name          string
	State         ProgressState
	Persona       string
	Message       string
	Progress      int // 0-100 percentage
	CurrentAction string
	Artifacts     []string
	StartTime     int64  // Unix nanoseconds
	EndTime       *int64 // Unix nanoseconds (null if not finished)
	Error         string
	TokensUsed    int
	DurationMs    int64
}

// PipelineProgress tracks overall pipeline execution progress.
type PipelineProgress struct {
	PipelineID     string
	PipelineName   string
	State          ProgressState
	TotalSteps     int
	CompletedSteps int
	CurrentStep    int
	Progress       int // 0-100 percentage
	Steps          map[string]*StepProgress
	StartTime      int64  // Unix nanoseconds
	EndTime        *int64 // Unix nanoseconds (null if not finished)
	Message        string
	Error          string
}

// DisplayConfig holds configuration for progress display.
type DisplayConfig struct {
	Enabled          bool
	AnimationType    AnimationType
	RefreshRate      int    // Updates per second (default: 10, range: 1-60)
	ShowDetails      bool   // Show detailed step information
	ShowArtifacts    bool   // Display artifact information
	CompactMode      bool   // Use compact display mode
	ColorMode        string // "auto", "on", "off" - control color usage
	ColorTheme       string // "default", "dark", "light", "high_contrast"
	AsciiOnly        bool   // Use ASCII-only characters (no Unicode)
	MaxHistoryLines  int    // Maximum lines of history to keep (default: 100)
	EnableTimestamps bool   // Show timestamps in output
	VerboseOutput    bool   // Enable verbose output
	AnimationEnabled bool   // Enable/disable animations
	ShowLogo         bool   // Display Wave logo in dashboard
	ShowMetrics      bool   // Display token/file counts and metrics
}

// ProgressRenderer defines the interface for rendering progress information.
type ProgressRenderer interface {
	// Render updates the display with current progress state
	Render(progress *PipelineProgress) error

	// Clear removes the current display
	Clear() error

	// Close cleans up resources
	Close() error
}

// ColorScheme provides color mappings for different terminal types.
var DefaultColorScheme = ColorPalette{
	Primary:    "\033[36m", // Standard cyan
	Success:    "\033[32m", // Standard green
	Warning:    "\033[33m", // Standard yellow
	Error:      "\033[31m", // Standard red
	Muted:      "\033[37m", // Standard white/light gray (readable)
	Background: "\033[40m", // Black background
	Reset:      "\033[0m",  // Reset
}

// AsciiOnlyColorScheme for terminals that don't support colors.
var AsciiOnlyColorScheme = ColorPalette{
	Primary:    "",
	Success:    "",
	Warning:    "",
	Error:      "",
	Muted:      "",
	Background: "",
	Reset:      "",
}

// DarkColorScheme optimized for dark terminal backgrounds.
var DarkColorScheme = ColorPalette{
	Primary:    "\033[36m", // Cyan
	Success:    "\033[32m", // Green
	Warning:    "\033[33m", // Yellow
	Error:      "\033[31m", // Red
	Muted:      "\033[90m", // Bright black/gray
	Background: "\033[40m", // Black background
	Reset:      "\033[0m",  // Reset
}

// LightColorScheme optimized for light terminal backgrounds.
var LightColorScheme = ColorPalette{
	Primary:    "\033[34m", // Blue (more readable on light)
	Success:    "\033[32m", // Green
	Warning:    "\033[33m", // Yellow
	Error:      "\033[31m", // Red
	Muted:      "\033[37m", // White/light gray
	Background: "\033[47m", // White background
	Reset:      "\033[0m",  // Reset
}

// HighContrastColorScheme for accessibility.
var HighContrastColorScheme = ColorPalette{
	Primary:    "\033[1;36m", // Bold cyan
	Success:    "\033[1;32m", // Bold green
	Warning:    "\033[1;33m", // Bold yellow
	Error:      "\033[1;31m", // Bold red
	Muted:      "\033[1;37m", // Bold white
	Background: "\033[40m",   // Black background
	Reset:      "\033[0m",    // Reset
}

// GetColorSchemeByName returns the appropriate color scheme based on theme name.
func GetColorSchemeByName(theme string) ColorPalette {
	switch theme {
	case "dark":
		return DarkColorScheme
	case "light":
		return LightColorScheme
	case "high_contrast":
		return HighContrastColorScheme
	case "default":
		return DefaultColorScheme
	default:
		return DefaultColorScheme
	}
}

// PipelineContext provides comprehensive context for overall pipeline progress tracking.
// It includes project metadata, step tracking, ETA calculations, and workspace information.
type PipelineContext struct {
	// Project metadata
	ManifestPath  string
	PipelineName  string
	PipelineID    string
	WorkspacePath string

	// Step tracking
	TotalSteps     int
	CurrentStepNum int // 1-based index of currently executing step
	CompletedSteps int
	FailedSteps    int
	SkippedSteps   int

	// Progress calculation
	OverallProgress int   // 0-100 percentage of overall pipeline completion
	EstimatedTimeMs int64 // ETA in milliseconds for remaining work

	// Current execution state
	CurrentStepID   string
	CurrentPersona  string
	CurrentAction   string
	CurrentStepName string

	// Timing information
	PipelineStartTime int64  // Unix nanoseconds
	CurrentStepStart  int64  // Unix nanoseconds
	AverageStepTimeMs int64  // Average time per completed step
	ElapsedTimeMs     int64  // Total elapsed time since pipeline start

	// Step status mapping
	StepStatuses map[string]ProgressState // stepID -> state
	StepOrder    []string                  // Ordered list of step IDs

	// Step durations in milliseconds
	StepDurations map[string]int64 // stepID -> duration in ms

	// Deliverables by step
	DeliverablesByStep map[string][]string // stepID -> deliverable strings

	// Tool activity (verbose mode)
	LastToolName   string // Most recent tool being used (Read, Write, Bash, etc.)
	LastToolTarget string // Most recent tool target (file path, command, pattern)

	// Additional context
	Message string
	Error   string
}

// DefaultDisplayConfig returns a display configuration with sensible defaults.
func DefaultDisplayConfig() DisplayConfig {
	return DisplayConfig{
		Enabled:          true,
		AnimationType:    AnimationSpinner,
		RefreshRate:      10, // 10 updates per second
		ShowDetails:      true,
		ShowArtifacts:    true,
		CompactMode:      false,
		ColorMode:        "auto",
		ColorTheme:       "default",
		AsciiOnly:        false,
		MaxHistoryLines:  100,
		EnableTimestamps: true,
		VerboseOutput:    false,
		AnimationEnabled: true,
		ShowLogo:         true,
		ShowMetrics:      true,
	}
}

// Validate checks the configuration for invalid values and corrects them.
func (dc *DisplayConfig) Validate() {
	// Refresh rate must be between 1 and 60
	if dc.RefreshRate < 1 {
		dc.RefreshRate = 1
	} else if dc.RefreshRate > 60 {
		dc.RefreshRate = 60
	}

	// Max history lines must be positive
	if dc.MaxHistoryLines < 1 {
		dc.MaxHistoryLines = 100
	}

	// Validate color mode
	if dc.ColorMode != "auto" && dc.ColorMode != "on" && dc.ColorMode != "off" {
		dc.ColorMode = "auto"
	}

	// Validate color theme
	validThemes := map[string]bool{
		"default":       true,
		"dark":          true,
		"light":         true,
		"high_contrast": true,
	}
	if !validThemes[dc.ColorTheme] {
		dc.ColorTheme = "default"
	}

	// Validate animation type
	validAnimations := map[AnimationType]bool{
		AnimationDots:        true,
		AnimationLine:        true,
		AnimationBars:        true,
		AnimationSpinner:     true,
		AnimationClock:       true,
		AnimationBouncingBar: true,
	}
	if !validAnimations[dc.AnimationType] {
		dc.AnimationType = AnimationSpinner
	}

	// If animations are disabled, use dots (simplest)
	if !dc.AnimationEnabled {
		dc.AnimationType = AnimationDots
	}
}
