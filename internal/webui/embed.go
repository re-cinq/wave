package webui

import (
	"embed"
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
	"templates/run_detail_v2.html",
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
		"statusClass":    statusClass,
		"statusIcon":     statusIcon,
		"formatDuration": formatDuration,
		"formatTime":     formatTime,
		"formatTimeISO":  formatTimeISO,
		"formatTokens":   formatTokensFunc,
		"contains":       strings.Contains,
		"hasPrefix":      strings.HasPrefix,
		"checkClass":     checkClass,
		"checkIcon":      checkIcon,
		"checkLabel":     checkLabel,
		"add":              func(a, b int) int { return a + b },
		"subtract":         func(a, b int) int { return a - b },
		"smoothnessLabel":  smoothnessLabel,
		"frictionLabel":    frictionLabel,
		"adapterIcon":      adapterIcon,
		"forgeIcon":        forgeIcon,
		"pluralize": func(n int, singular, plural string) string {
			if n == 1 {
				return singular
			}
			return plural
		},
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
	case "running":
		return "status-running"
	case "failed":
		return "status-failed"
	case "cancelled":
		return "status-cancelled"
	case "pending":
		return "status-pending"
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
