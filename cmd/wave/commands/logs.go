package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/recinq/wave/internal/audit"
	"github.com/recinq/wave/internal/state"
	"github.com/spf13/cobra"
)

// LogsOptions holds options for the logs command.
type LogsOptions struct {
	RunID    string // Specific run (from args, default: most recent)
	Step     string // Filter by step ID
	Errors   bool   // Only show errors
	Follow   bool   // Stream logs in real-time
	Tail     int    // Show last N lines
	Since    string // Filter by time (e.g., "10m", "1h")
	Level    string // Log level: all, info, error
	Format   string // text, json
	Manifest string
	Trace    bool // Show structured debug trace events
}

// LogsOutput represents the JSON output for logs command.
type LogsOutput struct {
	RunID string      `json:"run_id"`
	Logs  []LogsEntry `json:"logs"`
}

// LogsEntry represents a single log entry.
type LogsEntry struct {
	Timestamp  string `json:"timestamp"`
	State      string `json:"state"`
	StepID     string `json:"step_id,omitempty"`
	Persona    string `json:"persona,omitempty"`
	Message    string `json:"message,omitempty"`
	TokensUsed int    `json:"tokens_used,omitempty"`
	DurationMs int64  `json:"duration_ms,omitempty"`
}

// NewLogsCmd creates the logs command.
func NewLogsCmd() *cobra.Command {
	var opts LogsOptions

	cmd := &cobra.Command{
		Use:   "logs [run-id]",
		Short: "Show pipeline logs",
		Long: `Show logs from pipeline runs.

Without arguments, shows logs from the most recent run.
With a run-id argument, shows logs for that specific run.`,
		Example: `  wave logs                        # Show logs from most recent run
  wave logs debug-20260202-143022  # Show logs for specific run
  wave logs --step investigate     # Filter by step ID
  wave logs --errors               # Show only errors
  wave logs --tail 20              # Show last 20 log entries
  wave logs --since 10m            # Show logs from last 10 minutes
  wave logs --follow               # Stream logs in real-time
  wave logs --format json          # Output as JSON for scripting
  wave logs --trace                # Show debug trace events (requires --debug run)`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.RunID = args[0]
			}
			opts.Format = ResolveFormat(cmd, opts.Format)
			return runLogs(opts)
		},
	}

	cmd.Flags().StringVar(&opts.Step, "step", "", "Filter by step ID")
	cmd.Flags().BoolVar(&opts.Errors, "errors", false, "Only show errors (alias for --level error)")
	cmd.Flags().BoolVar(&opts.Follow, "follow", false, "Stream logs in real-time")
	cmd.Flags().IntVar(&opts.Tail, "tail", 0, "Show last N lines")
	cmd.Flags().StringVar(&opts.Since, "since", "", "Filter by time (e.g., \"10m\", \"1h\")")
	cmd.Flags().StringVar(&opts.Level, "level", "all", "Log level: all, info, error")
	cmd.Flags().StringVar(&opts.Format, "format", "text", "Output format (text, json)")
	cmd.Flags().StringVar(&opts.Manifest, "manifest", "wave.yaml", "Path to manifest file")
	cmd.Flags().BoolVar(&opts.Trace, "trace", false, "Show structured debug trace events (requires a --debug run)")

	return cmd
}

func runLogs(opts LogsOptions) error {
	// Handle --trace mode: read NDJSON trace file instead of state DB.
	if opts.Trace {
		return runLogsTrace(opts)
	}

	dbPath := ".agents/state.db"

	// Check if state database exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		if opts.Format == "json" {
			fmt.Println(`{"run_id":"","logs":[]}`)
			return nil
		}
		fmt.Println("No logs found (state database does not exist)")
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
				fmt.Println(`{"run_id":"","logs":[]}`)
				return nil
			}
			fmt.Println("No pipeline runs found")
			return nil
		}
	}

	// Verify run exists
	exists, err := store.RunExists(runID)
	if err != nil {
		return NewCLIError(CodeInternalError, fmt.Sprintf("failed to verify run: %s", err), "The state database may be corrupted -- try 'wave migrate validate'").WithCause(err)
	}
	if !exists {
		if opts.Format == "json" {
			fmt.Printf(`{"run_id":"%s","logs":[],"error":"run not found"}`, runID)
			fmt.Println()
			return nil
		}
		fmt.Printf("Run not found: %s\n", runID)
		return nil
	}

	// Handle --errors flag as alias for --level error
	if opts.Errors {
		opts.Level = "error"
	}

	if opts.Follow {
		return runLogsFollow(store, runID, opts)
	}

	return runLogsOnce(store, runID, opts)
}

