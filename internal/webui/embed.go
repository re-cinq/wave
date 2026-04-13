package webui

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/recinq/wave/internal/display"
)

//go:embed static/*
var staticFS embed.FS

//go:embed templates/*
var templatesFS embed.FS

// pageTemplates is the list of page templates that get their own clone of the
// base (layout + partials) so that each page can independently define "title",
// "content", and "scripts" blocks without colliding.
var pageTemplates = []string{
	"templates/runs.html",
	"templates/run_detail.html",
	"templates/personas.html",
	"templates/persona_detail.html",
	"templates/pipelines.html",
	"templates/pipeline_detail.html",
	"templates/contracts.html",
	"templates/contract_detail.html",
	"templates/skills.html",
	"templates/compose.html",
	"templates/issues.html",
	"templates/issue_detail.html",
	"templates/prs.html",
	"templates/pr_detail.html",
	"templates/health.html",
	"templates/ontology.html",
	"templates/analytics.html",
	"templates/retros.html",
	"templates/notfound.html",
	"templates/compare.html",
	"templates/webhooks.html",
	"templates/webhook_detail.html",
	"templates/admin.html",
}

// parseTemplates parses all embedded HTML templates using a clone-per-page
// strategy. The layout and partials form a shared base; each page template is
// parsed into its own clone so that block overrides (title, content, scripts)
// don't conflict across pages.
func parseTemplates(extraFuncs ...template.FuncMap) (map[string]*template.Template, error) {
	funcMap := template.FuncMap{
		"statusClass":       statusClass,
		"statusLabel":       statusLabel,
		"statusIcon":        statusIcon,
		"formatDuration":    formatDuration,
		"formatTime":        formatTime,
		"formatTimeISO":     formatTimeISO,
		"formatTokens":      formatTokensFunc,
		"formatBytes":       formatBytesFunc,
		"richInput":         richInputFunc,
		"friendlyModel":     friendlyModelFunc,
		"toJSON":            func(v interface{}) string { b, _ := json.Marshal(v); return string(b) },
		"formatTokensShort": formatTokensShort,
		"shortRunID": func(id string) string {
			if len(id) > 12 {
				return id[:12]
			}
			return id
		},
		"titleCase":  titleCaseFunc,
		"contains":   strings.Contains,
		"hasPrefix":  strings.HasPrefix,
		"checkClass": checkClass,
		"checkIcon":  checkIcon,
		"checkLabel": checkLabel,
		"add":        func(a, b int) int { return a + b },
		"subtract":   func(a, b int) int { return a - b },
		"multiply":   func(a int, b float64) float64 { return float64(a) * b },
		"smoothnessLabel": func(s string) string {
			switch s {
			case "smooth":
				return "Smooth"
			case "bumpy":
				return "Bumpy"
			case "rough":
				return "Rough"
			default:
				return s
			}
		},
		"frictionLabel": func(s string) string {
			switch s {
			case "stall":
				return "Stall"
			case "retry":
				return "Retry"
			case "rework":
				return "Rework"
			case "timeout":
				return "Timeout"
			default:
				return s
			}
		},
		"adapterIcon":    adapterIcon,
		"forgeIcon":      forgeIcon,
		"modelTierClass": modelTierClass,
		"pluralize": func(n int, singular, plural string) string {
			if n == 1 {
				return singular
			}
			return plural
		},
		"joinStrings": strings.Join,
	}
	for _, fm := range extraFuncs {
		for k, v := range fm {
			funcMap[k] = v
		}
	}

	// Parse layout into the base template.
	layoutData, err := templatesFS.ReadFile("templates/layout.html")
	if err != nil {
		return nil, fmt.Errorf("reading layout: %w", err)
	}
	base, err := template.New("base").Funcs(funcMap).Parse(string(layoutData))
	if err != nil {
		return nil, fmt.Errorf("parsing layout: %w", err)
	}

	// Parse all partials into the base.
	err = fs.WalkDir(templatesFS, "templates/partials", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		data, readErr := templatesFS.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		_, parseErr := base.New(path).Parse(string(data))
		return parseErr
	})
	if err != nil {
		return nil, fmt.Errorf("parsing partials: %w", err)
	}

	// Clone the base for each page template.
	pages := make(map[string]*template.Template, len(pageTemplates))
	for _, page := range pageTemplates {
		clone, cloneErr := base.Clone()
		if cloneErr != nil {
			return nil, fmt.Errorf("cloning base for %s: %w", page, cloneErr)
		}
		data, readErr := templatesFS.ReadFile(page)
		if readErr != nil {
			return nil, fmt.Errorf("reading %s: %w", page, readErr)
		}
		if _, parseErr := clone.Parse(string(data)); parseErr != nil {
			return nil, fmt.Errorf("parsing %s: %w", page, parseErr)
		}
		pages[page] = clone
	}

	return pages, nil
}

