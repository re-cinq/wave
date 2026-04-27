package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/recinq/wave/internal/display"
	"github.com/recinq/wave/internal/state"
	"github.com/spf13/cobra"
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

// conditionalColor returns the ANSI color code if NO_COLOR is not set,
// or an empty string when colors are disabled.
func conditionalColor(code string) string {
	if os.Getenv("NO_COLOR") != "" {
		return ""
	}
	return code
}

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
			opts.Format = ResolveFormat(cmd, opts.Format)
			return runStatus(opts)
		},
	}

	cmd.Flags().BoolVar(&opts.All, "all", false, "Show all recent pipelines (default 10)")
	cmd.Flags().StringVar(&opts.Format, "format", "table", "Output format (table, json)")
	cmd.Flags().StringVar(&opts.Manifest, "manifest", "wave.yaml", "Path to manifest file")

	return cmd
}

func runStatus(opts StatusOptions) error {
	dbPath := ".agents/state.db"

	// Check if state database exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		if opts.Format == "json" {
			fmt.Println(`{"runs":[]}`)
			return nil
		}
		fmt.Fprintln(os.Stderr, "No pipelines found")
		return nil
	}

	store, err := state.NewStateStore(dbPath)
	if err != nil {
		return NewCLIError(CodeStateDBError, fmt.Sprintf("failed to open state database: %s", err), "Check .agents/state.db file permissions or run 'wave run' to create it").WithCause(err)
	}
	defer store.Close()

	if opts.RunID != "" {
		return showRunDetails(store, opts)
	}

	if opts.All {
		return showAllRuns(store, opts, 10)
	}

	return showRunningRuns(store, opts)
}

// runRecordToStatusInfo converts a state.RunRecord to a StatusRunInfo for display.
func runRecordToStatusInfo(r *state.RunRecord) StatusRunInfo {
	info := StatusRunInfo{
		RunID:       r.RunID,
		Pipeline:    r.PipelineName,
		Status:      r.Status,
		CurrentStep: r.CurrentStep,
		Tokens:      r.TotalTokens,
		TokensStr:   formatTokens(r.TotalTokens),
		StartedAt:   r.StartedAt.Format("2006-01-02 15:04:05"),
		Input:       r.Input,
		Error:       r.ErrorMessage,
	}

	if r.CompletedAt != nil {
		info.CompletedAt = r.CompletedAt.Format("2006-01-02 15:04:05")
		info.Elapsed = formatElapsed(r.CompletedAt.Sub(r.StartedAt))
		info.ElapsedMs = r.CompletedAt.Sub(r.StartedAt).Milliseconds()
	} else {
		info.Elapsed = formatElapsed(time.Since(r.StartedAt))
		info.ElapsedMs = time.Since(r.StartedAt).Milliseconds()
	}

	return info
}

// showRunDetails shows detailed status for a specific run.
func showRunDetails(store state.StateStore, opts StatusOptions) error {
	record, err := store.GetRun(opts.RunID)
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

	run := runRecordToStatusInfo(record)

	if opts.Format == "json" {
		output := StatusOutput{Runs: []StatusRunInfo{run}}
		jsonBytes, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return NewCLIError(CodeInternalError, fmt.Sprintf("failed to marshal JSON: %s", err), "This is an internal serialization error").WithCause(err)
		}
		fmt.Println(string(jsonBytes))
		return nil
	}

	// Detailed table view for single run
	fmt.Printf("Run ID:     %s\n", run.RunID)
	fmt.Printf("Pipeline:   %s\n", run.Pipeline)
	fmt.Printf("Status:     %s%s%s\n", statusColor(run.Status), run.Status, conditionalColor("\033[0m"))
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
func showRunningRuns(store state.StateStore, opts StatusOptions) error {
	records, err := store.GetRunningRuns()
	if err != nil {
		return err
	}

	if state.ReconcileZombies(store, 0) > 0 {
		records, err = store.GetRunningRuns()
		if err != nil {
			return err
		}
	}

	if len(records) == 0 {
		if opts.Format == "json" {
			fmt.Println(`{"runs":[]}`)
			return nil
		}
		fmt.Fprintln(os.Stderr, "No running pipelines")
		return nil
	}

	runs := make([]StatusRunInfo, len(records))
	for i := range records {
		runs[i] = runRecordToStatusInfo(&records[i])
	}

	return outputRuns(runs, opts)
}

// showAllRuns shows recent pipelines.
func showAllRuns(store state.StateStore, opts StatusOptions, limit int) error {
	records, err := store.ListRuns(state.ListRunsOptions{Limit: limit})
	if err != nil {
		return err
	}

	if len(records) == 0 {
		if opts.Format == "json" {
			fmt.Println(`{"runs":[]}`)
			return nil
		}
		fmt.Fprintln(os.Stderr, "No pipelines found")
		return nil
	}

	runs := make([]StatusRunInfo, len(records))
	for i := range records {
		runs[i] = runRecordToStatusInfo(&records[i])
	}

	return outputRuns(runs, opts)
}

// outputRuns formats and outputs the run list.
func outputRuns(runs []StatusRunInfo, opts StatusOptions) error {
	if opts.Format == "json" {
		output := StatusOutput{Runs: runs}
		jsonBytes, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return NewCLIError(CodeInternalError, fmt.Sprintf("failed to marshal JSON: %s", err), "This is an internal serialization error").WithCause(err)
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

		statusColored := fmt.Sprintf("%s%-*s%s", statusColor(run.Status), statusWidth, run.Status, conditionalColor("\033[0m"))

		fmt.Printf("%-*s %-*s %s %-*s %-*s %s\n",
			runIDWidth, runID, pipelineWidth, pipeline, statusColored,
			stepWidth, step, elapsedWidth, run.Elapsed, run.TokensStr)
	}

	return nil
}

// statusColor returns the ANSI color code for a status.
func statusColor(status string) string {
	switch status {
	case "running":
		return conditionalColor("\033[33m")
	case "completed":
		return conditionalColor("\033[32m")
	case "failed":
		return conditionalColor("\033[31m")
	case "cancelled":
		return conditionalColor("\033[90m")
	default:
		return ""
	}
}
