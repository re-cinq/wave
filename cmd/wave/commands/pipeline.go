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

// NewPipelineCmd creates the pipeline parent command with subcommands.
func NewPipelineCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pipeline",
		Short: "Pipeline management commands",
		Long: `Create and manage Wave pipelines.

Subcommands:
  create    Scaffold a new pipeline from a built-in template
  list      List available pipelines (alias for wave list pipelines)`,
	}

	cmd.AddCommand(newPipelineCreateCmd())
	cmd.AddCommand(newPipelineListCmd())

	return cmd
}

func newPipelineCreateCmd() *cobra.Command {
	var name string
	var template string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Scaffold a new pipeline from a template",
		Long: `Create a new pipeline YAML in .agents/pipelines/<name>.yaml based on
an existing built-in pipeline template.

If --template is omitted, lists available templates grouped by category.`,
		Example: `  wave pipeline create --name my-pipeline --template impl-issue
  wave pipeline create --template impl-issue --name custom-impl
  wave pipeline create   # lists available templates`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPipelineCreate(name, template)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Name for the new pipeline")
	cmd.Flags().StringVar(&template, "template", "", "Built-in pipeline template to use")

	return cmd
}

func newPipelineListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available pipelines (alias for wave list pipelines)",
		RunE: func(cmd *cobra.Command, args []string) error {
			names := defaults.PipelineNames()
			sort.Strings(names)
			for _, n := range names {
				fmt.Println(strings.TrimSuffix(n, ".yaml"))
			}
			return nil
		},
	}
}

func runPipelineCreate(name, template string) error {
	// If no template specified, list available templates grouped by category
	if template == "" {
		return listPipelineTemplates()
	}

	if name == "" {
		return NewCLIError(CodeInvalidArgs, "--name is required when --template is specified", "Usage: wave pipeline create --name <name> --template <template>")
	}

	// Validate name: no path traversal, no path separators
	if strings.Contains(name, "..") || filepath.IsAbs(name) || strings.ContainsAny(name, `/\`) {
		return NewCLIError(CodeSecurityViolation, fmt.Sprintf("invalid pipeline name: %s", name), "Pipeline names must not contain path separators or '..' sequences")
	}

	// Load all pipeline templates from embedded FS
	pipelines, err := defaults.GetPipelines()
	if err != nil {
		return NewCLIError(CodeInternalError, fmt.Sprintf("failed to load pipeline templates: %v", err), "This is a bug in Wave")
	}

	// Look up the template (key is filename like "impl-issue.yaml")
	templateFile := template + ".yaml"
	content, ok := pipelines[templateFile]
	if !ok {
		return NewCLIError(CodePipelineNotFound, fmt.Sprintf("template %q not found", template), "Run 'wave pipeline create' without --template to see available templates")
	}

	// Check output path doesn't already exist
	outputPath := filepath.Join(".agents", "pipelines", name+".yaml")
	if _, err := os.Stat(outputPath); err == nil {
		return NewCLIError(CodeInvalidArgs, fmt.Sprintf("pipeline %q already exists at %s", name, outputPath), "Choose a different name or remove the existing file")
	}

	// Parse the YAML, update metadata.name, re-marshal
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(content), &doc); err != nil {
		return NewCLIError(CodeInternalError, fmt.Sprintf("failed to parse template %q: %v", template, err), "This is a bug in Wave")
	}

	updateMetadataName(&doc, name)

	out, err := yaml.Marshal(&doc)
	if err != nil {
		return NewCLIError(CodeInternalError, fmt.Sprintf("failed to marshal pipeline: %v", err), "This is a bug in Wave")
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return NewCLIError(CodeInternalError, fmt.Sprintf("failed to create directory: %v", err), "Check filesystem permissions")
	}

	if err := os.WriteFile(outputPath, out, 0644); err != nil {
		return NewCLIError(CodeInternalError, fmt.Sprintf("failed to write pipeline: %v", err), "Check filesystem permissions")
	}

	fmt.Fprintf(os.Stderr, "Created pipeline %q at %s (from template %q)\n", name, outputPath, template)
	return nil
}

// updateMetadataName finds metadata.name in a yaml.Node tree and updates its value.
func updateMetadataName(doc *yaml.Node, newName string) {
	if doc == nil {
		return
	}
	// The document node wraps the actual content
	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		updateMetadataName(doc.Content[0], newName)
		return
	}
	if doc.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i < len(doc.Content)-1; i += 2 {
		key := doc.Content[i]
		val := doc.Content[i+1]
		if key.Value == "metadata" && val.Kind == yaml.MappingNode {
			for j := 0; j < len(val.Content)-1; j += 2 {
				mKey := val.Content[j]
				mVal := val.Content[j+1]
				if mKey.Value == "name" {
					mVal.Value = newName
					return
				}
			}
		}
	}
}

// listPipelineTemplates prints available templates grouped by category prefix.
func listPipelineTemplates() error {
	pipelines, err := defaults.GetPipelines()
	if err != nil {
		return NewCLIError(CodeInternalError, fmt.Sprintf("failed to load pipelines: %v", err), "This is a bug in Wave")
	}

	// Group by prefix
	groups := make(map[string][]string)
	for filename := range pipelines {
		name := strings.TrimSuffix(filename, ".yaml")
		prefix := categoryPrefix(name)
		groups[prefix] = append(groups[prefix], name)
	}

	// Sort group keys
	prefixes := make([]string, 0, len(groups))
	for p := range groups {
		prefixes = append(prefixes, p)
	}
	sort.Strings(prefixes)

	fmt.Fprintf(os.Stderr, "Available pipeline templates:\n\n")
	for _, prefix := range prefixes {
		names := groups[prefix]
		sort.Strings(names)
		fmt.Fprintf(os.Stderr, "  %s:\n", prefix)
		for _, n := range names {
			fmt.Fprintf(os.Stderr, "    %s\n", n)
		}
		fmt.Fprintln(os.Stderr)
	}

	fmt.Fprintf(os.Stderr, "Usage: wave pipeline create --name <name> --template <template>\n")
	return nil
}

// categoryPrefix extracts the category prefix from a pipeline name (e.g. "impl" from "impl-issue").
func categoryPrefix(name string) string {
	knownPrefixes := []string{"impl", "plan", "ops", "audit", "doc", "test", "wave", "bench"}
	for _, p := range knownPrefixes {
		if strings.HasPrefix(name, p+"-") || name == p {
			return p
		}
	}
	return "other"
}
