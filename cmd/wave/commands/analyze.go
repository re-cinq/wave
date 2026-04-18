package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/display"
	"github.com/recinq/wave/internal/doctor"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/state"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// AnalyzeResult holds the output of the analyze command.
type AnalyzeResult struct {
	Telos         string                 `json:"telos,omitempty"`
	Contexts      []AnalyzeContext       `json:"contexts"`
	Conventions   map[string]string      `json:"conventions,omitempty"`
	SkillsWritten []string               `json:"skills_written,omitempty"`
	Profile       *doctor.ProjectProfile `json:"profile,omitempty"`
}

// AnalyzeContext represents a single bounded context with detected metadata.
type AnalyzeContext struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Packages    []string `json:"packages,omitempty"`
	FileCount   int      `json:"file_count,omitempty"`
	HasTests    bool     `json:"has_tests,omitempty"`
}

// NewAnalyzeCmd creates the analyze command for ontology generation.
func NewAnalyzeCmd() *cobra.Command {
	var deepFlag bool
	var evolveFlag bool
	var applyFlag bool
	var decisionsFlag bool

	cmd := &cobra.Command{
		Use:   "analyze",
		Short: "Analyze project domain and generate ontology context skills",
		Long: `Scan the project structure and generate ontology context skills
(.agents/skills/wave-ctx-*/SKILL.md) from the declared bounded contexts
in wave.yaml.

By default, performs a deterministic scan using directory structure and
package detection. Use --deep for AI-assisted analysis that extracts
invariants, domain vocabulary, and key decisions from code and tests.
Use --evolve to propose ontology updates based on pipeline run history.
Use --apply to auto-write proposed contexts to wave.yaml.
Use --decisions to show orchestration decision provenance table.`,
		Example: `  wave analyze
  wave analyze --json
  wave analyze --deep
  wave analyze --evolve
  wave analyze --apply
  wave analyze --decisions`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			if decisionsFlag {
				return runAnalyzeDecisions(cmd)
			}
			if deepFlag {
				return runAnalyzeDeep(cmd)
			}
			return runAnalyze(cmd, evolveFlag, applyFlag)
		},
	}

	cmd.Flags().BoolVar(&deepFlag, "deep", false, "Use AI-assisted analysis (requires adapter)")
	cmd.Flags().BoolVar(&evolveFlag, "evolve", false, "Propose updates based on pipeline run history")
	cmd.Flags().BoolVar(&applyFlag, "apply", false, "Auto-write proposed contexts to wave.yaml")
	cmd.Flags().BoolVar(&decisionsFlag, "decisions", false, "Show orchestration decision provenance")

	return cmd
}

func runAnalyze(cmd *cobra.Command, evolve bool, apply bool) error {
	outputCfg := GetOutputConfig(cmd)
	format := ResolveFormat(cmd, "text")
	if outputCfg.Format == OutputFormatJSON {
		format = "json"
	}

	// Load manifest
	manifestPath, _ := cmd.Root().PersistentFlags().GetString("manifest")
	if manifestPath == "" {
		manifestPath = "wave.yaml"
	}

	m, err := manifest.Load(manifestPath)
	if err != nil {
		return NewCLIError(CodeManifestMissing,
			fmt.Sprintf("failed to load manifest: %s", err),
			"Run 'wave init' to create a manifest")
	}

	// Route to evolve mode if requested
	if evolve {
		return runEvolve(cmd, m, manifestPath)
	}

	f := display.NewFormatter()

	// Scan project
	fmt.Fprintf(os.Stderr, "  Scanning project structure...\n")
	profile, err := doctor.ScanProject(".")
	if err != nil {
		return NewCLIError(CodeInternalError,
			fmt.Sprintf("project scan failed: %s", err),
			"Check that the project directory is accessible").WithCause(err)
	}

	// Build result
	result := buildAnalyzeResult(m, profile)

	// Generate SKILL.md files
	skillsWritten, writeErr := writeContextSkills(result)
	if writeErr != nil {
		return NewCLIError(CodeInternalError,
			fmt.Sprintf("failed to write context skills: %s", writeErr),
			"Check write permissions for .agents/skills/").WithCause(writeErr)
	}
	result.SkillsWritten = skillsWritten

	// Propose ontology updates to wave.yaml if contexts were detected but not declared
	proposedContexts := proposeNewContexts(m, profile)
	if len(proposedContexts) > 0 {
		result.Contexts = append(result.Contexts, proposedContexts...)
	}

	// Render output
	switch format {
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	default:
		renderAnalyzeText(cmd.OutOrStdout(), result, f)
	}

	// Auto-write proposed contexts to wave.yaml when --apply is set
	if apply && len(proposedContexts) > 0 {
		if err := writeContextsToManifest(manifestPath, proposedContexts); err != nil {
			return NewCLIError(CodeInternalError,
				fmt.Sprintf("failed to write contexts to manifest: %s", err),
				"Check write permissions for wave.yaml").WithCause(err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "\n  ✓ %d new context(s) written to %s\n", len(proposedContexts), manifestPath)
	} else if len(proposedContexts) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "\n  %s New contexts detected from project structure.\n", f.Muted("Tip:"))
		fmt.Fprintf(cmd.OutOrStdout(), "  Run 'wave analyze --apply' to add them to wave.yaml, or --deep for AI enrichment.\n")
	}

	return nil
}

