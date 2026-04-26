package commands

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/recinq/wave/internal/doctor"
	"github.com/recinq/wave/internal/forge"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/onboarding"
	"github.com/recinq/wave/internal/suggest"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// NewDoctorCmd creates the doctor command for checking project health.
func NewDoctorCmd() *cobra.Command {
	var fixFlag bool
	var skipCodebase bool
	var optimizeFlag bool
	var dryRunFlag bool
	var skipAIFlag bool
	var yesFlag bool

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check project health and environment setup",
		Long: `Run diagnostic checks on your Wave project configuration, tools,
and environment. Reports issues with remediation hints.

Use --optimize to scan the project and propose wave.yaml improvements based on
detected languages, build tools, test runners, and conventions.

Exit codes:
  0  All checks passed
  1  Warnings detected (non-blocking)
  2  Errors detected (action required)`,
		Example: `  wave doctor
  wave doctor --json
  wave doctor --fix
  wave doctor --optimize
  wave doctor --optimize --dry-run
  wave doctor --optimize --yes`,
		Args: cobra.NoArgs,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Validate flag dependencies: --dry-run, --skip-ai, --yes require --optimize
			if !optimizeFlag {
				if dryRunFlag {
					return NewCLIError(CodeFlagConflict, "--dry-run requires --optimize", "Add --optimize to use --dry-run")
				}
				if skipAIFlag {
					return NewCLIError(CodeFlagConflict, "--skip-ai requires --optimize", "Add --optimize to use --skip-ai")
				}
				if yesFlag {
					return NewCLIError(CodeFlagConflict, "--yes requires --optimize", "Add --optimize to use --yes")
				}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true

			if optimizeFlag {
				return runOptimize(cmd, optimizeOpts{
					dryRun: dryRunFlag,
					skipAI: skipAIFlag,
					yes:    yesFlag,
				})
			}

			outputCfg := GetOutputConfig(cmd)
			manifestPath, _ := cmd.Root().PersistentFlags().GetString("manifest")

			report, err := doctor.RunChecks(context.Background(), doctor.Options{
				ManifestPath:   manifestPath,
				Fix:            fixFlag,
				SkipCodebase:   skipCodebase,
				CheckOnboarded: onboarding.IsOnboarded,
			})
			if err != nil {
				return NewCLIError(CodeInternalError, fmt.Sprintf("doctor check failed: %s", err), "A health check encountered an unexpected error").WithCause(err)
			}

			format := ResolveFormat(cmd, "text")
			if outputCfg.Format == OutputFormatJSON {
				format = "json"
			}

			switch format {
			case "json":
				return renderDoctorJSON(report)
			default:
				renderDoctorText(report)
			}

			// Exit code based on summary
			switch report.Summary {
			case suggest.StatusErr:
				os.Exit(2)
			case suggest.StatusWarn:
				os.Exit(1)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&fixFlag, "fix", false, "Auto-install missing dependencies where possible")
	cmd.Flags().BoolVar(&skipCodebase, "skip-codebase", false, "Skip forge API codebase analysis")
	cmd.Flags().BoolVar(&optimizeFlag, "optimize", false, "Scan project and propose wave.yaml improvements")
	cmd.Flags().BoolVar(&dryRunFlag, "dry-run", false, "Show proposed changes without writing (requires --optimize)")
	cmd.Flags().BoolVar(&skipAIFlag, "skip-ai", false, "Skip AI-powered analysis, deterministic scan only (requires --optimize)")
	cmd.Flags().BoolVarP(&yesFlag, "yes", "y", false, "Accept all proposed changes without confirmation (requires --optimize)")

	return cmd
}

// optimizeOpts holds the flags specific to the optimize sub-flow.
type optimizeOpts struct {
	dryRun bool
	skipAI bool
	yes    bool
}

// runOptimize performs the project scan and optimization flow.
func runOptimize(cmd *cobra.Command, opts optimizeOpts) error {
	outputCfg := GetOutputConfig(cmd)
	format := ResolveFormat(cmd, "text")
	if outputCfg.Format == OutputFormatJSON {
		format = "json"
	}

	// 1. Scan project
	fmt.Fprint(os.Stderr, "Scanning project...")

	var scanOpts []doctor.ScanOption
	if opts.skipAI {
		scanOpts = append(scanOpts, doctor.WithSkipAI())
	}

	profile, err := doctor.ScanProject(".", scanOpts...)
	if err != nil {
		fmt.Fprintln(os.Stderr)
		return NewCLIError(CodeInternalError, fmt.Sprintf("project scan failed: %s", err), "Project scanning encountered an error").WithCause(err)
	}

	fmt.Fprintf(os.Stderr, " done (%s)\n\n", formatScanSummary(profile))

	// 2. Load manifest for current project config
	manifestPath, _ := cmd.Root().PersistentFlags().GetString("manifest")
	if manifestPath == "" {
		manifestPath = "wave.yaml"
	}

	m, err := manifest.Load(manifestPath)
	if err != nil {
		return NewCLIError(CodeManifestInvalid, fmt.Sprintf("failed to load manifest: %s", err), "Check wave.yaml syntax -- run 'wave validate' to diagnose").WithCause(err)
	}

	// 3. Detect forge info
	fi, _ := forge.DetectFromGitRemotes()
	var fiPtr *forge.ForgeInfo
	if fi.Type != forge.ForgeUnknown {
		fiPtr = &fi
	}

	// 4. Discover available pipeline names
	pipelineNames := discoverPipelineNames(".agents/pipelines")

	// 5. Run optimization
	result := doctor.Optimize(profile, m.Project, fiPtr, pipelineNames)

	// 6. Render result
	if format == "json" {
		return renderOptimizeJSON(result)
	}

	renderOptimizeText(os.Stdout, result)

	// 7. Dry-run stops here
	if opts.dryRun {
		fmt.Fprintln(os.Stdout, "\n(dry-run mode, no changes written)")
		return nil
	}

	// 8. If no changes, nothing to do
	if !result.HasChanges() {
		fmt.Fprintln(os.Stdout, "\nNo changes to apply.")
		return nil
	}

	// 9. Prompt for confirmation unless --yes
	if !opts.yes {
		accepted, err := promptConfirm(os.Stdin, os.Stdout, "Accept changes? [Y/n] ")
		if err != nil {
			return NewCLIError(CodeInternalError, fmt.Sprintf("confirmation failed: %s", err), "Interactive confirmation failed").WithCause(err)
		}
		if !accepted {
			fmt.Fprintln(os.Stdout, "Aborted.")
			return nil
		}
	}

	// 10. Apply changes
	if err := applyOptimizeChanges(manifestPath, m, result, profile); err != nil {
		return NewCLIError(CodeInternalError, fmt.Sprintf("failed to apply changes: %s", err), "Check file permissions for wave.yaml").WithCause(err)
	}

	fmt.Fprintln(os.Stdout, "Changes applied.")
	return nil
}

// formatScanSummary returns a short summary string for the scan result.
func formatScanSummary(profile *doctor.ProjectProfile) string {
	if profile == nil {
		return "no data"
	}
	if profile.FilesScanned > 0 {
		return fmt.Sprintf("%d files scanned", profile.FilesScanned)
	}
	return "scan complete"
}

// discoverPipelineNames reads .yaml/.yml filenames from the pipelines directory
// and returns them as pipeline names (extension stripped).
func discoverPipelineNames(pipelinesDir string) []string {
	entries, err := os.ReadDir(pipelinesDir)
	if err != nil {
		return nil
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ext)
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}

// promptConfirm shows the prompt and reads a Y/n response.
// Returns true for Y, y, or empty (default yes). Returns false for n/N.
func promptConfirm(in io.Reader, out io.Writer, prompt string) (bool, error) {
	fmt.Fprint(out, prompt)

	scanner := bufio.NewScanner(in)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return false, err
		}
		return false, nil // EOF
	}

	response := strings.TrimSpace(scanner.Text())
	switch strings.ToLower(response) {
	case "", "y", "yes":
		return true, nil
	case "n", "no":
		return false, nil
	default:
		return false, nil
	}
}

