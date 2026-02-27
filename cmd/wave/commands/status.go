package commands

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/recinq/wave/internal/display"
	"github.com/spf13/cobra"

	_ "modernc.org/sqlite"
)

// StatusOptions holds options for the status command.
type StatusOptions struct {
	All      bool   // Show all recent pipelines
	RunID    string // Specific run to show (from args)
	Format   string // table, json
	Manifest string
}

// StatusOutput represents the JSON output for status command.
type StatusOutput struct {
	Runs []StatusRunInfo `json:"runs"`
}

// StatusRunInfo holds status information about a pipeline run.
type StatusRunInfo struct {
	RunID       string `json:"run_id"`
	Pipeline    string `json:"pipeline"`
	Status      string `json:"status"`
	CurrentStep string `json:"current_step,omitempty"`
	Elapsed     string `json:"elapsed"`
	ElapsedMs   int64  `json:"elapsed_ms"`
	Tokens      int    `json:"tokens"`
	TokensStr   string `json:"tokens_str"`
	StartedAt   string `json:"started_at"`
	CompletedAt string `json:"completed_at,omitempty"`
	Input       string `json:"input,omitempty"`
	Error       string `json:"error,omitempty"`
}

// ANSI color codes for terminal output.
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorGray   = "\033[90m"
)

// NewStatusCmd creates the status command.
func NewStatusCmd() *cobra.Command {
	var opts StatusOptions

	cmd := &cobra.Command{
		Use:   "status [run-id]",
		Short: "Show pipeline status",
		Long: `Show the status of pipeline runs.

Without arguments, shows currently running pipelines.
With --all, shows recent pipelines (default 10).
With a run-id argument, shows detailed status for that specific run.

Examples:
  wave status                    # Show running pipelines
  wave status --all              # Show all recent pipelines
  wave status debug-20260202-143022  # Show specific run details
  wave status --format json      # Output as JSON for scripting`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.RunID = args[0]
			}
			return runStatus(opts)
		},
	}

	cmd.Flags().BoolVar(&opts.All, "all", false, "Show all recent pipelines (default 10)")
	cmd.Flags().StringVar(&opts.Format, "format", "table", "Output format (table, json)")
	cmd.Flags().StringVar(&opts.Manifest, "manifest", "wave.yaml", "Path to manifest file")

	return cmd
}

func runStatus(opts StatusOptions) error {
	dbPath := ".wave/state.db"

	// Check if state database exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		if opts.Format == "json" {
			fmt.Println(`{"runs":[]}`)
			return nil
		}
		fmt.Println("No pipelines found")
		return nil
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open state database: %w", err)
	}
	defer db.Close()

	// Configure SQLite for read-only access
	db.SetMaxOpenConns(1)

	if opts.RunID != "" {
		return showRunDetails(db, opts)
	}

	if opts.All {
		return showAllRuns(db, opts, 10)
	}

	return showRunningRuns(db, opts)
}

// showRunDetails shows detailed status for a specific run.
func showRunDetails(db *sql.DB, opts StatusOptions) error {
	run, err := queryRun(db, opts.RunID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			if opts.Format == "json" {
				fmt.Println(`{"runs":[],"error":"run not found"}`)
				return nil
			}
			fmt.Printf("Run not found: %s\n", opts.RunID)
			return nil
		}
		return err
	}

	if opts.Format == "json" {
		output := StatusOutput{Runs: []StatusRunInfo{run}}
		jsonBytes, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonBytes))
		return nil
	}

	// Detailed table view for single run
	fmt.Printf("Run ID:     %s\n", run.RunID)
	fmt.Printf("Pipeline:   %s\n", run.Pipeline)
	fmt.Printf("Status:     %s%s%s\n", statusColor(run.Status), run.Status, colorReset)
	if run.CurrentStep != "" {
		fmt.Printf("Step:       %s\n", run.CurrentStep)
	}
	fmt.Printf("Started:    %s\n", run.StartedAt)
	if run.CompletedAt != "" {
		fmt.Printf("Completed:  %s\n", run.CompletedAt)
	}
	fmt.Printf("Elapsed:    %s\n", run.Elapsed)
	fmt.Printf("Tokens:     %s\n", run.TokensStr)
	if run.Input != "" {
		// Truncate long input
		input := run.Input
		if len(input) > 100 {
			input = input[:97] + "..."
		}
		fmt.Printf("Input:      %s\n", input)
	}
	if run.Error != "" {
		fmt.Printf("Error:      %s\n", run.Error)
	}

	return nil
}

