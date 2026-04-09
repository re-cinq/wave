package adapter

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

// BrowserCommand represents a single browser action with parameters.
type BrowserCommand struct {
	Action    string `json:"action"`               // navigate, screenshot, get_text, get_html, click, type
	URL       string `json:"url,omitempty"`        // For navigate
	Selector  string `json:"selector,omitempty"`   // CSS selector for click, type, get_text, get_html
	Value     string `json:"value,omitempty"`      // Text value for type action
	Format    string `json:"format,omitempty"`     // Screenshot format (default: "png")
	TimeoutMs int    `json:"timeout_ms,omitempty"` // Per-command timeout (default: 30000)
	WaitFor   string `json:"wait_for,omitempty"`   // CSS selector to wait for before executing
}

// BrowserResult is the structured response from a browser command.
type BrowserResult struct {
	Status     string         `json:"status"`          // "success" or "error"
	Data       map[string]any `json:"data,omitempty"`  // Command-specific output
	Error      string         `json:"error,omitempty"` // Error message if status is "error"
	DurationMs int64          `json:"duration_ms"`     // Command execution time
}

// BrowserConfig holds browser-specific configuration.
type BrowserConfig struct {
	Headless        bool `yaml:"headless"`
	ViewportWidth   int  `yaml:"viewport_width"`
	ViewportHeight  int  `yaml:"viewport_height"`
	MaxRedirects    int  `yaml:"max_redirects"`
	MaxResponseSize int  `yaml:"max_response_size"` // bytes
	CommandTimeout  int  `yaml:"command_timeout"`   // seconds
}

// DefaultBrowserConfig returns a BrowserConfig with sensible defaults.
func DefaultBrowserConfig() BrowserConfig {
	return BrowserConfig{
		Headless:        true,
		ViewportWidth:   1280,
		ViewportHeight:  720,
		MaxRedirects:    10,
		MaxResponseSize: 5 * 1024 * 1024, // 5MB
		CommandTimeout:  30,
	}
}

// maxViewportWidth and maxViewportHeight cap viewport size to prevent oversized screenshots.
const (
	maxViewportWidth  = 3840
	maxViewportHeight = 2160
)

// BrowserAdapter implements AdapterRunner for browser automation via CDP.
type BrowserAdapter struct {
	Config BrowserConfig
}

// NewBrowserAdapter creates a BrowserAdapter with default configuration.
func NewBrowserAdapter() *BrowserAdapter {
	return &BrowserAdapter{
		Config: DefaultBrowserConfig(),
	}
}

// Run executes browser commands from cfg.Prompt (JSON array of BrowserCommand).
func (a *BrowserAdapter) Run(ctx context.Context, cfg AdapterRunConfig) (*AdapterResult, error) {
	var cancel context.CancelFunc
	if cfg.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, cfg.Timeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	// Parse commands from prompt
	commands, err := parseBrowserCommands(cfg.Prompt)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to parse browser commands: %v", err)), nil
	}

	if len(commands) == 0 {
		return errorResult("no browser commands provided"), nil
	}

	// Enforce viewport limits
	config := a.Config
	if config.ViewportWidth > maxViewportWidth {
		config.ViewportWidth = maxViewportWidth
	}
	if config.ViewportHeight > maxViewportHeight {
		config.ViewportHeight = maxViewportHeight
	}

	// Build chromedp allocator options
	opts := buildAllocatorOpts(config, cfg.SandboxEnabled)

	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx, opts...)
	defer allocCancel()

	browserCtx, browserCancel := chromedp.NewContext(allocCtx)
	defer browserCancel()

	// Set up domain filtering if allowed domains are specified
	if len(cfg.AllowedDomains) > 0 {
		if err := setupDomainFilter(browserCtx, cfg.AllowedDomains); err != nil {
			return errorResult(fmt.Sprintf("failed to set up domain filter: %v", err)), nil
		}
	}

	// Execute commands sequentially
	results := make([]BrowserResult, 0, len(commands))
	var redirectCount atomic.Int32
	for _, cmd := range commands {
		result := a.executeCommand(browserCtx, cmd, config, &redirectCount)
		results = append(results, result)

		// Emit stream event if callback is available
		if cfg.OnStreamEvent != nil {
			target := cmd.URL
			if target == "" {
				target = cmd.Selector
			}
			cfg.OnStreamEvent(StreamEvent{
				Type:     "tool_use",
				ToolName: "browser." + cmd.Action,
				Content:  fmt.Sprintf("%s: %s", result.Status, target),
			})
		}

		// Stop on error
		if result.Status == "error" {
			break
		}
	}

	// Marshal results to JSON
	resultJSON, err := json.Marshal(results)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to marshal results: %v", err)), nil
	}

	return &AdapterResult{
		ExitCode:      0,
		Stdout:        bytes.NewReader(resultJSON),
		ResultContent: string(resultJSON),
	}, nil
}

