# Data Model: Browser Automation Capability

**Feature**: #116 — Browser Automation for Personas
**Date**: 2026-03-16

## Entity Relationship

```
Manifest.Adapters["browser"] ──→ BrowserAdapter (AdapterRunner)
                                      │
Persona.Adapter = "browser" ──→ ResolveAdapter("browser")
                                      │
Pipeline Step ──→ executor.runStepExecution()
                      │
                      ├── Prompt = JSON array of BrowserCommand
                      │
                      └── BrowserAdapter.Run()
                              │
                              ├── chromedp.NewExecAllocator (launch Chromium)
                              ├── DomainFilter (network interception)
                              ├── CommandDispatcher (execute commands)
                              └── stdout = JSON array of BrowserResult
```

## Core Types

### BrowserCommand

Represents a single browser action to execute. Passed as JSON in `AdapterRunConfig.Prompt`.

```go
// internal/adapter/browser.go

// BrowserCommand represents a single browser action with parameters.
type BrowserCommand struct {
    Action    string `json:"action"`               // navigate, screenshot, get_text, get_html, click, type
    URL       string `json:"url,omitempty"`         // For navigate
    Selector  string `json:"selector,omitempty"`    // CSS selector for click, type, get_text, get_html
    Value     string `json:"value,omitempty"`       // Text value for type action
    Format    string `json:"format,omitempty"`      // Screenshot format (default: "png")
    TimeoutMs int    `json:"timeout_ms,omitempty"`  // Per-command timeout (default: 30000)
    WaitFor   string `json:"wait_for,omitempty"`    // CSS selector to wait for before executing
}
```

**Validation rules**:
- `action` is required, must be one of: `navigate`, `screenshot`, `get_text`, `get_html`, `click`, `type`
- `url` is required for `navigate`
- `selector` is optional for `get_text` and `get_html` (full page when absent), required for `click` and `type`
- `value` is required for `type`
- `timeout_ms` defaults to 30000 (30 seconds)

### BrowserResult

Structured response from a single browser command. Returned as JSON via stdout.

```go
// internal/adapter/browser.go

// BrowserResult is the structured response from a browser command.
type BrowserResult struct {
    Status     string                 `json:"status"`                // "success" or "error"
    Data       map[string]interface{} `json:"data,omitempty"`        // Command-specific output
    Error      string                 `json:"error,omitempty"`       // Error message if status is "error"
    DurationMs int64                  `json:"duration_ms"`           // Command execution time
}
```

**Data field contents by action**:
| Action | Data Fields |
|--------|------------|
| `navigate` | `title` (string), `url` (string), `status_code` (int) |
| `screenshot` | `image` (base64 string), `format` (string), `width` (int), `height` (int) |
| `get_text` | `text` (string) |
| `get_html` | `html` (string) |
| `click` | `selector` (string) |
| `type` | `selector` (string), `value` (string) |

### BrowserAdapter

The `AdapterRunner` implementation managing the Chromium lifecycle.

```go
// internal/adapter/browser.go

// BrowserAdapter implements AdapterRunner for browser automation via CDP.
type BrowserAdapter struct{}

func NewBrowserAdapter() *BrowserAdapter {
    return &BrowserAdapter{}
}

func (a *BrowserAdapter) Run(ctx context.Context, cfg AdapterRunConfig) (*AdapterResult, error)
```

**Lifecycle**:
1. Parse `cfg.Prompt` as `[]BrowserCommand`
2. Create `chromedp.ExecAllocator` with sandbox flags and viewport config
3. Set up domain filter via `fetch.Enable` using `cfg.AllowedDomains`
4. Execute commands sequentially, collecting `[]BrowserResult`
5. Marshal results to JSON, write to stdout buffer
6. Cleanup: cancel chromedp context (kills browser process)
7. Return `*AdapterResult` with stdout and exit code

### BrowserConfig

Optional configuration for the browser adapter, declared in manifest `adapters.browser`.

```go
// internal/adapter/browser.go

// BrowserConfig holds browser-specific configuration.
type BrowserConfig struct {
    Headless        bool   `yaml:"headless"`          // Default: true
    ViewportWidth   int    `yaml:"viewport_width"`    // Default: 1280
    ViewportHeight  int    `yaml:"viewport_height"`   // Default: 720
    MaxRedirects    int    `yaml:"max_redirects"`     // Default: 10
    MaxResponseSize int    `yaml:"max_response_size"` // Default: 5MB (bytes)
    CommandTimeout  int    `yaml:"command_timeout"`   // Default: 30 (seconds)
}
```

## Manifest Schema Extension

```yaml
# wave.yaml additions
adapters:
  browser:
    binary: chromium       # or google-chrome, chromium-browser
    mode: headless
    default_permissions:
      allowed_tools: []    # Browser adapter doesn't use Claude tools
      deny: []

personas:
  qa-tester:
    adapter: browser
    description: Browser automation for end-to-end testing
    system_prompt_file: .wave/personas/qa-tester.md
    sandbox:
      allowed_domains:
        - localhost
        - "*.example.com"
```

## Preflight Extension

The preflight checker gains browser-aware tool detection:

```go
// Browser binary search order (first found wins)
var browserBinaries = []string{
    "chromium",
    "chromium-browser",
    "google-chrome",
    "google-chrome-stable",
}
```

When a pipeline step uses a persona with `adapter: browser`, the preflight system checks for any of these binaries on PATH. The error message includes platform-specific install instructions.

## Event Integration

Browser commands emit structured events via the existing `event.EventEmitter`:

| Event State | When | Fields |
|------------|------|--------|
| `running` | Browser process starts | `adapter: "browser"` |
| `stream_activity` | Each command executes | `tool_name: "browser.<action>"`, `tool_target: "<url-or-selector>"` |
| `completed` / `failed` | Step ends | Standard completion fields |

## Security Boundaries

1. **Domain enforcement**: `fetch.Enable` intercepts all network requests. Non-allowed domains get `ErrorReasonBlockedByClient`.
2. **Process isolation**: Browser runs in its own process group. `killProcessGroup` ensures cleanup on timeout.
3. **Filesystem isolation**: User data dir is a temp directory inside the workspace. Destroyed with the workspace.
4. **No state leakage**: No cookies, localStorage, or session data survives step boundaries (fresh browser per step).
5. **Response size limits**: `get_text` and `get_html` enforce `MaxResponseSize` to prevent memory exhaustion.