// buildAnalyzeResult constructs the analysis result from manifest and profile.
func buildAnalyzeResult(m *manifest.Manifest, profile *doctor.ProjectProfile) *AnalyzeResult {
	result := &AnalyzeResult{
		Profile: profile,
	}

	if m.Ontology != nil {
		result.Telos = m.Ontology.Telos
		result.Conventions = m.Ontology.Conventions

		for _, ctx := range m.Ontology.Contexts {
			ac := AnalyzeContext{
				Name:        ctx.Name,
				Description: ctx.Description,
			}
			// Try to match context to packages by directory name
			ac.Packages, ac.FileCount, ac.HasTests = matchContextToPackages(ctx.Name)
			result.Contexts = append(result.Contexts, ac)
		}
	}

	return result
}

// matchContextToPackages scans the filesystem for directories matching a context name.
func matchContextToPackages(contextName string) (packages []string, fileCount int, hasTests bool) {
	// Normalize context name for directory matching
	normalized := strings.ReplaceAll(contextName, "-", "/")

	// Walk looking for matching directories
	_ = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() {
			return nil
		}
		// Skip hidden directories and vendor
		base := filepath.Base(path)
		if strings.HasPrefix(base, ".") || base == "vendor" || base == "node_modules" {
			return filepath.SkipDir
		}

		// Check if directory name matches context or its parts
		if matchesContext(path, contextName, normalized) {
			packages = append(packages, path)
			// Count files in this directory
			entries, _ := os.ReadDir(path)
			for _, e := range entries {
				if !e.IsDir() {
					fileCount++
					if strings.HasSuffix(e.Name(), "_test.go") ||
						strings.HasSuffix(e.Name(), ".test.ts") ||
						strings.HasSuffix(e.Name(), ".test.js") ||
						strings.HasSuffix(e.Name(), "_test.py") ||
						strings.HasSuffix(e.Name(), ".spec.ts") {
						hasTests = true
					}
				}
			}
		}
		return nil
	})
	return
}

// matchesContext checks if a directory path matches a context name.
func matchesContext(path, contextName, normalized string) bool {
	base := filepath.Base(path)
	// Direct match
	if base == contextName {
		return true
	}
	// Match with dashes replaced by directory separators
	if strings.HasSuffix(path, normalized) {
		return true
	}
	// Match with underscores
	if base == strings.ReplaceAll(contextName, "-", "_") {
		return true
	}
	return false
}

