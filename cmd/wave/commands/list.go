package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type ListOptions struct {
	Manifest string
	Format   string
}

func NewListCmd() *cobra.Command {
	var opts ListOptions

	cmd := &cobra.Command{
		Use:   "list [pipelines|personas|adapters]",
		Short: "List pipelines and personas",
		Long: `List available pipelines, personas, and their configurations.
Shows pipeline steps, persona bindings, and execution status.

Subcommands:
  pipelines   List available pipelines
  personas    List configured personas
  adapters    List configured adapters

With no arguments, lists everything.`,
		ValidArgs: []string{"pipelines", "personas", "adapters"},
		RunE: func(cmd *cobra.Command, args []string) error {
			filter := ""
			if len(args) > 0 {
				filter = args[0]
			}
			return runList(opts, filter)
		},
	}

	cmd.Flags().StringVar(&opts.Manifest, "manifest", "wave.yaml", "Path to manifest file")
	cmd.Flags().StringVar(&opts.Format, "format", "table", "Output format (table, json)")

	return cmd
}

func runList(opts ListOptions, filter string) error {
	showAll := filter == ""
	showPipelines := showAll || filter == "pipelines"
	showPersonas := showAll || filter == "personas"
	showAdapters := filter == "adapters"

	if showPipelines {
		if err := listPipelines(); err != nil {
			return err
		}
		if showAll {
			fmt.Println()
		}
	}

	// Load manifest for personas/adapters
	manifestData, err := os.ReadFile(opts.Manifest)
	if err != nil && (showPersonas || showAdapters) {
		fmt.Printf("(manifest not found: %s)\n", opts.Manifest)
		return nil
	}

	var m struct {
		Adapters map[string]struct {
			Binary       string `yaml:"binary"`
			Mode         string `yaml:"mode"`
			OutputFormat string `yaml:"output_format"`
		} `yaml:"adapters"`
		Personas map[string]struct {
			Adapter          string  `yaml:"adapter"`
			Description      string  `yaml:"description"`
			SystemPromptFile string  `yaml:"system_prompt_file"`
			Temperature      float64 `yaml:"temperature"`
			Permissions      struct {
				AllowedTools []string `yaml:"allowed_tools"`
				Deny         []string `yaml:"deny"`
			} `yaml:"permissions"`
		} `yaml:"personas"`
	}
	if err == nil {
		yaml.Unmarshal(manifestData, &m)
	}

	if showPersonas {
		listPersonas(m.Personas)
		if showAll && showAdapters {
			fmt.Println()
		}
	}

	if showAdapters {
		listAdapters(m.Adapters)
	}

	return nil
}

func listPipelines() error {
	pipelineDir := ".wave/pipelines"
	entries, err := os.ReadDir(pipelineDir)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read pipelines directory: %w", err)
	}

	fmt.Printf("Pipelines:\n")
	if len(entries) == 0 {
		fmt.Printf("  (none found in %s/)\n", pipelineDir)
		return nil
	}

	// Sort entries by name
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".yaml")
		pipelinePath := filepath.Join(pipelineDir, entry.Name())

		data, err := os.ReadFile(pipelinePath)
		if err != nil {
			fmt.Printf("  %-20s (error reading)\n", name)
			continue
		}

		var p struct {
			Metadata struct {
				Description string `yaml:"description"`
			} `yaml:"metadata"`
			Steps []struct {
				ID      string `yaml:"id"`
				Persona string `yaml:"persona"`
			} `yaml:"steps"`
		}
		if err := yaml.Unmarshal(data, &p); err != nil {
			fmt.Printf("  %-20s (error parsing)\n", name)
			continue
		}

		desc := p.Metadata.Description
		if desc == "" {
			desc = "(no description)"
		}

		stepIDs := []string{}
		for _, s := range p.Steps {
			stepIDs = append(stepIDs, s.ID)
		}
		fmt.Printf("  %-20s %d steps  %s\n", name, len(p.Steps), desc)
		fmt.Printf("  %-20s steps: %s\n", "", strings.Join(stepIDs, " → "))
	}

	return nil
}

func listPersonas(personas map[string]struct {
	Adapter          string  `yaml:"adapter"`
	Description      string  `yaml:"description"`
	SystemPromptFile string  `yaml:"system_prompt_file"`
	Temperature      float64 `yaml:"temperature"`
	Permissions      struct {
		AllowedTools []string `yaml:"allowed_tools"`
		Deny         []string `yaml:"deny"`
	} `yaml:"permissions"`
}) {
	fmt.Printf("Personas:\n")
	if len(personas) == 0 {
		fmt.Printf("  (none defined)\n")
		return
	}

	// Sort by name for stable output
	names := make([]string, 0, len(personas))
	for name := range personas {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		persona := personas[name]
		desc := persona.Description
		if desc == "" {
			desc = "(no description)"
		}
		// T089: Add permission summary
		permSummary := formatPermissionSummary(
			persona.Permissions.AllowedTools,
			persona.Permissions.Deny,
		)
		fmt.Printf("  %-20s adapter:%-10s temp:%.1f  %s\n",
			name,
			persona.Adapter,
			persona.Temperature,
			permSummary,
		)
		fmt.Printf("  %-20s %s\n", "", desc)
	}
}

// formatPermissionSummary creates a concise summary of persona permissions.
func formatPermissionSummary(allowed []string, denied []string) string {
	allowCount := len(allowed)
	denyCount := len(denied)

	if allowCount == 0 && denyCount == 0 {
		return "tools:(default)"
	}

	parts := []string{}
	if allowCount > 0 {
		parts = append(parts, fmt.Sprintf("allow:%d", allowCount))
	}
	if denyCount > 0 {
		parts = append(parts, fmt.Sprintf("deny:%d", denyCount))
	}

	return strings.Join(parts, " ")
}

// listAdapters lists all configured adapters with binary availability check.
func listAdapters(adapters map[string]struct {
	Binary       string `yaml:"binary"`
	Mode         string `yaml:"mode"`
	OutputFormat string `yaml:"output_format"`
}) {
	fmt.Printf("Adapters:\n")
	if len(adapters) == 0 {
		fmt.Printf("  (none defined)\n")
		return
	}

	// Sort by name for stable output
	names := make([]string, 0, len(adapters))
	for name := range adapters {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		adapter := adapters[name]
		// T087: Check binary availability
		available := "OK"
		if _, err := exec.LookPath(adapter.Binary); err != nil {
			available = "[X] not found"
		}
		fmt.Printf("  %-20s binary:%-10s mode:%-10s format:%-6s %s\n",
			name, adapter.Binary, adapter.Mode, adapter.OutputFormat, available)
	}
}
