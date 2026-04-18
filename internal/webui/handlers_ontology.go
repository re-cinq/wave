//go:build ontology

package webui

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/recinq/wave/internal/state"
)

// OntologyPageData holds data for the ontology page template.
type OntologyPageData struct {
	ActivePage  string
	Telos       string
	Contexts    []OntologyContextView
	Conventions map[string]string
	HasOntology bool
	Stale       bool // true when ontology needs re-analysis
}

// OntologyContextView is a single bounded context for display.
type OntologyContextView struct {
	Name           string
	Description    string
	Invariants     []string
	InvariantCount int
	HasSkill       bool
	SkillPath      string
	LastUpdated    time.Time
	LastUpdatedAgo string
	// Lineage stats from ontology_usage table
	TotalRuns   int
	Successes   int
	Failures    int
	SuccessRate float64
	LastUsed    time.Time
	LastUsedAgo string
	HasLineage  bool
	// Skill content
	SkillBody string // SKILL.md content (markdown)
	// Pipeline usage — which steps target this context
	UsedBySteps []ContextStepRef
}

// ContextStepRef links a context to a pipeline step that targets it.
type ContextStepRef struct {
	Pipeline string
	StepID   string
}

// handleOntologyPage handles GET /ontology - serves the HTML ontology page.
func (s *Server) handleOntologyPage(w http.ResponseWriter, r *http.Request) {
	data := s.buildOntologyData()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates["templates/ontology.html"].ExecuteTemplate(w, "templates/layout.html", data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

// handleAPIOntology handles GET /api/ontology - returns ontology data as JSON.
func (s *Server) handleAPIOntology(w http.ResponseWriter, r *http.Request) {
	data := s.buildOntologyData()
	writeJSON(w, http.StatusOK, data)
}

// buildOntologyData constructs the OntologyPageData from the loaded manifest.
func (s *Server) buildOntologyData() OntologyPageData {
	data := OntologyPageData{
		ActivePage: "ontology",
	}

	if s.manifest == nil || s.manifest.Ontology == nil {
		return data
	}

	o := s.manifest.Ontology
	data.HasOntology = true
	data.Telos = o.Telos
	data.Conventions = o.Conventions

	// Fetch lineage stats from state store
	statsMap := make(map[string]*state.OntologyStats)
	if s.store != nil {
		if allStats, err := s.store.GetOntologyStatsAll(); err == nil {
			for i := range allStats {
				statsMap[allStats[i].ContextName] = &allStats[i]
			}
		}
	}

	// Check for staleness sentinel
	if _, err := os.Stat(filepath.Join(".agents", ".ontology-stale")); err == nil {
		data.Stale = true
	}

	for _, ctx := range o.Contexts {
		view := OntologyContextView{
			Name:           ctx.Name,
			Description:    ctx.Description,
			Invariants:     ctx.Invariants,
			InvariantCount: len(ctx.Invariants),
		}

		// Check for a provisioned skill file for this context.
		skillPath := filepath.Join(".agents", "skills", "wave-ctx-"+ctx.Name, "SKILL.md")
		if stat, err := os.Stat(skillPath); err == nil {
			view.HasSkill = true
			view.SkillPath = skillPath
			view.LastUpdated = stat.ModTime()
			view.LastUpdatedAgo = formatTimeAgo(stat.ModTime())
			if body, err := os.ReadFile(skillPath); err == nil {
				view.SkillBody = string(body)
			}
		}

		// Merge lineage stats
		if stats, ok := statsMap[ctx.Name]; ok {
			view.TotalRuns = stats.TotalRuns
			view.Successes = stats.Successes
			view.Failures = stats.Failures
			view.SuccessRate = stats.SuccessRate
			view.LastUsed = stats.LastUsed
			view.LastUsedAgo = formatTimeAgo(stats.LastUsed)
			view.HasLineage = stats.TotalRuns > 0
		}

		data.Contexts = append(data.Contexts, view)
	}

	// Scan pipelines for steps that target each context
	contextStepRefs := buildContextStepRefs()
	for i := range data.Contexts {
		if refs, ok := contextStepRefs[data.Contexts[i].Name]; ok {
			data.Contexts[i].UsedBySteps = refs
		}
	}

	sort.Slice(data.Contexts, func(i, j int) bool {
		return data.Contexts[i].Name < data.Contexts[j].Name
	})

	return data
}

// buildContextStepRefs scans .agents/pipelines/ for steps that declare contexts.
func buildContextStepRefs() map[string][]ContextStepRef {
	refs := make(map[string][]ContextStepRef)
	pipelines, err := filepath.Glob(".agents/pipelines/*.yaml")
	if err != nil {
		return refs
	}
	for _, pf := range pipelines {
		data, err := os.ReadFile(pf)
		if err != nil {
			continue
		}
		pipelineName := filepath.Base(pf)
		pipelineName = pipelineName[:len(pipelineName)-5] // strip .yaml

		// Simple line-based scan for contexts: [...] in step blocks
		lines := splitLines(string(data))
		var currentStepID string
		for _, line := range lines {
			trimmed := trimSpace(line)
			if hasPrefix(trimmed, "- id:") || hasPrefix(trimmed, "id:") {
				currentStepID = trimSpace(trimAfter(trimmed, ":"))
			}
			if hasPrefix(trimmed, "contexts:") {
				ctxList := trimSpace(trimAfter(trimmed, "contexts:"))
				// Parse [ctx1, ctx2] format
				ctxList = trimBrackets(ctxList)
				for _, ctx := range splitComma(ctxList) {
					ctx = trimSpace(ctx)
					if ctx != "" {
						refs[ctx] = append(refs[ctx], ContextStepRef{
							Pipeline: pipelineName,
							StepID:   currentStepID,
						})
					}
				}
			}
		}
	}
	return refs
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func trimSpace(s string) string {
	i, j := 0, len(s)
	for i < j && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	for j > i && (s[j-1] == ' ' || s[j-1] == '\t' || s[j-1] == '\r') {
		j--
	}
	return s[i:j]
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func trimAfter(s, sep string) string {
	for i := 0; i <= len(s)-len(sep); i++ {
		if s[i:i+len(sep)] == sep {
			return s[i+len(sep):]
		}
	}
	return s
}

func trimBrackets(s string) string {
	if len(s) >= 2 && s[0] == '[' && s[len(s)-1] == ']' {
		return s[1 : len(s)-1]
	}
	return s
}

func splitComma(s string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}

// formatTimeAgo returns a human-readable "X ago" string for a past timestamp.
func formatTimeAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		return fmt.Sprintf("%dm ago", m)
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}
