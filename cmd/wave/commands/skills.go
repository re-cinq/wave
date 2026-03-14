package commands

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/recinq/wave/internal/display"
	"github.com/recinq/wave/internal/skill"
	"github.com/spf13/cobra"
)

// CLI output structs for wave skills subcommands.

// SkillListItem represents one skill in list output.
type SkillListItem struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Source      string   `json:"source"`
	UsedBy      []string `json:"used_by,omitempty"`
}

// SkillListOutput is the top-level output for wave skills list.
type SkillListOutput struct {
	Skills   []SkillListItem `json:"skills"`
	Warnings []string        `json:"warnings,omitempty"`
}

// SkillInstallOutput is the output for wave skills install.
type SkillInstallOutput struct {
	InstalledSkills []string `json:"installed_skills"`
	Source          string   `json:"source"`
	Warnings        []string `json:"warnings,omitempty"`
}

// SkillRemoveOutput is the output for wave skills remove.
type SkillRemoveOutput struct {
	Removed string `json:"removed"`
	Source  string `json:"source"`
}

// SkillSearchResult represents one search result item.
type SkillSearchResult struct {
	Name        string `json:"name"`
	Rating      string `json:"rating,omitempty"`
	Description string `json:"description"`
}

// SkillSyncOutput is the output for wave skills sync.
type SkillSyncOutput struct {
	SyncedSkills []string `json:"synced_skills"`
	Warnings     []string `json:"warnings,omitempty"`
	Status       string   `json:"status"`
}

// NewSkillsCmd creates the top-level wave skills command.
func NewSkillsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skills",
		Short: "Skill lifecycle management",
		Long: `Manage skills installed in your project and user directories.

Subcommands:
  list      List installed skills
  install   Install a skill from any source
  remove    Remove an installed skill
  search    Search the Tessl registry
  sync      Sync project dependencies from the Tessl registry`,
	}

	cmd.AddCommand(newSkillsListCmd())
	cmd.AddCommand(newSkillsInstallCmd())
	cmd.AddCommand(newSkillsRemoveCmd())
	cmd.AddCommand(newSkillsSearchCmd())
	cmd.AddCommand(newSkillsSyncCmd())

	return cmd
}

// newSkillStore creates a DirectoryStore with project and user skill sources.
func newSkillStore() *skill.DirectoryStore {
	sources := []skill.SkillSource{
		{Root: ".wave/skills", Precedence: 2},
	}
	home, err := os.UserHomeDir()
	if err == nil {
		sources = append(sources, skill.SkillSource{
			Root:       home + "/.claude/skills",
			Precedence: 1,
		})
	}
	return skill.NewDirectoryStore(sources...)
}

// --- List ---

func newSkillsListCmd() *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List installed skills",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSkillsList(cmd, format)
		},
	}
	cmd.Flags().StringVar(&format, "format", "table", "Output format: table, json")
	return cmd
}

func runSkillsList(cmd *cobra.Command, format string) error {
	format = ResolveFormat(cmd, format)

	store := newSkillStore()
	skills, err := store.List()

	var warnings []string
	var discErr *skill.DiscoveryError
	if errors.As(err, &discErr) {
		for _, se := range discErr.Errors {
			warnings = append(warnings, se.Error())
		}
	} else if err != nil {
		return err
	}

	usage := collectSkillPipelineUsage()

	output := SkillListOutput{
		Skills:   make([]SkillListItem, 0, len(skills)),
		Warnings: warnings,
	}
	for _, s := range skills {
		item := SkillListItem{
			Name:        s.Name,
			Description: s.Description,
			Source:      s.SourcePath,
			UsedBy:      usage[s.Name],
		}
		output.Skills = append(output.Skills, item)
	}

	switch format {
	case "json":
		return json.NewEncoder(os.Stdout).Encode(output)
	default:
		return renderSkillsListTable(cmd.OutOrStdout(), output)
	}
}

func renderSkillsListTable(w io.Writer, output SkillListOutput) error {
	f := display.NewFormatter()

	if len(output.Skills) == 0 {
		fmt.Fprintln(w, "No skills installed.")
		fmt.Fprintf(w, "  %s\n", f.Muted("Hint: use `wave skills install <source>` to install a skill"))
		return nil
	}

	fmt.Fprintln(w)
	fmt.Fprintf(w, "%-20s %-40s %-30s %s\n",
		f.Colorize("NAME", "\033[1;37m"),
		f.Colorize("DESCRIPTION", "\033[1;37m"),
		f.Colorize("SOURCE", "\033[1;37m"),
		f.Colorize("USED BY", "\033[1;37m"))

	for _, s := range output.Skills {
		usedBy := ""
		if len(s.UsedBy) > 0 {
			usedBy = strings.Join(s.UsedBy, ", ")
		}
		fmt.Fprintf(w, "%-20s %-40s %-30s %s\n",
			f.Primary(s.Name),
			s.Description,
			f.Muted(s.Source),
			usedBy)
	}

	if len(output.Warnings) > 0 {
		fmt.Fprintln(w)
		for _, warn := range output.Warnings {
			fmt.Fprintf(w, "  %s %s\n", f.Error("warning:"), warn)
		}
	}

	fmt.Fprintln(w)
	return nil
}

