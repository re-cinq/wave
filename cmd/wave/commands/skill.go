package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/recinq/wave/internal/defaults"
	"github.com/recinq/wave/internal/display"
	"github.com/recinq/wave/internal/skill"
	"github.com/spf13/cobra"
)

// SkillTemplateListItem represents one skill template in list output.
type SkillTemplateListItem struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	CheckCommand string `json:"check_command,omitempty"`
	Installed    bool   `json:"installed"`
}

// SkillTemplateListOutput is the top-level output for wave skill list.
type SkillTemplateListOutput struct {
	Templates []SkillTemplateListItem `json:"templates"`
}

// SkillTemplateInstallOutput is the output for wave skill install.
type SkillTemplateInstallOutput struct {
	Name        string `json:"name"`
	Destination string `json:"destination"`
}

// NewSkillCmd creates the top-level wave skill command for template management.
func NewSkillCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Manage shipped skill templates",
		Long: `Manage skill templates shipped with Wave.

Subcommands:
  list      List available and installed skill templates
  install   Install a skill template to .wave/skills/`,
	}

	cmd.AddCommand(newSkillListCmd())
	cmd.AddCommand(newSkillInstallCmd())

	return cmd
}

// --- List ---

func newSkillListCmd() *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available and installed skill templates",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSkillList(cmd, format)
		},
	}
	cmd.Flags().StringVar(&format, "format", "table", "Output format: table, json")
	return cmd
}

func runSkillList(cmd *cobra.Command, format string) error {
	format = ResolveFormat(cmd, format)

	templates := defaults.GetSkillTemplates()
	installed := installedSkillNames()

	output := SkillTemplateListOutput{
		Templates: make([]SkillTemplateListItem, 0, len(templates)),
	}

	for _, name := range defaults.SkillTemplateNames() {
		data := templates[name]
		item := SkillTemplateListItem{
			Name:      name,
			Installed: installed[name],
		}

		// Parse metadata from the template
		s, err := skill.ParseMetadata(data)
		if err == nil {
			item.Description = s.Description
			item.CheckCommand = s.CheckCommand
		}

		output.Templates = append(output.Templates, item)
	}

	switch format {
	case "json":
		return json.NewEncoder(cmd.OutOrStdout()).Encode(output)
	default:
		return renderSkillTemplateListTable(cmd.OutOrStdout(), output)
	}
}

func renderSkillTemplateListTable(w io.Writer, output SkillTemplateListOutput) error {
	f := display.NewFormatter()

	if len(output.Templates) == 0 {
		fmt.Fprintln(w, "No skill templates available.")
		return nil
	}

	fmt.Fprintln(w)
	fmt.Fprintf(w, "%-15s %-55s %s\n",
		f.Colorize("NAME", "\033[1;37m"),
		f.Colorize("DESCRIPTION", "\033[1;37m"),
		f.Colorize("STATUS", "\033[1;37m"))

	for _, t := range output.Templates {
		status := f.Muted("available")
		if t.Installed {
			status = f.Success("installed")
		}
		fmt.Fprintf(w, "%-15s %-55s %s\n",
			f.Primary(t.Name),
			t.Description,
			status)
	}

	fmt.Fprintln(w)
	return nil
}

// --- Install ---

func newSkillInstallCmd() *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "install <name>",
		Short: "Install a shipped skill template to .wave/skills/",
		Long: `Install a skill template that ships with Wave.

The template is copied to .wave/skills/<name>/SKILL.md in your project.

Examples:
  wave skill install gh-cli
  wave skill install docker
  wave skill install testing`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSkillInstall(cmd, args[0], format)
		},
	}
	cmd.Flags().StringVar(&format, "format", "table", "Output format: table, json")
	return cmd
}

func runSkillInstall(cmd *cobra.Command, name, format string) error {
	format = ResolveFormat(cmd, format)

	templates := defaults.GetSkillTemplates()
	data, ok := templates[name]
	if !ok {
		available := defaults.SkillTemplateNames()
		return NewCLIError(CodeSkillNotFound,
			fmt.Sprintf("skill template %q not found", name),
			fmt.Sprintf("Available templates: %s", strings.Join(available, ", ")))
	}

	destDir := filepath.Join(".wave", "skills", name)
	destFile := filepath.Join(destDir, "SKILL.md")

	// Check if already installed
	if _, err := os.Stat(destFile); err == nil {
		return NewCLIError(CodeSkillAlreadyExists,
			fmt.Sprintf("skill %q is already installed at %s", name, destDir),
			"Remove it first with `wave skills remove "+name+"` if you want to reinstall")
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create skill directory: %w", err)
	}

	if err := os.WriteFile(destFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write SKILL.md: %w", err)
	}

	output := SkillTemplateInstallOutput{
		Name:        name,
		Destination: destDir,
	}

	switch format {
	case "json":
		return json.NewEncoder(cmd.OutOrStdout()).Encode(output)
	default:
		f := display.NewFormatter()
		fmt.Fprintf(cmd.OutOrStdout(), "%s Installed skill template %s to %s\n",
			f.Success("OK"), f.Primary(name), destDir)
		return nil
	}
}

// installedSkillNames returns a set of skill names installed in .wave/skills/.
func installedSkillNames() map[string]bool {
	result := make(map[string]bool)
	entries, err := os.ReadDir(filepath.Join(".wave", "skills"))
	if err != nil {
		return result
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillFile := filepath.Join(".wave", "skills", entry.Name(), "SKILL.md")
		if _, err := os.Stat(skillFile); err == nil {
			result[entry.Name()] = true
		}
	}
	return result
}
