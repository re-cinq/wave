package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/recinq/wave/internal/doctor"
	"github.com/recinq/wave/internal/onboarding"
	"github.com/recinq/wave/internal/suggest"
	"github.com/spf13/cobra"
)

// NewSuggestCmd creates the suggest command for proposing pipeline runs.
func NewSuggestCmd() *cobra.Command {
	var limitFlag int
	var dryRunFlag bool

	cmd := &cobra.Command{
		Use:   "suggest",
		Short: "Propose pipeline runs based on codebase state",
		Long: `Analyze codebase health and suggest pipeline runs that would
be most impactful. Combines doctor diagnostics with forge API data
to generate prioritized recommendations.`,
		Example: `  wave suggest
  wave suggest --limit 3
  wave suggest --json
  wave suggest --dry-run`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true

			outputCfg := GetOutputConfig(cmd)
			manifestPath, _ := cmd.Root().PersistentFlags().GetString("manifest")

			// Run doctor checks first to get codebase state
			report, err := doctor.RunChecks(context.Background(), doctor.Options{
				ManifestPath:   manifestPath,
				CheckOnboarded: onboarding.IsOnboarded,
			})
			if err != nil {
				return fmt.Errorf("codebase analysis failed: %w", err)
			}

			// Generate suggestions
			proposal, err := suggest.Suggest(suggest.EngineOptions{
				Report: report,
				Limit:  limitFlag,
			})
			if err != nil {
				return fmt.Errorf("suggestion engine failed: %w", err)
			}

			format := ResolveFormat(cmd, "text")
			if outputCfg.Format == OutputFormatJSON {
				format = "json"
			}

			switch format {
			case "json":
				return renderSuggestJSON(proposal)
			default:
				renderSuggestText(proposal, dryRunFlag)
			}

			return nil
		},
	}

	cmd.Flags().IntVar(&limitFlag, "limit", 5, "Maximum number of suggestions")
	cmd.Flags().BoolVar(&dryRunFlag, "dry-run", false, "Show what would be suggested without executing")

	return cmd
}

func renderSuggestJSON(proposal *suggest.Proposal) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(proposal)
}

func renderSuggestText(proposal *suggest.Proposal, dryRun bool) {
	if len(proposal.Pipelines) == 0 {
		fmt.Fprintln(os.Stdout, "No pipeline suggestions available.")
		fmt.Fprintln(os.Stdout, "  Your codebase looks healthy, or no matching pipelines were found.")
		return
	}

	fmt.Fprintln(os.Stdout, "Suggested pipelines:")
	fmt.Fprintln(os.Stdout)

	for i, p := range proposal.Pipelines {
		fmt.Fprintf(os.Stdout, "  %d. [P%d] %s\n", i+1, p.Priority, p.Name)
		fmt.Fprintf(os.Stdout, "     Reason: %s\n", p.Reason)
		if p.Input != "" {
			fmt.Fprintf(os.Stdout, "     Input:  %s\n", p.Input)
		}
		fmt.Fprintln(os.Stdout)
	}

	if dryRun {
		fmt.Fprintln(os.Stdout, "(dry run — no pipelines executed)")
	} else {
		fmt.Fprintf(os.Stdout, "Run: wave compose %s\n", proposal.Pipelines[0].Name)
	}
}