// showRunningRuns shows currently running pipelines.
func showRunningRuns(db *sql.DB, opts StatusOptions) error {
	runs, err := queryRunningRuns(db)
	if err != nil {
		return err
	}

	if len(runs) == 0 {
		if opts.Format == "json" {
			fmt.Println(`{"runs":[]}`)
			return nil
		}
		fmt.Println("No running pipelines")
		return nil
	}

	return outputRuns(runs, opts)
}

// showAllRuns shows recent pipelines.
func showAllRuns(db *sql.DB, opts StatusOptions, limit int) error {
	runs, err := queryRecentRuns(db, limit)
	if err != nil {
		return err
	}

	if len(runs) == 0 {
		if opts.Format == "json" {
			fmt.Println(`{"runs":[]}`)
			return nil
		}
		fmt.Println("No pipelines found")
		return nil
	}

	return outputRuns(runs, opts)
}

// outputRuns formats and outputs the run list.
func outputRuns(runs []StatusRunInfo, opts StatusOptions) error {
	if opts.Format == "json" {
		output := StatusOutput{Runs: runs}
		jsonBytes, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonBytes))
		return nil
	}

	// Dynamic column width allocation based on terminal width
	termWidth := display.GetTerminalWidth()

	// Fixed-width columns: Status=12, Elapsed=10, Tokens=8
	// Gaps between 6 columns = 5 gaps x 1 space each = 5
	const statusWidth = 12
	const elapsedWidth = 10
	const tokensWidth = 8
	const gaps = 5

	fixedWidth := statusWidth + elapsedWidth + tokensWidth + gaps
	remaining := termWidth - fixedWidth
	if remaining < 30 {
		remaining = 30
	}

	// Allocate remaining: RunID 40%, Pipeline 30%, Step 30%
	runIDWidth := remaining * 40 / 100
	pipelineWidth := remaining * 30 / 100
	stepWidth := remaining - runIDWidth - pipelineWidth

	if runIDWidth < 10 {
		runIDWidth = 10
	}
	if pipelineWidth < 8 {
		pipelineWidth = 8
	}
	if stepWidth < 8 {
		stepWidth = 8
	}

	// Table format
	fmt.Printf("%-*s %-*s %-*s %-*s %-*s %s\n",
		runIDWidth, "RUN_ID", pipelineWidth, "PIPELINE",
		statusWidth, "STATUS", stepWidth, "STEP",
		elapsedWidth, "ELAPSED", "TOKENS")

	for _, run := range runs {
		runID := run.RunID
		if len(runID) > runIDWidth && runIDWidth > 3 {
			runID = runID[:runIDWidth-3] + "..."
		}
		pipeline := run.Pipeline
		if len(pipeline) > pipelineWidth && pipelineWidth > 3 {
			pipeline = pipeline[:pipelineWidth-3] + "..."
		}
		step := run.CurrentStep
		if step == "" {
			step = "-"
		}
		if len(step) > stepWidth && stepWidth > 3 {
			step = step[:stepWidth-3] + "..."
		}

		statusColored := fmt.Sprintf("%s%-*s%s", statusColor(run.Status), statusWidth, run.Status, colorReset)

		fmt.Printf("%-*s %-*s %s %-*s %-*s %s\n",
			runIDWidth, runID, pipelineWidth, pipeline, statusColored,
			stepWidth, step, elapsedWidth, run.Elapsed, run.TokensStr)
	}

	return nil
}

