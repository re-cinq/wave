package commands

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/recinq/wave/internal/display"
	"github.com/recinq/wave/internal/skill"
	"github.com/recinq/wave/internal/timeouts"
	"github.com/spf13/cobra"
	"io"
	"os"
	"os/exec"
	"strings"
)

// tesslTimeout is the maximum duration for tessl subprocess calls.
// Configured via runtime.timeouts.skill_publish_seconds in wave.yaml.
var tesslTimeout = timeouts.SkillPublish

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

// SkillAuditItem represents one skill in audit output.
type SkillAuditItem struct {
	Name           string   `json:"name"`
	Classification string   `json:"classification"`
	WaveRefCount   int      `json:"wave_ref_count"`
	Warnings       []string `json:"warnings,omitempty"`
	Source         string   `json:"source"`
}

// SkillAuditOutput is the top-level output for wave skills audit.
type SkillAuditOutput struct {
	Skills  []SkillAuditItem `json:"skills"`
	Summary AuditSummary     `json:"summary"`
}

// AuditSummary provides aggregate counts.
type AuditSummary struct {
	Total        int `json:"total"`
	Standalone   int `json:"standalone"`
	WaveSpecific int `json:"wave_specific"`
	Both         int `json:"both"`
}

// PublishResultItem represents one publish result in CLI output.
type PublishResultItem struct {
	Name     string   `json:"name"`
	Status   string   `json:"status"`
	URL      string   `json:"url,omitempty"`
	Digest   string   `json:"digest,omitempty"`
	Reason   string   `json:"reason,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// SkillPublishOutput is the top-level output for wave skills publish.
type SkillPublishOutput struct {
	Results  []PublishResultItem `json:"results"`
	Lockfile string              `json:"lockfile"`
}

// SkillVerifyItem represents one verify result.
type SkillVerifyItem struct {
	Name           string   `json:"name"`
	Status         string   `json:"status"`
	ExpectedDigest string   `json:"expected_digest,omitempty"`
	ActualDigest   string   `json:"actual_digest,omitempty"`
	Warnings       []string `json:"warnings,omitempty"`
}

// SkillVerifyOutput is the top-level output for wave skills verify.
type SkillVerifyOutput struct {
	Results []SkillVerifyItem `json:"results"`
	Summary VerifySummary     `json:"summary"`
}

// VerifySummary provides aggregate verify counts.
type VerifySummary struct {
	Total    int `json:"total"`
	OK       int `json:"ok"`
	Modified int `json:"modified"`
	Missing  int `json:"missing"`
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
  sync      Sync project dependencies from the Tessl registry
  audit     Audit and classify skills for publishing
  publish   Publish skills to a registry
  verify    Verify published skill integrity`,
	}

	cmd.AddCommand(newSkillsListCmd())
	cmd.AddCommand(newSkillsInstallCmd())
	cmd.AddCommand(newSkillsRemoveCmd())
	cmd.AddCommand(newSkillsSearchCmd())
	cmd.AddCommand(newSkillsSyncCmd())
	cmd.AddCommand(newSkillsAuditCmd())
	cmd.AddCommand(newSkillsPublishCmd())
	cmd.AddCommand(newSkillsVerifyCmd())

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
	var ontologyOnly bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List installed skills",
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

	usage := collectSkillPipelineUsage()

	output := SkillListOutput{
		Skills:   make([]SkillListItem, 0, len(skills)),
		Warnings: warnings,
	}
	for _, s := range skills {
		if ontologyOnly && !strings.HasPrefix(s.Name, "wave-ctx-") {
			continue
		}
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
		return json.NewEncoder(cmd.OutOrStdout()).Encode(output)
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
		return json.NewEncoder(cmd.OutOrStdout()).Encode(output)
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
		return json.NewEncoder(cmd.OutOrStdout()).Encode(output)
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
	ctx, cancel := context.WithTimeout(ctx, tesslTimeout)
	defer cancel()

	var stderr bytes.Buffer
	searchCmd := exec.CommandContext(ctx, "tessl", "search", query)
	searchCmd.Stderr = &stderr
	out, err := searchCmd.Output()
	if err != nil {
		detail := fmt.Sprintf("tessl search failed: %v", err)
		if stderr.Len() > 0 {
			detail += ": " + strings.TrimSpace(stderr.String())
		}
		return NewCLIError(CodeSkillSourceError, detail,
			"Check that tessl is configured correctly")
	}

	results := parseTesslSearchOutput(string(out))

	switch format {
	case "json":
		return json.NewEncoder(cmd.OutOrStdout()).Encode(results)
	default:
		renderSearchTable(cmd.OutOrStdout(), results)
		return nil
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

func renderSearchTable(w io.Writer, results []SkillSearchResult) {
	if len(results) == 0 {
		fmt.Fprintln(w, "No results found.")
		return
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
	ctx, cancel := context.WithTimeout(ctx, tesslTimeout)
	defer cancel()

	var stderr bytes.Buffer
	syncCmd := exec.CommandContext(ctx, "tessl", "install", "--project-dependencies")
	syncCmd.Stderr = &stderr
	out, err := syncCmd.Output()
	if err != nil {
		detail := fmt.Sprintf("tessl sync failed: %v", err)
		if stderr.Len() > 0 {
			detail += ": " + strings.TrimSpace(stderr.String())
		}
		return NewCLIError(CodeSkillSourceError, detail,
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
		return json.NewEncoder(cmd.OutOrStdout()).Encode(output)
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

// --- Audit ---

func newSkillsAuditCmd() *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Audit and classify skills for publishing",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSkillsAudit(cmd, format)
		},
	}
	cmd.Flags().StringVar(&format, "format", "table", "Output format: table, json")
	return cmd
}

func runSkillsAudit(cmd *cobra.Command, format string) error {
	format = ResolveFormat(cmd, format)

	store := newSkillStore()
	classifications, err := skill.ClassifyAll(store)
	if err != nil {
		return err
	}

	output := SkillAuditOutput{
		Skills: make([]SkillAuditItem, 0, len(classifications)),
	}

	for _, c := range classifications {
		item := SkillAuditItem{
			Name:           c.Name,
			Classification: c.Tag,
			WaveRefCount:   c.WaveRefCount,
			Warnings:       c.Warnings,
			Source:         c.SourcePath,
		}
		output.Skills = append(output.Skills, item)

		switch c.Tag {
		case skill.TagStandalone:
			output.Summary.Standalone++
		case skill.TagWaveSpecific:
			output.Summary.WaveSpecific++
		case skill.TagBoth:
			output.Summary.Both++
		}
	}
	output.Summary.Total = len(classifications)

	switch format {
	case "json":
		return json.NewEncoder(cmd.OutOrStdout()).Encode(output)
	default:
		return renderSkillsAuditTable(cmd.OutOrStdout(), output)
	}
}

func renderSkillsAuditTable(w io.Writer, output SkillAuditOutput) error {
	f := display.NewFormatter()

	if len(output.Skills) == 0 {
		fmt.Fprintln(w, "No skills found.")
		return nil
	}

	fmt.Fprintln(w)
	fmt.Fprintf(w, "%-25s %-18s %-12s %-30s %s\n",
		f.Colorize("NAME", "\033[1;37m"),
		f.Colorize("CLASSIFICATION", "\033[1;37m"),
		f.Colorize("WAVE REFS", "\033[1;37m"),
		f.Colorize("WARNINGS", "\033[1;37m"),
		f.Colorize("SOURCE", "\033[1;37m"))

	for _, s := range output.Skills {
		tag := s.Classification
		switch s.Classification {
		case skill.TagStandalone:
			tag = f.Success(s.Classification)
		case skill.TagWaveSpecific:
			tag = f.Error(s.Classification)
		case skill.TagBoth:
			tag = f.Warning(s.Classification)
		}

		warns := ""
		if len(s.Warnings) > 0 {
			warns = strings.Join(s.Warnings, ", ")
		}

		fmt.Fprintf(w, "%-25s %-18s %-12d %-30s %s\n",
			f.Primary(s.Name),
			tag,
			s.WaveRefCount,
			warns,
			f.Muted(s.Source))
	}

	fmt.Fprintf(w, "\n%d standalone, %d wave-specific, %d both (%d total)\n",
		output.Summary.Standalone, output.Summary.WaveSpecific,
		output.Summary.Both, output.Summary.Total)

	return nil
}

// --- Publish ---

func newSkillsPublishCmd() *cobra.Command {
	var format string
	var force, dryRun, all bool
	var registry string
	cmd := &cobra.Command{
		Use:   "publish [name]",
		Short: "Publish skills to a registry",
		Long: `Publish a skill to the Tessl registry.

Examples:
  wave skills publish golang             # Publish a single skill
  wave skills publish --all              # Publish all standalone skills
  wave skills publish golang --dry-run   # Validate without publishing
  wave skills publish golang --force     # Force publish wave-specific skills`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if all && len(args) > 0 {
				return NewCLIError(CodeFlagConflict,
					"--all cannot be used with a skill name argument",
					"Use either `wave skills publish <name>` or `wave skills publish --all`")
			}
			if !all && len(args) == 0 {
				return NewCLIError(CodeInvalidArgs,
					"skill name is required (or use --all)",
					"Usage: wave skills publish <name> or wave skills publish --all")
			}
			name := ""
			if len(args) > 0 {
				name = args[0]
			}
			return runSkillsPublish(cmd, name, format, all, force, dryRun, registry)
		},
	}
	cmd.Flags().StringVar(&format, "format", "table", "Output format: table, json")
	cmd.Flags().BoolVar(&force, "force", false, "Force publish wave-specific skills")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate and compute digest without publishing")
	cmd.Flags().BoolVar(&all, "all", false, "Publish all standalone-eligible skills")
	cmd.Flags().StringVar(&registry, "registry", "tessl", "Target registry name")
	return cmd
}

func runSkillsPublish(cmd *cobra.Command, name, format string, all, force, dryRun bool, registry string) error {
	format = ResolveFormat(cmd, format)

	store := newSkillStore()
	lockfilePath := ".wave/skills.lock"
	publisher := skill.NewPublisher(store, lockfilePath, registry, exec.LookPath)

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	opts := skill.PublishOpts{
		Force:    force,
		DryRun:   dryRun,
		Registry: registry,
	}

	var results []skill.PublishResult

	if all {
		var err error
		results, err = publisher.PublishAll(ctx, opts)
		if err != nil {
			return NewCLIError(CodeSkillPublishFailed, err.Error(),
				"Check skill store and retry")
		}
	} else {
		result := publisher.PublishOne(ctx, name, opts)
		results = append(results, result)
	}

	output := SkillPublishOutput{
		Results:  make([]PublishResultItem, 0, len(results)),
		Lockfile: lockfilePath,
	}

	for _, r := range results {
		item := PublishResultItem{
			Name:     r.Name,
			Digest:   r.Digest,
			URL:      r.URL,
			Warnings: r.Warnings,
		}
		switch {
		case r.Skipped:
			item.Status = "skipped"
			item.Reason = r.SkipReason
		case r.Success:
			item.Status = "published"
		default:
			item.Status = "failed"
			item.Reason = r.Error
		}

		// Check body size against agentskills.io recommendation
		if s, readErr := store.Read(r.Name); readErr == nil {
			bodyLines := strings.Count(s.Body, "\n")
			if bodyLines > 500 {
				item.Warnings = append(item.Warnings, fmt.Sprintf("SKILL.md body exceeds 500 lines (%d lines) — consider splitting into core + references/", bodyLines))
			}
		}

		output.Results = append(output.Results, item)
	}

	switch format {
	case "json":
		return json.NewEncoder(cmd.OutOrStdout()).Encode(output)
	default:
		return renderSkillsPublishTable(cmd.OutOrStdout(), output)
	}
}

func renderSkillsPublishTable(w io.Writer, output SkillPublishOutput) error {
	f := display.NewFormatter()

	published, skipped, failed := 0, 0, 0
	for _, r := range output.Results {
		var status string
		switch r.Status {
		case "published":
			status = f.Success("Published")
			published++
		case "skipped":
			status = f.Warning("Skipped")
			skipped++
		case "failed":
			status = f.Error("Failed")
			failed++
		}

		detail := r.URL
		if r.Reason != "" {
			detail = r.Reason
		}

		fmt.Fprintf(w, "  %s %s", status, f.Primary(r.Name))
		if detail != "" {
			fmt.Fprintf(w, " — %s", detail)
		}
		fmt.Fprintln(w)

		for _, warn := range r.Warnings {
			fmt.Fprintf(w, "    %s %s\n", f.Warning("warning:"), warn)
		}
	}

	if len(output.Results) > 1 {
		fmt.Fprintf(w, "\n%d published, %d skipped, %d failed\n", published, skipped, failed)
	}

	return nil
}

// --- Verify ---

func newSkillsVerifyCmd() *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify published skill integrity",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSkillsVerify(cmd, format)
		},
	}
	cmd.Flags().StringVar(&format, "format", "table", "Output format: table, json")
	return cmd
}

