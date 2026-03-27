package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/recinq/wave/internal/retro"
	"github.com/recinq/wave/internal/state"
	"github.com/spf13/cobra"
)

// NewRetroCmd creates the retro command group for viewing and analyzing
// run retrospectives.
func NewRetroCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "retro",
		Short: "View and analyze run retrospectives",
		Long: `View and analyze run retrospectives.

Retrospectives are structured post-execution analyses of pipeline runs,
combining quantitative metrics (duration, retries, tokens) with an optional
qualitative narrative (smoothness, friction points, learnings).

Subcommands:
  view   View a single retrospective by run ID
  list   List recent retrospectives with optional filters
  stats  Show aggregate retrospective statistics`,
	}

	cmd.AddCommand(newRetroViewCmd())
	cmd.AddCommand(newRetroListCmd())
	cmd.AddCommand(newRetroStatsCmd())

	return cmd
}

// newRetroViewCmd creates the "retro view" subcommand.
func newRetroViewCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "view <run-id>",
		Short: "View retrospective for a run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			runID := args[0]

			store, err := openRetroStateStore()
			if err != nil {
				return err
			}
			defer store.Close()

			retroStore := retro.NewFileStore(filepath.Join(".wave", "retros"), store)
			r, err := retroStore.Get(runID)
			if err != nil {
				return NewCLIError(CodeInternalError, fmt.Sprintf("failed to get retrospective: %s", err), "Check the state database and .wave/retros/ directory").WithCause(err)
			}
			if r == nil {
				return NewCLIError(CodeRunNotFound, fmt.Sprintf("no retrospective found for run %s", runID), "Use 'wave retro list' to see available retrospectives")
			}

			if jsonOutput {
				return printRetroJSON(r)
			}
			printRetroText(r)
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")

	return cmd
}

// newRetroListCmd creates the "retro list" subcommand.
func newRetroListCmd() *cobra.Command {
	var pipeline string
	var since string
	var limit int
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List recent retrospectives",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openRetroStateStore()
			if err != nil {
				return err
			}
			defer store.Close()

			opts := retro.ListOptions{
				PipelineName: pipeline,
				Limit:        limit,
			}

			if since != "" {
				d, err := parseDuration(since)
				if err != nil {
					return NewCLIError(CodeInvalidArgs, fmt.Sprintf("invalid --since value: %s", err), "Use formats like '7d', '30d', '24h'")
				}
				opts.Since = time.Now().Add(-d)
			}

			retroStore := retro.NewFileStore(filepath.Join(".wave", "retros"), store)
			retros, err := retroStore.List(opts)
			if err != nil {
				return NewCLIError(CodeInternalError, fmt.Sprintf("failed to list retrospectives: %s", err), "Check the state database").WithCause(err)
			}

			if jsonOutput {
				return printRetroListJSON(retros)
			}
			printRetroListText(retros)
			return nil
		},
	}

	cmd.Flags().StringVar(&pipeline, "pipeline", "", "Filter by pipeline name")
	cmd.Flags().StringVar(&since, "since", "", "Show retros since duration (e.g., 7d, 30d)")
	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum number of retros to show")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")

	return cmd
}

// newRetroStatsCmd creates the "retro stats" subcommand.
func newRetroStatsCmd() *cobra.Command {
	var jsonOutput bool
	var pipeline string
	var since string

	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Show aggregate retrospective statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openRetroStateStore()
			if err != nil {
				return err
			}
			defer store.Close()

			opts := retro.ListOptions{
				PipelineName: pipeline,
			}
			if since != "" {
				d, err := parseDuration(since)
				if err != nil {
					return NewCLIError(CodeInvalidArgs, fmt.Sprintf("invalid --since value: %s", err), "Use formats like '7d', '30d', '24h'")
				}
				opts.Since = time.Now().Add(-d)
			}

			retroStore := retro.NewFileStore(filepath.Join(".wave", "retros"), store)
			retros, err := retroStore.List(opts)
			if err != nil {
				return NewCLIError(CodeInternalError, fmt.Sprintf("failed to list retrospectives: %s", err), "Check the state database").WithCause(err)
			}

			if len(retros) == 0 {
				fmt.Println("No retrospectives found.")
				return nil
			}

			stats := computeRetroStats(retros)

			if jsonOutput {
				return printRetroStatsJSON(stats)
			}
			printRetroStatsText(stats)
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	cmd.Flags().StringVar(&pipeline, "pipeline", "", "Filter by pipeline name")
	cmd.Flags().StringVar(&since, "since", "", "Show stats since duration (e.g., 7d, 30d)")

	return cmd
}

