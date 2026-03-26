package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/recinq/wave/internal/display"
	"github.com/recinq/wave/internal/doctor"
	"github.com/recinq/wave/internal/manifest"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// AnalyzeResult holds the output of the analyze command.
type AnalyzeResult struct {
	Telos          string                  `json:"telos,omitempty"`
	Contexts       []AnalyzeContext        `json:"contexts"`
	Conventions    map[string]string       `json:"conventions,omitempty"`
	SkillsWritten  []string                `json:"skills_written,omitempty"`
	Profile        *doctor.ProjectProfile  `json:"profile,omitempty"`
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

	cmd := &cobra.Command{
		Use:   "analyze",
		Short: "Analyze project domain and generate ontology context skills",
		Long: `Scan the project structure and generate ontology context skills
(.wave/skills/wave-ctx-*/SKILL.md) from the declared bounded contexts
in wave.yaml.

By default, performs a deterministic scan using directory structure and
package detection. Use --deep for AI-assisted analysis (not yet implemented).
Use --evolve to propose ontology updates based on pipeline run history.`,
		Example: `  wave analyze
  wave analyze --json
  wave analyze --deep
  wave analyze --evolve`,
		Args: cobra.NoArgs,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if deepFlag {
				return NewCLIError(CodeInvalidArgs,
					"--deep requires an adapter and is not yet implemented",
					"Use deterministic scan (no flags) for now")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			return runAnalyze(cmd, evolveFlag)
		},
	}

	cmd.Flags().BoolVar(&deepFlag, "deep", false, "Use AI-assisted analysis (requires adapter)")
	cmd.Flags().BoolVar(&evolveFlag, "evolve", false, "Propose updates based on pipeline run history")

	return cmd
}

func runAnalyze(cmd *cobra.Command, evolve bool) error {
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
			"Check write permissions for .wave/skills/").WithCause(writeErr)
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

	// Suggest updating wave.yaml if new contexts were proposed
	if len(proposedContexts) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "\n  %s New contexts detected from project structure.\n", f.Muted("Tip:"))
		fmt.Fprintf(cmd.OutOrStdout(), "  Add them to wave.yaml under ontology.contexts, or re-run with --deep for AI enrichment.\n")
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

	// Look for top-level packages that could be contexts
	var proposed []AnalyzeContext

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
			if existing[name] || strings.HasPrefix(name, ".") {
				continue
			}
			// Count files to determine if this is a significant package
			count := countFiles(filepath.Join(dir, name))
			if count >= 3 { // Only propose packages with 3+ files
				proposed = append(proposed, AnalyzeContext{
					Name:      name,
					Packages:  []string{filepath.Join(dir, name)},
					FileCount: count,
				})
			}
		}
	}

	return proposed
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

// writeContextSkills generates SKILL.md files for declared ontology contexts.
func writeContextSkills(result *AnalyzeResult) ([]string, error) {
	var written []string

	for _, ctx := range result.Contexts {
		skillDir := filepath.Join(".wave", "skills", "wave-ctx-"+ctx.Name)
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

// updateManifestOntology writes proposed ontology changes back to wave.yaml.
// This is used when new contexts are detected that should be added to the manifest.
func updateManifestOntology(manifestPath string, ontology *manifest.Ontology) error {
	rawData, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}

	var raw map[string]interface{}
	if err := yaml.Unmarshal(rawData, &raw); err != nil {
		return err
	}

	// Marshal ontology to generic map
	ontologyBytes, err := yaml.Marshal(ontology)
	if err != nil {
		return err
	}
	var ontologyMap map[string]interface{}
	if err := yaml.Unmarshal(ontologyBytes, &ontologyMap); err != nil {
		return err
	}

	raw["ontology"] = ontologyMap

	outData, err := yaml.Marshal(raw)
	if err != nil {
		return err
	}

	return os.WriteFile(manifestPath, outData, 0o644)
}
