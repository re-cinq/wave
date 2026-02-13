//go:build webui

package webui

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"time"
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
	"templates/pipelines.html",
}

// parseTemplates parses all embedded HTML templates using a clone-per-page
// strategy. The layout and partials form a shared base; each page template is
// parsed into its own clone so that block overrides (title, content, scripts)
// don't conflict across pages.
func parseTemplates() (map[string]*template.Template, error) {
	funcMap := template.FuncMap{
		"statusClass":    statusClass,
		"formatDuration": formatDuration,
		"formatTime":     formatTime,
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
	default:
		return "status-unknown"
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
	return fmt.Sprintf("%dm%ds", m, s)
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
