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
	if _, err := os.Stat(filepath.Join(".wave", ".ontology-stale")); err == nil {
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
		skillPath := filepath.Join(".wave", "skills", "wave-ctx-"+ctx.Name, "SKILL.md")
		if stat, err := os.Stat(skillPath); err == nil {
			view.HasSkill = true
			view.SkillPath = skillPath
			view.LastUpdated = stat.ModTime()
			view.LastUpdatedAgo = formatTimeAgo(stat.ModTime())
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

	sort.Slice(data.Contexts, func(i, j int) bool {
		return data.Contexts[i].Name < data.Contexts[j].Name
	})

	return data
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