// queryEventOptions translates LogsOptions to state.EventQueryOptions.
func queryEventOptions(opts LogsOptions) (state.EventQueryOptions, error) {
	q := state.EventQueryOptions{
		StepID:     opts.Step,
		ErrorsOnly: opts.Level == "error",
		TailLimit:  opts.Tail,
	}
	if opts.Since != "" {
		duration, err := parseDuration(opts.Since)
		if err != nil {
			return q, NewCLIError(CodeInvalidArgs, fmt.Sprintf("invalid --since duration: %s", err), "Use a duration like '10m', '1h', or '30s'").WithCause(err)
		}
		q.SinceUnix = time.Now().Add(-duration).Unix()
	}
	return q, nil
}

// recordsToEntries converts state.LogRecord values into the CLI's LogsEntry DTO.
func recordsToEntries(records []state.LogRecord) []LogsEntry {
	out := make([]LogsEntry, 0, len(records))
	for _, r := range records {
		out = append(out, LogsEntry{
			Timestamp:  r.Timestamp.Format("15:04:05"),
			State:      r.State,
			StepID:     r.StepID,
			Persona:    r.Persona,
			Message:    r.Message,
			TokensUsed: r.TokensUsed,
			DurationMs: r.DurationMs,
		})
	}
	return out
}

// runLogsOnce retrieves and displays logs once.
func runLogsOnce(store state.StateStore, runID string, opts LogsOptions) error {
	q, err := queryEventOptions(opts)
	if err != nil {
		return err
	}

	records, err := store.GetEvents(runID, q)
	if err != nil {
		return NewCLIError(CodeInternalError, fmt.Sprintf("failed to query logs: %s", err), "The state database may be corrupted -- try 'wave migrate validate'").WithCause(err)
	}

	logs := recordsToEntries(records)

	if len(logs) == 0 {
		if opts.Format == "json" {
			output := LogsOutput{RunID: runID, Logs: []LogsEntry{}}
			jsonBytes, _ := json.MarshalIndent(output, "", "  ")
			fmt.Println(string(jsonBytes))
			return nil
		}
		fmt.Printf("No logs found for run: %s\n", runID)
		return nil
	}

	if err := outputLogs(runID, logs, opts); err != nil {
		return err
	}

	// Show performance summary in text mode (not for --errors or --step filters)
	if opts.Format == "text" && !opts.Errors && opts.Step == "" {
		renderPerformanceSummary(store, runID)
	}

	return nil
}