func runSkillsVerify(cmd *cobra.Command, format string) error {
	format = ResolveFormat(cmd, format)

	lockfilePath := ".wave/skills.lock"
	lf, err := skill.LoadLockfile(lockfilePath)
	if err != nil {
		return err
	}

	if len(lf.Published) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No published skills.")
		return nil
	}

	store := newSkillStore()
	output := SkillVerifyOutput{
		Results: make([]SkillVerifyItem, 0, len(lf.Published)),
	}

	for _, rec := range lf.Published {
		item := SkillVerifyItem{
			Name:           rec.Name,
			ExpectedDigest: rec.Digest,
		}

		s, readErr := store.Read(rec.Name)
		if readErr != nil {
			item.Status = "missing"
			output.Summary.Missing++
		} else {
			digest, digestErr := skill.ComputeDigest(s)
			if digestErr != nil {
				item.Status = "missing"
				output.Summary.Missing++
			} else {
				item.ActualDigest = digest
				if digest == rec.Digest {
					item.Status = "ok"
					output.Summary.OK++
				} else {
					item.Status = "modified"
					output.Summary.Modified++
				}
			}

			// Check body size against agentskills.io recommendation
			bodyLines := strings.Count(s.Body, "\n")
			if bodyLines > 500 {
				item.Warnings = append(item.Warnings, fmt.Sprintf("SKILL.md body exceeds 500 lines (%d lines) — consider splitting into core + references/", bodyLines))
			}
		}

		output.Results = append(output.Results, item)
	}
	output.Summary.Total = len(lf.Published)

	switch format {
	case "json":
		return json.NewEncoder(cmd.OutOrStdout()).Encode(output)
	default:
		return renderSkillsVerifyTable(cmd.OutOrStdout(), output)
	}
}

