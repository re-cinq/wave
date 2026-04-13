package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/retro"
	"github.com/recinq/wave/internal/state"
	"github.com/spf13/cobra"
)

// NewRetroCmd creates the wave retro command with subcommands.
func NewRetroCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "retro [run-id]",
		Short: "View and manage run retrospectives",
		Long: `View, list, and generate retrospectives for pipeline runs.
Retrospectives combine quantitative metrics with optional LLM-powered
narrative analysis to identify friction points and learnings.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			if len(args) == 0 {
				return cmd.Help()
			}
			narrate, _ := cmd.Flags().GetBool("narrate")
			jsonOutput, _ := cmd.Flags().GetBool("json")
			return runRetroView(args[0], narrate, jsonOutput)
		},
	}

	cmd.Flags().Bool("narrate", false, "Generate or regenerate LLM narrative")
	cmd.Flags().Bool("json", false, "Output in JSON format")

	cmd.AddCommand(newRetroListCmd())
	cmd.AddCommand(newRetroStatsCmd())

	return cmd
}

func runRetroView(runID string, narrate bool, jsonOutput bool) error {
	store, err := state.NewStateStore(".wave/state.db")
	if err != nil {
		return NewCLIError(CodeStateDBError, "failed to open state store: "+err.Error(),
			"Check that .wave/state.db exists and is not corrupted.").WithCause(err)
	}
	defer store.Close()

	storage := retro.NewStorage(".wave/retros", store)

	if narrate {
		m, runner, err := loadManifestAndRunner()
		if err != nil {
			return err
		}
		gen := retro.NewGenerator(store, runner, ".wave/retros", &m.Runtime.Retros)
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		if err := gen.GenerateNarrativeSync(ctx, runID); err != nil {
			return NewCLIError(CodeInternalError, "narrative generation failed: "+err.Error(),
				"Check adapter configuration and try again.").WithCause(err)
		}
		fmt.Fprintf(os.Stderr, "Narrative generated for run %s\n", runID)
	}

	r, err := storage.Load(runID)
	if err != nil {
		return NewCLIError(CodeRunNotFound, fmt.Sprintf("retrospective not found for run %s", runID),
			"Run 'wave retro list' to see available retrospectives.").WithCause(err)
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(r)
	}

	printRetro(r)
	return nil
}

func printRetro(r *retro.Retrospective) {
	fmt.Printf("Retrospective: %s\n", r.RunID)
	fmt.Printf("Pipeline:      %s\n", r.Pipeline)
	fmt.Printf("Timestamp:     %s\n", r.Timestamp.Format(time.RFC3339))
	fmt.Println()

	if r.Quantitative != nil {
		q := r.Quantitative
		fmt.Printf("Duration:  %s\n", formatDurationMs(q.TotalDurationMs))
		fmt.Printf("Steps:     %d total, %d succeeded, %d failed\n", q.TotalSteps, q.SuccessCount, q.FailureCount)
		fmt.Printf("Retries:   %d\n", q.TotalRetries)
		fmt.Printf("Tokens:    %d\n", q.TotalTokens)
		fmt.Println()

		if len(q.Steps) > 0 {
			fmt.Println("Steps:")
			for _, s := range q.Steps {
				retryStr := ""
				if s.Retries > 0 {
					retryStr = fmt.Sprintf(" (%d retries)", s.Retries)
				}
				fmt.Printf("  %-20s %8s  %s%s\n", s.Name, formatDurationMs(s.DurationMs), s.Status, retryStr)
			}
			fmt.Println()
		}
	}

	if r.Narrative != nil {
		n := r.Narrative
		fmt.Printf("Smoothness: %s\n", n.Smoothness)
		fmt.Printf("Intent:     %s\n", n.Intent)
		fmt.Printf("Outcome:    %s\n", n.Outcome)

		if len(n.FrictionPoints) > 0 {
			fmt.Println("\nFriction Points:")
			for _, f := range n.FrictionPoints {
				fmt.Printf("  [%s] %s: %s\n", f.Type, f.Step, f.Detail)
			}
		}

		if len(n.Learnings) > 0 {
			fmt.Println("\nLearnings:")
			for _, l := range n.Learnings {
				fmt.Printf("  [%s] %s\n", l.Category, l.Detail)
			}
		}

		if len(n.OpenItems) > 0 {
			fmt.Println("\nOpen Items:")
			for _, o := range n.OpenItems {
				fmt.Printf("  [%s] %s\n", o.Type, o.Detail)
			}
		}

		if len(n.Recommendations) > 0 {
			fmt.Println("\nRecommendations:")
			for _, r := range n.Recommendations {
				fmt.Printf("  - %s\n", r)
			}
		}
	} else {
		fmt.Println("(no narrative — run with --narrate to generate)")
	}
}

func newRetroListCmd() *cobra.Command {
	var pipelineFilter string
	var since string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List retrospectives",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			return runRetroList(pipelineFilter, since)
		},
	}

	cmd.Flags().StringVar(&pipelineFilter, "pipeline", "", "Filter by pipeline name")
	cmd.Flags().StringVar(&since, "since", "", "Show retros since duration (e.g. 7d, 24h)")

	return cmd
}

func runRetroList(pipelineFilter, since string) error {
	store, err := state.NewStateStore(".wave/state.db")
	if err != nil {
		return NewCLIError(CodeStateDBError, "failed to open state store: "+err.Error(),
			"Check that .wave/state.db exists and is not corrupted.").WithCause(err)
	}
	defer store.Close()

	storage := retro.NewStorage(".wave/retros", store)

	var sinceTime time.Time
	if since != "" {
		d, err := parseSinceDuration(since)
		if err != nil {
			return NewCLIError(CodeInvalidArgs, "invalid --since value: "+err.Error(),
				"Use a duration like '7d', '24h', or '30m'.").WithCause(err)
		}
		sinceTime = time.Now().Add(-d)
	}

	records, err := storage.List(pipelineFilter, sinceTime, 50)
	if err != nil {
		return NewCLIError(CodeStateDBError, "failed to list retrospectives: "+err.Error(),
			"Check that .wave/state.db is accessible.").WithCause(err)
	}

	if len(records) == 0 {
		fmt.Println("No retrospectives found")
		return nil
	}

	fmt.Printf("%-40s %-20s %-12s %-14s %s\n", "RUN ID", "PIPELINE", "SMOOTHNESS", "STATUS", "TIMESTAMP")
	fmt.Println(strings.Repeat("-", 100))
	for _, r := range records {
		smoothness := r.Smoothness
		if smoothness == "" {
			smoothness = "-"
		}
		fmt.Printf("%-40s %-20s %-12s %-14s %s\n",
			truncate(r.RunID, 40),
			truncate(r.PipelineName, 20),
			smoothness,
			r.Status,
			r.CreatedAt.Format("2006-01-02 15:04"),
		)
	}

	return nil
}

func newRetroStatsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stats",
		Short: "Show aggregate retrospective statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			return runRetroStats()
		},
	}
}

func runRetroStats() error {
	store, err := state.NewStateStore(".wave/state.db")
	if err != nil {
		return NewCLIError(CodeStateDBError, "failed to open state store: "+err.Error(),
			"Check that .wave/state.db exists and is not corrupted.").WithCause(err)
	}
	defer store.Close()

	storage := retro.NewStorage(".wave/retros", store)

	records, err := storage.List("", time.Time{}, 1000)
	if err != nil {
		return NewCLIError(CodeStateDBError, "failed to list retrospectives: "+err.Error(),
			"Check that .wave/state.db is accessible.").WithCause(err)
	}

	if len(records) == 0 {
		fmt.Println("No retrospectives found")
		return nil
	}

	// Smoothness distribution
	smoothnessCounts := make(map[string]int)
	pipelineCounts := make(map[string]int)
	pipelineSmooth := make(map[string]map[string]int)
	total := len(records)

	for _, r := range records {
		if r.Smoothness != "" {
			smoothnessCounts[r.Smoothness]++
		}
		pipelineCounts[r.PipelineName]++
		if _, ok := pipelineSmooth[r.PipelineName]; !ok {
			pipelineSmooth[r.PipelineName] = make(map[string]int)
		}
		if r.Smoothness != "" {
			pipelineSmooth[r.PipelineName][r.Smoothness]++
		}
	}

	fmt.Printf("Total Retrospectives: %d\n\n", total)

	fmt.Println("Smoothness Distribution:")
	for _, s := range []string{"effortless", "smooth", "bumpy", "struggled", "failed"} {
		count := smoothnessCounts[s]
		pct := float64(count) / float64(total) * 100
		bar := strings.Repeat("#", int(pct/2))
		fmt.Printf("  %-12s %3d (%5.1f%%) %s\n", s, count, pct, bar)
	}

	fmt.Println("\nPipeline Breakdown:")
	fmt.Printf("  %-30s %5s\n", "PIPELINE", "RUNS")
	fmt.Println("  " + strings.Repeat("-", 40))
	for pipeline, count := range pipelineCounts {
		fmt.Printf("  %-30s %5d\n", truncate(pipeline, 30), count)
	}

	return nil
}

func formatDurationMs(ms int64) string {
	d := time.Duration(ms) * time.Millisecond
	if d < time.Second {
		return fmt.Sprintf("%dms", ms)
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%.1fm", d.Minutes())
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func parseSinceDuration(s string) (time.Duration, error) {
	// Support day suffix (e.g., "7d")
	if strings.HasSuffix(s, "d") {
		s = strings.TrimSuffix(s, "d")
		var days int
		if _, err := fmt.Sscanf(s, "%d", &days); err != nil {
			return 0, fmt.Errorf("invalid day format: %s", s)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}

func loadManifestAndRunner() (*manifest.Manifest, adapter.AdapterRunner, error) {
	mp, err := loadManifestStrict("wave.yaml")
	if err != nil {
		return nil, nil, err
	}
	m := *mp

	var adapterName string
	for name := range m.Adapters {
		adapterName = name
		break
	}
	runner := adapter.ResolveAdapter(adapterName)

	return &m, runner, nil
}
