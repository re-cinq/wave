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
	"github.com/recinq/wave/internal/humanize"
)

//go:embed static/*
var staticFS embed.FS

//go:embed templates/*
var templatesFS embed.FS

// pageTemplates is the list of page templates that get their own clone of the
// base (layout + partials) so that each page can independently define "title",
// "content", and "scripts" blocks without colliding.
var pageTemplates = []string{
	"templates/run_detail.html",
	"templates/personas.html",
	"templates/persona_detail.html",
	"templates/pipeline_detail.html",
	"templates/contracts.html",
	"templates/contract_detail.html",
	"templates/skills.html",
	"templates/skill_detail.html",
	"templates/compose.html",
	"templates/issues.html",
	"templates/issue_detail.html",
	"templates/prs.html",
	"templates/pr_detail.html",
	"templates/health.html",
	"templates/analytics.html",
	"templates/retros.html",
	"templates/notfound.html",
	"templates/compare.html",
	"templates/webhooks.html",
	"templates/webhook_detail.html",
	"templates/admin.html",
}

// standalonePageTemplates is the list of templates that do NOT extend
// templates/layout.html. They render fully self-contained pages — typically
// to avoid Tailwind utility-class collisions with the project stylesheet.
// Each entry is parsed into its own root template and merged into the
// returned page map alongside the layout-clone pages. The Tailwind utility
// classes referenced by these templates are served from the vendored,
// embedded /static/tailwind.css (compiled via `make tailwind`).
var standalonePageTemplates = []string{
	"templates/work/board.html",
	"templates/work/detail.html",
	"templates/onboard/index.html",
	"templates/proposals/list.html",
	"templates/proposals/detail.html",
	"templates/runs.html",
	"templates/pipelines.html",
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
		"formatDuration":    humanize.DurationSeconds,
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
		"modelTierClass":   modelTierClass,
		"modelTierTooltip": modelTierTooltip,
		"runKindLabel":     runKindLabel,
		"addInt":           func(a, b int) int { return a + b },
		"deref": func(p *int) int {
			if p == nil {
				return 0
			}
			return *p
		},
		"subtreeIsLarger": func(r RunSummary) bool {
			return r.SubtreeTokens > int64(r.TotalTokens)
		},
		"hasResumeChildren": func(children []RunSummary) bool {
			for _, c := range children {
				if c.RunKind == "resume" {
					return true
				}
			}
			return false
		},
		"countResumeChildren": func(children []RunSummary) int {
			count := 0
			for _, c := range children {
				if c.RunKind == "resume" {
					count++
				}
			}
			return count
		},
		// hasCompositionChildren reports whether any composition (iterate /
		// aggregate / branch / loop / sub-pipeline) child runs are attached.
		// Resumes are excluded so the "Children" pill on /runs and the
		// composition section on the run detail page can be rendered
		// independently of the existing "Resumed by" pill (#1450 follow-up).
		"hasCompositionChildren": func(children []RunSummary) bool {
			for _, c := range children {
				if isCompositionRunKind(c.RunKind) {
					return true
				}
			}
			return false
		},
		"countCompositionChildren": func(children []RunSummary) int {
			count := 0
			for _, c := range children {
				if isCompositionRunKind(c.RunKind) {
					count++
				}
			}
			return count
		},
		// filterCompositionChildren returns only the composition children
		// for use by the run-detail composition-children section.
		"filterCompositionChildren": func(children []RunSummary) []RunSummary {
			var out []RunSummary
			for _, c := range children {
				if isCompositionRunKind(c.RunKind) {
					out = append(out, c)
				}
			}
			return out
		},
		// groupChildrenByKind buckets children by run_kind so the run-detail
		// page can render one section per kind ("Iterate children", etc.).
		"groupChildrenByKind": groupChildrenByKind,
		// runKindBreadcrumbLabel renders the parent breadcrumb arrow + text
		// based on the child's run_kind ("← iterate parent", etc.).
		"runKindBreadcrumbLabel": runKindBreadcrumbLabel,
		"pluralize": func(n int, singular, plural string) string {
			if n == 1 {
				return singular
			}
			return plural
		},
		"joinStrings":              strings.Join,
		"proposalStatusBadgeClass": proposalStatusBadgeClass,
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

	// Parse onboarding chat partials (templates/onboard/_*.html). These are
	// shared partials used by the standalone onboard page template.
	err = fs.WalkDir(templatesFS, "templates/onboard", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if !strings.HasPrefix(name, "_") {
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
		return nil, fmt.Errorf("parsing onboard partials: %w", err)
	}

	// Clone the base for each page template.
	pages := make(map[string]*template.Template, len(pageTemplates)+len(standalonePageTemplates))
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

	// Standalone pages are NOT cloned from the layout-bearing base — they
	// render their own <html> shell so Tailwind utility classes don't
	// collide with the project stylesheet. They still get access to shared
	// partials (templates/partials/*) for template composition.
	for _, page := range standalonePageTemplates {
		data, readErr := templatesFS.ReadFile(page)
		if readErr != nil {
			return nil, fmt.Errorf("reading %s: %w", page, readErr)
		}
		t, parseErr := template.New(page).Funcs(funcMap).Parse(string(data))
		if parseErr != nil {
			return nil, fmt.Errorf("parsing %s: %w", page, parseErr)
		}
		// Parse partials into each standalone template so {{template "partials/..."}} works.
		if err := parsePartialsInto(t); err != nil {
			return nil, fmt.Errorf("parsing partials for standalone %s: %w", page, err)
		}
		pages[page] = t
	}

	return pages, nil
}

// parsePartialsInto walks templates/partials/ and parses every file into the
// given template. This lets standalone pages (which don't share the layout
// base) still use {{template "partials/..."}} blocks.
func parsePartialsInto(t *template.Template) error {
	return fs.WalkDir(templatesFS, "templates/partials", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		data, err := templatesFS.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = t.New(path).Parse(string(data))
		return err
	})
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
	case "rejected":
		// Bang glyph signals "stop, take a look" without the red-cross
		// "this broke" connotation of failure. Used by run_row + run_detail.
		return "!"
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
	case "rejected":
		// Rejected = design-rejection terminal state (e.g. fetch-assess
		// reported `implementable: false`). Distinct from `failed` so the
		// UI doesn't misrepresent a legitimate verdict as a runtime bug.
		return "status-rejected"
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
	case "rejected":
		// "rejected" alone reads ambiguously in tables — be explicit about
		// the design-rejection meaning so operators don't confuse it with
		// PR/issue rejection vocabulary.
		return "rejected (no-op)"
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

func modelTierTooltip(model string) string {
	switch strings.ToLower(model) {
	case "strongest":
		return "Strongest tier: uses the most capable model for complex tasks"
	case "balanced":
		return "Balanced tier: good quality-to-cost ratio for typical tasks"
	case "cheapest", "fastest":
		return "Cheapest tier: fast and low-cost for simple tasks"
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
