package webui

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/recinq/wave/internal/defaults"
	"github.com/recinq/wave/internal/pipeline"
	"github.com/recinq/wave/internal/sandbox"
	"github.com/recinq/wave/internal/skill"
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
	if err := s.assets.templates["templates/skills.html"].Execute(w, data); err != nil {
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

	destDir := filepath.Join(".agents", "skills", name)
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
// Installed includes both bundled templates and non-template skills found in .agents/skills/.
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

	// Include non-template skills installed in .agents/skills/
	for name := range installedSet {
		if seen[name] {
			continue
		}
		desc := ""
		skillFile := filepath.Join(".agents", "skills", name, "SKILL.md")
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

// installedSkillSet returns a set of skill names installed in .agents/skills/.
func installedSkillSet() map[string]bool {
	result := make(map[string]bool)
	entries, err := os.ReadDir(filepath.Join(".agents", "skills"))
	if err != nil {
		return result
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillFile := filepath.Join(".agents", "skills", entry.Name(), "SKILL.md")
		if _, err := os.Stat(skillFile); err == nil {
			result[entry.Name()] = true
		}
	}
	return result
}

// getSkillSummaries scans pipeline YAML files to extract skill declarations,
// mirroring the TUI's DefaultSkillDataProvider logic.
func getSkillSummaries() []SkillSummary {
	pipelinesDir := filepath.Join(".agents", "pipelines")
	skillMap := make(map[string]*SkillSummary)

	for _, pl := range pipeline.ScanPipelinesDir(pipelinesDir) {
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

				pkg, source, method := parseInstallCmd(skillConfig.Install)
				skillMap[skillName] = &SkillSummary{
					Name:          skillName,
					CommandsGlob:  glob,
					CommandFiles:  commandFiles,
					InstallCmd:    skillConfig.Install,
					InstallPkg:    pkg,
					InstallSource: source,
					InstallMethod: method,
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

// parseInstallCmd extracts package name, source repo, and method from an install command.
// Examples:
//
//	"uv tool install --force specify-cli --from git+https://github.com/github/spec-kit.git"
//	  → pkg="specify-cli", source="github.com/github/spec-kit", method="uv"
//	"pip install requests"  → pkg="requests", source="", method="pip"
//	"npm install -g typescript" → pkg="typescript", source="", method="npm"
//	"brew install gh"       → pkg="gh", source="", method="brew"
func parseInstallCmd(cmd string) (pkg, source, method string) {
	if cmd == "" {
		return "", "", ""
	}
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return "", "", ""
	}

	// Derive method from first token (or second for "uv tool")
	method = parts[0]
	if method == "uv" && len(parts) > 1 && parts[1] == "tool" {
		method = "uv"
	}

	// Find "install" verb index
	installIdx := -1
	for i, p := range parts {
		if p == "install" {
			installIdx = i
			break
		}
	}
	if installIdx < 0 {
		return "", "", method
	}

	// First non-flag arg after "install" is the package name
	for i := installIdx + 1; i < len(parts); i++ {
		p := parts[i]
		if strings.HasPrefix(p, "-") {
			continue
		}
		// skip "tool" sub-command for "uv tool install"
		if p == "tool" {
			continue
		}
		pkg = p
		break
	}

	// Look for --from <url> (uv-style)
	for i, p := range parts {
		if p == "--from" && i+1 < len(parts) {
			src := parts[i+1]
			src = strings.TrimPrefix(src, "git+")
			src = strings.TrimSuffix(src, ".git")
			src = strings.TrimPrefix(src, "https://")
			src = strings.TrimPrefix(src, "http://")
			source = src
			break
		}
	}

	return pkg, source, method
}

// handleAPISkillRunInstall handles POST /api/skills/{name}/run-install.
// It runs the install command declared in requires.skills for a pipeline tool requirement.
func (s *Server) handleAPISkillRunInstall(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeJSONError(w, http.StatusBadRequest, "skill name is required")
		return
	}

	skills := getSkillSummaries()
	var installCmd string
	for _, sk := range skills {
		if sk.Name == name {
			installCmd = sk.InstallCmd
			break
		}
	}

	if installCmd == "" {
		writeJSONError(w, http.StatusNotFound, "no install command found for skill: "+name)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	// Route the install command through internal/sandbox so it inherits
	// the same backend policy (none/docker/bubblewrap) and audit log line
	// that pipeline-driven shellouts use. The webui historically called
	// `exec.CommandContext("sh", "-c", ...)` directly, bypassing both.
	cfg := sandbox.Config{}
	if s.runtime.manifest != nil {
		cfg.Backend = sandbox.SandboxBackendType(s.runtime.manifest.Runtime.Sandbox.ResolveBackend())
		cfg.DockerImage = s.runtime.manifest.Runtime.Sandbox.DockerImage
		cfg.AllowedDomains = s.runtime.manifest.Runtime.Sandbox.DefaultAllowedDomains
		cfg.EnvPassthrough = s.runtime.manifest.Runtime.Sandbox.EnvPassthrough
	}
	out, err := sandbox.RunShell(ctx, installCmd, cfg)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
			"output":  string(out),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"output":  string(out),
	})
}

// SkillDetail holds all data for the skill detail page.
type SkillDetail struct {
	ActivePage  string
	Name        string
	Description string
	Body        string // full SKILL.md body (after frontmatter)
	IsInstalled bool   // found in .agents/skills/
	Requirement *SkillSummary
}

// handleSkillDetailPage handles GET /skills/{name}.
func (s *Server) handleSkillDetailPage(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.NotFound(w, r)
		return
	}

	detail := SkillDetail{
		ActivePage: "skills",
		Name:       name,
	}

	// Check installed SKILL.md
	skillFile := filepath.Join(".agents", "skills", name, "SKILL.md")
	if data, err := os.ReadFile(skillFile); err == nil {
		detail.IsInstalled = true
		if meta, err := skill.ParseMetadata(data); err == nil {
			detail.Description = meta.Description
		}
		// Strip frontmatter (--- ... ---) for body
		body := string(data)
		if strings.HasPrefix(body, "---") {
			if idx := strings.Index(body[3:], "---"); idx >= 0 {
				body = strings.TrimSpace(body[3+idx+3:])
			}
		}
		detail.Body = body
	}

	// Check pipeline requirements
	for _, sk := range getSkillSummaries() {
		if sk.Name == name {
			sk := sk
			detail.Requirement = &sk
			break
		}
	}

	if !detail.IsInstalled && detail.Requirement == nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.assets.templates["templates/skill_detail.html"].ExecuteTemplate(w, "templates/layout.html", detail); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}
