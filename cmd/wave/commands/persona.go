package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/recinq/wave/internal/defaults"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// NewPersonaCmd creates the persona parent command with subcommands.
func NewPersonaCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "persona",
		Short: "Persona management commands",
		Long: `Create and manage Wave personas.

Subcommands:
  create    Scaffold a new persona from a built-in template
  list      List available personas`,
	}

	cmd.AddCommand(newPersonaCreateCmd())
	cmd.AddCommand(newPersonaListCmd())

	return cmd
}

func newPersonaCreateCmd() *cobra.Command {
	var name string
	var template string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Scaffold a new persona from a template",
		Long: `Create a new persona in .agents/personas/ based on an existing built-in
persona template. This creates both the .md system prompt and .yaml config file.

If --template is omitted, lists available persona templates.`,
		Example: `  wave persona create --name my-reviewer --template reviewer
  wave persona create --template implementer --name fast-impl
  wave persona create   # lists available templates`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPersonaCreate(name, template)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Name for the new persona")
	cmd.Flags().StringVar(&template, "template", "", "Built-in persona template to use")

	return cmd
}

func newPersonaListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available persona templates",
		RunE: func(cmd *cobra.Command, args []string) error {
			names := defaults.PersonaNames()
			// Deduplicate: PersonaNames returns .md filenames; strip extension
			seen := make(map[string]bool)
			var sorted []string
			for _, n := range names {
				base := strings.TrimSuffix(n, ".md")
				if !seen[base] {
					seen[base] = true
					sorted = append(sorted, base)
				}
			}
			sort.Strings(sorted)
			for _, n := range sorted {
				fmt.Println(n)
			}
			return nil
		},
	}
}

func runPersonaCreate(name, template string) error {
	// If no template specified, list available persona templates
	if template == "" {
		return listPersonaTemplates()
	}

	if name == "" {
		return NewCLIError(CodeInvalidArgs, "--name is required when --template is specified", "Usage: wave persona create --name <name> --template <template>")
	}

	// Validate name: no path traversal, no path separators
	if strings.Contains(name, "..") || filepath.IsAbs(name) || strings.ContainsAny(name, `/\`) {
		return NewCLIError(CodeSecurityViolation, fmt.Sprintf("invalid persona name: %s", name), "Persona names must not contain path separators or '..' sequences")
	}

	// Load persona system prompts (.md files)
	personas, err := defaults.GetPersonas()
	if err != nil {
		return NewCLIError(CodeInternalError, fmt.Sprintf("failed to load persona templates: %v", err), "This is a bug in Wave")
	}

	// Look up the template .md
	templateMDFile := template + ".md"
	mdContent, ok := personas[templateMDFile]
	if !ok {
		return NewCLIError(CodeInvalidArgs, fmt.Sprintf("persona template %q not found", template), "Run 'wave persona create' without --template to see available templates")
	}

	// Load persona configs (.yaml files)
	personaConfigs, err := defaults.GetPersonaConfigs()
	if err != nil {
		return NewCLIError(CodeInternalError, fmt.Sprintf("failed to load persona configs: %v", err), "This is a bug in Wave")
	}

	config, hasConfig := personaConfigs[template]

	// Check output paths don't already exist
	personaDir := filepath.Join(".agents", "personas")
	mdPath := filepath.Join(personaDir, name+".md")
	yamlPath := filepath.Join(personaDir, name+".yaml")

	if _, err := os.Stat(mdPath); err == nil {
		return NewCLIError(CodeInvalidArgs, fmt.Sprintf("persona %q already exists at %s", name, mdPath), "Choose a different name or remove the existing files")
	}
	if _, err := os.Stat(yamlPath); err == nil {
		return NewCLIError(CodeInvalidArgs, fmt.Sprintf("persona config %q already exists at %s", name, yamlPath), "Choose a different name or remove the existing files")
	}

	// Ensure directory exists
	if err := os.MkdirAll(personaDir, 0755); err != nil {
		return NewCLIError(CodeInternalError, fmt.Sprintf("failed to create directory: %v", err), "Check filesystem permissions")
	}

	// Write the .md system prompt
	if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
		return NewCLIError(CodeInternalError, fmt.Sprintf("failed to write persona prompt: %v", err), "Check filesystem permissions")
	}

	// Write the .yaml config (update description to reference new name)
	if hasConfig {
		configData, err := yaml.Marshal(&config)
		if err != nil {
			return NewCLIError(CodeInternalError, fmt.Sprintf("failed to marshal persona config: %v", err), "This is a bug in Wave")
		}
		if err := os.WriteFile(yamlPath, configData, 0644); err != nil {
			// Clean up the .md we already wrote
			_ = os.Remove(mdPath)
			return NewCLIError(CodeInternalError, fmt.Sprintf("failed to write persona config: %v", err), "Check filesystem permissions")
		}
		fmt.Fprintf(os.Stderr, "Created persona %q at:\n  %s\n  %s\n(from template %q)\n", name, mdPath, yamlPath, template)
	} else {
		fmt.Fprintf(os.Stderr, "Created persona %q at:\n  %s\n(from template %q, no config template found)\n", name, mdPath, template)
	}

	return nil
}

// listPersonaTemplates prints available persona templates.
func listPersonaTemplates() error {
	personas, err := defaults.GetPersonas()
	if err != nil {
		return NewCLIError(CodeInternalError, fmt.Sprintf("failed to load personas: %v", err), "This is a bug in Wave")
	}

	var names []string
	for filename := range personas {
		names = append(names, strings.TrimSuffix(filename, ".md"))
	}
	sort.Strings(names)

	fmt.Fprintf(os.Stderr, "Available persona templates:\n\n")
	for _, n := range names {
		fmt.Fprintf(os.Stderr, "  %s\n", n)
	}
	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "Usage: wave persona create --name <name> --template <template>\n")
	return nil
}
