package display

import (
	"os"
	"sync"
	"testing"
	"time"
)

func TestNewTerminalInfo(t *testing.T) {
	ti := NewTerminalInfo()
	if ti == nil {
		t.Fatal("NewTerminalInfo returned nil")
	}
	if ti.capabilities == nil {
		t.Error("capabilities should not be nil")
	}
}

func TestTerminalInfo_IsTTY(t *testing.T) {
	ti := NewTerminalInfo()
	// Just verify it doesn't panic and returns a boolean
	_ = ti.IsTTY()
}

func TestTerminalInfo_Dimensions(t *testing.T) {
	ti := NewTerminalInfo()

	width := ti.GetWidth()
	if width <= 0 {
		t.Errorf("Width should be positive, got %d", width)
	}

	height := ti.GetHeight()
	if height <= 0 {
		t.Errorf("Height should be positive, got %d", height)
	}
}

func TestTerminalInfo_SupportsANSI(t *testing.T) {
	ti := NewTerminalInfo()
	// Just verify it doesn't panic
	_ = ti.SupportsANSI()
}

func TestTerminalInfo_SupportsColor(t *testing.T) {
	ti := NewTerminalInfo()
	_ = ti.SupportsColor()
}

func TestTerminalInfo_Supports256Colors(t *testing.T) {
	ti := NewTerminalInfo()
	_ = ti.Supports256Colors()
}

func TestTerminalInfo_SupportsUnicode(t *testing.T) {
	ti := NewTerminalInfo()
	_ = ti.SupportsUnicode()
}

func TestTerminalInfo_SupportsAlternateBuffer(t *testing.T) {
	ti := NewTerminalInfo()
	_ = ti.SupportsAlternateBuffer()
}

func TestTerminalInfo_HasMouseSupport(t *testing.T) {
	ti := NewTerminalInfo()
	_ = ti.HasMouseSupport()
}

func TestTerminalInfo_GetColorScheme(t *testing.T) {
	ti := NewTerminalInfo()
	scheme := ti.GetColorScheme()
	if scheme != "dark" && scheme != "light" {
		t.Errorf("ColorScheme should be 'dark' or 'light', got %q", scheme)
	}
}

func TestTerminalInfo_Capabilities(t *testing.T) {
	ti := NewTerminalInfo()
	caps := ti.Capabilities()
	if caps == nil {
		t.Error("Capabilities() should not return nil")
	}
}

func TestDetectCapabilities(t *testing.T) {
	caps := detectCapabilities()
	if caps == nil {
		t.Fatal("detectCapabilities returned nil")
	}

	// Width and height should have reasonable defaults
	if caps.Width <= 0 {
		t.Errorf("Width should be positive, got %d", caps.Width)
	}
	if caps.Height <= 0 {
		t.Errorf("Height should be positive, got %d", caps.Height)
	}

	// ColorScheme should be set
	if caps.ColorScheme != "dark" && caps.ColorScheme != "light" {
		t.Errorf("ColorScheme should be 'dark' or 'light', got %q", caps.ColorScheme)
	}
}

func TestGetTerminalWidth_WithEnvVar(t *testing.T) {
	// Save and restore COLUMNS
	oldColumns := os.Getenv("COLUMNS")
	defer os.Setenv("COLUMNS", oldColumns)

	os.Setenv("COLUMNS", "120")
	width := getTerminalWidth()
	// Should return at least the default or detected value
	if width <= 0 {
		t.Errorf("Width should be positive, got %d", width)
	}
}

func TestGetTerminalWidth_InvalidEnvVar(t *testing.T) {
	// Save and restore COLUMNS
	oldColumns := os.Getenv("COLUMNS")
	defer os.Setenv("COLUMNS", oldColumns)

	os.Setenv("COLUMNS", "invalid")
	width := getTerminalWidth()
	// Should fall back to default
	if width <= 0 {
		t.Errorf("Width should be positive even with invalid COLUMNS, got %d", width)
	}
}

func TestGetTerminalHeight_WithEnvVar(t *testing.T) {
	// Save and restore LINES
	oldLines := os.Getenv("LINES")
	defer os.Setenv("LINES", oldLines)

	os.Setenv("LINES", "50")
	height := getTerminalHeight()
	if height <= 0 {
		t.Errorf("Height should be positive, got %d", height)
	}
}

