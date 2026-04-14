package webui

import (
	"net/http"
	"os"
	"path/filepath"
	"sort"

	"github.com/recinq/wave/internal/defaults"
	"github.com/recinq/wave/internal/pipeline"
	"github.com/recinq/wave/internal/skill"
	"gopkg.in/yaml.v3"
)

// SkillTemplateSummary represents a bundled skill template for the web UI.
type SkillTemplateSummary struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Installed   bool   `json:"installed"`
}

// handleSkillsPage handles GET /skills - serves the HTML skills page.
func (s *Server) handleSkillsPage(w http.ResponseWriter, r *http.Request) {
	skills := getSkillSummaries()
	installed, available := getSkillTemplateSummaries()

	data := struct {
		ActivePage string
		Skills     []SkillSummary
		Installed  []SkillTemplateSummary
		Available  []SkillTemplateSummary
	}{
		ActivePage: "skills",
		Skills:     skills,
		Installed:  installed,
		Available:  available,
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

// handleAPISkillInstall handles POST /api/skills/{name}/install - installs a bundled skill template.
func (s *Server) handleAPISkillInstall(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeJSONError(w, http.StatusBadRequest, "skill name is required")
		return
	}

	templates := defaults.GetSkillTemplates()
	data, ok := templates[name]
	if !ok {
		writeJSONError(w, http.StatusNotFound, "skill template not found: "+name)
		return
	}

	destDir := filepath.Join(".wave", "skills", name)
	destFile := filepath.Join(destDir, "SKILL.md")

	// Check if already installed
	if _, err := os.Stat(destFile); err == nil {
		writeJSONError(w, http.StatusConflict, "skill already installed: "+name)
		return
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to create skill directory: "+err.Error())
		return
	}

	if err := os.WriteFile(destFile, data, 0644); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to write SKILL.md: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"name":    name,
	})
}

// getSkillTemplateSummaries returns installed and available skill templates.
// Installed includes both bundled templates and non-template skills found in .wave/skills/.
func getSkillTemplateSummaries() (installed []SkillTemplateSummary, available []SkillTemplateSummary) {
	templates := defaults.GetSkillTemplates()
	installedSet := installedSkillSet()
	seen := make(map[string]bool)

	for _, name := range defaults.SkillTemplateNames() {
		seen[name] = true
		data := templates[name]
		desc := ""
		if s, err := skill.ParseMetadata(data); err == nil {
			desc = s.Description
		}

		summary := SkillTemplateSummary{
			Name:        name,
			Description: desc,
			Installed:   installedSet[name],
		}

		if summary.Installed {
			installed = append(installed, summary)
		} else {
			available = append(available, summary)
		}
	}

	// Include non-template skills installed in .wave/skills/
	for name := range installedSet {
		if seen[name] {
			continue
		}
		desc := ""
		skillFile := filepath.Join(".wave", "skills", name, "SKILL.md")
		if data, err := os.ReadFile(skillFile); err == nil {
			if s, err := skill.ParseMetadata(data); err == nil {
				desc = s.Description
			}
		}
		installed = append(installed, SkillTemplateSummary{
			Name:        name,
			Description: desc,
			Installed:   true,
		})
	}

	sort.Slice(installed, func(i, j int) bool {
		return installed[i].Name < installed[j].Name
	})

	return installed, available
}

// installedSkillSet returns a set of skill names installed in .wave/skills/.
func installedSkillSet() map[string]bool {
	result := make(map[string]bool)
	entries, err := os.ReadDir(filepath.Join(".wave", "skills"))
	if err != nil {
		return result
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillFile := filepath.Join(".wave", "skills", entry.Name(), "SKILL.md")
		if _, err := os.Stat(skillFile); err == nil {
			result[entry.Name()] = true
		}
	}
	return result
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
