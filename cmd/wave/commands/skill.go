package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
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
		Short: "Manage skill templates and install from remote sources",
		Long: `Manage skill templates shipped with Wave and install skills from remote sources.

Subcommands:
  list      List available and installed skill templates
  install   Install a skill from bundled templates, GitHub, Tessl, or URL
  check     Run check commands for installed skills`,
	}

	cmd.AddCommand(newSkillListCmd())
	cmd.AddCommand(newSkillInstallCmd())
	cmd.AddCommand(newSkillCheckCmd())

	return cmd
}

// --- List ---

func newSkillListCmd() *cobra.Command {
	var format string
	var remote bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available and installed skill templates",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSkillList(cmd, format, remote)
		},
	}
	cmd.Flags().StringVar(&format, "format", "table", "Output format: table, json")
	cmd.Flags().BoolVar(&remote, "remote", false, "Show available remote sources (bundled templates plus hints for github: and tessl: sources)")
	return cmd
}

func runSkillList(cmd *cobra.Command, format string, remote bool) error {
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
		if err := renderSkillTemplateListTable(cmd.OutOrStdout(), output); err != nil {
			return err
		}
		if remote {
			renderRemoteSourceHints(cmd.OutOrStdout())
		}
		return nil
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
		Use:   "install <name-or-source>",
		Short: "Install a skill from bundled templates, GitHub, Tessl, or URL",
		Long: `Install a skill template that ships with Wave, or fetch from a remote source.

Source detection:
  <bare-name>             Bundled template (e.g., gh-cli, docker)
  github:<owner>/<repo>   Fetch from GitHub repository
  tessl:<name>            Fetch from Tessl registry
  https://<url>           Fetch from direct URL (archive)

Examples:
  wave skill install gh-cli
  wave skill install docker
  wave skill install github:re-cinq/wave-skills/golang
  wave skill install tessl:spec-kit
  wave skill install https://example.com/skills.tar.gz`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSkillInstall(cmd, args[0], format)
		},
	}
	cmd.Flags().StringVar(&format, "format", "table", "Output format: table, json")
	return cmd
}

// isRemoteSource returns true if the source string contains a recognized
// remote prefix (github:, tessl:, https://, or any other registered adapter prefix).
func isRemoteSource(source string) bool {
	if strings.HasPrefix(source, "https://") {
		return true
	}
	remotePrefixes := []string{"github:", "tessl:", "bmad:", "openspec:", "speckit:", "file:"}
	for _, prefix := range remotePrefixes {
		if strings.HasPrefix(source, prefix) {
			return true
		}
	}
	return false
}

func runSkillInstall(cmd *cobra.Command, source, format string) error {
	format = ResolveFormat(cmd, format)

	// Dispatch to remote installer if a source prefix is detected
	if isRemoteSource(source) {
		return runSkillInstallRemote(cmd, source, format)
	}

	// Bare name — install from bundled templates
	return runSkillInstallBundled(cmd, source, format)
}