func TestGetTerminalHeight_InvalidEnvVar(t *testing.T) {
	// Save and restore LINES
	oldLines := os.Getenv("LINES")
	defer os.Setenv("LINES", oldLines)

	os.Setenv("LINES", "invalid")
	height := getTerminalHeight()
	if height <= 0 {
		t.Errorf("Height should be positive even with invalid LINES, got %d", height)
	}
}

func TestCheckANSISupport_NoColor(t *testing.T) {
	// Save and restore NO_COLOR
	oldNoColor := os.Getenv("NO_COLOR")
	defer os.Setenv("NO_COLOR", oldNoColor)

	os.Setenv("NO_COLOR", "1")
	result := checkANSISupport()
	// When NO_COLOR is set, ANSI should be disabled
	if result && isTerminal() {
		t.Error("ANSI support should be disabled when NO_COLOR is set")
	}
}

func TestCheckANSISupport_DumbTerm(t *testing.T) {
	// Save and restore TERM
	oldTerm := os.Getenv("TERM")
	defer os.Setenv("TERM", oldTerm)

	os.Setenv("TERM", "dumb")
	result := checkANSISupport()
	if result && isTerminal() {
		t.Error("ANSI support should be disabled for dumb terminal")
	}
}

func TestCheckColorSupport(t *testing.T) {
	// Test with COLORTERM=truecolor
	oldColorterm := os.Getenv("COLORTERM")
	defer os.Setenv("COLORTERM", oldColorterm)

	os.Setenv("COLORTERM", "truecolor")
	result := checkColorSupport()
	// Should be enabled if ANSI is supported
	if !result && checkANSISupport() {
		t.Error("Color support should be enabled with COLORTERM=truecolor")
	}
}

func TestCheck256ColorSupport(t *testing.T) {
	// Save and restore TERM
	oldTerm := os.Getenv("TERM")
	defer os.Setenv("TERM", oldTerm)

	os.Setenv("TERM", "xterm-256color")
	result := check256ColorSupport()
	// Should be enabled if ANSI is supported
	if !result && checkANSISupport() {
		t.Error("256-color support should be enabled with 256color TERM")
	}
}

func TestCheckUnicodeSupport_UTF8(t *testing.T) {
	// Save and restore LANG
	oldLang := os.Getenv("LANG")
	defer os.Setenv("LANG", oldLang)

	os.Setenv("LANG", "en_US.UTF-8")
	result := checkUnicodeSupport()
	if !result {
		t.Error("Unicode support should be enabled with UTF-8 LANG")
	}
}

func TestCheckUnicodeSupport_NoUnicode(t *testing.T) {
	// Save and restore NO_UNICODE
	oldNoUnicode := os.Getenv("NO_UNICODE")
	defer os.Setenv("NO_UNICODE", oldNoUnicode)

	os.Setenv("NO_UNICODE", "1")
	result := checkUnicodeSupport()
	if result {
		t.Error("Unicode support should be disabled when NO_UNICODE is set")
	}
}

func TestCheckAlternateBufferSupport(t *testing.T) {
	// Just verify it doesn't panic
	_ = checkAlternateBufferSupport()
}

func TestCheckMouseSupport(t *testing.T) {
	// Just verify it doesn't panic
	_ = checkMouseSupport()
}

func TestDetectColorScheme_WithCOLORFGBG(t *testing.T) {
	// Save and restore COLORFGBG
	oldColorFGBG := os.Getenv("COLORFGBG")
	defer os.Setenv("COLORFGBG", oldColorFGBG)

	tests := []struct {
		name     string
		value    string
		wantDark bool
	}{
		{"dark background", "15;0", true},
		{"light background", "0;15", false},
		{"invalid format", "invalid", true},
		{"single value", "15", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("COLORFGBG", tt.value)
			result := detectColorScheme()
			if tt.wantDark && result != "dark" {
				t.Errorf("Expected dark, got %q", result)
			}
			if !tt.wantDark && result != "light" {
				t.Errorf("Expected light, got %q", result)
			}
		})
	}
}

func TestIsCI(t *testing.T) {
	// Save and restore CI vars
	ciVars := []string{
		"CI", "GITHUB_ACTIONS", "GITLAB_CI", "CIRCLECI", "TRAVIS",
	}
	oldValues := make(map[string]string)
	for _, v := range ciVars {
		oldValues[v] = os.Getenv(v)
	}
	defer func() {
		for v, val := range oldValues {
			os.Setenv(v, val)
		}
	}()

	// Clear all CI vars
	for _, v := range ciVars {
		os.Unsetenv(v)
	}

	// Test with CI=true
	os.Setenv("CI", "true")
	if !isCI() {
		t.Error("isCI should return true when CI is set")
	}

	os.Unsetenv("CI")
	os.Setenv("GITHUB_ACTIONS", "true")
	if !isCI() {
		t.Error("isCI should return true when GITHUB_ACTIONS is set")
	}
}

