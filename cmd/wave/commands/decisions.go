package commands

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	_ "modernc.org/sqlite"
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
	dbPath := ".wave/state.db"

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		if opts.Format == "json" {
			fmt.Println(`{"run_id":"","decisions":[]}`)
			return nil
		}
		fmt.Println("No decisions found (state database does not exist)")
		return nil
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return NewCLIError(CodeStateDBError, fmt.Sprintf("failed to open state database: %s", err), "Check .wave/state.db file permissions or run 'wave run' to create it").WithCause(err)
	}
	defer db.Close()

	db.SetMaxOpenConns(1)

	// Resolve run ID if not provided
	runID := opts.RunID
	if runID == "" {
		runID, err = getMostRecentRunID(db)
		if err != nil {
			if opts.Format == "json" {
				fmt.Println(`{"run_id":"","decisions":[]}`)
				return nil
			}
			fmt.Println("No pipeline runs found")
			return nil
		}
	}

	// Verify run exists
	if !runExists(db, runID) {
		if opts.Format == "json" {
			fmt.Printf(`{"run_id":"%s","decisions":[],"error":"run not found"}`, runID)
			fmt.Println()
			return nil
		}
		fmt.Printf("Run not found: %s\n", runID)
		return nil
	}

	decisions, err := queryDecisions(db, runID, opts)
	if err != nil {
		return err
	}

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

// queryDecisions retrieves decisions matching the options.
func queryDecisions(db *sql.DB, runID string, opts DecisionsOptions) ([]DecisionEntry, error) {
	query := `SELECT id, timestamp, step_id, category, decision, rationale, context_json
	          FROM decision_log
	          WHERE run_id = ?`
	args := []any{runID}

	if opts.Step != "" {
		query += " AND step_id = ?"
		args = append(args, opts.Step)
	}

	if opts.Category != "" {
		query += " AND category = ?"
		args = append(args, opts.Category)
	}

	query += " ORDER BY timestamp ASC, id ASC"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, NewCLIError(CodeInternalError, fmt.Sprintf("failed to query decisions: %s", err), "The state database may need migration -- try 'wave migrate up'").WithCause(err)
	}
	defer rows.Close()

	var decisions []DecisionEntry
	for rows.Next() {
		var d DecisionEntry
		var timestamp int64
		var stepID, rationale, contextJSON sql.NullString

		err := rows.Scan(
			&d.ID,
			&timestamp,
			&stepID,
			&d.Category,
			&d.Decision,
			&rationale,
			&contextJSON,
		)
		if err != nil {
			return nil, NewCLIError(CodeInternalError, fmt.Sprintf("failed to scan decision: %s", err), "The state database may have schema issues -- try 'wave migrate up'").WithCause(err)
		}

		d.Timestamp = time.Unix(timestamp, 0).Format("15:04:05")
		if stepID.Valid {
			d.StepID = stepID.String
		}
		if rationale.Valid {
			d.Rationale = rationale.String
		}
		if contextJSON.Valid && contextJSON.String != "" && contextJSON.String != "{}" {
			d.Context = json.RawMessage(contextJSON.String)
		}

		decisions = append(decisions, d)
	}

	if err := rows.Err(); err != nil {
		return nil, NewCLIError(CodeInternalError, fmt.Sprintf("error iterating decisions: %s", err), "The state database may have schema issues -- try 'wave migrate up'").WithCause(err)
	}

	return decisions, nil
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

	// Text format
	for _, d := range decisions {
		printDecisionEntry(d)
	}

	return nil
}

// printDecisionEntry prints a single decision entry in text format.
func printDecisionEntry(d DecisionEntry) {
	var parts []string
	parts = append(parts, fmt.Sprintf("[%s]", d.Timestamp))

	// Color-code categories
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
