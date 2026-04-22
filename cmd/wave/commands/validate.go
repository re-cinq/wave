package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/recinq/wave/internal/forge"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/pipeline"
	"github.com/spf13/cobra"
)

type ValidateOptions struct {
	ManifestPath    string
	Pipeline        string
	All             bool
	Verbose         bool
	PromptToolsWarn bool // Downgrade prompt/tool permission mismatches to warnings.
}

func NewValidateCmd() *cobra.Command {
	var opts ValidateOptions

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate Wave configuration",
		Long: `Validate the wave.yaml manifest and project structure.
Checks manifest syntax, references, and system dependencies.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Verbose, _ = cmd.Root().PersistentFlags().GetBool("verbose")
			return runValidate(opts)
		},
	}

	cmd.Flags().StringVar(&opts.ManifestPath, "manifest", "wave.yaml", "Path to manifest file")
	cmd.Flags().StringVar(&opts.Pipeline, "pipeline", "", "Specific pipeline to validate")
	cmd.Flags().BoolVar(&opts.All, "all", false, "Validate all pipelines in .agents/pipelines/")
	cmd.Flags().BoolVar(&opts.PromptToolsWarn, "prompt-tools-warn", false,
		"Downgrade prompt/tool permission mismatches to warnings (honours WAVE_PROMPT_TOOLS_WARN env)")

	return cmd
}

func runValidate(opts ValidateOptions) error {
	if opts.Verbose {
		fmt.Printf("Validating manifest: %s\n", opts.ManifestPath)
	}

	mp, err := loadManifestStrict(opts.ManifestPath)
	if err != nil {
		return err
	}
	m := *mp

	if opts.Verbose {
		fmt.Printf("✓ Manifest syntax is valid\n")
	}

	// Validate adapter references in personas
	for name, persona := range m.Personas {
		if persona.Adapter != "" && m.GetAdapter(persona.Adapter) == nil {
			availableAdapters := make([]string, 0, len(m.Adapters))
			for adapterName := range m.Adapters {
				availableAdapters = append(availableAdapters, adapterName)
			}
			fmt.Printf("✗ Persona '%s' references unknown adapter '%s'\n", name, persona.Adapter)
			if len(availableAdapters) > 0 {
				fmt.Printf("  Available adapters: %v\n", availableAdapters)
			}
			return NewCLIError(CodeValidationFailed, fmt.Sprintf("manifest validation failed: persona '%s' references unknown adapter '%s'", name, persona.Adapter), "Update the persona's adapter field to reference a defined adapter")
		}
	}

	if errs := validateManifestStructure(&m); len(errs) > 0 {
		fmt.Printf("✗ Manifest validation failed:\n")
		for _, err := range errs {
			fmt.Printf("  - %s\n", err)
		}
		return NewCLIError(CodeValidationFailed, "manifest validation failed", "Fix the issues listed above and re-run 'wave validate'")
	}

	if opts.Verbose {
		fmt.Printf("✓ Manifest structure is valid\n")
	}

	if errs := validateSystemReferences(&m, opts.ManifestPath); len(errs) > 0 {
		fmt.Printf("✗ System reference validation failed:\n")
		for _, err := range errs {
			fmt.Printf("  - %s\n", err)
		}
		return NewCLIError(CodeValidationFailed, "system reference validation failed", "Create missing files or update references in wave.yaml")
	}

	if opts.Verbose {
		fmt.Printf("✓ System references are valid\n")
	}

	adapterWarnings := validateAdapterBinaries(&m, opts.Verbose)
	if len(adapterWarnings) > 0 {
		for _, warn := range adapterWarnings {
			fmt.Printf("⚠ Warning: %s\n", warn)
		}
	}

	if opts.Verbose {
		fmt.Printf("✓ Adapter configuration checked\n")
		// Print summary in verbose mode
		fmt.Printf("\nSummary:\n")
		fmt.Printf("  Adapters:  %d defined\n", len(m.Adapters))
		fmt.Printf("  Personas:  %d defined\n", len(m.Personas))
		fmt.Printf("\n")
	}

	// Detect forge for template resolution
	forgeInfo, _ := forge.DetectFromGitRemotes()

	warnPromptTools := promptToolWarnEnabled(opts)

	if opts.All {
		pipelineDir := filepath.Join(filepath.Dir(opts.ManifestPath), ".agents", "pipelines")
		entries, err := os.ReadDir(pipelineDir)
		if err != nil {
			return NewCLIError(CodeInternalError, fmt.Sprintf("failed to read pipeline directory: %s", err), "Run 'wave init' to create pipeline directory").WithCause(err)
		}
		var allErrs []string
		var allPromptFindings []promptToolFinding
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
				continue
			}
			name := strings.TrimSuffix(entry.Name(), ".yaml")
			structErrs, findings := validatePipelineWithPromptTools(name, &m, forgeInfo)
			if len(structErrs) > 0 {
				for _, e := range structErrs {
					allErrs = append(allErrs, fmt.Sprintf("%s: %s", name, e))
				}
			} else if opts.Verbose && len(findings) == 0 {
				fmt.Printf("✓ Pipeline '%s' is valid\n", name)
			}
			allPromptFindings = append(allPromptFindings, findings...)
		}
		if len(allPromptFindings) > 0 {
			label := "✗ Pipeline prompt/tool mismatches:"
			if warnPromptTools {
				label = "⚠ Pipeline prompt/tool mismatches (warning mode):"
			}
			fmt.Println(label)
			for _, f := range allPromptFindings {
				fmt.Printf("  - %s\n", f)
			}
			if !warnPromptTools {
				return NewCLIError(CodeValidationFailed,
					fmt.Sprintf("%d prompt/tool mismatch(es) found across pipelines", len(allPromptFindings)),
					"Either grant the persona the missing tool, change the prompt, or re-run with --prompt-tools-warn to downgrade to warnings")
			}
		}
		if len(allErrs) > 0 {
			fmt.Printf("✗ Pipeline validation failed:\n")
			for _, e := range allErrs {
				fmt.Printf("  - %s\n", e)
			}
			return NewCLIError(CodeValidationFailed, fmt.Sprintf("%d pipeline issue(s) found", len(allErrs)), "Fix the issues listed above and re-run 'wave validate --all'")
		}
	} else if opts.Pipeline != "" {
		structErrs, findings := validatePipelineWithPromptTools(opts.Pipeline, &m, forgeInfo)
		if len(findings) > 0 {
			label := "✗ Pipeline '%s' prompt/tool mismatches:\n"
			if warnPromptTools {
				label = "⚠ Pipeline '%s' prompt/tool mismatches (warning mode):\n"
			}
			fmt.Printf(label, opts.Pipeline)
			for _, f := range findings {
				fmt.Printf("  - %s\n", f)
			}
			if !warnPromptTools {
				return NewCLIError(CodeValidationFailed,
					fmt.Sprintf("pipeline '%s' has %d prompt/tool mismatch(es)", opts.Pipeline, len(findings)),
					"Either grant the persona the missing tool, change the prompt, or re-run with --prompt-tools-warn to downgrade to warnings")
			}
		}
		if len(structErrs) > 0 {
			fmt.Printf("✗ Pipeline '%s' validation failed:\n", opts.Pipeline)
			for _, e := range structErrs {
				fmt.Printf("  - %s\n", e)
			}
			return NewCLIError(CodeValidationFailed, fmt.Sprintf("pipeline '%s' validation failed: %s", opts.Pipeline, strings.Join(structErrs, "; ")), "Fix the pipeline definition and re-run validation")
		}
		if opts.Verbose && len(findings) == 0 {
			fmt.Printf("✓ Pipeline '%s' is valid\n", opts.Pipeline)
		}
	}

	fmt.Printf("✓ Validation successful\n")
	return nil
}

// validatePipelineWithPromptTools runs both the structural pipeline validator
// and the prompt/tool permission check in a single pipeline file read. It
// returns the structural error list (which is fatal) and the prompt/tool
// findings (which the caller renders separately, optionally as warnings).
func validatePipelineWithPromptTools(pipelineName string, m *manifest.Manifest, fi forge.ForgeInfo) ([]string, []promptToolFinding) {
	structErrs := validatePipelineFull(pipelineName, m, fi)
	pipelinePath := filepath.Join(".agents", "pipelines", pipelineName+".yaml")
	pipelineData, err := os.ReadFile(pipelinePath)
	if err != nil {
		// Structural validator already reports the read failure; nothing to scan.
		return structErrs, nil
	}
	loader := &pipeline.YAMLPipelineLoader{}
	pParsed, err := loader.Unmarshal(pipelineData)
	if err != nil {
		// Same — YAML errors surfaced by the structural pass.
		return structErrs, nil
	}
	findings := validatePromptToolPermissions(pipelineName, pParsed, m)
	return structErrs, findings
}

func validateManifestStructure(m *manifest.Manifest) []string {
	var errs []string

	if m.APIVersion == "" {
		errs = append(errs, "apiVersion is required")
	}

	if m.Kind != "WaveManifest" && m.Kind != "Wave" {
		errs = append(errs, fmt.Sprintf("kind must be 'WaveManifest', got '%s'", m.Kind))
	}

	if m.Metadata.Name == "" {
		errs = append(errs, "metadata.name is required")
	}

	if m.Runtime.WorkspaceRoot == "" {
		errs = append(errs, "runtime.workspace_root is required")
	}

	for name, adapter := range m.Adapters {
		if adapter.Binary == "" {
			errs = append(errs, fmt.Sprintf("adapters.%s.binary is required", name))
		}
		if adapter.Mode == "" {
			errs = append(errs, fmt.Sprintf("adapters.%s.mode is required", name))
		}
	}

	for name, persona := range m.Personas {
		if persona.Adapter == "" {
			errs = append(errs, fmt.Sprintf("personas.%s.adapter is required", name))
		}
		if persona.SystemPromptFile == "" {
			errs = append(errs, fmt.Sprintf("personas.%s.system_prompt_file is required", name))
		}
	}

	return errs
}

func validateSystemReferences(m *manifest.Manifest, manifestPath string) []string {
	var errs []string
	manifestDir := filepath.Dir(manifestPath)

	for name, persona := range m.Personas {
		promptPath := persona.GetSystemPromptPath(manifestDir)
		if _, err := os.Stat(promptPath); os.IsNotExist(err) {
			errs = append(errs, fmt.Sprintf("personas.%s.system_prompt_file '%s' does not exist", name, promptPath))
		}
	}

	return errs
}

func validateAdapterBinaries(m *manifest.Manifest, verbose bool) []string {
	var warnings []string

	for name, adapter := range m.Adapters {
		binaryPath, err := exec.LookPath(adapter.Binary)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("adapter '%s' binary '%s' not found in PATH", name, adapter.Binary))
		} else if verbose {
			fmt.Printf("  Adapter '%s': binary found at %s\n", name, binaryPath)
		}
	}

	return warnings
}

// isCompositionStep returns true if the step is an orchestration primitive
// (sub-pipeline, branch, gate, loop) that does not require a persona.
func isCompositionStep(step pipeline.Step) bool {
	if step.Type == pipeline.StepTypeCommand || step.Type == pipeline.StepTypeConditional {
		return true
	}
	return step.SubPipeline != "" || step.Branch != nil || step.Gate != nil || step.Loop != nil || step.Aggregate != nil
}

// resolveForgeTemplate expands {{ forge.type }} in a persona name using the detected forge.
// Returns all possible expansions if forge is unknown.
func resolveForgeTemplate(persona string, fi forge.ForgeInfo) []string {
	if !strings.Contains(persona, "{{ forge.type }}") && !strings.Contains(persona, "{{forge.type}}") {
		return []string{persona}
	}
	if fi.Type != forge.ForgeUnknown {
		resolved := strings.ReplaceAll(persona, "{{ forge.type }}", string(fi.Type))
		resolved = strings.ReplaceAll(resolved, "{{forge.type}}", string(fi.Type))
		return []string{resolved}
	}
	// Unknown forge: expand to all variants
	var results []string
	for _, ft := range []string{"github", "gitlab", "gitea", "bitbucket"} {
		resolved := strings.ReplaceAll(persona, "{{ forge.type }}", ft)
		resolved = strings.ReplaceAll(resolved, "{{forge.type}}", ft)
		results = append(results, resolved)
	}
	return results
}

// validatePipelineFull performs comprehensive validation of a pipeline against the manifest.
// Returns a list of error strings (empty = valid).
func validatePipelineFull(pipelineName string, m *manifest.Manifest, fi forge.ForgeInfo) []string {
	pipelinePath := filepath.Join(".agents", "pipelines", pipelineName+".yaml")
	pipelineData, err := os.ReadFile(pipelinePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{fmt.Sprintf("pipeline file does not exist: %s", pipelinePath)}
		}
		return []string{fmt.Sprintf("cannot read pipeline file: %s", err)}
	}

	loader := &pipeline.YAMLPipelineLoader{}
	pParsed, err := loader.Unmarshal(pipelineData)
	if err != nil {
		return []string{fmt.Sprintf("invalid YAML: %s", err)}
	}
	p := *pParsed

	var errs []string

	// First pass: collect all step IDs and check for duplicates.
	// This must happen before dependency validation so that YAML ordering
	// does not produce false positives (the executor topologically sorts at runtime).
	stepIDs := make(map[string]bool)
	for i, step := range p.Steps {
		if stepIDs[step.ID] {
			errs = append(errs, fmt.Sprintf("step[%d] duplicate id '%s'", i, step.ID))
		}
		stepIDs[step.ID] = true
	}

	// Second pass: validate references now that all step IDs are known.
	for _, step := range p.Steps {
		// Persona validation (skip composition steps)
		if !isCompositionStep(step) {
			if step.Persona == "" {
				errs = append(errs, fmt.Sprintf("step '%s' has no persona (and is not a composition step)", step.ID))
			} else {
				// Resolve forge templates and check at least one variant exists in manifest
				candidates := resolveForgeTemplate(step.Persona, fi)
				found := false
				for _, c := range candidates {
					if m.GetPersona(c) != nil {
						found = true
						break
					}
				}
				if !found {
					errs = append(errs, fmt.Sprintf("step '%s' persona '%s' not found in manifest", step.ID, step.Persona))
				}
			}
		}

		// Sub-pipeline existence
		if step.SubPipeline != "" && !strings.Contains(step.SubPipeline, "{{") {
			subPath := filepath.Join(".agents", "pipelines", step.SubPipeline+".yaml")
			if _, err := os.Stat(subPath); os.IsNotExist(err) {
				errs = append(errs, fmt.Sprintf("step '%s' references sub-pipeline '%s' which does not exist", step.ID, step.SubPipeline))
			}
		}

		// Contract schema file existence
		if sp := step.Handover.Contract.SchemaPath; sp != "" && !strings.Contains(sp, "{{") {
			if _, err := os.Stat(sp); os.IsNotExist(err) {
				errs = append(errs, fmt.Sprintf("step '%s' references contract schema '%s' which does not exist", step.ID, sp))
			}
		}

		// Dependency validation
		for _, dep := range step.Dependencies {
			if !stepIDs[dep] {
				errs = append(errs, fmt.Sprintf("step '%s' depends on non-existent step '%s'", step.ID, dep))
			}
		}

		// Prompt file existence
		if sp := step.Exec.SourcePath; sp != "" && !strings.Contains(sp, "{{") {
			if _, err := os.Stat(sp); os.IsNotExist(err) {
				errs = append(errs, fmt.Sprintf("step '%s' references prompt file '%s' which does not exist", step.ID, sp))
			}
		}
	}

	return errs
}