// --- Install ---

func newSkillsInstallCmd() *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "install <source>",
		Short: "Install a skill from any source",
		Long: `Install a skill using a source prefix to select the adapter.

Recognized prefixes: tessl:, bmad:, openspec:, speckit:, github:, file:, https://

Examples:
  wave skills install tessl:github/spec-kit
  wave skills install github:re-cinq/wave-skills/golang
  wave skills install file:./my-skill`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSkillsInstall(cmd, args[0], format)
		},
	}
	cmd.Flags().StringVar(&format, "format", "table", "Output format: table, json")
	return cmd
}

func runSkillsInstall(cmd *cobra.Command, source, format string) error {
	format = ResolveFormat(cmd, format)

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

	output := SkillInstallOutput{
		InstalledSkills: names,
		Source:          source,
		Warnings:        result.Warnings,
	}

	switch format {
	case "json":
		return json.NewEncoder(os.Stdout).Encode(output)
	default:
		f := display.NewFormatter()
		for _, name := range names {
			fmt.Fprintf(cmd.OutOrStdout(), "%s Installed skill %s from %s\n", f.Success("✓"), f.Primary(name), source)
		}
		for _, warn := range result.Warnings {
			fmt.Fprintf(cmd.OutOrStdout(), "  %s %s\n", f.Error("warning:"), warn)
		}
		return nil
	}
}

func classifySkillError(err error) *CLIError {
	var depErr *skill.DependencyError
	if errors.As(err, &depErr) {
		return NewCLIError(CodeSkillDependencyMissing,
			fmt.Sprintf("required tool %q is not installed", depErr.Binary),
			depErr.Instructions)
	}
	if errors.Is(err, skill.ErrNotFound) {
		return NewCLIError(CodeSkillNotFound,
			err.Error(),
			"Check the source reference and try again")
	}
	msg := err.Error()
	if strings.Contains(msg, "unknown source prefix") || strings.Contains(msg, "no source prefix") {
		return NewCLIError(CodeSkillSourceError, msg,
			"Recognized prefixes: tessl:, bmad:, openspec:, speckit:, github:, file:, https://")
	}
	return NewCLIError(CodeSkillSourceError, msg, "Check the source string and try again")
}

// --- Remove ---

func newSkillsRemoveCmd() *cobra.Command {
	var format string
	var yes bool
	cmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove an installed skill",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSkillsRemove(cmd, args[0], format, yes, os.Stdin, os.Stderr)
		},
	}
	cmd.Flags().StringVar(&format, "format", "table", "Output format: table, json")
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}

func runSkillsRemove(cmd *cobra.Command, name, format string, yes bool, in io.Reader, out io.Writer) error {
	format = ResolveFormat(cmd, format)

	store := newSkillStore()

	// Read the skill first to get source path for output
	s, err := store.Read(name)
	if err != nil {
		if errors.Is(err, skill.ErrNotFound) {
			return NewCLIError(CodeSkillNotFound,
				fmt.Sprintf("skill %q not found", name),
				"Run `wave skills list` to see installed skills")
		}
		return err
	}

	if !yes {
		confirmed, promptErr := promptConfirm(in, out, fmt.Sprintf("Remove skill %q? [Y/n] ", name))
		if promptErr != nil {
			return promptErr
		}
		if !confirmed {
			fmt.Fprintln(out, "Cancelled.")
			return nil
		}
	}

	if err := store.Delete(name); err != nil {
		if errors.Is(err, skill.ErrNotFound) {
			return NewCLIError(CodeSkillNotFound,
				fmt.Sprintf("skill %q not found", name),
				"Run `wave skills list` to see installed skills")
		}
		return err
	}

	output := SkillRemoveOutput{
		Removed: name,
		Source:  s.SourcePath,
	}

	switch format {
	case "json":
		return json.NewEncoder(os.Stdout).Encode(output)
	default:
		f := display.NewFormatter()
		fmt.Fprintf(cmd.OutOrStdout(), "%s Removed skill %s\n", f.Success("✓"), f.Primary(name))
		return nil
	}
}