// applyOptimizeChanges writes the updated manifest and project profile.
func applyOptimizeChanges(manifestPath string, m *manifest.Manifest, result *doctor.OptimizeResult, profile *doctor.ProjectProfile) error {
	// Apply proposed changes to the project config
	newProject := result.ApplyTo(m.Project)

	// Read existing wave.yaml as a generic map to preserve non-project sections
	rawData, err := os.ReadFile(manifestPath)
	if err != nil {
		return NewCLIError(CodeManifestMissing, fmt.Sprintf("failed to read %s: %s", manifestPath, err), "Check that wave.yaml exists and is readable").WithCause(err)
	}

	var raw map[string]interface{}
	if err := yaml.Unmarshal(rawData, &raw); err != nil {
		return NewCLIError(CodeManifestInvalid, fmt.Sprintf("failed to parse %s: %s", manifestPath, err), "Check wave.yaml YAML syntax").WithCause(err)
	}

	// Marshal the new project block and unmarshal into a generic map
	projectBytes, err := yaml.Marshal(newProject)
	if err != nil {
		return NewCLIError(CodeInternalError, fmt.Sprintf("failed to marshal project config: %s", err), "Internal serialization error").WithCause(err)
	}

	var projectMap map[string]interface{}
	if err := yaml.Unmarshal(projectBytes, &projectMap); err != nil {
		return NewCLIError(CodeInternalError, fmt.Sprintf("failed to unmarshal project config: %s", err), "Internal deserialization error").WithCause(err)
	}

	raw["project"] = projectMap

	// Write back wave.yaml
	outData, err := yaml.Marshal(raw)
	if err != nil {
		return NewCLIError(CodeInternalError, fmt.Sprintf("failed to marshal wave.yaml: %s", err), "Internal serialization error").WithCause(err)
	}

	if err := os.WriteFile(manifestPath, outData, 0o644); err != nil {
		return NewCLIError(CodeInternalError, fmt.Sprintf("failed to write %s: %s", manifestPath, err), "Check write permissions for wave.yaml").WithCause(err)
	}

	// Write project profile
	profilePath := filepath.Join(".agents", "project-profile.json")
	if err := os.MkdirAll(filepath.Dir(profilePath), 0o755); err != nil {
		return NewCLIError(CodeInternalError, fmt.Sprintf("failed to create directory for %s: %s", profilePath, err), "Check write permissions for .agents/ directory").WithCause(err)
	}

	profileData, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return NewCLIError(CodeInternalError, fmt.Sprintf("failed to marshal project profile: %s", err), "Internal serialization error").WithCause(err)
	}

	if err := os.WriteFile(profilePath, profileData, 0o644); err != nil {
		return NewCLIError(CodeInternalError, fmt.Sprintf("failed to write %s: %s", profilePath, err), "Check write permissions for .agents/ directory").WithCause(err)
	}

	return nil
}