func renderSkillsVerifyTable(w io.Writer, output SkillVerifyOutput) error {
	f := display.NewFormatter()

	fmt.Fprintln(w)
	fmt.Fprintf(w, "%-25s %-12s %-30s %s\n",
		f.Colorize("NAME", "\033[1;37m"),
		f.Colorize("STATUS", "\033[1;37m"),
		f.Colorize("EXPECTED", "\033[1;37m"),
		f.Colorize("ACTUAL", "\033[1;37m"))

	for _, r := range output.Results {
		var status string
		switch r.Status {
		case "ok":
			status = f.Success("ok")
		case "modified":
			status = f.Warning("modified")
		case "missing":
			status = f.Error("missing")
		}

		expected := r.ExpectedDigest
		if len(expected) > 20 {
			expected = expected[:20] + "..."
		}
		actual := r.ActualDigest
		if len(actual) > 20 {
			actual = actual[:20] + "..."
		}

		fmt.Fprintf(w, "%-25s %-12s %-30s %s\n",
			f.Primary(r.Name),
			status,
			expected,
			actual)

		for _, warn := range r.Warnings {
			fmt.Fprintf(w, "    %s %s\n", f.Warning("warning:"), warn)
		}
	}

	fmt.Fprintf(w, "\n%d ok, %d modified, %d missing\n",
		output.Summary.OK, output.Summary.Modified, output.Summary.Missing)

	return nil
}
