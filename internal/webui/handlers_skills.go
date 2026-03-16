package webui

import (
	"net/http"
	"os"
	"path/filepath"
	"sort"

	"github.com/recinq/wave/internal/pipeline"
	"gopkg.in/yaml.v3"
)

// handleSkillsPage handles GET /skills - serves the HTML skills page.
func (s *Server) handleSkillsPage(w http.ResponseWriter, r *http.Request) {
	skills := getSkillSummaries()

	data := struct {
		Skills []SkillSummary
	}{
		Skills: skills,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates["templates/skills.html"].ExecuteTemplate(w, "templates/layout.html", data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

// handleAPISkills handles GET /api/skills - returns skill list as JSON.
func (s *Server) handleAPISkills(w http.ResponseWriter, r *http.Request) {
	skills := getSkillSummaries()
	writeJSON(w, http.StatusOK, SkillListResponse{Skills: skills})
}

// getSkillSummaries scans pipeline YAML files to extract skill declarations,
// mirroring the TUI's DefaultSkillDataProvider logic.
func getSkillSummaries() []SkillSummary {
	pipelinesDir := filepath.Join(".wave", "pipelines")
	entries, err := os.ReadDir(pipelinesDir)
	if err != nil {
		return nil
	}

	skillMap := make(map[string]*SkillSummary)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(pipelinesDir, entry.Name()))
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

				skillMap[skillName] = &SkillSummary{
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

	var skills []SkillSummary
	for _, s := range skillMap {
		skills = append(skills, *s)
	}

	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Name < skills[j].Name
	})

	return skills
}