// runLogsFollow streams logs in real-time.
func runLogsFollow(store state.StateStore, runID string, opts LogsOptions) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	go func() {
		<-sigChan
		cancel()
	}()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	q, err := queryEventOptions(opts)
	if err != nil {
		return err
	}

	// Print initial logs and seed lastID from the highest event ID seen.
	initial, err := store.GetEvents(runID, q)
	if err != nil {
		return NewCLIError(CodeInternalError, fmt.Sprintf("failed to query logs: %s", err), "The state database may be corrupted -- try 'wave migrate validate'").WithCause(err)
	}
	for _, r := range initial {
		printLogEntry(LogsEntry{
			Timestamp:  r.Timestamp.Format("15:04:05"),
			State:      r.State,
			StepID:     r.StepID,
			Persona:    r.Persona,
			Message:    r.Message,
			TokensUsed: r.TokensUsed,
			DurationMs: r.DurationMs,
		}, opts.Format)
	}

	var lastID int64
	for _, r := range initial {
		if r.ID > lastID {
			lastID = r.ID
		}
	}

	// Follow mode polls AfterID — TailLimit/SinceUnix only apply to the initial fetch.
	pollOpts := state.EventQueryOptions{
		StepID:     opts.Step,
		ErrorsOnly: opts.Level == "error",
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			status, statusErr := store.GetRunStatus(runID)
			if statusErr != nil {
				return nil
			}

			pollOpts.AfterID = lastID
			newRecords, err := store.GetEvents(runID, pollOpts)
			if err != nil {
				continue
			}

			for _, r := range newRecords {
				printLogEntry(LogsEntry{
					Timestamp:  r.Timestamp.Format("15:04:05"),
					State:      r.State,
					StepID:     r.StepID,
					Persona:    r.Persona,
					Message:    r.Message,
					TokensUsed: r.TokensUsed,
					DurationMs: r.DurationMs,
				}, opts.Format)
				if r.ID > lastID {
					lastID = r.ID
				}
			}

			// Exit if pipeline completed (or run vanished).
			if status == "" || (status != "running" && status != "pending") {
				return nil
			}
		}
	}
}

// outputLogs formats and outputs the logs.
func outputLogs(runID string, logs []LogsEntry, opts LogsOptions) error {
	if opts.Format == "json" {
		output := LogsOutput{RunID: runID, Logs: logs}
		jsonBytes, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return NewCLIError(CodeInternalError, fmt.Sprintf("failed to marshal JSON: %s", err), "This is an internal serialization error").WithCause(err)
		}
		fmt.Println(string(jsonBytes))
		return nil
	}

	for _, log := range logs {
		printLogEntry(log, opts.Format)
	}

	return nil
}

// printLogEntry prints a single log entry.
func printLogEntry(log LogsEntry, format string) {
	if format == "json" {
		jsonBytes, _ := json.Marshal(log)
		fmt.Println(string(jsonBytes))
		return
	}

	// Format: [HH:MM:SS] state     step_id (persona) duration tokens message
	var parts []string
	parts = append(parts, fmt.Sprintf("[%s]", log.Timestamp))

	// Highlight ontology events with color
	stateStr := fmt.Sprintf("%-18s", log.State)
	switch log.State {
	case "ontology_inject":
		stateStr = fmt.Sprintf("\033[1;35m%-18s\033[0m", log.State) // bold magenta
	case "ontology_lineage":
		stateStr = fmt.Sprintf("\033[35m%-18s\033[0m", log.State) // magenta
	case "failed":
		stateStr = fmt.Sprintf("\033[31m%-18s\033[0m", log.State) // red
	case "completed":
		stateStr = fmt.Sprintf("\033[32m%-18s\033[0m", log.State) // green
	}
	parts = append(parts, stateStr)

	if log.StepID != "" {
		if log.Persona != "" {
			parts = append(parts, fmt.Sprintf("%s (%s)", log.StepID, log.Persona))
		} else {
			parts = append(parts, log.StepID)
		}
	}

	if log.DurationMs > 0 {
		parts = append(parts, fmt.Sprintf("%.1fs", float64(log.DurationMs)/1000))
	}

	if log.TokensUsed > 0 {
		parts = append(parts, formatTokens(log.TokensUsed)+" tokens")
	}

	if log.Message != "" {
		parts = append(parts, log.Message)
	}

	fmt.Println(strings.Join(parts, " "))
}