// --- Search ---

func newSkillsSearchCmd() *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search the Tessl registry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSkillsSearch(cmd, args[0], format)
		},
	}
	cmd.Flags().StringVar(&format, "format", "table", "Output format: table, json")
	return cmd
}

func runSkillsSearch(cmd *cobra.Command, query, format string) error {
	format = ResolveFormat(cmd, format)

	if _, err := exec.LookPath("tessl"); err != nil {
		return NewCLIError(CodeSkillDependencyMissing,
			"the tessl CLI is required for registry search",
			"Install tessl: see https://tessl.io/docs/install")
	}

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	out, err := exec.CommandContext(ctx, "tessl", "search", query).Output()
	if err != nil {
		return NewCLIError(CodeSkillSourceError,
			fmt.Sprintf("tessl search failed: %v", err),
			"Check that tessl is configured correctly")
	}

	results := parseTesslSearchOutput(string(out))

	switch format {
	case "json":
		return json.NewEncoder(os.Stdout).Encode(results)
	default:
		return renderSearchTable(cmd.OutOrStdout(), results)
	}
}

func parseTesslSearchOutput(output string) []SkillSearchResult {
	var results []SkillSearchResult
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		// Parse tab-separated or space-separated output: name rating description
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		result := SkillSearchResult{
			Name: fields[0],
		}
		if len(fields) >= 3 {
			result.Rating = fields[1]
			result.Description = strings.Join(fields[2:], " ")
		} else {
			result.Description = strings.Join(fields[1:], " ")
		}
		results = append(results, result)
	}
	return results
}

func renderSearchTable(w io.Writer, results []SkillSearchResult) error {
	if len(results) == 0 {
		fmt.Fprintln(w, "No results found.")
		return nil
	}

	f := display.NewFormatter()
	fmt.Fprintln(w)
	fmt.Fprintf(w, "%-30s %-10s %s\n",
		f.Colorize("NAME", "\033[1;37m"),
		f.Colorize("RATING", "\033[1;37m"),
		f.Colorize("DESCRIPTION", "\033[1;37m"))

	for _, r := range results {
		fmt.Fprintf(w, "%-30s %-10s %s\n",
			f.Primary(r.Name),
			r.Rating,
			r.Description)
	}

	fmt.Fprintln(w)
	return nil
}

// --- Sync ---

func newSkillsSyncCmd() *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync project dependencies from the Tessl registry",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSkillsSync(cmd, format)
		},
	}
	cmd.Flags().StringVar(&format, "format", "table", "Output format: table, json")
	return cmd
}

func runSkillsSync(cmd *cobra.Command, format string) error {
	format = ResolveFormat(cmd, format)

	if _, err := exec.LookPath("tessl"); err != nil {
		return NewCLIError(CodeSkillDependencyMissing,
			"the tessl CLI is required for dependency sync",
			"Install tessl: see https://tessl.io/docs/install")
	}

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	out, err := exec.CommandContext(ctx, "tessl", "install", "--project-dependencies").Output()
	if err != nil {
		return NewCLIError(CodeSkillSourceError,
			fmt.Sprintf("tessl sync failed: %v", err),
			"Check that tessl is configured and your project has skill dependencies declared")
	}

	synced, warnings := parseTesslSyncOutput(string(out))

	status := "up_to_date"
	if len(synced) > 0 {
		status = "synced"
	}

	output := SkillSyncOutput{
		SyncedSkills: synced,
		Warnings:     warnings,
		Status:       status,
	}

	switch format {
	case "json":
		return json.NewEncoder(os.Stdout).Encode(output)
	default:
		f := display.NewFormatter()
		w := cmd.OutOrStdout()
		if len(synced) == 0 {
			fmt.Fprintln(w, "All skills up to date.")
		} else {
			for _, name := range synced {
				fmt.Fprintf(w, "%s Synced skill %s\n", f.Success("✓"), f.Primary(name))
			}
		}
		for _, warn := range warnings {
			fmt.Fprintf(w, "  %s %s\n", f.Error("warning:"), warn)
		}
		return nil
	}
}

func parseTesslSyncOutput(output string) (synced []string, warnings []string) {
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "warning:") || strings.HasPrefix(line, "Warning:") {
			warnings = append(warnings, strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(line, "Warning:"), "warning:")))
			continue
		}
		// Lines like "installed golang" or "updated spec-kit"
		fields := strings.Fields(line)
		if len(fields) >= 2 && (fields[0] == "installed" || fields[0] == "updated") {
			synced = append(synced, fields[1])
		}
	}
	return synced, warnings
}
