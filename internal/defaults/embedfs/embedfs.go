// Package embedfs holds the embedded persona/pipeline/contract/prompt assets
// shipped in the Wave binary, exposed via plain string maps. It deliberately
// avoids importing internal/manifest so callers like internal/adapter and
// internal/pipeline can pull cold-start fallback content without creating an
// import cycle through internal/manifest's tests.
package embedfs

import (
	"embed"
	"io/fs"
	"path/filepath"
	"strings"
)

//go:embed personas/*.md
var personasFS embed.FS

//go:embed personas/*.yaml
var personaConfigsFS embed.FS

//go:embed pipelines/*.yaml
var pipelinesFS embed.FS

//go:embed contracts/*.json contracts/*.md
var contractsFS embed.FS

//go:embed prompts/**/*.md
var promptsFS embed.FS

//go:embed schemas/*.json
var schemasFS embed.FS

//go:embed skills/*/SKILL.md
var skillsFS embed.FS

// PersonaConfigsFS exposes the embedded persona yaml configs for callers that
// need to parse them as manifest.Persona (kept out of this package to avoid
// importing internal/manifest and creating an import cycle).
func PersonaConfigsFS() embed.FS { return personaConfigsFS }

// SchemasFS / SkillsFS expose embedded JSON schemas + skill templates.
func SchemasFS() embed.FS { return schemasFS }
func SkillsFS() embed.FS  { return skillsFS }

// GetPersonas returns filename → content for embedded persona prompts.
func GetPersonas() (map[string]string, error) { return readDir(personasFS, "personas") }

// GetPipelines returns filename → content for embedded pipeline yaml.
func GetPipelines() (map[string]string, error) { return readDir(pipelinesFS, "pipelines") }

// GetContracts returns filename → content for embedded contract files.
func GetContracts() (map[string]string, error) { return readDir(contractsFS, "contracts") }

// GetPrompts returns relative path → content for embedded prompts (subdirs
// preserved, e.g. "onboard/detect.md").
func GetPrompts() (map[string]string, error) {
	out := make(map[string]string)
	err := fs.WalkDir(promptsFS, "prompts", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, rerr := promptsFS.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		out[strings.TrimPrefix(path, "prompts/")] = string(data)
		return nil
	})
	return out, err
}

func readDir(fsys embed.FS, dir string) (map[string]string, error) {
	out := make(map[string]string)
	err := fs.WalkDir(fsys, dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, rerr := fsys.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		out[filepath.Base(path)] = string(data)
		return nil
	})
	return out, err
}