// staticHandler returns an http.Handler that serves embedded static files.
func staticHandler() http.Handler {
	sub, _ := fs.Sub(staticFS, "static")
	return http.StripPrefix("/static/", http.FileServer(http.FS(sub)))
}

// statusIcon returns a Unicode icon for a pipeline status.
func statusIcon(status string) string {
	switch status {
	case "completed":
		return "✓"
	case "running":
		return "●"
	case "failed":
		return "✕"
	case "cancelled":
		return "○"
	case "pending":
		return "◌"
	default:
		return "·"
	}
}

// statusClass returns a CSS class name for a pipeline status.
func statusClass(status string) string {
	switch status {
	case "completed":
		return "status-completed"
	case "completed_empty":
		return "status-completed-empty"
	case "running":
		return "status-running"
	case "failed":
		return "status-failed"
	case "cancelled":
		return "status-cancelled"
	case "pending":
		return "status-pending"
	case "skipped":
		return "status-skipped"
	case "hook_started":
		return "status-hook-started"
	case "hook_passed":
		return "status-hook-passed"
	case "hook_failed":
		return "status-hook-failed"
	default:
		return "status-unknown"
	}
}

// statusLabel returns a human-readable label for a pipeline status.
func statusLabel(status string) string {
	switch status {
	case "completed_empty":
		return "No Changes"
	default:
		return status
	}
}

// checkClass returns a CSS class name for a CI check status/conclusion.
func checkClass(status, conclusion string) string {
	if status != "completed" {
		return "status-running"
	}
	switch conclusion {
	case "success":
		return "status-completed"
	case "failure", "timed_out", "action_required":
		return "status-failed"
	case "cancelled":
		return "status-cancelled"
	case "skipped", "neutral":
		return "status-pending"
	default:
		return "status-unknown"
	}
}

// checkIcon returns a Unicode icon for a CI check status/conclusion.
func checkIcon(status, conclusion string) string {
	if status != "completed" {
		return "●"
	}
	switch conclusion {
	case "success":
		return "✓"
	case "failure", "timed_out", "action_required":
		return "✕"
	case "skipped", "neutral":
		return "—"
	case "cancelled":
		return "✕"
	default:
		return "?"
	}
}

// checkLabel returns a human-readable label for a CI check status/conclusion.
func checkLabel(status, conclusion string) string {
	if status != "completed" {
		if status == "in_progress" {
			return "In Progress"
		}
		return "Queued"
	}
	switch conclusion {
	case "success":
		return "Passed"
	case "failure":
		return "Failed"
	case "cancelled":
		return "Cancelled"
	case "skipped":
		return "Skipped"
	case "neutral":
		return "Neutral"
	case "timed_out":
		return "Timed Out"
	case "action_required":
		return "Action Required"
	default:
		return conclusion
	}
}

// formatDuration formats a duration in human-readable form.
func formatDuration(seconds float64) string {
	if seconds < 60 {
		return template.HTMLEscapeString(formatDurationShort(seconds))
	}
	m := int(seconds) / 60
	s := int(seconds) % 60
	return template.HTMLEscapeString(formatMinSec(m, s))
}

func formatDurationShort(seconds float64) string {
	if seconds < 1 {
		return "<1s"
	}
	return fmt.Sprintf("%.0fs", seconds)
}

func formatMinSec(m, s int) string {
	return fmt.Sprintf("%dm %ds", m, s)
}