// executeCommand dispatches a single browser command to its handler.
func (a *BrowserAdapter) executeCommand(ctx context.Context, cmd BrowserCommand, config BrowserConfig, redirectCount *atomic.Int32) BrowserResult {
	start := time.Now()

	// Per-command timeout: create a separate timeout context but do NOT use it
	// as the chromedp context — chromedp reacts to context cancellation by closing
	// the CDP session. Instead, we use a timer-based approach.
	timeoutMs := cmd.TimeoutMs
	if timeoutMs <= 0 {
		timeoutMs = config.CommandTimeout * 1000
	}

	// Use a goroutine + channel pattern for timeout instead of context.WithTimeout
	// to avoid prematurely closing the browser session.
	type cmdResult struct {
		result BrowserResult
	}
	done := make(chan cmdResult, 1)

	go func() {
		var result BrowserResult

		// Execute wait_for if specified
		if cmd.WaitFor != "" {
			if err := chromedp.Run(ctx, chromedp.WaitVisible(cmd.WaitFor)); err != nil {
				done <- cmdResult{BrowserResult{
					Status:     "error",
					Error:      fmt.Sprintf("wait_for selector %q failed: %v", cmd.WaitFor, err),
					DurationMs: time.Since(start).Milliseconds(),
				}}
				return
			}
		}

		switch cmd.Action {
		case "navigate":
			result = executeNavigate(ctx, cmd, config, redirectCount)
		case "screenshot":
			result = executeScreenshot(ctx, cmd)
		case "get_text":
			result = executeGetText(ctx, cmd, config)
		case "get_html":
			result = executeGetHTML(ctx, cmd, config)
		case "click":
			result = executeClick(ctx, cmd)
		case "type":
			result = executeType(ctx, cmd)
		default:
			result = BrowserResult{
				Status: "error",
				Error:  fmt.Sprintf("unknown action: %q", cmd.Action),
			}
		}

		result.DurationMs = time.Since(start).Milliseconds()
		done <- cmdResult{result}
	}()

	timer := time.NewTimer(time.Duration(timeoutMs) * time.Millisecond)
	defer timer.Stop()

	select {
	case r := <-done:
		return r.result
	case <-timer.C:
		return BrowserResult{
			Status:     "error",
			Error:      fmt.Sprintf("command %q timed out after %dms", cmd.Action, timeoutMs),
			DurationMs: time.Since(start).Milliseconds(),
		}
	case <-ctx.Done():
		return BrowserResult{
			Status:     "error",
			Error:      fmt.Sprintf("command %q canceled: %v", cmd.Action, ctx.Err()),
			DurationMs: time.Since(start).Milliseconds(),
		}
	}
}

func executeNavigate(ctx context.Context, cmd BrowserCommand, config BrowserConfig, redirectCount *atomic.Int32) BrowserResult {
	if cmd.URL == "" {
		return BrowserResult{Status: "error", Error: "navigate requires 'url' field"}
	}

	// Listen for redirect events to enforce max redirects
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		if _, ok := ev.(*network.EventRequestWillBeSent); ok {
			redirectCount.Add(1)
		}
	})

	var title string
	err := chromedp.Run(ctx,
		chromedp.Navigate(cmd.URL),
		chromedp.Title(&title),
	)
	if err != nil {
		if int(redirectCount.Load()) > config.MaxRedirects {
			return BrowserResult{
				Status: "error",
				Error:  fmt.Sprintf("exceeded max redirects (%d)", config.MaxRedirects),
			}
		}
		return BrowserResult{
			Status: "error",
			Error:  fmt.Sprintf("navigate failed: %v", err),
		}
	}

	if int(redirectCount.Load()) > config.MaxRedirects {
		return BrowserResult{
			Status: "error",
			Error:  fmt.Sprintf("exceeded max redirects (%d)", config.MaxRedirects),
		}
	}

	// Get the final URL
	var currentURL string
	if err := chromedp.Run(ctx, chromedp.Location(&currentURL)); err != nil {
		currentURL = cmd.URL
	}

	return BrowserResult{
		Status: "success",
		Data: map[string]any{
			"title": title,
			"url":   currentURL,
		},
	}
}