// runLogsTrace reads and displays structured NDJSON trace events from a debug trace file.
func runLogsTrace(opts LogsOptions) error {
	traceDir := ".agents/traces"

	runID := opts.RunID
	if runID == "" {
		dbPath := ".agents/state.db"
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			fmt.Println("No trace files found (no runs recorded)")
			return nil
		}
		store, err := state.NewReadOnlyStateStore(dbPath)
		if err != nil {
			return NewCLIError(CodeStateDBError, fmt.Sprintf("failed to open state database: %s", err), "Check .agents/state.db file permissions").WithCause(err)
		}
		defer store.Close()

		runID, err = store.GetMostRecentRunID()
		if err != nil {
			return NewCLIError(CodeInternalError, fmt.Sprintf("failed to query most recent run: %s", err), "The state database may be corrupted").WithCause(err)
		}
		if runID == "" {
			fmt.Println("No pipeline runs found")
			return nil
		}
	}

	tracePath, err := audit.FindTraceFile(traceDir, runID)
	if err != nil {
		if opts.Format == "json" {
			fmt.Printf(`{"run_id":"%s","trace_events":[],"error":"no trace file"}`, runID)
			fmt.Println()
			return nil
		}
		fmt.Printf("No trace file found for run %s (was it run with --debug?)\n", runID)
		return nil
	}

	events, err := audit.ReadTraceFile(tracePath)
	if err != nil {
		return NewCLIError(CodeInternalError, fmt.Sprintf("failed to read trace file: %s", err), "The trace file may be corrupted or incomplete").WithCause(err)
	}

	if opts.Step != "" {
		var filtered []audit.TraceEvent
		for _, ev := range events {
			if ev.StepID == opts.Step {
				filtered = append(filtered, ev)
			}
		}
		events = filtered
	}

	if opts.Format == "json" {
		output := struct {
			RunID  string             `json:"run_id"`
			Events []audit.TraceEvent `json:"trace_events"`
		}{RunID: runID, Events: events}
		if output.Events == nil {
			output.Events = []audit.TraceEvent{}
		}
		jsonBytes, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(jsonBytes))
		return nil
	}

	if len(events) == 0 {
		fmt.Printf("No trace events found for run %s\n", runID)
		return nil
	}

	for _, ev := range events {
		ts := ev.Timestamp
		if len(ts) > 19 {
			if t, parseErr := time.Parse(time.RFC3339Nano, ts); parseErr == nil {
				ts = t.Format("15:04:05.000")
			}
		}

		line := fmt.Sprintf("[%s] %-28s", ts, ev.EventType)
		if ev.StepID != "" {
			line += fmt.Sprintf(" step=%-20s", ev.StepID)
		}
		if ev.DurationMs > 0 {
			line += fmt.Sprintf(" %dms", ev.DurationMs)
		}

		for k, v := range ev.Metadata {
			line += fmt.Sprintf(" %s=%s", k, v)
		}

		fmt.Println(line)
	}

	return nil
}

// renderPerformanceSummary displays aggregated performance metrics for a run.
func renderPerformanceSummary(store state.StateStore, runID string) {
	stats, err := store.GetEventAggregateStats(runID)
	if err != nil || stats == nil || stats.TotalEvents == 0 {
		return
	}

	fmt.Fprintln(os.Stderr, "\n--- Performance Summary ---")
	fmt.Fprintf(os.Stderr, "Total Steps: %d\n", stats.TotalEvents)

	if stats.TotalTokens > 0 {
		fmt.Fprintf(os.Stderr, "Total Tokens: %s\n", formatTokens(stats.TotalTokens))
	}

	if stats.AvgDurationMs > 0 {
		fmt.Fprintf(os.Stderr, "Avg Duration: %.1fs\n", stats.AvgDurationMs/1000.0)
	}

	if stats.MinDurationMs > 0 {
		fmt.Fprintf(os.Stderr, "Duration Range: %.1fs - %.1fs\n",
			stats.MinDurationMs/1000.0,
			stats.MaxDurationMs/1000.0)
	}

	if stats.AvgDurationMs > 0 && stats.TotalTokens > 0 {
		totalDuration := stats.AvgDurationMs * float64(stats.TotalEvents)
		burnRate := float64(stats.TotalTokens) / (totalDuration / 1000.0)
		if burnRate >= 1.0 {
			fmt.Fprintf(os.Stderr, "Token Burn Rate: %.1f tokens/s\n", burnRate)
		}
	}
}
