package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/recinq/wave/internal/manifest"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type ValidateOptions struct {
	ManifestPath string
	Pipeline     string
	Verbose      bool
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

	return cmd
}

func runValidate(opts ValidateOptions) error {
	if opts.Verbose {
		fmt.Printf("Validating manifest: %s\n", opts.ManifestPath)
	}

	manifestData, err := os.ReadFile(opts.ManifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return NewCLIError(CodeManifestMissing, fmt.Sprintf("failed to read manifest: %s", err), "Run 'wave init' to create a new Wave project")
		}
		return NewCLIError(CodeManifestMissing, fmt.Sprintf("failed to read manifest: %s", err), "Check file permissions and path")
	}

	var m manifest.Manifest
	if err := yaml.Unmarshal(manifestData, &m); err != nil {
		return NewCLIError(CodeManifestInvalid, fmt.Sprintf("failed to parse manifest YAML: %s", err), "Check for syntax errors like incorrect indentation or invalid characters")
	}

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

	if opts.Pipeline != "" {
		if err := validatePipeline(opts.Pipeline, &m); err != nil {
			return NewCLIError(CodeValidationFailed, fmt.Sprintf("pipeline validation failed: %s", err), "Fix the pipeline definition and re-run validation")
		}
		if opts.Verbose {
			fmt.Printf("✓ Pipeline '%s' is valid\n", opts.Pipeline)
		}
	}

	fmt.Printf("✓ Validation successful\n")
	return nil
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

func validatePipeline(pipelineName string, m *manifest.Manifest) error {
	pipelinePath := filepath.Join(".wave", "pipelines", pipelineName+".yaml")
	if _, err := os.Stat(pipelinePath); os.IsNotExist(err) {
		return NewCLIError(CodePipelineNotFound, fmt.Sprintf("pipeline file '%s' does not exist", pipelinePath), "Create the pipeline file or check the pipeline name")
	}

	pipelineData, err := os.ReadFile(pipelinePath)
	if err != nil {
		return NewCLIError(CodeInternalError, fmt.Sprintf("failed to read pipeline file: %s", err), "Check file permissions")
	}

	var pipeline map[string]interface{}
	if err := yaml.Unmarshal(pipelineData, &pipeline); err != nil {
		return NewCLIError(CodeValidationFailed, fmt.Sprintf("failed to parse pipeline: %s", err), "Check pipeline YAML syntax")
	}

	if pipeline["kind"] != "WavePipeline" {
		return NewCLIError(CodeValidationFailed, "invalid pipeline kind, expected 'WavePipeline'", "Set kind: WavePipeline in the pipeline file")
	}

	steps, ok := pipeline["steps"].([]interface{})
	if !ok {
		return NewCLIError(CodeValidationFailed, "pipeline must have steps", "Add at least one step to the pipeline")
	}

	stepIDs := make(map[string]bool)
	for i, stepInterface := range steps {
		step, ok := stepInterface.(map[string]interface{})
		if !ok {
			return NewCLIError(CodeValidationFailed, fmt.Sprintf("step[%d] must be an object", i), "Ensure each step is a YAML mapping with id, persona, etc.")
		}

		stepID, ok := step["id"].(string)
		if !ok || stepID == "" {
			return NewCLIError(CodeValidationFailed, fmt.Sprintf("step[%d].id is required", i), "Add an id field to each step")
		}

		if stepIDs[stepID] {
			return NewCLIError(CodeValidationFailed, fmt.Sprintf("duplicate step id: %s", stepID), "Rename one of the duplicate steps")
		}
		stepIDs[stepID] = true

		// Composition steps use `pipeline:` instead of `persona:`
		_, hasSubPipeline := step["pipeline"].(string)

		persona, hasPersona := step["persona"].(string)
		if !hasSubPipeline && (!hasPersona || persona == "") {
			return NewCLIError(CodeValidationFailed, fmt.Sprintf("step[%d].persona is required (or use pipeline: for composition steps)", i), "Add a persona field or pipeline field to the step")
		}

		if hasPersona && persona != "" && m.GetPersona(persona) == nil {
			return NewCLIError(CodeValidationFailed, fmt.Sprintf("step[%d].persona '%s' not found in manifest", i, persona), "Add the persona to wave.yaml or use an existing persona")
		}

		if deps, ok := step["dependencies"].([]interface{}); ok {
			for _, depInterface := range deps {
				dep, ok := depInterface.(string)
				if !ok || dep == "" {
					return NewCLIError(CodeValidationFailed, fmt.Sprintf("step[%d] has invalid dependency", i), "Ensure all dependency values are valid step ID strings")
				}
				if !stepIDs[dep] {
					return NewCLIError(CodeValidationFailed, fmt.Sprintf("step[%d] depends on non-existent step '%s'", i, dep), "Check that the dependency step ID matches an earlier step")
				}
			}
		}
	}

	return nil
}