// openRetroStateStore opens the state store at the standard path, returning
// a CLIError when the database does not exist or cannot be opened.
func openRetroStateStore() (state.StateStore, error) {
	dbPath := ".wave/state.db"
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil, NewCLIError(CodeStateDBError, "state database not found", "Run 'wave run' to create the state database")
	}
	store, err := state.NewStateStore(dbPath)
	if err != nil {
		return nil, NewCLIError(CodeStateDBError, fmt.Sprintf("failed to open state database: %s", err), "Check .wave/state.db file permissions").WithCause(err)
	}
	return store, nil
}

// ---------------------------------------------------------------------------
// View output
// ---------------------------------------------------------------------------

func printRetroText(r *retro.Retrospective) {
	fmt.Printf("Retrospective: %s\n", r.RunID)
	fmt.Printf("Pipeline:      %s\n", r.Pipeline)
	fmt.Printf("Timestamp:     %s\n", r.Timestamp.Format("2006-01-02 15:04"))

	dur := time.Duration(r.Quantitative.TotalDurationMs) * time.Millisecond
	fmt.Printf("Duration:      %s\n", formatElapsed(dur))

	// Steps table
	if len(r.Quantitative.Steps) > 0 {
		fmt.Println()
		fmt.Println("Steps:")
		for _, s := range r.Quantitative.Steps {
			icon := "+"
			if s.Status == "completed" {
				icon = "v"
			} else if s.Status == "failed" {
				icon = "x"
			}
			stepDur := time.Duration(s.DurationMs) * time.Millisecond
			retryLabel := "retries"
			if s.Retries == 1 {
				retryLabel = "retry"
			}
			fmt.Printf("  %s %-16s %8s   %d %s   %s tokens\n",
				icon, s.Name, formatElapsed(stepDur), s.Retries, retryLabel, formatTokens(s.TokensUsed))
		}
	}

	// Summary line
	fmt.Printf("\nSummary: %d steps, %d succeeded, %d failed, %d retries\n",
		r.Quantitative.TotalSteps,
		r.Quantitative.SuccessCount,
		r.Quantitative.FailureCount,
		r.Quantitative.TotalRetries)

	// Narrative section
	if r.Narrative != nil {
		n := r.Narrative
		fmt.Println()
		fmt.Println("Narrative:")
		fmt.Printf("  Smoothness: %s\n", capitalize(n.Smoothness))
		if n.Intent != "" {
			fmt.Printf("  Intent:     %s\n", n.Intent)
		}
		if n.Outcome != "" {
			fmt.Printf("  Outcome:    %s\n", n.Outcome)
		}

		if len(n.FrictionPoints) > 0 {
			fmt.Println()
			fmt.Println("  Friction Points:")
			for _, fp := range n.FrictionPoints {
				fmt.Printf("    - [%s] %s: %s\n", fp.Type, fp.Step, fp.Detail)
			}
		}

		if len(n.Learnings) > 0 {
			fmt.Println()
			fmt.Println("  Learnings:")
			for _, l := range n.Learnings {
				fmt.Printf("    - [%s] %s\n", l.Category, l.Detail)
			}
		}

		if len(n.OpenItems) > 0 {
			fmt.Println()
			fmt.Println("  Open Items:")
			for _, oi := range n.OpenItems {
				fmt.Printf("    - [%s] %s\n", oi.Type, oi.Detail)
			}
		}
	}
}

func printRetroJSON(r *retro.Retrospective) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

// ---------------------------------------------------------------------------
// List output
// ---------------------------------------------------------------------------

func printRetroListText(retros []retro.Retrospective) {
	if len(retros) == 0 {
		fmt.Println("No retrospectives found.")
		return
	}

	fmt.Printf("%-24s %-20s %-12s %-12s %s\n", "RUN ID", "PIPELINE", "SMOOTHNESS", "DURATION", "TIMESTAMP")
	for _, r := range retros {
		smoothness := "-"
		if r.Narrative != nil && r.Narrative.Smoothness != "" {
			smoothness = r.Narrative.Smoothness
		}
		dur := time.Duration(r.Quantitative.TotalDurationMs) * time.Millisecond
		ts := r.Timestamp.Format("2006-01-02 15:04")

		// Truncate long run IDs for display
		runID := r.RunID
		if len(runID) > 24 {
			runID = runID[:21] + "..."
		}

		fmt.Printf("%-24s %-20s %-12s %-12s %s\n", runID, r.Pipeline, smoothness, formatElapsed(dur), ts)
	}
}

func printRetroListJSON(retros []retro.Retrospective) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(retros)
}

// ---------------------------------------------------------------------------
// Stats computation and output
// ---------------------------------------------------------------------------

