package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/recinq/wave/internal/doctor"
	"github.com/spf13/cobra"
)

// NewDoctorCmd creates the doctor command for checking project health.
func NewDoctorCmd() *cobra.Command {
	var fixFlag bool
	var skipCodebase bool

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check project health and environment setup",
		Long: `Run diagnostic checks on your Wave project configuration, tools,
and environment. Reports issues with remediation hints.

Exit codes:
  0  All checks passed
  1  Warnings detected (non-blocking)
  2  Errors detected (action required)`,
		Example: `  wave doctor
  wave doctor --json
  wave doctor --fix`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true

			outputCfg := GetOutputConfig(cmd)
			manifestPath, _ := cmd.Root().PersistentFlags().GetString("manifest")

			report, err := doctor.RunChecks(context.Background(), doctor.Options{
				ManifestPath: manifestPath,
				Fix:          fixFlag,
				SkipCodebase: skipCodebase,
			})
			if err != nil {
				return fmt.Errorf("doctor check failed: %w", err)
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
			case doctor.StatusErr:
				os.Exit(2)
			case doctor.StatusWarn:
				os.Exit(1)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&fixFlag, "fix", false, "Auto-install missing dependencies where possible")
	cmd.Flags().BoolVar(&skipCodebase, "skip-codebase", false, "Skip forge API codebase analysis")

	return cmd
}

func renderDoctorJSON(report *doctor.Report) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}

func renderDoctorText(report *doctor.Report) {
	for _, r := range report.Results {
		var icon string
		switch r.Status {
		case doctor.StatusOK:
			icon = "✓"
		case doctor.StatusWarn:
			icon = "!"
		case doctor.StatusErr:
			icon = "✗"
		}

		fmt.Fprintf(os.Stdout, "  %s %s: %s\n", icon, r.Name, r.Message)
		if r.Fix != "" && r.Status != doctor.StatusOK {
			fmt.Fprintf(os.Stdout, "    Fix: %s\n", r.Fix)
		}
	}

	fmt.Fprintln(os.Stdout)

	switch report.Summary {
	case doctor.StatusOK:
		fmt.Fprintln(os.Stdout, "All checks passed.")
	case doctor.StatusWarn:
		fmt.Fprintln(os.Stdout, "Some checks have warnings.")
	case doctor.StatusErr:
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