// proposeNewContexts suggests contexts from package structure that aren't already declared.
// It scans standard code directories, monorepo packages, manifest services, and compose services.
func proposeNewContexts(m *manifest.Manifest, profile *doctor.ProjectProfile) []AnalyzeContext {
	if profile == nil {
		return nil
	}

	// Build set of existing context names
	existing := make(map[string]bool)
	if m.Ontology != nil {
		for _, ctx := range m.Ontology.Contexts {
			existing[ctx.Name] = true
		}
	}

	// Track already-proposed names to avoid duplicates
	proposed := make(map[string]*AnalyzeContext)

	// Check internal/ or src/ directories for potential bounded contexts
	for _, dir := range []string{"internal", "src", "pkg", "lib", "app"} {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			name := entry.Name()
			if existing[name] || strings.HasPrefix(name, ".") || proposed[name] != nil {
				continue
			}
			count := countFiles(filepath.Join(dir, name))
			if count >= 3 {
				proposed[name] = &AnalyzeContext{
					Name:      name,
					Packages:  []string{filepath.Join(dir, name)},
					FileCount: count,
				}
			}
		}
	}

	// Scan monorepo packages from doctor profile
	if profile.MonorepoLayout != nil {
		for _, pkg := range profile.MonorepoLayout.Packages {
			name := filepath.Base(pkg)
			if existing[name] || strings.HasPrefix(name, ".") || proposed[name] != nil {
				continue
			}
			count := countFiles(pkg)
			if count >= 1 { // monorepo packages are already curated, lower threshold
				proposed[name] = &AnalyzeContext{
					Name:      name,
					Packages:  []string{pkg},
					FileCount: count,
				}
			}
		}
	}

	// Scan monorepo service dirs (services/, apps/, packages/)
	for _, scanDir := range []string{"services", "apps", "packages"} {
		entries, err := os.ReadDir(scanDir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			name := entry.Name()
			if existing[name] || strings.HasPrefix(name, ".") || proposed[name] != nil {
				continue
			}
			dirPath := filepath.Join(scanDir, name)
			count := countFiles(dirPath)
			if count >= 1 {
				proposed[name] = &AnalyzeContext{
					Name:      name,
					Packages:  []string{dirPath},
					FileCount: count,
				}
			}
		}
	}

	// Propose from manifest services
	if m.Project != nil {
		for svcName := range m.Project.Services {
			if existing[svcName] || proposed[svcName] != nil {
				continue
			}
			svc := m.Project.Services[svcName]
			proposed[svcName] = &AnalyzeContext{
				Name:     svcName,
				Packages: []string{svc.Path},
			}
		}
	}

	// Convert map to sorted slice
	var result []AnalyzeContext
	for _, ctx := range proposed {
		result = append(result, *ctx)
	}
	return result
}

// countFiles returns the number of non-directory files in a directory (non-recursive).
func countFiles(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if !e.IsDir() {
			count++
		}
	}
	return count
}

// writeContextsToManifest appends proposed contexts to the ontology section of wave.yaml.
// It preserves existing YAML structure by reading, modifying, and rewriting.
func writeContextsToManifest(manifestPath string, contexts []AnalyzeContext) error {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read manifest: %w", err)
	}

	// Parse into generic map to preserve structure
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("parse manifest: %w", err)
	}

	// Get or create ontology section
	ontology, _ := raw["ontology"].(map[string]interface{})
	if ontology == nil {
		ontology = make(map[string]interface{})
		raw["ontology"] = ontology
	}

	// Get existing contexts
	existingContexts, _ := ontology["contexts"].([]interface{})

	// Build set of existing context names
	existingNames := make(map[string]bool)
	for _, ctx := range existingContexts {
		if m, ok := ctx.(map[string]interface{}); ok {
			if name, ok := m["name"].(string); ok {
				existingNames[name] = true
			}
		}
	}

	// Append new contexts
	for _, ctx := range contexts {
		if existingNames[ctx.Name] {
			continue
		}
		entry := map[string]interface{}{
			"name": ctx.Name,
		}
		if ctx.Description != "" {
			entry["description"] = ctx.Description
		} else if len(ctx.Packages) > 0 {
			entry["description"] = fmt.Sprintf("Auto-detected from %s", strings.Join(ctx.Packages, ", "))
		}
		existingContexts = append(existingContexts, entry)
	}

	ontology["contexts"] = existingContexts

	// Write back
	out, err := yaml.Marshal(raw)
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}

	return os.WriteFile(manifestPath, out, 0644)
}

