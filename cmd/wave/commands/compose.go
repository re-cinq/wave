package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/recinq/wave/internal/tui"
	"github.com/spf13/cobra"
)

// NewComposeCmd creates the compose command for validating and executing
// pipeline sequences.
func NewComposeCmd() *cobra.Command {
	var validateOnly bool

	cmd := &cobra.Command{
		Use:   "compose [pipelines...]",
		Short: "Validate and execute a pipeline sequence",
		Long: `Validate artifact compatibility between adjacent pipelines in a sequence
and optionally execute them in order.

The compose command checks that each pipeline's output artifacts match the
next pipeline's expected input artifacts. This ensures data flows correctly
across pipeline boundaries before execution begins.

Use --validate-only to check compatibility without executing.`,
		Example: `  wave compose speckit-flow wave-evolve wave-review
  wave compose speckit-flow wave-evolve --validate-only
  wave compose pipeline-a pipeline-b pipeline-c`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkOnboarding(); err != nil {
				return NewCLIError(CodeOnboardingRequired,
					"onboarding not complete",
					"Run 'wave init' to complete setup before running pipelines")
			}

			outputCfg := GetOutputConfig(cmd)
			if err := ValidateOutputFormat(outputCfg.Format); err != nil {
				return err
			}

			cmd.SilenceUsage = true
			cmd.SilenceErrors = true

			pDir := pipelinesDir()

			// Load all pipelines from arguments
			var seq tui.Sequence
			for _, name := range args {
				p, err := tui.LoadPipelineByName(pDir, name)
				if err != nil {
					return NewCLIError(CodePipelineNotFound,
						fmt.Sprintf("pipeline not found: %s", name),
						"Run 'wave list pipelines' to see available pipelines")
				}
				seq.Add(name, p)
			}

			// Validate artifact compatibility across the sequence
			result := tui.ValidateSequence(seq)

			if validateOnly {
				return renderValidationReport(args, result)
			}

			// Not validate-only: check for errors before execution
			if result.Status == tui.CompatibilityError {
				renderValidationReport(args, result) //nolint:errcheck
				return NewCLIError(CodeContractViolation,
					"sequence has incompatible artifact flows",
					"Run 'wave compose --validate-only' to see details, then fix pipeline artifacts")
			}

			// Sequence is valid or has warnings only — print informational message
			if result.Status == tui.CompatibilityWarning {
				fmt.Fprintf(os.Stderr, "Sequence validated with warnings:\n")
				for _, diag := range result.Diagnostics {
					fmt.Fprintf(os.Stderr, "  ! %s\n", diag)
				}
				fmt.Fprintln(os.Stderr)
			}

			fmt.Fprintf(os.Stderr, "Sequence %s is ready for execution.\n", formatSequenceArrow(args))
			fmt.Fprintf(os.Stderr, "Sequential execution is not yet implemented (see #249).\n")
			return nil
		},
	}

	cmd.Flags().BoolVar(&validateOnly, "validate-only", false, "Check compatibility without executing")

	return cmd
}

// renderValidationReport prints the compatibility report to stdout and
// returns an error if the result has CompatibilityError status.
func renderValidationReport(names []string, result tui.CompatibilityResult) error {
	fmt.Fprintf(os.Stdout, "Sequence validation: %s\n", formatSequenceArrow(names))
	fmt.Fprintln(os.Stdout)

	for i, flow := range result.Flows {
		fmt.Fprintf(os.Stdout, "Boundary %d: %s → %s\n", i+1, flow.SourcePipeline, flow.TargetPipeline)

		if len(flow.Matches) == 0 {
			fmt.Fprintf(os.Stdout, "  (no artifact flow)\n")
		}

		for _, match := range flow.Matches {
			switch match.Status {
			case tui.MatchCompatible:
				fmt.Fprintf(os.Stdout, "  ✓ %s → %s (compatible)\n", match.OutputName, match.InputName)
			case tui.MatchMissing:
				qualifier := "missing"
				if match.Optional {
					qualifier = "missing, optional"
				}
				fmt.Fprintf(os.Stdout, "  ✗ %s (%s — no matching output from %s)\n",
					match.InputName, qualifier, flow.SourcePipeline)
			case tui.MatchUnmatched:
				fmt.Fprintf(os.Stdout, "  ~ %s (output not consumed by %s)\n",
					match.OutputName, flow.TargetPipeline)
			}
		}

		fmt.Fprintln(os.Stdout)
	}

	// Count errors and warnings
	var errorCount, warningCount int
	for _, flow := range result.Flows {
		for _, match := range flow.Matches {
			if match.Status == tui.MatchMissing {
				if match.Optional {
					warningCount++
				} else {
					errorCount++
				}
			}
		}
	}

	fmt.Fprintf(os.Stdout, "Result: %d error(s), %d warning(s)\n", errorCount, warningCount)

	if result.Status == tui.CompatibilityError {
		return NewCLIError(CodeContractViolation,
			"sequence validation failed: incompatible artifact flows",
			"Fix pipeline artifacts to ensure outputs match expected inputs")
	}

	return nil
}

// formatSequenceArrow joins pipeline names with arrow separators.
func formatSequenceArrow(names []string) string {
	return strings.Join(names, " → ")
}