func executeScreenshot(ctx context.Context, cmd BrowserCommand) BrowserResult {
	var buf []byte
	err := chromedp.Run(ctx, chromedp.CaptureScreenshot(&buf))
	if err != nil {
		return BrowserResult{
			Status: "error",
			Error:  fmt.Sprintf("screenshot failed: %v", err),
		}
	}

	encoded := base64.StdEncoding.EncodeToString(buf)
	format := cmd.Format
	if format == "" {
		format = "png"
	}

	return BrowserResult{
		Status: "success",
		Data: map[string]any{
			"image":  encoded,
			"format": format,
		},
	}
}

func executeGetText(ctx context.Context, cmd BrowserCommand, config BrowserConfig) BrowserResult {
	var text string
	var err error

	if cmd.Selector != "" {
		err = chromedp.Run(ctx, chromedp.Text(cmd.Selector, &text))
	} else {
		err = chromedp.Run(ctx, chromedp.Evaluate(`document.body.innerText`, &text))
	}

	if err != nil {
		return BrowserResult{
			Status: "error",
			Error:  fmt.Sprintf("get_text failed: %v", err),
		}
	}

	// Enforce response size limit
	truncated := false
	if config.MaxResponseSize > 0 && len(text) > config.MaxResponseSize {
		text = text[:config.MaxResponseSize]
		truncated = true
	}

	data := map[string]any{"text": text}
	if truncated {
		data["truncated"] = true
		data["warning"] = fmt.Sprintf("response truncated to %d bytes", config.MaxResponseSize)
	}

	return BrowserResult{
		Status: "success",
		Data:   data,
	}
}

func executeGetHTML(ctx context.Context, cmd BrowserCommand, config BrowserConfig) BrowserResult {
	var html string
	var err error

	if cmd.Selector != "" {
		err = chromedp.Run(ctx, chromedp.OuterHTML(cmd.Selector, &html))
	} else {
		err = chromedp.Run(ctx, chromedp.Evaluate(`document.documentElement.outerHTML`, &html))
	}

	if err != nil {
		return BrowserResult{
			Status: "error",
			Error:  fmt.Sprintf("get_html failed: %v", err),
		}
	}

	// Enforce response size limit
	truncated := false
	if config.MaxResponseSize > 0 && len(html) > config.MaxResponseSize {
		html = html[:config.MaxResponseSize]
		truncated = true
	}

	data := map[string]any{"html": html}
	if truncated {
		data["truncated"] = true
		data["warning"] = fmt.Sprintf("response truncated to %d bytes", config.MaxResponseSize)
	}

	return BrowserResult{
		Status: "success",
		Data:   data,
	}
}

func executeClick(ctx context.Context, cmd BrowserCommand) BrowserResult {
	if cmd.Selector == "" {
		return BrowserResult{Status: "error", Error: "click requires 'selector' field"}
	}

	err := chromedp.Run(ctx, chromedp.Click(cmd.Selector))
	if err != nil {
		return BrowserResult{
			Status: "error",
			Error:  fmt.Sprintf("click failed for selector %q: %v", cmd.Selector, err),
		}
	}

	return BrowserResult{
		Status: "success",
		Data:   map[string]any{"selector": cmd.Selector},
	}
}

func executeType(ctx context.Context, cmd BrowserCommand) BrowserResult {
	if cmd.Selector == "" {
		return BrowserResult{Status: "error", Error: "type requires 'selector' field"}
	}
	if cmd.Value == "" {
		return BrowserResult{Status: "error", Error: "type requires 'value' field"}
	}

	err := chromedp.Run(ctx, chromedp.SendKeys(cmd.Selector, cmd.Value))
	if err != nil {
		return BrowserResult{
			Status: "error",
			Error:  fmt.Sprintf("type failed for selector %q: %v", cmd.Selector, err),
		}
	}

	return BrowserResult{
		Status: "success",
		Data: map[string]any{
			"selector": cmd.Selector,
			"value":    cmd.Value,
		},
	}
}

// parseBrowserCommands parses JSON input into a slice of BrowserCommand.
func parseBrowserCommands(input string) ([]BrowserCommand, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("empty input")
	}

	var commands []BrowserCommand
	if err := json.Unmarshal([]byte(input), &commands); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	// Validate each command
	for i, cmd := range commands {
		if err := validateCommand(cmd); err != nil {
			return nil, fmt.Errorf("command %d: %w", i, err)
		}
	}

	return commands, nil
}

