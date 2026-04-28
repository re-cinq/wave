package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/recinq/wave/internal/display"
	"github.com/recinq/wave/internal/listing"
	"github.com/recinq/wave/internal/skill"
	"github.com/spf13/cobra"
)

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

// SkillCheckOutput is the output for wave skills check.
type SkillCheckOutput struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Source      string   `json:"source"`
	UsedBy      []string `json:"used_by,omitempty"`
	OK          bool     `json:"ok"`
}

// SkillAddOutput is the output for wave skills add.
type SkillAddOutput struct {
	Installed []string `json:"installed"`
	Source    string   `json:"source"`
	Target    string   `json:"target"`
}

// SkillDoctorOutput is the output for wave skills doctor.
type SkillDoctorOutput struct {
	Scanned    []string         `json:"scanned"`
	Duplicates map[string][]int `json:"duplicates,omitempty"`
	Malformed  []string         `json:"malformed,omitempty"`
	Deprecated []string         `json:"deprecated,omitempty"`
	OK         bool             `json:"ok"`
}

// NewSkillsCmd creates the top-level wave skills command.
func NewSkillsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skills",
		Short: "Skill lifecycle management",
		Long: `Manage SKILL.md files discovered across project and user directories.

Subcommands:
  list      List discovered skills
  check     Validate a single skill and show usage
  add       Install a skill from a local path or file:// URL
  doctor    Diagnose skill discovery issues`,
	}

	cmd.AddCommand(newSkillsListCmd())
	cmd.AddCommand(newSkillsCheckCmd())
	cmd.AddCommand(newSkillsAddCmd())
	cmd.AddCommand(newSkillsDoctorCmd())

	return cmd
}

// skillSourcePaths returns the ordered detection paths (project then user-global).
// First match wins per skill name (earlier = higher precedence).
func skillSourcePaths() []string {
	paths := []string{
		".agents/skills",
		".claude/skills",
		".opencode/skills",
		".gemini/skills",
	}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths,
			filepath.Join(home, ".agents", "skills"),
			filepath.Join(home, ".claude", "skills"),
			filepath.Join(home, ".config", "opencode", "skills"),
			filepath.Join(home, ".gemini", "skills"),
		)
	}
	return paths
}

// newSkillStore creates a DirectoryStore covering all detection paths.
// Precedence is descending by index (project paths first, then user-global).
func newSkillStore() *skill.DirectoryStore {
	paths := skillSourcePaths()
	sources := make([]skill.SkillSource, 0, len(paths))
	for i, p := range paths {
		sources = append(sources, skill.SkillSource{
			Root:       p,
			Precedence: len(paths) - i,
		})
	}
	return skill.NewDirectoryStore(sources...)
}

// --- list ---

func newSkillsListCmd() *cobra.Command {
	var format string
	var ontologyOnly bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List discovered skills",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSkillsList(cmd, format, ontologyOnly)
		},
	}
	cmd.Flags().StringVar(&format, "format", "table", "Output format: table, json")
	cmd.Flags().BoolVar(&ontologyOnly, "ontology", false, "Show only ontology context skills (wave-ctx-*)")
	return cmd
}

func runSkillsList(cmd *cobra.Command, format string, ontologyOnly bool) error {
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

	usage := listing.CollectSkillPipelineUsage()

	output := SkillListOutput{
		Skills:   make([]SkillListItem, 0, len(skills)),
		Warnings: warnings,
	}
	for _, s := range skills {
		if ontologyOnly && !strings.HasPrefix(s.Name, "wave-ctx-") {
			continue
		}
		output.Skills = append(output.Skills, SkillListItem{
			Name:        s.Name,
			Description: s.Description,
			Source:      s.SourcePath,
			UsedBy:      usage[s.Name],
		})
	}

	switch format {
	case "json":
		return json.NewEncoder(cmd.OutOrStdout()).Encode(output)
	default:
		return renderSkillsListTable(cmd.OutOrStdout(), output)
	}
}

func renderSkillsListTable(w io.Writer, output SkillListOutput) error {
	f := display.NewFormatter()

	if len(output.Skills) == 0 {
		fmt.Fprintln(w, "No skills installed.")
		fmt.Fprintf(w, "  %s\n", f.Muted("Hint: use `wave skills add <path>` to install a skill"))
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

// --- check ---

func newSkillsCheckCmd() *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "check <name>",
		Short: "Validate a single skill and show usage",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSkillsCheck(cmd, args[0], format)
		},
	}
	cmd.Flags().StringVar(&format, "format", "table", "Output format: table, json")
	return cmd
}

func runSkillsCheck(cmd *cobra.Command, name, format string) error {
	format = ResolveFormat(cmd, format)

	store := newSkillStore()
	s, err := store.Read(name)
	if err != nil {
		if errors.Is(err, skill.ErrNotFound) {
			return NewCLIError(CodeSkillNotFound,
				fmt.Sprintf("skill %q not found", name),
				"Run `wave skills list` to see installed skills")
		}
		return err
	}

	usage := listing.CollectSkillPipelineUsage()
	output := SkillCheckOutput{
		Name:        s.Name,
		Description: s.Description,
		Source:      s.SourcePath,
		UsedBy:      usage[s.Name],
		OK:          true,
	}

	switch format {
	case "json":
		return json.NewEncoder(cmd.OutOrStdout()).Encode(output)
	default:
		f := display.NewFormatter()
		w := cmd.OutOrStdout()
		fmt.Fprintf(w, "%s %s\n", f.Success("OK"), f.Primary(s.Name))
		fmt.Fprintf(w, "  description: %s\n", s.Description)
		fmt.Fprintf(w, "  source:      %s\n", s.SourcePath)
		if len(output.UsedBy) > 0 {
			fmt.Fprintf(w, "  used by:     %s\n", strings.Join(output.UsedBy, ", "))
		}
		return nil
	}
}

