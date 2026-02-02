package commands

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	_ "modernc.org/sqlite"
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
}

// LogsOutput represents the JSON output for logs command.
type LogsOutput struct {
	RunID string        `json:"run_id"`
	Logs  []LogsEntry   `json:"logs"`
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
With a run-id argument, shows logs for that specific run.

Examples:
  wave logs                      # Show logs from most recent run
  wave logs debug-20260202-143022  # Show logs for specific run
  wave logs --step investigate   # Filter by step ID
  wave logs --errors             # Show only errors
  wave logs --tail 20            # Show last 20 log entries
  wave logs --since 10m          # Show logs from last 10 minutes
  wave logs --follow             # Stream logs in real-time
  wave logs --format json        # Output as JSON for scripting`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.RunID = args[0]
			}
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

	return cmd
}

func runLogs(opts LogsOptions) error {
	dbPath := ".wave/state.db"

	// Check if state database exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		if opts.Format == "json" {
			fmt.Println(`{"run_id":"","logs":[]}`)
			return nil
		}
		fmt.Println("No logs found (state database does not exist)")
		return nil
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open state database: %w", err)
	}
	defer db.Close()

	// Configure SQLite
	db.SetMaxOpenConns(1)

	// Resolve run ID if not provided
	runID := opts.RunID
	if runID == "" {
		runID, err = getMostRecentRunID(db)
		if err != nil {
			if opts.Format == "json" {
				fmt.Println(`{"run_id":"","logs":[]}`)
				return nil
			}
			fmt.Println("No pipeline runs found")
			return nil
		}
	}

	// Verify run exists
	if !runExists(db, runID) {
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
		return runLogsFollow(db, runID, opts)
	}

	return runLogsOnce(db, runID, opts)
}

// runLogsOnce retrieves and displays logs once.
func runLogsOnce(db *sql.DB, runID string, opts LogsOptions) error {
	logs, err := queryLogs(db, runID, opts)
	if err != nil {
		return err
	}

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

	return outputLogs(runID, logs, opts)
}

// runLogsFollow streams logs in real-time.
func runLogsFollow(db *sql.DB, runID string, opts LogsOptions) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	go func() {
		<-sigChan
		cancel()
	}()

	var lastID int64 = 0
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	// Print initial logs
	logs, err := queryLogs(db, runID, opts)
	if err != nil {
		return err
	}

	for _, log := range logs {
		printLogEntry(log, opts.Format)
		// Track the highest ID we've seen
		id := getLogID(db, runID, log)
		if id > lastID {
			lastID = id
		}
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			// Check if pipeline is still running
			status, err := getRunStatus(db, runID)
			if err != nil {
				return nil
			}

			// Query new logs since lastID
			newLogs, newLastID, err := queryNewLogs(db, runID, lastID, opts)
			if err != nil {
				continue
			}

			for _, log := range newLogs {
				printLogEntry(log, opts.Format)
			}
			if newLastID > lastID {
				lastID = newLastID
			}

			// Exit if pipeline completed
			if status != "running" && status != "pending" {
				return nil
			}
		}
	}
}

// queryLogs retrieves logs matching the options.
func queryLogs(db *sql.DB, runID string, opts LogsOptions) ([]LogsEntry, error) {
	query := `SELECT timestamp, step_id, state, persona, message, tokens_used, duration_ms
	          FROM event_log
	          WHERE run_id = ?`
	args := []any{runID}

	if opts.Step != "" {
		query += " AND step_id = ?"
		args = append(args, opts.Step)
	}

	if opts.Level == "error" {
		query += " AND state = 'failed'"
	}

	if opts.Since != "" {
		duration, err := parseSinceDuration(opts.Since)
		if err != nil {
			return nil, fmt.Errorf("invalid --since duration: %w", err)
		}
		cutoff := time.Now().Add(-duration).Unix()
		query += " AND timestamp >= ?"
		args = append(args, cutoff)
	}

	query += " ORDER BY timestamp ASC, id ASC"

	if opts.Tail > 0 {
		// To get last N, we need to wrap in a subquery
		query = fmt.Sprintf(`SELECT * FROM (%s) ORDER BY timestamp ASC`, query+" LIMIT "+strconv.Itoa(opts.Tail))
		// Actually, let's use a simpler approach - order DESC first, then re-order
		query = `SELECT timestamp, step_id, state, persona, message, tokens_used, duration_ms
		         FROM event_log
		         WHERE run_id = ?`
		args = []any{runID}

		if opts.Step != "" {
			query += " AND step_id = ?"
			args = append(args, opts.Step)
		}

		if opts.Level == "error" {
			query += " AND state = 'failed'"
		}

		if opts.Since != "" {
			duration, _ := parseSinceDuration(opts.Since)
			cutoff := time.Now().Add(-duration).Unix()
			query += " AND timestamp >= ?"
			args = append(args, cutoff)
		}

		// Order DESC, limit, then we'll reverse in code
		query += " ORDER BY timestamp DESC, id DESC LIMIT ?"
		args = append(args, opts.Tail)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query logs: %w", err)
	}
	defer rows.Close()

	var logs []LogsEntry
	for rows.Next() {
		var log LogsEntry
		var timestamp int64
		var stepID, persona, message sql.NullString
		var tokensUsed, durationMs sql.NullInt64

		err := rows.Scan(
			&timestamp,
			&stepID,
			&log.State,
			&persona,
			&message,
			&tokensUsed,
			&durationMs,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan log: %w", err)
		}

		log.Timestamp = time.Unix(timestamp, 0).Format("15:04:05")
		if stepID.Valid {
			log.StepID = stepID.String
		}
		if persona.Valid {
			log.Persona = persona.String
		}
		if message.Valid {
			log.Message = message.String
		}
		if tokensUsed.Valid {
			log.TokensUsed = int(tokensUsed.Int64)
		}
		if durationMs.Valid {
			log.DurationMs = durationMs.Int64
		}

		logs = append(logs, log)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating logs: %w", err)
	}

	// Reverse if we used tail with DESC order
	if opts.Tail > 0 && len(logs) > 0 {
		for i, j := 0, len(logs)-1; i < j; i, j = i+1, j-1 {
			logs[i], logs[j] = logs[j], logs[i]
		}
	}

	return logs, nil
}