// queryRun queries a specific run by ID.
func queryRun(db *sql.DB, runID string) (StatusRunInfo, error) {
	query := `SELECT run_id, pipeline_name, status, input, current_step, total_tokens,
	                 started_at, completed_at, error_message
	          FROM pipeline_run
	          WHERE run_id = ?`

	var run StatusRunInfo
	var startedAt int64
	var completedAt sql.NullInt64
	var input, currentStep, errorMessage sql.NullString
	var tokens int

	err := db.QueryRow(query, runID).Scan(
		&run.RunID,
		&run.Pipeline,
		&run.Status,
		&input,
		&currentStep,
		&tokens,
		&startedAt,
		&completedAt,
		&errorMessage,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return run, fmt.Errorf("run not found: %s", runID)
		}
		return run, fmt.Errorf("failed to query run: %w", err)
	}

	run.Tokens = tokens
	run.TokensStr = formatTokens(tokens)
	run.StartedAt = time.Unix(startedAt, 0).Format("2006-01-02 15:04:05")

	if input.Valid {
		run.Input = input.String
	}
	if currentStep.Valid {
		run.CurrentStep = currentStep.String
	}
	if completedAt.Valid {
		run.CompletedAt = time.Unix(completedAt.Int64, 0).Format("2006-01-02 15:04:05")
		run.Elapsed = formatElapsed(time.Unix(completedAt.Int64, 0).Sub(time.Unix(startedAt, 0)))
		run.ElapsedMs = completedAt.Int64*1000 - startedAt*1000
	} else {
		run.Elapsed = formatElapsed(time.Since(time.Unix(startedAt, 0)))
		run.ElapsedMs = time.Since(time.Unix(startedAt, 0)).Milliseconds()
	}
	if errorMessage.Valid {
		run.Error = errorMessage.String
	}

	return run, nil
}

// queryRunningRuns queries currently running pipelines.
func queryRunningRuns(db *sql.DB) ([]StatusRunInfo, error) {
	query := `SELECT run_id, pipeline_name, status, input, current_step, total_tokens,
	                 started_at, completed_at, error_message
	          FROM pipeline_run
	          WHERE status = 'running'
	          ORDER BY started_at DESC`

	return queryRunsInternal(db, query)
}

// queryRecentRuns queries recent pipelines.
func queryRecentRuns(db *sql.DB, limit int) ([]StatusRunInfo, error) {
	query := `SELECT run_id, pipeline_name, status, input, current_step, total_tokens,
	                 started_at, completed_at, error_message
	          FROM pipeline_run
	          ORDER BY started_at DESC
	          LIMIT ?`

	return queryRunsInternalWithArgs(db, query, limit)
}

// queryRunsInternal executes a query and returns StatusRunInfo slice.
func queryRunsInternal(db *sql.DB, query string) ([]StatusRunInfo, error) {
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query runs: %w", err)
	}
	defer rows.Close()

	return scanRuns(rows)
}

// queryRunsInternalWithArgs executes a query with arguments.
func queryRunsInternalWithArgs(db *sql.DB, query string, args ...any) ([]StatusRunInfo, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query runs: %w", err)
	}
	defer rows.Close()

	return scanRuns(rows)
}

// scanRuns scans rows into StatusRunInfo slice.
func scanRuns(rows *sql.Rows) ([]StatusRunInfo, error) {
	var runs []StatusRunInfo

	for rows.Next() {
		var run StatusRunInfo
		var startedAt int64
		var completedAt sql.NullInt64
		var input, currentStep, errorMessage sql.NullString
		var tokens int

		err := rows.Scan(
			&run.RunID,
			&run.Pipeline,
			&run.Status,
			&input,
			&currentStep,
			&tokens,
			&startedAt,
			&completedAt,
			&errorMessage,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan run: %w", err)
		}

		run.Tokens = tokens
		run.TokensStr = formatTokens(tokens)
		run.StartedAt = time.Unix(startedAt, 0).Format("2006-01-02 15:04:05")

		if input.Valid {
			run.Input = input.String
		}
		if currentStep.Valid {
			run.CurrentStep = currentStep.String
		}
		if completedAt.Valid {
			run.CompletedAt = time.Unix(completedAt.Int64, 0).Format("2006-01-02 15:04:05")
			run.Elapsed = formatElapsed(time.Unix(completedAt.Int64, 0).Sub(time.Unix(startedAt, 0)))
			run.ElapsedMs = completedAt.Int64*1000 - startedAt*1000
		} else {
			run.Elapsed = formatElapsed(time.Since(time.Unix(startedAt, 0)))
			run.ElapsedMs = time.Since(time.Unix(startedAt, 0)).Milliseconds()
		}
		if errorMessage.Valid {
			run.Error = errorMessage.String
		}

		runs = append(runs, run)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating runs: %w", err)
	}

	return runs, nil
}

// statusColor returns the ANSI color code for a status.
func statusColor(status string) string {
	switch status {
	case "running":
		return colorYellow
	case "completed":
		return colorGreen
	case "failed":
		return colorRed
	case "cancelled":
		return colorGray
	default:
		return ""
	}
}
