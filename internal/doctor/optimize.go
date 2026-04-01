package doctor

import (
	"fmt"
	"strings"

	"github.com/recinq/wave/internal/forge"
	"github.com/recinq/wave/internal/manifest"
)

// OptimizeResult contains proposed changes to wave.yaml.
type OptimizeResult struct {
	ProjectChanges []ConfigChange           `json:"project_changes"`
	PipelineRecs   []PipelineRecommendation `json:"pipeline_recommendations"`
	Conventions    []string                 `json:"conventions_detected"`
	Profile        *ProjectProfile          `json:"profile"`
}

// ConfigChange represents a single proposed change to a config value.
type ConfigChange struct {
	Field    string `json:"field"`    // e.g. "project.test_command"
	Current  string `json:"current"`  // current value (empty if not set)
	Proposed string `json:"proposed"` // recommended value
	Reason   string `json:"reason"`   // why this change is recommended
	Source   string `json:"source"`   // where the recommendation came from
}

// PipelineRecommendation describes a pipeline and whether it's recommended.
type PipelineRecommendation struct {
	Name        string `json:"name"`
	Recommended bool   `json:"recommended"`
	Reason      string `json:"reason"`
}

// HasChanges returns true if any ConfigChange has Current != Proposed.
func (r *OptimizeResult) HasChanges() bool {
	for _, c := range r.ProjectChanges {
		if c.Current != c.Proposed {
			return true
		}
	}
	return false
}

// ApplyTo returns a new Project with proposed values applied.
func (r *OptimizeResult) ApplyTo(p *manifest.Project) *manifest.Project {
	out := &manifest.Project{}
	if p != nil {
		*out = *p
	}
	for _, c := range r.ProjectChanges {
		if c.Current == c.Proposed {
			continue
		}
		switch c.Field {
		case "project.language":
			out.Language = c.Proposed
		case "project.test_command":
			out.TestCommand = c.Proposed
		case "project.lint_command":
			out.LintCommand = c.Proposed
		case "project.build_command":
			out.BuildCommand = c.Proposed
		case "project.source_glob":
			out.SourceGlob = c.Proposed
		}
	}
	return out
}

// Optimize analyzes the project profile against current config and proposes improvements.
func Optimize(profile *ProjectProfile, current *manifest.Project, fi *forge.ForgeInfo, pipelines []string) *OptimizeResult {
	if current == nil {
		current = &manifest.Project{}
	}

	result := &OptimizeResult{
		Profile: profile,
	}

	if profile == nil {
		return result
	}

	optimizeLanguage(result, profile, current)
	optimizeTestCommand(result, profile, current)
	optimizeLintCommand(result, profile, current)
	optimizeBuildCommand(result, profile, current)
	optimizeSourceGlob(result, profile, current)
	optimizePipelines(result, fi, pipelines)
	collectConventions(result, profile)

	return result
}

// optimizeLanguage proposes the primary language from the profile.
func optimizeLanguage(result *OptimizeResult, profile *ProjectProfile, current *manifest.Project) {
	if len(profile.Languages) == 0 {
		return
	}

	primary := primaryLanguage(profile)
	if primary == "" {
		return
	}

	if current.Language != "" && strings.EqualFold(current.Language, primary) {
		return
	}

	if current.Language == "" {
		result.ProjectChanges = append(result.ProjectChanges, ConfigChange{
			Field:    "project.language",
			Current:  "",
			Proposed: primary,
			Reason:   fmt.Sprintf("detected primary language from %d source files", profile.Languages[0].FileCount),
			Source:   "profile",
		})
		return
	}

	// Current is set but doesn't match detected primary
	result.ProjectChanges = append(result.ProjectChanges, ConfigChange{
		Field:    "project.language",
		Current:  current.Language,
		Proposed: primary,
		Reason:   fmt.Sprintf("detected primary language %q (%.0f%% of files) differs from configured %q", primary, profile.Languages[0].Percentage, current.Language),
		Source:   "profile",
	})
}