// renderOptimizeJSON outputs the full optimization result as JSON.
func renderOptimizeJSON(result *doctor.OptimizeResult) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

// renderOptimizeText outputs the optimization result as a human-readable diff.
func renderOptimizeText(w io.Writer, result *doctor.OptimizeResult) {
	// Project changes
	if len(result.ProjectChanges) > 0 {
		fmt.Fprintln(w, "Proposed wave.yaml changes:")
		fmt.Fprintln(w)

		for _, c := range result.ProjectChanges {
			renderConfigChange(w, c)
		}
	} else {
		fmt.Fprintln(w, "No wave.yaml changes proposed.")
		fmt.Fprintln(w)
	}

	// Pipeline recommendations
	if len(result.PipelineRecs) > 0 {
		fmt.Fprintln(w, "Pipeline recommendations:")

		for _, rec := range result.PipelineRecs {
			renderPipelineRec(w, rec)
		}
		fmt.Fprintln(w)
	}

	// Detected conventions
	if len(result.Conventions) > 0 {
		fmt.Fprintln(w, "Detected conventions:")

		for _, conv := range result.Conventions {
			fmt.Fprintf(w, "  * %s\n", conv)
		}
		fmt.Fprintln(w)
	}
}

// renderConfigChange renders a single config change with diff-style markers.
func renderConfigChange(w io.Writer, c doctor.ConfigChange) {
	if c.Current == c.Proposed {
		// No change — show confirmation
		sourceNote := ""
		if c.Source != "" {
			sourceNote = fmt.Sprintf(" (source: %s)", c.Source)
		}
		fmt.Fprintf(w, "  %s:\n", c.Field)
		fmt.Fprintf(w, "    current:  %s  (no change)%s\n", displayValue(c.Current), sourceNote)
		fmt.Fprintln(w)
		return
	}

	// Changed value — show diff
	fmt.Fprintf(w, "  %s:\n", c.Field)

	currentDisplay := displayValue(c.Current)
	proposedDisplay := displayValue(c.Proposed)
	reasonNote := ""
	if c.Reason != "" {
		reasonNote = fmt.Sprintf("  (%s)", c.Reason)
	}

	fmt.Fprintf(w, "  - current:  %s\n", currentDisplay)
	fmt.Fprintf(w, "  + proposed: %s%s\n", proposedDisplay, reasonNote)
	fmt.Fprintln(w)
}

