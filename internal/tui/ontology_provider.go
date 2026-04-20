package tui

import (
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/ontology"
	"github.com/recinq/wave/internal/state"
)

// OntologyInfo is the TUI data projection for an ontology context.
type OntologyInfo struct {
	Name        string
	Description string
	Invariants  []string
	SkillPath   string    // path to SKILL.md if exists
	LastUpdated time.Time // mtime of SKILL.md
	HasSkill    bool
	// Lineage stats from ontology_usage table
	TotalRuns   int
	Successes   int
	Failures    int
	SuccessRate float64
	LastUsed    time.Time
	HasLineage  bool
}

// OntologyOverview holds the top-level ontology summary.
type OntologyOverview struct {
	Telos       string
	Conventions map[string]string
	Contexts    []OntologyInfo
	Stale       bool // true when ontology needs re-analysis
}

// OntologyDataProvider fetches ontology data for the Ontology view.
type OntologyDataProvider interface {
	FetchOntology() (*OntologyOverview, error)
}

// DefaultOntologyDataProvider implements OntologyDataProvider from manifest and skill files.
type DefaultOntologyDataProvider struct {
	manifest  *manifest.Manifest
	skillsDir string
	store     state.StateStore
}

// NewDefaultOntologyDataProvider creates a new ontology data provider.
func NewDefaultOntologyDataProvider(m *manifest.Manifest, skillsDir string, store state.StateStore) *DefaultOntologyDataProvider {
	return &DefaultOntologyDataProvider{manifest: m, skillsDir: skillsDir, store: store}
}

// FetchOntology reads the manifest ontology section and enriches with skill file metadata and lineage stats.
func (p *DefaultOntologyDataProvider) FetchOntology() (*OntologyOverview, error) {
	overview := &OntologyOverview{}

	if p.manifest == nil || p.manifest.Ontology == nil {
		return overview, nil
	}

	o := p.manifest.Ontology
	overview.Telos = o.Telos
	overview.Conventions = o.Conventions

	// Check staleness sentinel (ownership lives in internal/ontology)
	if ontology.IsStaleInDir(".agents") {
		overview.Stale = true
	}

	// Fetch lineage stats
	statsMap := make(map[string]*state.OntologyStats)
	if p.store != nil {
		if allStats, err := p.store.GetOntologyStatsAll(); err == nil {
			for i := range allStats {
				statsMap[allStats[i].ContextName] = &allStats[i]
			}
		}
	}

	for _, ctx := range o.Contexts {
		info := OntologyInfo{
			Name:        ctx.Name,
			Description: ctx.Description,
			Invariants:  ctx.Invariants,
		}

		// Check for skill file
		skillPath := filepath.Join(p.skillsDir, "wave-ctx-"+ctx.Name, "SKILL.md")
		if stat, err := os.Stat(skillPath); err == nil {
			info.HasSkill = true
			info.SkillPath = skillPath
			info.LastUpdated = stat.ModTime()
		}

		// Merge lineage stats
		if stats, ok := statsMap[ctx.Name]; ok {
			info.TotalRuns = stats.TotalRuns
			info.Successes = stats.Successes
			info.Failures = stats.Failures
			info.SuccessRate = stats.SuccessRate
			info.LastUsed = stats.LastUsed
			info.HasLineage = stats.TotalRuns > 0
		}

		overview.Contexts = append(overview.Contexts, info)
	}

	sort.Slice(overview.Contexts, func(i, j int) bool {
		return overview.Contexts[i].Name < overview.Contexts[j].Name
	})

	return overview, nil
}