// optimizeTestCommand proposes the best test command.
func optimizeTestCommand(result *OptimizeResult, profile *ProjectProfile, current *manifest.Project) {
	proposed := bestTestCommand(profile, current)
	if proposed == "" {
		return
	}

	if current.TestCommand == proposed {
		return
	}

	reason := "detected from project analysis"
	source := "profile"

	if profile.TestRunner.Source == "ci" {
		reason = "detected from CI configuration"
		source = "ci"
	} else if profile.TestRunner.Source == "makefile" {
		reason = "detected from Makefile target"
		source = "makefile"
	} else if isSimpleLanguageDefault(current.TestCommand) && proposed != current.TestCommand {
		reason = fmt.Sprintf("upgrade from basic default %q to project-specific command", current.TestCommand)
		source = profile.TestRunner.Source
	}

	result.ProjectChanges = append(result.ProjectChanges, ConfigChange{
		Field:    "project.test_command",
		Current:  current.TestCommand,
		Proposed: proposed,
		Reason:   reason,
		Source:   source,
	})
}

// optimizeLintCommand proposes the best lint command.
func optimizeLintCommand(result *OptimizeResult, profile *ProjectProfile, current *manifest.Project) {
	proposed := bestLintCommand(profile)
	if proposed == "" {
		return
	}

	if current.LintCommand == proposed {
		return
	}

	reason := "detected lint tool configuration"
	source := "profile"

	if len(profile.LintTools) > 0 {
		tool := profile.LintTools[0]
		if tool.ConfigFile != "" {
			reason = fmt.Sprintf("detected %s configuration (%s)", tool.Name, tool.ConfigFile)
			source = tool.ConfigFile
		}
	}

	if isBasicLintDefault(current.LintCommand) && proposed != current.LintCommand {
		reason = fmt.Sprintf("upgrade from basic default %q to dedicated linter", current.LintCommand)
	}

	result.ProjectChanges = append(result.ProjectChanges, ConfigChange{
		Field:    "project.lint_command",
		Current:  current.LintCommand,
		Proposed: proposed,
		Reason:   reason,
		Source:   source,
	})
}

// optimizeBuildCommand proposes the best build command.
func optimizeBuildCommand(result *OptimizeResult, profile *ProjectProfile, current *manifest.Project) {
	proposed := bestBuildCommand(profile)
	if proposed == "" {
		return
	}

	if current.BuildCommand == proposed {
		return
	}

	reason := "detected from build system"
	source := "profile"

	if profile.BuildSystem.Name == "make" {
		reason = "detected Makefile build target"
		source = "makefile"
	} else if profile.BuildSystem.File != "" {
		reason = fmt.Sprintf("detected from %s", profile.BuildSystem.File)
		source = profile.BuildSystem.File
	}

	result.ProjectChanges = append(result.ProjectChanges, ConfigChange{
		Field:    "project.build_command",
		Current:  current.BuildCommand,
		Proposed: proposed,
		Reason:   reason,
		Source:   source,
	})
}

// optimizeSourceGlob proposes the source glob based on primary language.
func optimizeSourceGlob(result *OptimizeResult, profile *ProjectProfile, current *manifest.Project) {
	if len(profile.Languages) == 0 {
		return
	}

	proposed := sourceGlobForLanguage(profile.Languages[0])
	if proposed == "" {
		return
	}

	if current.SourceGlob == proposed {
		return
	}

	result.ProjectChanges = append(result.ProjectChanges, ConfigChange{
		Field:    "project.source_glob",
		Current:  current.SourceGlob,
		Proposed: proposed,
		Reason:   fmt.Sprintf("derived from primary language %q extensions", profile.Languages[0].Name),
		Source:   "profile",
	})
}