// retroStats holds aggregated retrospective statistics.
type retroStats struct {
	TotalRuns             int                    `json:"total_runs"`
	SmoothnessDistribution map[string]int         `json:"smoothness_distribution"`
	FrictionFrequency     map[string]int         `json:"friction_frequency"`
	PipelineStats         map[string]pipelineStat `json:"pipeline_stats"`
}

// pipelineStat holds per-pipeline aggregate statistics.
type pipelineStat struct {
	Runs           int     `json:"runs"`
	AvgDurationMs  int64   `json:"avg_duration_ms"`
	SuccessRate    float64 `json:"success_rate"`
	TotalSuccesses int     `json:"total_successes"`
	TotalFailures  int     `json:"total_failures"`
}

func computeRetroStats(retros []retro.Retrospective) retroStats {
	stats := retroStats{
		TotalRuns:             len(retros),
		SmoothnessDistribution: make(map[string]int),
		FrictionFrequency:     make(map[string]int),
		PipelineStats:         make(map[string]pipelineStat),
	}

	for _, r := range retros {
		// Smoothness distribution
		if r.Narrative != nil && r.Narrative.Smoothness != "" {
			stats.SmoothnessDistribution[r.Narrative.Smoothness]++
		}

		// Friction frequency
		if r.Narrative != nil {
			for _, fp := range r.Narrative.FrictionPoints {
				stats.FrictionFrequency[fp.Type]++
			}
		}

		// Pipeline stats
		ps := stats.PipelineStats[r.Pipeline]
		ps.Runs++
		ps.AvgDurationMs += r.Quantitative.TotalDurationMs
		ps.TotalSuccesses += r.Quantitative.SuccessCount
		ps.TotalFailures += r.Quantitative.FailureCount
		stats.PipelineStats[r.Pipeline] = ps
	}

	// Compute averages and rates
	for name, ps := range stats.PipelineStats {
		if ps.Runs > 0 {
			ps.AvgDurationMs = ps.AvgDurationMs / int64(ps.Runs)
			total := ps.TotalSuccesses + ps.TotalFailures
			if total > 0 {
				ps.SuccessRate = float64(ps.TotalSuccesses) / float64(total) * 100
			}
		}
		stats.PipelineStats[name] = ps
	}

	return stats
}

func printRetroStatsText(stats retroStats) {
	fmt.Printf("Retrospective Statistics (%d runs)\n", stats.TotalRuns)

	// Smoothness distribution
	fmt.Println()
	fmt.Println("Smoothness Distribution:")
	smoothnessOrder := []string{
		retro.SmoothnessEffortless,
		retro.SmoothnessSmooth,
		retro.SmoothnessBumpy,
		retro.SmoothnessStruggled,
		retro.SmoothnessFailed,
	}
	for _, s := range smoothnessOrder {
		count := stats.SmoothnessDistribution[s]
		pct := 0.0
		if stats.TotalRuns > 0 {
			pct = float64(count) / float64(stats.TotalRuns) * 100
		}
		fmt.Printf("  %-12s %3d (%4.0f%%)\n", capitalize(s)+":", count, pct)
	}

	// Top friction points
	if len(stats.FrictionFrequency) > 0 {
		fmt.Println()
		fmt.Println("Top Friction Points:")

		// Sort by frequency descending
		type frictionEntry struct {
			Type  string
			Count int
		}
		var entries []frictionEntry
		for t, c := range stats.FrictionFrequency {
			entries = append(entries, frictionEntry{Type: t, Count: c})
		}
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Count > entries[j].Count
		})

		for _, e := range entries {
			fmt.Printf("  %-22s %3d occurrences\n", e.Type+":", e.Count)
		}
	}

	// Pipeline comparison
	if len(stats.PipelineStats) > 0 {
		fmt.Println()
		fmt.Println("Pipeline Comparison:")
		fmt.Printf("  %-20s %5s  %-14s  %s\n", "PIPELINE", "RUNS", "AVG DURATION", "SUCCESS RATE")

		// Sort pipelines by run count descending
		type pipelineEntry struct {
			Name string
			Stat pipelineStat
		}
		var pEntries []pipelineEntry
		for name, ps := range stats.PipelineStats {
			pEntries = append(pEntries, pipelineEntry{Name: name, Stat: ps})
		}
		sort.Slice(pEntries, func(i, j int) bool {
			return pEntries[i].Stat.Runs > pEntries[j].Stat.Runs
		})

		for _, pe := range pEntries {
			avgDur := time.Duration(pe.Stat.AvgDurationMs) * time.Millisecond
			fmt.Printf("  %-20s %5d  %-14s  %.0f%%\n",
				pe.Name, pe.Stat.Runs, formatElapsed(avgDur), pe.Stat.SuccessRate)
		}
	}
}

func printRetroStatsJSON(stats retroStats) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(stats)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// capitalize returns s with the first letter uppercased.
func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