// writeContextSkills generates SKILL.md files for declared ontology contexts.
func writeContextSkills(result *AnalyzeResult) ([]string, error) {
	var written []string

	for _, ctx := range result.Contexts {
		skillDir := filepath.Join(".agents", "skills", "wave-ctx-"+ctx.Name)
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			return written, fmt.Errorf("failed to create %s: %w", skillDir, err)
		}

		skillPath := filepath.Join(skillDir, "SKILL.md")
		content := generateSkillContent(ctx, result.Telos)

		if err := os.WriteFile(skillPath, []byte(content), 0644); err != nil {
			return written, fmt.Errorf("failed to write %s: %w", skillPath, err)
		}
		written = append(written, skillPath)
	}

	return written, nil
}

// generateSkillContent creates the SKILL.md content for a context.
func generateSkillContent(ctx AnalyzeContext, telos string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "# %s Context\n\n", ctx.Name)

	if telos != "" {
		fmt.Fprintf(&b, "> Project telos: %s\n\n", telos)
	}

	if ctx.Description != "" {
		fmt.Fprintf(&b, "%s\n\n", ctx.Description)
	}

	if len(ctx.Packages) > 0 {
		fmt.Fprintf(&b, "## Packages\n\n")
		for _, pkg := range ctx.Packages {
			fmt.Fprintf(&b, "- `%s`", pkg)
			if ctx.FileCount > 0 {
				fmt.Fprintf(&b, " (%d files)", ctx.FileCount)
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	if ctx.HasTests {
		fmt.Fprintf(&b, "## Testing\n\nTest files detected in this context.\n\n")
	}

	fmt.Fprintf(&b, "## Invariants\n\n")
	fmt.Fprintf(&b, "_Run `wave analyze --deep` to extract invariants from code and tests._\n")

	return b.String()
}

// renderAnalyzeText outputs the analysis result as human-readable text.
func renderAnalyzeText(w io.Writer, result *AnalyzeResult, f *display.Formatter) {
	fmt.Fprintln(w)

	if result.Telos != "" {
		fmt.Fprintf(w, "  %s %s\n\n", f.Colorize("Telos:", "\033[1;37m"), result.Telos)
	}

	if result.Profile != nil && result.Profile.FilesScanned > 0 {
		fmt.Fprintf(w, "  %s %d files scanned", f.Muted("Project:"), result.Profile.FilesScanned)
		if len(result.Profile.Languages) > 0 {
			langs := make([]string, 0, len(result.Profile.Languages))
			for _, l := range result.Profile.Languages {
				langs = append(langs, l.Name)
			}
			fmt.Fprintf(w, ", languages: %s", strings.Join(langs, ", "))
		}
		fmt.Fprintln(w)
		fmt.Fprintln(w)
	}

	if len(result.Contexts) > 0 {
		fmt.Fprintf(w, "  %s\n\n", f.Colorize("Bounded Contexts:", "\033[1;37m"))
		for _, ctx := range result.Contexts {
			fmt.Fprintf(w, "    %s", f.Primary(ctx.Name))
			if ctx.Description != "" {
				fmt.Fprintf(w, " — %s", ctx.Description)
			}
			fmt.Fprintln(w)

			if len(ctx.Packages) > 0 {
				fmt.Fprintf(w, "      %s %s\n", f.Muted("packages:"), strings.Join(ctx.Packages, ", "))
			}
			if ctx.FileCount > 0 {
				fmt.Fprintf(w, "      %s %d files", f.Muted("files:"), ctx.FileCount)
				if ctx.HasTests {
					fmt.Fprintf(w, " (tests found)")
				}
				fmt.Fprintln(w)
			}
			fmt.Fprintln(w)
		}
	} else {
		fmt.Fprintf(w, "  No bounded contexts declared in wave.yaml.\n")
		fmt.Fprintf(w, "  %s Add contexts under ontology.contexts or run 'wave init' to configure.\n\n",
			f.Muted("Hint:"))
	}

	if len(result.Conventions) > 0 {
		fmt.Fprintf(w, "  %s\n", f.Colorize("Conventions:", "\033[1;37m"))
		for k, v := range result.Conventions {
			fmt.Fprintf(w, "    %s: %s\n", k, v)
		}
		fmt.Fprintln(w)
	}

	if len(result.SkillsWritten) > 0 {
		fmt.Fprintf(w, "  %s Context skills written:\n", f.Success("✓"))
		for _, path := range result.SkillsWritten {
			fmt.Fprintf(w, "    %s\n", path)
		}
		fmt.Fprintln(w)
	}
}

// deepAnalysisResult is the expected JSON output from the deep analysis agent.
type deepAnalysisResult struct {
	Contexts []deepContextResult `json:"contexts"`
}

type deepContextResult struct {
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	Invariants       []string `json:"invariants"`
	KeyDecisions     []string `json:"key_decisions"`
	DomainVocabulary []struct {
		Term    string `json:"term"`
		Meaning string `json:"meaning"`
	} `json:"domain_vocabulary"`
	NeighboringContexts []string `json:"neighboring_contexts"`
	KeyFiles            []string `json:"key_files"`
}

// runAnalyzeDeep launches a navigator persona to deep-scan the project
// and extract invariants, domain vocabulary, and key decisions per context.
func runAnalyzeDeep(cmd *cobra.Command) error {
	manifestPath, _ := cmd.Root().PersistentFlags().GetString("manifest")
	if manifestPath == "" {
		manifestPath = "wave.yaml"
	}

	m, err := manifest.Load(manifestPath)
	if err != nil {
		return NewCLIError(CodeManifestMissing,
			fmt.Sprintf("failed to load manifest: %s", err),
			"Run 'wave init' to create a manifest")
	}

	if m.Ontology == nil || len(m.Ontology.Contexts) == 0 {
		return NewCLIError(CodeInvalidArgs,
			"no ontology contexts declared in wave.yaml",
			"Run 'wave init' or add contexts under ontology.contexts first, then re-run with --deep")
	}

	// Build the context list for the prompt
	var contextList strings.Builder
	for _, ctx := range m.Ontology.Contexts {
		fmt.Fprintf(&contextList, "- **%s**: %s\n", ctx.Name, ctx.Description)
	}

	prompt := fmt.Sprintf(`Analyze this project's codebase and extract detailed domain knowledge for each bounded context.

The project's telos: %s

Declared bounded contexts:
%s

For EACH bounded context above, search the codebase to find:

1. **Invariants**: Rules that must ALWAYS hold true. Look for:
   - Assertions and validation logic
   - Error conditions that panic or return errors
   - Comments with "must", "always", "never", "invariant"
   - Test assertions that verify invariant behavior

2. **Key decisions**: Architectural choices that shape the context. Look for:
   - Design patterns used (interfaces, factories, strategies)
   - Trade-offs documented in comments or commit messages
   - Configuration defaults with reasoning

3. **Domain vocabulary**: Key terms specific to this context. Look for:
   - Type names and their doc comments
   - Constants and enums
   - Function names that encode domain concepts

4. **Neighboring contexts**: What other bounded contexts this one interacts with.

5. **Key files**: The 5-10 most important source files for this context.

Output your findings as a JSON object with this exact structure:
`+"```json\n"+`{
  "contexts": [
    {
      "name": "<context-name>",
      "description": "<updated description based on code analysis>",
      "invariants": ["<invariant 1>", "<invariant 2>"],
      "key_decisions": ["<decision 1>", "<decision 2>"],
      "domain_vocabulary": [{"term": "<term>", "meaning": "<meaning>"}],
      "neighboring_contexts": ["<context-name>"],
      "key_files": ["<relative/path/to/file.go>"]
    }
  ]
}
`+"```\n"+`
Write the JSON to .agents/output/deep-analysis.json`, m.Ontology.Telos, contextList.String())

	// Resolve adapter binary
	adapterName := "claude"
	if len(m.Adapters) > 0 {
		for name := range m.Adapters {
			adapterName = name
			break
		}
	}
	adapterDef := m.GetAdapter(adapterName)
	if adapterDef == nil {
		return NewCLIError(CodeInvalidArgs,
			fmt.Sprintf("adapter %q not found in manifest", adapterName),
			"Check wave.yaml adapters section")
	}

	// Get navigator persona for read-only analysis
	persona := m.GetPersona("navigator")
	if persona == nil {
		return NewCLIError(CodeInvalidArgs,
			"navigator persona not found in manifest",
			"Add a navigator persona to .agents/personas/")
	}

	fmt.Fprintf(os.Stderr, "  Launching deep analysis with %s adapter...\n", adapterName)
	fmt.Fprintf(os.Stderr, "  Analyzing %d bounded contexts...\n\n", len(m.Ontology.Contexts))

	// Create output directory
	if err := os.MkdirAll(".agents/output", 0o755); err != nil {
		return err
	}

	// Build adapter config
	cwd, _ := os.Getwd()
	cfg := adapter.AdapterRunConfig{
		Adapter:       adapterDef.Binary,
		Persona:       "navigator",
		WorkspacePath: cwd,
		Prompt:        prompt,
		SystemPrompt:  "You are a codebase analysis specialist. Extract domain knowledge from code structure, tests, and documentation. Output structured JSON.",
		// No timeout — deep analysis scales with codebase size
		Model:        persona.Model,
		AllowedTools: []string{"Read", "Glob", "Grep", "Bash(git log*)"},
		DenyTools:    []string{"Write", "Edit", "Bash(rm*)", "Bash(mv*)"},
		OutputFormat: adapterDef.OutputFormat,
	}

	// Run the adapter
	a := adapter.NewClaudeAdapter()
	ctx := context.Background()
	result, err := a.Run(ctx, cfg)
	if err != nil {
		return NewCLIError(CodeInternalError,
			fmt.Sprintf("deep analysis failed: %s", err),
			"Check that the adapter binary is available and authenticated")
	}

	if result.ExitCode != 0 {
		return NewCLIError(CodeInternalError,
			fmt.Sprintf("deep analysis exited with code %d", result.ExitCode),
			"Check adapter logs for details")
	}

	// Read the output file the agent wrote
	outputPath := filepath.Join(cwd, ".agents/output/deep-analysis.json")
	outputData, err := os.ReadFile(outputPath)
	if err != nil {
		return NewCLIError(CodeInternalError,
			"deep analysis completed but output file not found at .agents/output/deep-analysis.json",
			"The agent may not have written the file — check adapter output")
	}

	var deepResult deepAnalysisResult
	if err := json.Unmarshal(outputData, &deepResult); err != nil {
		return NewCLIError(CodeInternalError,
			fmt.Sprintf("failed to parse deep analysis output: %s", err),
			"The output may not be valid JSON — check .agents/output/deep-analysis.json")
	}

	// Write enriched SKILL.md files
	fmt.Fprintf(os.Stderr, "  Writing enriched context skills...\n")
	for _, ctx := range deepResult.Contexts {
		skillDir := filepath.Join(".agents", "skills", "wave-ctx-"+ctx.Name)
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: could not create %s: %v\n", skillDir, err)
			continue
		}

		content := generateDeepSkillContent(ctx, m.Ontology.Telos)
		skillPath := filepath.Join(skillDir, "SKILL.md")
		if err := os.WriteFile(skillPath, []byte(content), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: could not write %s: %v\n", skillPath, err)
			continue
		}
		fmt.Fprintf(os.Stderr, "  ✓ %s (%d invariants, %d decisions, %d terms)\n",
			skillPath, len(ctx.Invariants), len(ctx.KeyDecisions), len(ctx.DomainVocabulary))
	}

	// Remove staleness sentinel after deep refresh
	_ = os.Remove(".agents/.ontology-stale")

	fmt.Fprintf(os.Stderr, "\n  Deep analysis complete. Review the generated skills before committing.\n")
	return nil
}

// generateDeepSkillContent creates a rich SKILL.md from deep analysis results.
func generateDeepSkillContent(ctx deepContextResult, _ string) string {
	var b strings.Builder

	b.WriteString("---\n")
	fmt.Fprintf(&b, "name: wave-ctx-%s\n", ctx.Name)
	fmt.Fprintf(&b, "description: Domain context for %s\n", ctx.Description)
	b.WriteString("---\n\n")

	// Capitalize first letter for heading
	ctxTitle := ctx.Name
	if len(ctxTitle) > 0 {
		ctxTitle = strings.ToUpper(ctxTitle[:1]) + ctxTitle[1:]
	}
	fmt.Fprintf(&b, "# %s Context\n\n", ctxTitle)
	fmt.Fprintf(&b, "%s\n\n", ctx.Description)

	if len(ctx.Invariants) > 0 {
		b.WriteString("## Invariants\n\n")
		for _, inv := range ctx.Invariants {
			fmt.Fprintf(&b, "- %s\n", inv)
		}
		b.WriteString("\n")
	}

	if len(ctx.KeyDecisions) > 0 {
		b.WriteString("## Key Decisions\n\n")
		for _, dec := range ctx.KeyDecisions {
			fmt.Fprintf(&b, "- %s\n", dec)
		}
		b.WriteString("\n")
	}

	if len(ctx.DomainVocabulary) > 0 {
		b.WriteString("## Domain Vocabulary\n\n")
		b.WriteString("| Term | Meaning |\n")
		b.WriteString("|------|--------|\n")
		for _, v := range ctx.DomainVocabulary {
			fmt.Fprintf(&b, "| %s | %s |\n", v.Term, v.Meaning)
		}
		b.WriteString("\n")
	}

	if len(ctx.NeighboringContexts) > 0 {
		b.WriteString("## Neighboring Contexts\n\n")
		for _, nc := range ctx.NeighboringContexts {
			fmt.Fprintf(&b, "- **%s**\n", nc)
		}
		b.WriteString("\n")
	}

	if len(ctx.KeyFiles) > 0 {
		b.WriteString("## Key Files\n\n")
		for _, f := range ctx.KeyFiles {
			fmt.Fprintf(&b, "- `%s`\n", f)
		}
		b.WriteString("\n")
	}

	return b.String()
}

// runAnalyzeDecisions shows orchestration decision provenance as a table.
func runAnalyzeDecisions(cmd *cobra.Command) error {
	store, err := state.NewStateStore(".agents/state.db")
	if err != nil {
		return NewCLIError(CodeInternalError,
			fmt.Sprintf("failed to open state database: %s", err),
			"Check that .agents/state.db exists")
	}
	defer store.Close()

	summaries, err := store.ListOrchestrationDecisionSummary(50)
	if err != nil {
		return NewCLIError(CodeInternalError,
			fmt.Sprintf("failed to query decisions: %s", err),
			"Check state database integrity")
	}

	f := display.NewFormatter()

	if len(summaries) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "\n  No orchestration decisions recorded yet.\n")
		fmt.Fprintf(cmd.OutOrStdout(), "  %s Run 'wave do <task>' to create orchestrated pipeline runs.\n\n", f.Muted("Hint:"))
		return nil
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\n  %s\n\n", f.Colorize("Orchestration Decisions:", "\033[1;37m"))

	// Header
	fmt.Fprintf(cmd.OutOrStdout(), "  %-14s %-16s %-24s %6s %8s %10s %12s\n",
		f.Muted("DOMAIN"), f.Muted("COMPLEXITY"), f.Muted("PIPELINE"),
		f.Muted("TOTAL"), f.Muted("SUCCESS"), f.Muted("AVG TOKENS"), f.Muted("AVG DURATION"))
	fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", f.Muted(strings.Repeat("─", 96)))

	for _, s := range summaries {
		successStr := fmt.Sprintf("%.0f%%", s.SuccessRate)
		durationStr := formatDurationMs(s.AvgDurationMs)
		fmt.Fprintf(cmd.OutOrStdout(), "  %-14s %-16s %-24s %6d %8s %10d %12s\n",
			s.Domain, s.Complexity, s.PipelineName,
			s.Total, successStr, s.AvgTokens, durationStr)
	}
	fmt.Fprintln(cmd.OutOrStdout())
	return nil
}