func runSkillInstallBundled(cmd *cobra.Command, name, format string) error {
	templates := defaults.GetSkillTemplates()
	data, ok := templates[name]
	if !ok {
		available := defaults.SkillTemplateNames()
		return NewCLIError(CodeSkillNotFound,
			fmt.Sprintf("skill template %q not found", name),
			fmt.Sprintf("Available templates: %s\nYou can also install from remote sources: github:<owner>/<repo>, tessl:<name>, https://<url>",
				strings.Join(available, ", ")))
	}

	destDir := filepath.Join(".agents", "skills", name)
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

func runSkillInstallRemote(cmd *cobra.Command, source, format string) error {
	store := newSkillStore()
	router := skill.NewDefaultRouter(".")

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	result, err := router.Install(ctx, source, store)
	if err != nil {
		return classifySkillError(err)
	}

	names := make([]string, 0, len(result.Skills))
	for _, s := range result.Skills {
		names = append(names, s.Name)
	}

	switch format {
	case "json":
		output := SkillInstallOutput{
			InstalledSkills: names,
			Source:          source,
			Warnings:        result.Warnings,
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(output)
	default:
		f := display.NewFormatter()
		for _, name := range names {
			fmt.Fprintf(cmd.OutOrStdout(), "%s Installed skill %s from %s\n",
				f.Success("OK"), f.Primary(name), source)
		}
		for _, warn := range result.Warnings {
			fmt.Fprintf(cmd.OutOrStdout(), "  %s %s\n", f.Error("warning:"), warn)
		}
		return nil
	}
}

// renderRemoteSourceHints prints hints about remote skill sources.
func renderRemoteSourceHints(w io.Writer) {
	f := display.NewFormatter()
	fmt.Fprintln(w, f.Colorize("Remote sources also supported:", "\033[1;37m"))
	fmt.Fprintf(w, "  %s  %s\n", f.Primary("github:<owner>/<repo>"), "Install from a GitHub repository")
	fmt.Fprintf(w, "  %s  %s\n", f.Primary("tessl:<name>"), "Install from the Tessl registry")
	fmt.Fprintf(w, "  %s  %s\n", f.Primary("https://<url>"), "Install from a direct URL (archive)")
	fmt.Fprintln(w)
}

// --- Check ---

// SkillCheckItem represents one skill check result in output.
type SkillCheckItem struct {
	Name         string `json:"name"`
	CheckCommand string `json:"check_command"`
	Passed       bool   `json:"passed"`
	Error        string `json:"error,omitempty"`
}

// SkillCheckOutput is the top-level output for wave skill check.
type SkillCheckOutput struct {
	Results []SkillCheckItem `json:"results"`
	Passed  int              `json:"passed"`
	Failed  int              `json:"failed"`
	Skipped int              `json:"skipped"`
}

func newSkillCheckCmd() *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Run check commands for installed skills",
		Long: `For each installed skill in .agents/skills/, parse its SKILL.md frontmatter
to find a check_command. If present, run it and report pass/fail.

Returns non-zero exit code if any check fails.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSkillCheck(cmd, format)
		},
	}
	cmd.Flags().StringVar(&format, "format", "table", "Output format: table, json")
	return cmd
}

func runSkillCheck(cmd *cobra.Command, format string) error {
	format = ResolveFormat(cmd, format)

	skillsDir := filepath.Join(".agents", "skills")
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintln(cmd.OutOrStdout(), "No skills installed.")
			return nil
		}
		return fmt.Errorf("failed to read skills directory: %w", err)
	}

	output := SkillCheckOutput{
		Results: make([]SkillCheckItem, 0),
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillFile := filepath.Join(skillsDir, entry.Name(), "SKILL.md")
		data, readErr := os.ReadFile(skillFile)
		if readErr != nil {
			continue
		}

		s, parseErr := skill.ParseMetadata(data)
		if parseErr != nil {
			continue
		}

		if s.CheckCommand == "" {
			output.Skipped++
			continue
		}

		item := SkillCheckItem{
			Name:         s.Name,
			CheckCommand: s.CheckCommand,
		}

		// Run check command via shell
		checkCmd := exec.Command("sh", "-c", s.CheckCommand)
		if runErr := checkCmd.Run(); runErr != nil {
			item.Passed = false
			item.Error = runErr.Error()
			output.Failed++
		} else {
			item.Passed = true
			output.Passed++
		}

		output.Results = append(output.Results, item)
	}

	switch format {
	case "json":
		return json.NewEncoder(cmd.OutOrStdout()).Encode(output)
	default:
		if err := renderSkillCheckTable(cmd.OutOrStdout(), output); err != nil {
			return err
		}
		if output.Failed > 0 {
			return NewCLIError(CodeSkillValidationFailed,
				fmt.Sprintf("%d skill check(s) failed", output.Failed),
				"Run the failing check commands manually to diagnose")
		}
		return nil
	}
}

func renderSkillCheckTable(w io.Writer, output SkillCheckOutput) error {
	f := display.NewFormatter()

	if len(output.Results) == 0 {
		fmt.Fprintln(w, "No skills with check commands found.")
		return nil
	}

	fmt.Fprintln(w)
	fmt.Fprintf(w, "%-15s %-40s %s\n",
		f.Colorize("NAME", "\033[1;37m"),
		f.Colorize("CHECK COMMAND", "\033[1;37m"),
		f.Colorize("STATUS", "\033[1;37m"))

	for _, r := range output.Results {
		status := f.Success("pass")
		if !r.Passed {
			status = f.Error("FAIL")
		}
		fmt.Fprintf(w, "%-15s %-40s %s\n",
			f.Primary(r.Name),
			r.CheckCommand,
			status)
	}

	fmt.Fprintf(w, "\n%d passed, %d failed, %d skipped\n",
		output.Passed, output.Failed, output.Skipped)

	return nil
}

// installedSkillNames returns a set of skill names installed in .agents/skills/.
func installedSkillNames() map[string]bool {
	result := make(map[string]bool)
	entries, err := os.ReadDir(filepath.Join(".agents", "skills"))
	if err != nil {
		return result
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillFile := filepath.Join(".agents", "skills", entry.Name(), "SKILL.md")
		if _, err := os.Stat(skillFile); err == nil {
			result[entry.Name()] = true
		}
	}
	return result
}
