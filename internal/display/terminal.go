package display

import (
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"golang.org/x/term"
)

// TerminalInfo provides terminal detection and information.
type TerminalInfo struct {
	capabilities *TerminalCapabilities
}

// NewTerminalInfo creates a new TerminalInfo with detected capabilities.
func NewTerminalInfo() *TerminalInfo {
	return &TerminalInfo{
		capabilities: detectCapabilities(),
	}
}

// IsTTY returns true if output is directed to a terminal.
func (ti *TerminalInfo) IsTTY() bool {
	return ti.capabilities.IsTTY
}

// GetWidth returns the terminal width in columns.
func (ti *TerminalInfo) GetWidth() int {
	return ti.capabilities.Width
}

// GetHeight returns the terminal height in rows.
func (ti *TerminalInfo) GetHeight() int {
	return ti.capabilities.Height
}

// SupportsANSI returns true if terminal supports ANSI escape sequences.
func (ti *TerminalInfo) SupportsANSI() bool {
	return ti.capabilities.SupportsANSI
}

// SupportsColor returns true if terminal supports 24-bit RGB colors.
func (ti *TerminalInfo) SupportsColor() bool {
	return ti.capabilities.SupportsColor
}

// Supports256Colors returns true if terminal supports 256-color palette.
func (ti *TerminalInfo) Supports256Colors() bool {
	return ti.capabilities.Supports256Colors
}

// SupportsUnicode returns true if terminal supports Unicode characters.
func (ti *TerminalInfo) SupportsUnicode() bool {
	return ti.capabilities.SupportsUnicode
}

// SupportsAlternateBuffer returns true if terminal supports alternate screen buffer.
func (ti *TerminalInfo) SupportsAlternateBuffer() bool {
	return ti.capabilities.SupportsAlternate
}

// HasMouseSupport returns true if terminal supports mouse input.
func (ti *TerminalInfo) HasMouseSupport() bool {
	return ti.capabilities.HasMouseSupport
}

// GetColorScheme returns the detected color scheme ("light" or "dark").
func (ti *TerminalInfo) GetColorScheme() string {
	return ti.capabilities.ColorScheme
}

// Capabilities returns the full TerminalCapabilities structure.
func (ti *TerminalInfo) Capabilities() *TerminalCapabilities {
	return ti.capabilities
}

// detectCapabilities detects terminal capabilities by checking environment
// variables, TTY status, and terminal type information.
func detectCapabilities() *TerminalCapabilities {
	caps := &TerminalCapabilities{
		IsTTY:             isTerminal(),
		Width:             getTerminalWidth(),
		Height:            getTerminalHeight(),
		SupportsANSI:      checkANSISupport(),
		SupportsColor:     checkColorSupport(),
		Supports256Colors: check256ColorSupport(),
		SupportsUnicode:   checkUnicodeSupport(),
		SupportsAlternate: checkAlternateBufferSupport(),
		HasMouseSupport:   checkMouseSupport(),
		ColorScheme:       detectColorScheme(),
	}
	return caps
}

// isTerminal checks if stdout is connected to a terminal.
// Honors WAVE_FORCE_TTY=1/0 for testing auto-mode behavior in CI/scripts.
func isTerminal() bool {
	if v := os.Getenv("WAVE_FORCE_TTY"); v != "" {
		return v == "1" || v == "true"
	}
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// getTerminalWidth returns the terminal width, defaulting to 80 if not available.
func getTerminalWidth() int {
	if width, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
		if width > 0 {
			return width
		}
	}
	// Fallback to environment variable or default
	if widthStr := os.Getenv("COLUMNS"); widthStr != "" {
		if width, err := strconv.Atoi(widthStr); err == nil && width > 0 {
			return width
		}
	}
	return 80
}

// getTerminalHeight returns the terminal height, defaulting to 24 if not available.
func getTerminalHeight() int {
	if _, height, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
		if height > 0 {
			return height
		}
	}
	// Fallback to environment variable or default
	if heightStr := os.Getenv("LINES"); heightStr != "" {
		if height, err := strconv.Atoi(heightStr); err == nil && height > 0 {
			return height
		}
	}
	return 24
}