// queryNewLogs retrieves logs newer than lastID.
func queryNewLogs(db *sql.DB, runID string, lastID int64, opts LogsOptions) ([]LogsEntry, int64, error) {
	query := `SELECT id, timestamp, step_id, state, persona, message, tokens_used, duration_ms
	          FROM event_log
	          WHERE run_id = ? AND id > ?`
	args := []any{runID, lastID}

	if opts.Step != "" {
		query += " AND step_id = ?"
		args = append(args, opts.Step)
	}

	if opts.Level == "error" {
		query += " AND state = 'failed'"
	}

	query += " ORDER BY id ASC"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, lastID, fmt.Errorf("failed to query new logs: %w", err)
	}
	defer rows.Close()

	var logs []LogsEntry
	newLastID := lastID

	for rows.Next() {
		var log LogsEntry
		var id, timestamp int64
		var stepID, persona, message sql.NullString
		var tokensUsed, durationMs sql.NullInt64

		err := rows.Scan(
			&id,
			&timestamp,
			&stepID,
			&log.State,
			&persona,
			&message,
			&tokensUsed,
			&durationMs,
		)
		if err != nil {
			return nil, lastID, fmt.Errorf("failed to scan log: %w", err)
		}

		if id > newLastID {
			newLastID = id
		}

		log.Timestamp = time.Unix(timestamp, 0).Format("15:04:05")
		if stepID.Valid {
			log.StepID = stepID.String
		}
		if persona.Valid {
			log.Persona = persona.String
		}
		if message.Valid {
			log.Message = message.String
		}
		if tokensUsed.Valid {
			log.TokensUsed = int(tokensUsed.Int64)
		}
		if durationMs.Valid {
			log.DurationMs = durationMs.Int64
		}

		logs = append(logs, log)
	}

	return logs, newLastID, nil
}

// outputLogs formats and outputs the logs.
func outputLogs(runID string, logs []LogsEntry, opts LogsOptions) error {
	if opts.Format == "json" {
		output := LogsOutput{RunID: runID, Logs: logs}
		jsonBytes, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonBytes))
		return nil
	}

	// Text format
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
	parts = append(parts, fmt.Sprintf("%-10s", log.State))

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

// getMostRecentRunID gets the most recent run ID.
func getMostRecentRunID(db *sql.DB) (string, error) {
	var runID string
	err := db.QueryRow(`SELECT run_id FROM pipeline_run ORDER BY started_at DESC LIMIT 1`).Scan(&runID)
	if err != nil {
		return "", err
	}
	return runID, nil
}

// runExists checks if a run exists.
func runExists(db *sql.DB, runID string) bool {
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM pipeline_run WHERE run_id = ?`, runID).Scan(&count)
	return err == nil && count > 0
}

// getRunStatus gets the status of a run.
func getRunStatus(db *sql.DB, runID string) (string, error) {
	var status string
	err := db.QueryRow(`SELECT status FROM pipeline_run WHERE run_id = ?`, runID).Scan(&status)
	return status, err
}

// getLogID gets the ID of a log entry (for follow mode tracking).
func getLogID(db *sql.DB, runID string, log LogsEntry) int64 {
	// Parse timestamp back to unix time
	t, err := time.Parse("15:04:05", log.Timestamp)
	if err != nil {
		return 0
	}
	// Combine with today's date
	now := time.Now()
	fullTime := time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), t.Second(), 0, time.Local)

	var id int64
	err = db.QueryRow(`SELECT id FROM event_log WHERE run_id = ? AND timestamp = ? AND state = ? LIMIT 1`,
		runID, fullTime.Unix(), log.State).Scan(&id)
	if err != nil {
		return 0
	}
	return id
}

// parseSinceDuration parses duration strings like "10m", "1h", "2d".
func parseSinceDuration(s string) (time.Duration, error) {
	if s == "" {
		return 0, nil
	}

	// Check for day suffix (not supported by time.ParseDuration)
	dayRegex := regexp.MustCompile(`^(\d+)d(.*)$`)
	if matches := dayRegex.FindStringSubmatch(s); len(matches) == 3 {
		days, err := strconv.Atoi(matches[1])
		if err != nil {
			return 0, fmt.Errorf("invalid days value: %s", matches[1])
		}
		remaining := matches[2]
		var extraDuration time.Duration
		if remaining != "" {
			extraDuration, err = time.ParseDuration(remaining)
			if err != nil {
				return 0, fmt.Errorf("invalid duration: %s", s)
			}
		}
		return time.Duration(days)*24*time.Hour + extraDuration, nil
	}

	return time.ParseDuration(s)
}