// optimizePipelines produces pipeline recommendations based on forge type.
func optimizePipelines(result *OptimizeResult, fi *forge.ForgeInfo, pipelines []string) {
	if len(pipelines) == 0 {
		return
	}

	forgeType := forge.ForgeUnknown
	forgePrefix := ""
	if fi != nil {
		forgeType = fi.Type
		forgePrefix = fi.PipelinePrefix
	}

	for _, name := range pipelines {
		rec := classifyPipeline(name, forgeType, forgePrefix)
		result.PipelineRecs = append(result.PipelineRecs, rec)
	}
}

// classifyPipeline determines whether a pipeline is recommended for the given forge.
func classifyPipeline(name string, forgeType forge.ForgeType, forgePrefix string) PipelineRecommendation {
	pipelinePrefix := extractForgePrefix(name)

	// Universal pipeline (no forge prefix)
	if pipelinePrefix == "" {
		return PipelineRecommendation{
			Name:        name,
			Recommended: true,
			Reason:      "universal pipeline, works with any forge",
		}
	}

	// Unknown forge: include everything
	if forgeType == forge.ForgeUnknown {
		return PipelineRecommendation{
			Name:        name,
			Recommended: true,
			Reason:      "no forge detected, including all pipelines",
		}
	}

	// Matching forge prefix
	if pipelinePrefix == forgePrefix {
		return PipelineRecommendation{
			Name:        name,
			Recommended: true,
			Reason:      fmt.Sprintf("matches detected forge (%s)", forgeType),
		}
	}

	// Wrong forge prefix
	return PipelineRecommendation{
		Name:        name,
		Recommended: false,
		Reason:      fmt.Sprintf("requires %s forge, but project uses %s", forgeLabelForPrefix(pipelinePrefix), forgeType),
	}
}

// collectConventions gathers detected conventions as human-readable strings.
func collectConventions(result *OptimizeResult, profile *ProjectProfile) {
	if profile.Conventions.CommitFormat != "" {
		result.Conventions = append(result.Conventions,
			fmt.Sprintf("commit format: %s", profile.Conventions.CommitFormat))
	}
	if profile.Conventions.HasPRTemplate {
		result.Conventions = append(result.Conventions, "pull request template configured")
	}
	if profile.Conventions.BranchNaming != "" {
		result.Conventions = append(result.Conventions,
			fmt.Sprintf("branch naming: %s", profile.Conventions.BranchNaming))
	}
	if profile.Conventions.HasEditorConfig {
		result.Conventions = append(result.Conventions, "editorconfig configured")
	}
	if profile.HasClaudeMD {
		result.Conventions = append(result.Conventions, "AGENTS.md project instructions present")
	}
	if profile.HasDocker {
		result.Conventions = append(result.Conventions, "Docker configuration present")
	}
}

// --- helpers ---

// primaryLanguage returns the name of the language with the highest file count.
func primaryLanguage(profile *ProjectProfile) string {
	if len(profile.Languages) == 0 {
		return ""
	}
	best := profile.Languages[0]
	for _, lang := range profile.Languages[1:] {
		if lang.FileCount > best.FileCount {
			best = lang
		}
	}
	return strings.ToLower(best.Name)
}

// bestTestCommand determines the best test command from the profile and current config.
func bestTestCommand(profile *ProjectProfile, current *manifest.Project) string {
	runner := profile.TestRunner

	// CI or makefile source = high confidence, prefer that
	if runner.Source == "ci" || runner.Source == "makefile" {
		if runner.Command != "" {
			return runner.Command
		}
	}

	// If profile has a command and current is a simple default, upgrade
	if runner.Command != "" && (current.TestCommand == "" || isSimpleLanguageDefault(current.TestCommand)) {
		return runner.Command
	}

	// If current is empty and profile has something, propose it
	if current.TestCommand == "" && runner.Command != "" {
		return runner.Command
	}

	return ""
}

// bestLintCommand determines the best lint command from the profile.
func bestLintCommand(profile *ProjectProfile) string {
	if len(profile.LintTools) == 0 {
		return ""
	}

	// Check for Makefile lint target first
	if profile.BuildSystem.Name == "make" {
		for _, target := range profile.BuildSystem.Targets {
			if target == "lint" {
				return "make lint"
			}
		}
	}

	// Use the first (primary) lint tool's command
	primary := profile.LintTools[0]
	if primary.Command != "" {
		return primary.Command
	}

	return ""
}

