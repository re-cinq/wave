package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/recinq/wave/internal/state"
	"github.com/spf13/cobra"
)

// DecisionsOptions holds options for the decisions command.
type DecisionsOptions struct {
	RunID    string
	Step     string
	Format   string
	Category string
	Manifest string
}

// DecisionsOutput represents the JSON output for the decisions command.
type DecisionsOutput struct {
	RunID     string          `json:"run_id"`
	Decisions []DecisionEntry `json:"decisions"`
}

// DecisionEntry represents a single decision in the output.
type DecisionEntry struct {
	ID        int64           `json:"id"`
	Timestamp string          `json:"timestamp"`
	StepID    string          `json:"step_id,omitempty"`
	Category  string          `json:"category"`
	Decision  string          `json:"decision"`
	Rationale string          `json:"rationale,omitempty"`
	Context   json.RawMessage `json:"context,omitempty"`
}

// NewDecisionsCmd creates the decisions command.
func NewDecisionsCmd() *cobra.Command {
	var opts DecisionsOptions

	cmd := &cobra.Command{
		Use:   "decisions [run-id]",
		Short: "Show decision log for a pipeline run",
		Long: `Show the structured decision log from pipeline runs.

Displays model routing, retry, contract validation, and other decisions
made during pipeline execution. Useful for understanding why the orchestrator
chose specific models, retried steps, or handled contract failures.

Without arguments, shows decisions from the most recent run.
With a run-id argument, shows decisions for that specific run.`,
		Example: `  wave decisions                          # Decisions from most recent run
  wave decisions run-20260328-143022      # Decisions for specific run
  wave decisions --step implement         # Filter by step ID
  wave decisions --category retry         # Filter by decision category
  wave decisions --format json            # Output as JSON`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.RunID = args[0]
			}
			opts.Format = ResolveFormat(cmd, opts.Format)
			return runDecisions(opts)
		},
	}

	cmd.Flags().StringVar(&opts.Step, "step", "", "Filter by step ID")
	cmd.Flags().StringVar(&opts.Category, "category", "", "Filter by category (model_routing, retry, contract, budget, composition)")
	cmd.Flags().StringVar(&opts.Format, "format", "text", "Output format (text, json)")
	cmd.Flags().StringVar(&opts.Manifest, "manifest", "wave.yaml", "Path to manifest file")

	return cmd
}

func runDecisions(opts DecisionsOptions) error {
	dbPath := ".agents/state.db"

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		if opts.Format == "json" {
			fmt.Println(`{"run_id":"","decisions":[]}`)
			return nil
		}
		fmt.Println("No decisions found (state database does not exist)")
		return nil
	}

	store, err := state.NewReadOnlyStateStore(dbPath)
	if err != nil {
		return NewCLIError(CodeStateDBError, fmt.Sprintf("failed to open state database: %s", err), "Check .agents/state.db file permissions or run 'wave run' to create it").WithCause(err)
	}
	defer store.Close()

	// Resolve run ID if not provided
	runID := opts.RunID
	if runID == "" {
		runID, err = store.GetMostRecentRunID()
		if err != nil {
			return NewCLIError(CodeInternalError, fmt.Sprintf("failed to query most recent run: %s", err), "The state database may be corrupted -- try 'wave migrate validate'").WithCause(err)
		}
		if runID == "" {
			if opts.Format == "json" {
				fmt.Println(`{"run_id":"","decisions":[]}`)
				return nil
			}
			fmt.Println("No pipeline runs found")
			return nil
		}
	}

	exists, err := store.RunExists(runID)
	if err != nil {
		return NewCLIError(CodeInternalError, fmt.Sprintf("failed to verify run: %s", err), "The state database may be corrupted -- try 'wave migrate validate'").WithCause(err)
	}
	if !exists {
		if opts.Format == "json" {
			fmt.Printf(`{"run_id":"%s","decisions":[],"error":"run not found"}`, runID)
			fmt.Println()
			return nil
		}
		fmt.Printf("Run not found: %s\n", runID)
		return nil
	}

	records, err := store.GetDecisionsFiltered(runID, state.DecisionQueryOptions{
		StepID:   opts.Step,
		Category: opts.Category,
	})
	if err != nil {
		return NewCLIError(CodeInternalError, fmt.Sprintf("failed to query decisions: %s", err), "The state database may need migration -- try 'wave migrate up'").WithCause(err)
	}

	decisions := decisionRecordsToEntries(records)

	if len(decisions) == 0 {
		if opts.Format == "json" {
			output := DecisionsOutput{RunID: runID, Decisions: []DecisionEntry{}}
			jsonBytes, _ := json.MarshalIndent(output, "", "  ")
			fmt.Println(string(jsonBytes))
			return nil
		}
		fmt.Printf("No decisions found for run: %s\n", runID)
		return nil
	}

	return outputDecisions(runID, decisions, opts)
}

// decisionRecordsToEntries converts state.DecisionRecord pointers into the
// CLI's presentation DTO. Empty/`{}` context fields are stripped to keep the
// JSON shape backward compatible with the prior raw-SQL implementation.
func decisionRecordsToEntries(records []*state.DecisionRecord) []DecisionEntry {
	out := make([]DecisionEntry, 0, len(records))
	for _, r := range records {
		entry := DecisionEntry{
			ID:        r.ID,
			Timestamp: r.Timestamp.Format("15:04:05"),
			StepID:    r.StepID,
			Category:  r.Category,
			Decision:  r.Decision,
			Rationale: r.Rationale,
		}
		if r.Context != "" && r.Context != "{}" {
			entry.Context = json.RawMessage(r.Context)
		}
		out = append(out, entry)
	}
	return out
}

// outputDecisions formats and outputs the decisions.
func outputDecisions(runID string, decisions []DecisionEntry, opts DecisionsOptions) error {
	if opts.Format == "json" {
		output := DecisionsOutput{RunID: runID, Decisions: decisions}
		jsonBytes, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return NewCLIError(CodeInternalError, fmt.Sprintf("failed to marshal JSON: %s", err), "This is an internal serialization error").WithCause(err)
		}
		fmt.Println(string(jsonBytes))
		return nil
	}

	for _, d := range decisions {
		printDecisionEntry(d)
	}

	return nil
}

// printDecisionEntry prints a single decision entry in text format.
func printDecisionEntry(d DecisionEntry) {
	var parts []string
	parts = append(parts, fmt.Sprintf("[%s]", d.Timestamp))

	categoryStr := fmt.Sprintf("%-15s", d.Category)
	switch d.Category {
	case "model_routing":
		categoryStr = fmt.Sprintf("\033[36m%-15s\033[0m", d.Category) // cyan
	case "retry":
		categoryStr = fmt.Sprintf("\033[33m%-15s\033[0m", d.Category) // yellow
	case "contract":
		categoryStr = fmt.Sprintf("\033[35m%-15s\033[0m", d.Category) // magenta
	case "budget":
		categoryStr = fmt.Sprintf("\033[31m%-15s\033[0m", d.Category) // red
	case "composition":
		categoryStr = fmt.Sprintf("\033[34m%-15s\033[0m", d.Category) // blue
	}
	parts = append(parts, categoryStr)

	if d.StepID != "" {
		parts = append(parts, fmt.Sprintf("[%s]", d.StepID))
	}

	parts = append(parts, d.Decision)

	if d.Rationale != "" {
		parts = append(parts, fmt.Sprintf("(%s)", d.Rationale))
	}

	fmt.Println(strings.Join(parts, " "))
}
