package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/display"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/persona"
	"github.com/spf13/cobra"
)

// AgentListItem represents one persona in the agent list output.
type AgentListItem struct {
	Name         string   `json:"name"`
	Adapter      string   `json:"adapter"`
	Model        string   `json:"model,omitempty"`
	Description  string   `json:"description,omitempty"`
	AllowedTools []string `json:"allowed_tools,omitempty"`
	DenyTools    []string `json:"deny_tools,omitempty"`
}

// AgentListOutput is the top-level output for wave agent list.
type AgentListOutput struct {
	Personas []AgentListItem `json:"personas"`
}

// NewAgentCmd creates the top-level wave agent command.
func NewAgentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Persona-to-agent compiler utilities",
		Long: `Inspect and export personas as Claude Code agent .md files.

The agent compiler translates Wave personas into self-contained Claude Code
agent definitions with YAML frontmatter (model, tools, disallowedTools,
permissionMode) that can be passed directly to Claude Code via --agent <path>.

Subcommands:
  list              List all personas with their agent .md equivalent summary
  inspect <name>    Show the full generated agent markdown for a persona
  export <name>     Write the agent .md file to disk`,
	}

	cmd.AddCommand(newAgentListCmd())
	cmd.AddCommand(newAgentInspectCmd())
	cmd.AddCommand(newAgentExportCmd())

	return cmd
}

// --- List ---

func newAgentListCmd() *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all personas and their agent .md summary",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAgentList(cmd, format)
		},
	}
	cmd.Flags().StringVar(&format, "format", "table", "Output format: table, json")
	return cmd
}

func runAgentList(cmd *cobra.Command, format string) error {
	format = ResolveFormat(cmd, format)

	manifestPath, _ := cmd.Root().PersistentFlags().GetString("manifest")
	if manifestPath == "" {
		manifestPath = "wave.yaml"
	}

	m, err := loadManifestForAgent(manifestPath)
	if err != nil {
		return err
	}

	items := buildAgentListItems(m)
	output := AgentListOutput{Personas: items}

	switch format {
	case "json":
		return json.NewEncoder(cmd.OutOrStdout()).Encode(output)
	default:
		return renderAgentListTable(cmd.OutOrStdout(), output)
	}
}

func buildAgentListItems(m *manifest.Manifest) []AgentListItem {
	items := make([]AgentListItem, 0, len(m.Personas))
	for name, p := range m.Personas {
		item := AgentListItem{
			Name:         name,
			Adapter:      p.Adapter,
			Model:        p.Model,
			Description:  p.Description,
			AllowedTools: p.Permissions.AllowedTools,
			DenyTools:    p.Permissions.Deny,
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})
	return items
}

func renderAgentListTable(w io.Writer, output AgentListOutput) error {
	f := display.NewFormatter()

	if len(output.Personas) == 0 {
		fmt.Fprintln(w, "No personas defined in manifest.")
		fmt.Fprintf(w, "  %s\n", f.Muted("Hint: define personas in wave.yaml under the `personas:` key"))
		return nil
	}

	fmt.Fprintln(w)
	fmt.Fprintf(w, "%-22s %-12s %-14s %-30s %s\n",
		f.Colorize("PERSONA", "\033[1;37m"),
		f.Colorize("ADAPTER", "\033[1;37m"),
		f.Colorize("MODEL", "\033[1;37m"),
		f.Colorize("ALLOWED TOOLS", "\033[1;37m"),
		f.Colorize("DESCRIPTION", "\033[1;37m"))

	for _, item := range output.Personas {
		model := item.Model
		if model == "" {
			model = f.Muted("(default)")
		}
		tools := strings.Join(item.AllowedTools, ", ")
		if len(tools) > 28 {
			tools = tools[:25] + "..."
		}
		if tools == "" {
			tools = f.Muted("(none)")
		}
		desc := item.Description
		if desc == "" {
			desc = f.Muted("-")
		}
		fmt.Fprintf(w, "%-22s %-12s %-14s %-30s %s\n",
			f.Primary(item.Name),
			item.Adapter,
			model,
			tools,
			desc)
	}

	fmt.Fprintln(w)
	fmt.Fprintf(w, "  %s Use `wave agent inspect <name>` to see the full agent .md\n", f.Muted("Tip:"))
	fmt.Fprintln(w)
	return nil
}

// --- Inspect ---

func newAgentInspectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inspect <name>",
		Short: "Show the generated agent markdown for a persona",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAgentInspect(cmd, args[0])
		},
	}
	return cmd
}

func runAgentInspect(cmd *cobra.Command, name string) error {
	manifestPath, _ := cmd.Root().PersistentFlags().GetString("manifest")
	if manifestPath == "" {
		manifestPath = "wave.yaml"
	}

	m, err := loadManifestForAgent(manifestPath)
	if err != nil {
		return err
	}

	agentMd, err := compilePersonaToAgentMd(name, m)
	if err != nil {
		return err
	}

	fmt.Fprint(cmd.OutOrStdout(), agentMd)
	return nil
}

// --- Export ---

func newAgentExportCmd() *cobra.Command {
	var outputPath string
	cmd := &cobra.Command{
		Use:   "export <name>",
		Short: "Write the agent .md file for a persona to disk",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAgentExport(cmd, args[0], outputPath)
		},
	}
	cmd.Flags().StringVar(&outputPath, "export-path", "", "Export file path (default: <name>.agent.md)")
	return cmd
}

func runAgentExport(cmd *cobra.Command, name, outputPath string) error {
	manifestPath, _ := cmd.Root().PersistentFlags().GetString("manifest")
	if manifestPath == "" {
		manifestPath = "wave.yaml"
	}

	m, err := loadManifestForAgent(manifestPath)
	if err != nil {
		return err
	}

	agentMd, err := compilePersonaToAgentMd(name, m)
	if err != nil {
		return err
	}

	dest := outputPath
	if dest == "" {
		dest = name + ".agent.md"
	}

	if err := os.WriteFile(dest, []byte(agentMd), 0644); err != nil {
		return fmt.Errorf("failed to write agent .md to %s: %w", dest, err)
	}

	f := display.NewFormatter()
	fmt.Fprintf(cmd.OutOrStdout(), "%s Wrote agent .md for %s to %s\n",
		f.Success("✓"), f.Primary(name), dest)
	return nil
}

// --- Helpers ---

// loadManifestForAgent loads the manifest and returns a CLIError if it is
// missing or invalid.
func loadManifestForAgent(manifestPath string) (*manifest.Manifest, error) {
	return loadManifestStrict(manifestPath)
}

// compilePersonaToAgentMd resolves a persona by name and compiles it to agent
// markdown. It reads the base protocol and persona system prompt from disk.
func compilePersonaToAgentMd(name string, m *manifest.Manifest) (string, error) {
	p := m.GetPersona(name)
	if p == nil {
		return "", NewCLIError(CodeInvalidArgs,
			fmt.Sprintf("persona %q not found in manifest", name),
			"Run 'wave agent list' to see available personas")
	}

	// Load base protocol
	baseProtocolPath := filepath.Join(".agents", "personas", "base-protocol.md")
	baseProtocolData, err := os.ReadFile(baseProtocolPath)
	if err != nil {
		return "", fmt.Errorf("failed to read base protocol from %s: %w", baseProtocolPath, err)
	}

	// Load persona system prompt
	var systemPrompt string
	if p.SystemPromptFile != "" {
		promptPath := p.GetSystemPromptPath(".")
		data, err := os.ReadFile(promptPath)
		if err != nil {
			return "", fmt.Errorf("failed to read system prompt for persona %q from %s: %w", name, promptPath, err)
		}
		systemPrompt = string(data)
	} else {
		// Fall back to the default .agents/personas/<name>.md location
		promptPath := filepath.Join(".agents", "personas", name+".md")
		if data, err := os.ReadFile(promptPath); err == nil {
			systemPrompt = string(data)
		} else {
			systemPrompt = fmt.Sprintf("# %s\n\nYou are operating as the %s persona.\n", name, name)
		}
	}

	// Map manifest.Persona to the neutral persona.Persona used by the agent
	// compiler (internal/persona breaks the would-be adapter↔manifest cycle).
	spec := persona.Persona{
		Model:        p.Model,
		AllowedTools: p.Permissions.AllowedTools,
		DenyTools:    p.Permissions.Deny,
	}

	agentMd := adapter.PersonaToAgentMarkdown(
		spec,
		string(baseProtocolData),
		systemPrompt,
		"", // no runtime contract section during static inspection
		"", // no runtime restrictions section during static inspection
	)

	return agentMd, nil
}