// formatTime formats a time.Time for display.
func formatTime(t interface{}) string {
	switch v := t.(type) {
	case time.Time:
		if v.IsZero() {
			return "-"
		}
		return v.Format("2006-01-02 15:04:05")
	case *time.Time:
		if v == nil || v.IsZero() {
			return "-"
		}
		return v.Format("2006-01-02 15:04:05")
	default:
		return "-"
	}
}

// formatTokensFunc formats a token count for display in templates.
// Accepts int or int64 values and delegates to display.FormatTokenCount.
func formatTokensFunc(v interface{}) string {
	switch n := v.(type) {
	case int:
		return display.FormatTokenCount(n)
	case int64:
		return display.FormatTokenCount(int(n))
	default:
		return "0"
	}
}

// richInputFunc parses a pipeline input string and returns a human-friendly display.
// Recognizes GitHub/GitLab/Bitbucket URLs for issues, PRs, and commits.
func richInputFunc(input, linkedURL string) string {
	url := linkedURL
	if url == "" {
		url = input
	}

	// GitHub: /owner/repo/pull/123 or /owner/repo/issues/123
	if strings.Contains(url, "github.com") {
		parts := strings.Split(url, "/")
		for i, p := range parts {
			if p == "pull" && i+1 < len(parts) {
				return "PR #" + parts[i+1]
			}
			if p == "issues" && i+1 < len(parts) {
				return "Issue #" + parts[i+1]
			}
		}
	}
	// GitLab: /-/merge_requests/123 or /-/issues/123
	if strings.Contains(url, "gitlab") {
		parts := strings.Split(url, "/")
		for i, p := range parts {
			if p == "merge_requests" && i+1 < len(parts) {
				return "MR !" + parts[i+1]
			}
			if p == "issues" && i+1 < len(parts) {
				return "Issue #" + parts[i+1]
			}
		}
	}

	// Truncate long non-URL inputs
	if len(input) > 80 {
		return input[:77] + "..."
	}
	return input
}

// friendlyModelFunc converts raw model IDs to tier display names.
// All models are normalized to the Wave tier vocabulary:
// cheapest / balanced / strongest.
func friendlyModelFunc(model string) string {
	m := strings.ToLower(model)
	switch {
	case strings.Contains(m, "opus"), m == "strongest":
		return "strongest"
	case strings.Contains(m, "sonnet"), m == "balanced":
		return "balanced"
	case strings.Contains(m, "haiku"), m == "cheapest", m == "fastest":
		return "cheapest"
	default:
		if len(model) > 20 {
			return model[:20] + "…"
		}
		return model
	}
}

// modelTierClass returns a CSS class for the model's capability tier.
// Maps model names to tier-strongest / tier-balanced / tier-cheapest so
// badges are color-coded by capability rather than a single color.
func modelTierClass(model string) string {
	m := strings.ToLower(model)
	switch {
	case strings.Contains(m, "opus"), m == "strongest":
		return "tier-strongest"
	case strings.Contains(m, "sonnet"), m == "balanced":
		return "tier-balanced"
	case strings.Contains(m, "haiku"), m == "cheapest", m == "fastest":
		return "tier-cheapest"
	default:
		return ""
	}
}

// formatBytesFunc formats a byte count for human-readable display.
func formatBytesFunc(v interface{}) string {
	var n int64
	switch x := v.(type) {
	case int64:
		n = x
	case int:
		n = int64(x)
	default:
		return "0B"
	}
	switch {
	case n >= 1024*1024:
		return fmt.Sprintf("%.1fMB", float64(n)/1024/1024)
	case n >= 1024:
		return fmt.Sprintf("%.1fKB", float64(n)/1024)
	default:
		return fmt.Sprintf("%dB", n)
	}
}

// formatTimeISO formats a time.Time as an ISO 8601 string for use in HTML
// datetime attributes and JavaScript relative time calculation.
func formatTimeISO(t interface{}) string {
	switch v := t.(type) {
	case time.Time:
		if v.IsZero() {
			return ""
		}
		return v.Format(time.RFC3339)
	case *time.Time:
		if v == nil || v.IsZero() {
			return ""
		}
		return v.Format(time.RFC3339)
	default:
		return ""
	}
}

func titleCaseFunc(s string) string {
	words := strings.Fields(strings.NewReplacer("_", " ", "-", " ").Replace(s))
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}