// GetTerminalWidth returns the current terminal width.
// Exported wrapper for use by display-related code in other packages.
func GetTerminalWidth() int {
	return getTerminalWidth()
}

// GetTerminalHeight returns the current terminal height.
// Exported wrapper for use by display-related code in other packages.
func GetTerminalHeight() int {
	return getTerminalHeight()
}

// checkANSISupport checks if terminal supports ANSI escape sequences.
func checkANSISupport() bool {
	// ANSI is supported if:
	// 1. We're on a TTY
	// 2. TERM is not "dumb"
	// 3. Not explicitly disabled via NO_COLOR
	if !isTerminal() {
		return false
	}

	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	term := os.Getenv("TERM")
	if term == "dumb" || term == "" {
		return false
	}

	return true
}

// checkColorSupport checks for 24-bit RGB color support.
func checkColorSupport() bool {
	// 24-bit color support is indicated by:
	// 1. COLORTERM=truecolor
	// 2. TERM contains "256color" or "truecolor"
	// 3. CI environments (GitHub Actions, GitLab CI, etc.)
	// And we're on a TTY with ANSI support

	if !checkANSISupport() {
		return false
	}

	colorterm := os.Getenv("COLORTERM")
	if colorterm == "truecolor" || colorterm == "24bit" {
		return true
	}

	term := os.Getenv("TERM")
	if strings.Contains(term, "256color") || strings.Contains(term, "truecolor") {
		return true
	}

	// Check for CI/CD environments that support colors
	if isCI() {
		return true
	}

	return false
}

// check256ColorSupport checks for 256-color palette support.
func check256ColorSupport() bool {
	if !checkANSISupport() {
		return false
	}

	term := os.Getenv("TERM")
	// 256-color support is present in most modern terminals
	if strings.Contains(term, "256") {
		return true
	}

	// Fallback: if color is supported, we likely have 256 colors
	if checkColorSupport() {
		return true
	}

	return false
}

// checkUnicodeSupport checks if terminal supports Unicode characters.
func checkUnicodeSupport() bool {
	// Unicode support is indicated by:
	// 1. LANG or LC_ALL containing "UTF-8"
	// 2. Being on a TTY (most modern terminals support Unicode)
	// 3. Not explicitly disabled

	if os.Getenv("NO_UNICODE") != "" {
		return false
	}

	lang := os.Getenv("LANG")
	if strings.Contains(strings.ToUpper(lang), "UTF-8") {
		return true
	}

	lcAll := os.Getenv("LC_ALL")
	if strings.Contains(strings.ToUpper(lcAll), "UTF-8") {
		return true
	}

	// Assume Unicode support on TTY (reasonable for modern terminals)
	if isTerminal() {
		return true
	}

	return false
}

// checkAlternateBufferSupport checks if terminal supports alternate screen buffer.
func checkAlternateBufferSupport() bool {
	// Alternate buffer support is indicated by:
	// 1. Being on a TTY
	// 2. TERM not being "dumb"
	// Most modern terminals support this

	if !isTerminal() {
		return false
	}

	term := os.Getenv("TERM")
	return term != "dumb" && term != ""
}

// checkMouseSupport checks if terminal supports mouse input.
func checkMouseSupport() bool {
	// Mouse support is rare in headless environments
	// Most terminal emulators support it when interactive

	if !isTerminal() {
		return false
	}

	// Disable mouse in non-interactive CI environments
	if isCI() {
		return false
	}

	return true
}

// detectColorScheme determines if the terminal uses a light or dark background.
func detectColorScheme() string {
	// Check for explicit override
	scheme := os.Getenv("COLORFGBG")
	if scheme != "" {
		// COLORFGBG format is "foreground;background"
		parts := strings.Split(scheme, ";")
		if len(parts) == 2 {
			if bg, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
				// Light backgrounds are typically high values (>7)
				if bg > 7 {
					return "light"
				}
				return "dark"
			}
		}
	}

	// Check for common light terminal preferences
	if os.Getenv("ITERM_PROFILE") != "" {
		// iTerm2 - check if "Light" is in profile name (heuristic)
		if strings.Contains(strings.ToLower(os.Getenv("ITERM_PROFILE")), "light") {
			return "light"
		}
	}

	// Default to dark (most common for developer terminals)
	return "dark"
}

