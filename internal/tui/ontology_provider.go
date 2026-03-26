package tui

import (
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/recinq/wave/internal/manifest"
)

// OntologyInfo is the TUI data projection for an ontology context.
type OntologyInfo struct {
	Name        string
	Description string
	Invariants  []string
	SkillPath   string    // path to SKILL.md if exists
	LastUpdated time.Time // mtime of SKILL.md
	HasSkill    bool
}

// OntologyOverview holds the top-level ontology summary.
type OntologyOverview struct {
	Telos       string
	Conventions map[string]string
	Contexts    []OntologyInfo
}

// OntologyDataProvider fetches ontology data for the Ontology view.
type OntologyDataProvider interface {
	FetchOntology() (*OntologyOverview, error)
}

// DefaultOntologyDataProvider implements OntologyDataProvider from manifest and skill files.
type DefaultOntologyDataProvider struct {
	manifest  *manifest.Manifest
	skillsDir string
}

// NewDefaultOntologyDataProvider creates a new ontology data provider.
func NewDefaultOntologyDataProvider(m *manifest.Manifest, skillsDir string) *DefaultOntologyDataProvider {
	return &DefaultOntologyDataProvider{manifest: m, skillsDir: skillsDir}
}

// FetchOntology reads the manifest ontology section and enriches with skill file metadata.
func (p *DefaultOntologyDataProvider) FetchOntology() (*OntologyOverview, error) {
	overview := &OntologyOverview{}

	if p.manifest == nil || p.manifest.Ontology == nil {
		return overview, nil
	}

	o := p.manifest.Ontology
	overview.Telos = o.Telos
	overview.Conventions = o.Conventions

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

		overview.Contexts = append(overview.Contexts, info)
	}

	sort.Slice(overview.Contexts, func(i, j int) bool {
		return overview.Contexts[i].Name < overview.Contexts[j].Name
	})

	return overview, nil
}
