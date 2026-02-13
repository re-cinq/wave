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

// parseTemplates parses all embedded HTML templates.
func parseTemplates() (*template.Template, error) {
	funcMap := template.FuncMap{
		"statusClass":    statusClass,
		"formatDuration": formatDuration,
		"formatTime":     formatTime,
	}

	tmpl := template.New("").Funcs(funcMap)

	err := fs.WalkDir(templatesFS, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := templatesFS.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = tmpl.New(path).Parse(string(data))
		return err
	})
	if err != nil {
		return nil, err
	}

	return tmpl, nil
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