// validateCommand checks that a BrowserCommand has valid fields for its action.
func validateCommand(cmd BrowserCommand) error {
	switch cmd.Action {
	case "navigate":
		if cmd.URL == "" {
			return fmt.Errorf("navigate requires 'url' field")
		}
	case "screenshot":
		// No required fields
	case "get_text", "get_html":
		// Selector is optional
	case "click":
		if cmd.Selector == "" {
			return fmt.Errorf("click requires 'selector' field")
		}
	case "type":
		if cmd.Selector == "" {
			return fmt.Errorf("type requires 'selector' field")
		}
		if cmd.Value == "" {
			return fmt.Errorf("type requires 'value' field")
		}
	case "":
		return fmt.Errorf("'action' field is required")
	default:
		return fmt.Errorf("unknown action: %q", cmd.Action)
	}
	return nil
}

// buildAllocatorOpts creates chromedp allocator options from config.
func buildAllocatorOpts(config BrowserConfig, sandboxEnabled bool) []chromedp.ExecAllocatorOption {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", config.Headless),
		chromedp.WindowSize(config.ViewportWidth, config.ViewportHeight),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("no-default-browser-check", true),
		// Disable Chrome's OS-level namespace sandbox. Wave provides its
		// own security model (adapter sandbox, workspace isolation, domain
		// filtering). Without this flag Chrome fails in CI runners and
		// containers that lack unprivileged user-namespace support.
		chromedp.NoSandbox,
	)

	if sandboxEnabled {
		opts = append(opts,
			chromedp.Flag("disable-extensions", true),
			chromedp.Flag("disable-plugins", true),
			chromedp.Flag("disable-popup-blocking", true),
		)
	}

	return opts
}

// setupDomainFilter configures CDP fetch interception to block non-allowed domains.
func setupDomainFilter(ctx context.Context, allowedDomains []string) error {
	// Enable fetch interception for all URL patterns
	err := chromedp.Run(ctx, fetch.Enable().WithPatterns([]*fetch.RequestPattern{
		{URLPattern: "*", RequestStage: fetch.RequestStageRequest},
	}))
	if err != nil {
		return fmt.Errorf("failed to enable fetch interception: %w", err)
	}

	// Listen for request events and filter by domain.
	// Each paused request must be handled in a goroutine because chromedp.Run
	// blocks on the event loop, and the listener fires from that same loop.
	// We capture the context once and pass request-local data to avoid races.
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		e, ok := ev.(*fetch.EventRequestPaused)
		if !ok {
			return
		}
		requestID := e.RequestID
		requestURL := e.Request.URL
		go func() {
			host := extractHost(requestURL)
			if isDomainAllowed(host, allowedDomains) {
				_ = chromedp.Run(ctx, fetch.ContinueRequest(requestID))
			} else {
				_ = chromedp.Run(ctx, fetch.FailRequest(requestID, network.ErrorReasonBlockedByClient))
			}
		}()
	})

	return nil
}

// isDomainAllowed checks if a host matches the allowed domains list.
// Supports exact match and wildcard patterns (e.g., "*.example.com").
func isDomainAllowed(host string, allowedDomains []string) bool {
	for _, domain := range allowedDomains {
		if domain == host {
			return true
		}
		// Wildcard support: *.example.com matches sub.example.com
		if strings.HasPrefix(domain, "*.") {
			suffix := domain[1:] // ".example.com"
			if strings.HasSuffix(host, suffix) {
				return true
			}
		}
	}
	return false
}

// extractHost extracts the hostname from a URL string.
func extractHost(rawURL string) string {
	// Simple host extraction without importing net/url to keep it lightweight.
	// Format: scheme://host:port/path
	s := rawURL
	if idx := strings.Index(s, "://"); idx >= 0 {
		s = s[idx+3:]
	}
	if idx := strings.Index(s, "/"); idx >= 0 {
		s = s[:idx]
	}
	if idx := strings.Index(s, ":"); idx >= 0 {
		s = s[:idx]
	}
	return s
}

// errorResult creates an AdapterResult representing a top-level error.
func errorResult(message string) *AdapterResult {
	result := []BrowserResult{{
		Status: "error",
		Error:  message,
	}}
	data, _ := json.Marshal(result)
	return &AdapterResult{
		ExitCode:      1,
		Stdout:        bytes.NewReader(data),
		ResultContent: string(data),
	}
}