func TestNewResizeHandler(t *testing.T) {
	rh := NewResizeHandler()
	if rh == nil {
		t.Fatal("NewResizeHandler returned nil")
	}
	if rh.callbacks == nil {
		t.Error("callbacks should not be nil")
	}
	if rh.currentWidth <= 0 {
		t.Error("currentWidth should be positive")
	}
	if rh.currentHeight <= 0 {
		t.Error("currentHeight should be positive")
	}
}

func TestResizeHandler_OnResize(t *testing.T) {
	rh := NewResizeHandler()

	callCount := 0
	rh.OnResize(func(width, height int) {
		callCount++
	})

	if len(rh.callbacks) != 1 {
		t.Errorf("Expected 1 callback, got %d", len(rh.callbacks))
	}

	rh.OnResize(func(width, height int) {})
	if len(rh.callbacks) != 2 {
		t.Errorf("Expected 2 callbacks, got %d", len(rh.callbacks))
	}
}

func TestResizeHandler_GetCurrentSize(t *testing.T) {
	rh := NewResizeHandler()

	width, height := rh.GetCurrentSize()
	if width <= 0 {
		t.Errorf("Width should be positive, got %d", width)
	}
	if height <= 0 {
		t.Errorf("Height should be positive, got %d", height)
	}
}

func TestResizeHandler_StartStop(t *testing.T) {
	rh := NewResizeHandler()

	// Initially not running
	if rh.running {
		t.Error("Handler should not be running initially")
	}

	// Start
	rh.Start()
	if !rh.running {
		t.Error("Handler should be running after Start()")
	}

	// Start again (should be idempotent)
	rh.Start()
	if !rh.running {
		t.Error("Handler should still be running after second Start()")
	}

	// Stop
	rh.Stop()
	// Give goroutine time to stop
	time.Sleep(50 * time.Millisecond)
	if rh.running {
		t.Error("Handler should not be running after Stop()")
	}

	// Stop again (should be idempotent)
	rh.Stop()
}

func TestResizeHandler_RefreshSize(t *testing.T) {
	rh := NewResizeHandler()

	width, height := rh.RefreshSize()
	if width <= 0 {
		t.Errorf("Width should be positive, got %d", width)
	}
	if height <= 0 {
		t.Errorf("Height should be positive, got %d", height)
	}

	// Verify current size is updated
	currentWidth, currentHeight := rh.GetCurrentSize()
	if currentWidth != width || currentHeight != height {
		t.Error("Current size should be updated after RefreshSize")
	}
}

func TestResizeHandler_RefreshSize_WithCallback(t *testing.T) {
	rh := NewResizeHandler()

	callCount := 0
	var mu sync.Mutex
	rh.OnResize(func(width, height int) {
		mu.Lock()
		callCount++
		mu.Unlock()
	})

	// Force a "change" by setting current size to 0
	rh.mu.Lock()
	rh.currentWidth = 0
	rh.currentHeight = 0
	rh.mu.Unlock()

	rh.RefreshSize()
	// Give callback time to execute
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	if callCount < 1 {
		t.Error("Callback should have been called on size change")
	}
	mu.Unlock()
}

func TestResizeHandler_ConcurrentAccess(t *testing.T) {
	rh := NewResizeHandler()
	rh.Start()
	defer rh.Stop()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_, _ = rh.GetCurrentSize()
				rh.RefreshSize()
			}
		}()
	}
	wg.Wait()
}

func TestResizeHandler_CallbackExecution(t *testing.T) {
	rh := NewResizeHandler()

	var receivedWidth, receivedHeight int
	var mu sync.Mutex
	rh.OnResize(func(width, height int) {
		mu.Lock()
		receivedWidth = width
		receivedHeight = height
		mu.Unlock()
	})

	// Force a change
	rh.mu.Lock()
	rh.currentWidth = 0
	rh.currentHeight = 0
	rh.mu.Unlock()

	w, h := rh.RefreshSize()
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	if receivedWidth != w || receivedHeight != h {
		t.Errorf("Callback received (%d, %d), expected (%d, %d)",
			receivedWidth, receivedHeight, w, h)
	}
	mu.Unlock()
}