// --- add ---

func newSkillsAddCmd() *cobra.Command {
	var format string
	var project bool
	cmd := &cobra.Command{
		Use:   "add <path-or-url>",
		Short: "Install a skill from a local path or file:// URL",
		Long: `Install a skill from a local path or file:// URL.

By default skills install to ~/.agents/skills/<name>/ (user-global).
Use --project to install to .agents/skills/<name>/ (project-scoped, committed).

Examples:
  wave skills add ./my-skill
  wave skills add file:///abs/path/to/skill
  wave skills add ./my-skill --project`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSkillsAdd(cmd, args[0], format, project)
		},
	}
	cmd.Flags().StringVar(&format, "format", "table", "Output format: table, json")
	cmd.Flags().BoolVar(&project, "project", false, "Install to project (.agents/skills/) instead of user-global")
	return cmd
}

func runSkillsAdd(cmd *cobra.Command, source, format string, project bool) error {
	format = ResolveFormat(cmd, format)

	var target string
	if project {
		target = ".agents/skills"
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("resolve home dir: %w", err)
		}
		target = filepath.Join(home, ".agents", "skills")
	}
	if err := os.MkdirAll(target, 0o755); err != nil {
		return fmt.Errorf("create target dir %s: %w", target, err)
	}

	store := skill.NewDirectoryStore(skill.SkillSource{Root: target, Precedence: 1})
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

	output := SkillAddOutput{
		Installed: names,
		Source:    source,
		Target:    target,
	}

	switch format {
	case "json":
		return json.NewEncoder(cmd.OutOrStdout()).Encode(output)
	default:
		f := display.NewFormatter()
		for _, name := range names {
			fmt.Fprintf(cmd.OutOrStdout(), "%s Installed skill %s into %s\n", f.Success("✓"), f.Primary(name), f.Muted(target))
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
			"Recognized prefixes: file:, https://, or a local path")
	}
	return NewCLIError(CodeSkillSourceError, msg, "Check the source string and try again")
}

// --- doctor ---

func newSkillsDoctorCmd() *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose skill discovery issues",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSkillsDoctor(cmd, format)
		},
	}
	cmd.Flags().StringVar(&format, "format", "table", "Output format: table, json")
	return cmd
}

func runSkillsDoctor(cmd *cobra.Command, format string) error {
	format = ResolveFormat(cmd, format)

	paths := skillSourcePaths()
	output := SkillDoctorOutput{
		Scanned:    paths,
		Duplicates: map[string][]int{},
		OK:         true,
	}

	// duplicate detection: scan each path independently, aggregate by name
	seen := map[string][]int{}
	for i, root := range paths {
		store := skill.NewDirectoryStore(skill.SkillSource{Root: root, Precedence: 1})
		skills, err := store.List()
		var discErr *skill.DiscoveryError
		if errors.As(err, &discErr) {
			for _, se := range discErr.Errors {
				output.Malformed = append(output.Malformed, se.Error())
				output.OK = false
			}
		}
		for _, s := range skills {
			seen[s.Name] = append(seen[s.Name], i)
		}
	}
	for name, indices := range seen {
		if len(indices) > 1 {
			output.Duplicates[name] = indices
		}
	}

	// deprecated .wave/skills/ mention
	if _, err := os.Stat(".wave/skills"); err == nil {
		output.Deprecated = append(output.Deprecated, ".wave/skills/ exists — move to .agents/skills/")
		output.OK = false
	}

	switch format {
	case "json":
		return json.NewEncoder(cmd.OutOrStdout()).Encode(output)
	default:
		return renderSkillsDoctorTable(cmd.OutOrStdout(), output)
	}
}

func renderSkillsDoctorTable(w io.Writer, output SkillDoctorOutput) error {
	f := display.NewFormatter()
	fmt.Fprintln(w)
	fmt.Fprintln(w, f.Colorize("Scanned paths:", "\033[1;37m"))
	for _, p := range output.Scanned {
		fmt.Fprintf(w, "  %s\n", f.Muted(p))
	}

	if len(output.Duplicates) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, f.Colorize("Duplicates:", "\033[1;37m"))
		names := make([]string, 0, len(output.Duplicates))
		for n := range output.Duplicates {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			indices := output.Duplicates[n]
			roots := make([]string, 0, len(indices))
			for _, i := range indices {
				roots = append(roots, output.Scanned[i])
			}
			fmt.Fprintf(w, "  %s in %s\n", f.Primary(n), strings.Join(roots, ", "))
		}
	}

	if len(output.Malformed) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, f.Colorize("Malformed:", "\033[1;37m"))
		for _, m := range output.Malformed {
			fmt.Fprintf(w, "  %s %s\n", f.Error("error:"), m)
		}
	}

	if len(output.Deprecated) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, f.Colorize("Deprecated:", "\033[1;37m"))
		for _, d := range output.Deprecated {
			fmt.Fprintf(w, "  %s %s\n", f.Error("warning:"), d)
		}
	}

	fmt.Fprintln(w)
	if output.OK {
		fmt.Fprintf(w, "%s skill discovery healthy\n", f.Success("OK"))
	} else {
		fmt.Fprintf(w, "%s issues found\n", f.Error("FAIL"))
	}
	return nil
}