// bestBuildCommand determines the best build command from the profile.
func bestBuildCommand(profile *ProjectProfile) string {
	bs := profile.BuildSystem
	if bs.Name == "" {
		return ""
	}

	// Makefile with build target
	if bs.Name == "make" {
		for _, target := range bs.Targets {
			if target == "build" {
				return "make build"
			}
		}
	}

	// Language-specific defaults
	primary := primaryLanguage(profile)
	switch primary {
	case "go":
		return "go build ./..."
	case "rust":
		return "cargo build"
	case "typescript", "javascript":
		if bs.Name == "npm" || bs.File == "package.json" {
			return "npm run build"
		}
	}

	return ""
}

// sourceGlobForLanguage returns a source glob pattern for the given language.
func sourceGlobForLanguage(lang LanguageInfo) string {
	if len(lang.Extensions) == 0 {
		return ""
	}

	name := strings.ToLower(lang.Name)
	switch name {
	case "go":
		return "**/*.go"
	case "typescript":
		return "**/*.{ts,tsx}"
	case "javascript":
		return "**/*.{js,jsx}"
	case "python":
		return "**/*.py"
	case "rust":
		return "**/*.rs"
	case "java":
		return "**/*.java"
	case "ruby":
		return "**/*.rb"
	case "c":
		return "**/*.{c,h}"
	case "c++", "cpp":
		return "**/*.{cpp,hpp,cc,hh}"
	case "c#", "csharp":
		return "**/*.cs"
	case "php":
		return "**/*.php"
	case "swift":
		return "**/*.swift"
	case "kotlin":
		return "**/*.{kt,kts}"
	case "scala":
		return "**/*.scala"
	default:
		// Build from extensions
		if len(lang.Extensions) == 1 {
			return "**/*" + lang.Extensions[0]
		}
		exts := make([]string, len(lang.Extensions))
		for i, ext := range lang.Extensions {
			exts[i] = strings.TrimPrefix(ext, ".")
		}
		return "**/*.{" + strings.Join(exts, ",") + "}"
	}
}

// isSimpleLanguageDefault returns true if the command is a basic language default.
var simpleDefaults = []string{
	"go test ./...",
	"npm test",
	"yarn test",
	"pytest",
	"python -m pytest",
	"cargo test",
	"mvn test",
	"gradle test",
	"bundle exec rspec",
	"mix test",
}

func isSimpleLanguageDefault(cmd string) bool {
	cmd = strings.TrimSpace(cmd)
	for _, d := range simpleDefaults {
		if cmd == d {
			return true
		}
	}
	return false
}

// isBasicLintDefault returns true if the command is a basic lint default.
func isBasicLintDefault(cmd string) bool {
	cmd = strings.TrimSpace(cmd)
	basics := []string{
		"go vet ./...",
		"npm run lint",
		"yarn lint",
		"python -m py_compile",
	}
	for _, b := range basics {
		if cmd == b {
			return true
		}
	}
	return false
}

// extractForgePrefix returns the forge prefix from a pipeline name, or empty string.
func extractForgePrefix(name string) string {
	knownPrefixes := []string{"gh", "gl", "bb", "gt", "local"}
	for _, prefix := range knownPrefixes {
		if strings.HasPrefix(name, prefix+"-") {
			return prefix
		}
	}
	return ""
}

// forgeLabelForPrefix returns a human-readable forge label for a pipeline prefix.
func forgeLabelForPrefix(prefix string) string {
	switch prefix {
	case "gh":
		return "GitHub"
	case "gl":
		return "GitLab"
	case "bb":
		return "Bitbucket"
	case "gt":
		return "Gitea/Forgejo/Codeberg"
	case "local":
		return "local"
	default:
		return prefix
	}
}
