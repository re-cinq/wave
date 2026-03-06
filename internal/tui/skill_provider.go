package tui

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/recinq/wave/internal/pipeline"
	"gopkg.in/yaml.v3"
)

// SkillInfo is the TUI data projection for a skill.
type SkillInfo struct {
	Name          string
	CommandsGlob  string
	CommandFiles  []string
	InstallCmd    string
	CheckCmd      string
	PipelineUsage []string
}

// SkillDataProvider fetches skill data for the Skills view.
type SkillDataProvider interface {
	FetchSkills() ([]SkillInfo, error)
}

// DefaultSkillDataProvider implements SkillDataProvider by scanning pipeline YAML files.
type DefaultSkillDataProvider struct {
	pipelinesDir string
}

// NewDefaultSkillDataProvider creates a new skill data provider.
func NewDefaultSkillDataProvider(pipelinesDir string) *DefaultSkillDataProvider {
	return &DefaultSkillDataProvider{pipelinesDir: pipelinesDir}
}

// FetchSkills scans all pipeline YAML files and returns deduplicated skills.
func (p *DefaultSkillDataProvider) FetchSkills() ([]SkillInfo, error) {
	if p.pipelinesDir == "" {
		return nil, nil
	}

	entries, err := os.ReadDir(p.pipelinesDir)
	if err != nil {
		return nil, nil
	}

	// Map by skill name for deduplication
	skillMap := make(map[string]*SkillInfo)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(p.pipelinesDir, entry.Name()))
		if err != nil {
			continue
		}

		var pl pipeline.Pipeline
		if err := yaml.Unmarshal(data, &pl); err != nil {
			continue
		}

		if pl.Requires == nil || len(pl.Requires.Skills) == 0 {
			continue
		}

		for skillName, skillConfig := range pl.Requires.Skills {
			if existing, ok := skillMap[skillName]; ok {
				// Deduplicate: add pipeline name if not already present
				found := false
				for _, name := range existing.PipelineUsage {
					if name == pl.Metadata.Name {
						found = true
						break
					}
				}
				if !found {
					existing.PipelineUsage = append(existing.PipelineUsage, pl.Metadata.Name)
				}
			} else {
				glob := skillConfig.CommandsGlob
				var commandFiles []string
				if glob != "" {
					matches, _ := filepath.Glob(glob)
					if matches != nil {
						commandFiles = matches
					}
				}

				skillMap[skillName] = &SkillInfo{
					Name:          skillName,
					CommandsGlob:  glob,
					CommandFiles:  commandFiles,
					InstallCmd:    skillConfig.Install,
					CheckCmd:      skillConfig.Check,
					PipelineUsage: []string{pl.Metadata.Name},
				}
			}
		}
	}

	// Convert map to sorted slice
	var skills []SkillInfo
	for _, s := range skillMap {
		skills = append(skills, *s)
	}

	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Name < skills[j].Name
	})

	return skills, nil
}