// isCI checks if running in a CI/CD environment.
func isCI() bool {
	ciVars := []string{
		"CI",
		"CONTINUOUS_INTEGRATION",
		"BUILD_ID",
		"BUILD_NUMBER",
		"RUN_ID",
		"GITHUB_ACTIONS",
		"GITLAB_CI",
		"CIRCLECI",
		"TRAVIS",
		"DRONE",
	}

	for _, envVar := range ciVars {
		if os.Getenv(envVar) != "" {
			return true
		}
	}

	return false
}

// ResizeHandler manages terminal window resize events.
type ResizeHandler struct {
	mu             sync.RWMutex
	callbacks      []func(width, height int)
	stopChan       chan struct{}
	signalChan     chan os.Signal
	currentWidth   int
	currentHeight  int
	running        bool
}

// NewResizeHandler creates a new resize handler.
func NewResizeHandler() *ResizeHandler {
	width, height := getTerminalWidth(), getTerminalHeight()
	return &ResizeHandler{
		callbacks:     make([]func(width, height int), 0),
		stopChan:      make(chan struct{}),
		signalChan:    make(chan os.Signal, 1),
		currentWidth:  width,
		currentHeight: height,
		running:       false,
	}
}

// OnResize registers a callback to be called when terminal is resized.
func (rh *ResizeHandler) OnResize(callback func(width, height int)) {
	rh.mu.Lock()
	defer rh.mu.Unlock()
	rh.callbacks = append(rh.callbacks, callback)
}

// Start begins listening for resize events.
func (rh *ResizeHandler) Start() {
	rh.mu.Lock()
	if rh.running {
		rh.mu.Unlock()
		return
	}
	rh.running = true
	rh.mu.Unlock()

	// Register for SIGWINCH (window change signal)
	signal.Notify(rh.signalChan, syscall.SIGWINCH)

	go rh.handleResizeEvents()
}

// Stop stops listening for resize events.
func (rh *ResizeHandler) Stop() {
	rh.mu.Lock()
	defer rh.mu.Unlock()

	if !rh.running {
		return
	}

	rh.running = false
	signal.Stop(rh.signalChan)
	close(rh.stopChan)
}

// GetCurrentSize returns the current terminal size.
func (rh *ResizeHandler) GetCurrentSize() (width, height int) {
	rh.mu.RLock()
	defer rh.mu.RUnlock()
	return rh.currentWidth, rh.currentHeight
}

// handleResizeEvents processes resize signals and invokes callbacks.
func (rh *ResizeHandler) handleResizeEvents() {
	for {
		select {
		case <-rh.signalChan:
			// Get new terminal size
			width := getTerminalWidth()
			height := getTerminalHeight()

			rh.mu.Lock()
			// Only trigger callbacks if size actually changed
			if width != rh.currentWidth || height != rh.currentHeight {
				rh.currentWidth = width
				rh.currentHeight = height

				// Invoke all callbacks
				for _, callback := range rh.callbacks {
					if callback != nil {
						// Run callback in goroutine to avoid blocking
						go callback(width, height)
					}
				}
			}
			rh.mu.Unlock()

		case <-rh.stopChan:
			return
		}
	}
}

// RefreshSize manually updates the current size without waiting for a signal.
// Useful for checking size on demand.
func (rh *ResizeHandler) RefreshSize() (width, height int) {
	width = getTerminalWidth()
	height = getTerminalHeight()

	rh.mu.Lock()
	changed := width != rh.currentWidth || height != rh.currentHeight
	rh.currentWidth = width
	rh.currentHeight = height
	rh.mu.Unlock()

	// If size changed, trigger callbacks
	if changed {
		rh.mu.RLock()
		for _, callback := range rh.callbacks {
			if callback != nil {
				go callback(width, height)
			}
		}
		rh.mu.RUnlock()
	}

	return width, height
}