// renderPipelineRec renders a single pipeline recommendation.
func renderPipelineRec(w io.Writer, rec doctor.PipelineRecommendation) {
	icon := "x"
	if rec.Recommended {
		icon = "+"
	}
	fmt.Fprintf(w, "  %s %-20s %s\n", icon, rec.Name, rec.Reason)
}

// displayValue formats a config value for display, showing "(not set)" for empty.
func displayValue(v string) string {
	if v == "" {
		return "(not set)"
	}
	return v
}

func renderDoctorJSON(report *suggest.Report) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}

func renderDoctorText(report *suggest.Report) {
	for _, r := range report.Results {
		var icon string
		switch r.Status {
		case suggest.StatusOK:
			icon = "✓"
		case suggest.StatusWarn:
			icon = "!"
		case suggest.StatusErr:
			icon = "✗"
		}

		fmt.Fprintf(os.Stdout, "  %s %s: %s\n", icon, r.Name, r.Message)
		if r.Fix != "" && r.Status != suggest.StatusOK {
			fmt.Fprintf(os.Stdout, "    Fix: %s\n", r.Fix)
		}
	}

	fmt.Fprintln(os.Stdout)

	switch report.Summary {
	case suggest.StatusOK:
		fmt.Fprintln(os.Stdout, "All checks passed.")
	case suggest.StatusWarn:
		fmt.Fprintln(os.Stdout, "Some checks have warnings.")
	case suggest.StatusErr:
		fmt.Fprintln(os.Stdout, "Some checks failed. See above for remediation.")
	}

	if report.ForgeInfo != nil {
		fmt.Fprintf(os.Stdout, "\nForge: %s (%s)\n", report.ForgeInfo.Type, report.ForgeInfo.Slug())
	}

	if report.Codebase != nil {
		cb := report.Codebase
		fmt.Fprintln(os.Stdout, "\nCodebase:")
		fmt.Fprintf(os.Stdout, "  PRs: %d open, %d needs review, %d stale\n",
			cb.PRs.Open, cb.PRs.NeedsReview, cb.PRs.Stale)
		fmt.Fprintf(os.Stdout, "  Issues: %d open, %d poor quality, %d unassigned\n",
			cb.Issues.Open, cb.Issues.PoorQuality, cb.Issues.Unassigned)
		fmt.Fprintf(os.Stdout, "  CI: %s (%d recent runs, %d failures)\n",
			cb.CI.Status, cb.CI.RecentRuns, cb.CI.Failures)
	}
}
