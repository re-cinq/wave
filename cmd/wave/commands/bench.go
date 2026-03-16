package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/recinq/wave/internal/bench"
	"github.com/spf13/cobra"
)

// NewBenchCmd creates the bench command group for SWE-bench benchmarking.
func NewBenchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bench",
		Short: "Run and analyze SWE-bench benchmarks",
		Long: `Run Wave pipelines against SWE-bench benchmark tasks and generate
pass/fail reports. Useful for measuring pipeline quality and comparing
different pipeline configurations.

Subcommands:
  run      Execute a pipeline against benchmark tasks
  report   Generate a summary from benchmark results
  list     List available benchmark datasets`,
	}

	cmd.AddCommand(newBenchRunCmd())
	cmd.AddCommand(newBenchReportCmd())
	cmd.AddCommand(newBenchListCmd())

	return cmd
}

func newBenchRunCmd() *cobra.Command {
	var (
		dataset     string
		pipeline    string
		limit       int
		timeout     int
		outputPath  string
		datasetsDir string
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run a pipeline against benchmark tasks",
		Long: `Execute a Wave pipeline against each task in a JSONL benchmark dataset.
Tasks are run sequentially. Results are written to a JSON file.`,
		Example: `  wave bench run --dataset swe-bench-lite.jsonl --pipeline impl-issue
  wave bench run --dataset tasks.jsonl --pipeline impl-issue --limit 10
  wave bench run --dataset tasks.jsonl --pipeline impl-issue --output results.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true

			if dataset == "" {
				return fmt.Errorf("--dataset is required")
			}
			if pipeline == "" {
				return fmt.Errorf("--pipeline is required")
			}

			// Resolve dataset path
			datasetPath := dataset
			if !filepath.IsAbs(datasetPath) {
				// Check in datasets directory first
				if datasetsDir == "" {
					datasetsDir = ".wave/bench/datasets"
				}
				candidate := filepath.Join(datasetsDir, datasetPath)
				if _, err := os.Stat(candidate); err == nil {
					datasetPath = candidate
				}
			}

			tasks, err := bench.LoadDataset(datasetPath)
			if err != nil {
				return fmt.Errorf("load dataset: %w", err)
			}

			fmt.Fprintf(os.Stderr, "Loaded %d tasks from %s\n", len(tasks), datasetPath)

			cfg := bench.RunConfig{
				Pipeline:    pipeline,
				Limit:       limit,
				DatasetPath: datasetPath,
				WorkDir:     ".wave/bench/workspaces",
			}
			if timeout > 0 {
				cfg.TaskTimeout = time.Duration(timeout) * time.Second
			}

			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
			defer stop()

			runner := &bench.SubprocessRunner{}
			report, err := bench.RunBenchmark(ctx, tasks, cfg, runner)
			if err != nil && report == nil {
				return fmt.Errorf("benchmark failed: %w", err)
			}

			// Determine output format
			format := ResolveFormat(cmd, "text")
			outputCfg := GetOutputConfig(cmd)
			if outputCfg.Format == OutputFormatJSON {
				format = "json"
			}

			// Write results file if requested
			if outputPath != "" {
				if err := writeReportFile(report, outputPath); err != nil {
					return fmt.Errorf("write results: %w", err)
				}
				fmt.Fprintf(os.Stderr, "Results written to %s\n", outputPath)
			}

			switch format {
			case "json":
				return renderBenchReportJSON(report)
			default:
				renderBenchReportText(report)
			}

			if report.Errors > 0 {
				os.Exit(2)
			}
			if report.Failed > 0 {
				os.Exit(1)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&dataset, "dataset", "", "Path to JSONL dataset file")
	cmd.Flags().StringVar(&pipeline, "pipeline", "", "Pipeline name to execute per task")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of tasks to run (0 = all)")
	cmd.Flags().IntVar(&timeout, "timeout", 0, "Per-task timeout in seconds (0 = no limit)")
	cmd.Flags().StringVar(&outputPath, "output", "", "Path to write JSON results file")
	cmd.Flags().StringVar(&datasetsDir, "datasets-dir", ".wave/bench/datasets", "Directory to search for dataset files")

	return cmd
}

func newBenchReportCmd() *cobra.Command {
	var resultsPath string

	cmd := &cobra.Command{
		Use:   "report",
		Short: "Generate a summary from benchmark results",
		Long:  `Read a JSON results file from a previous bench run and display a summary.`,
		Example: `  wave bench report --results results.json
  wave bench report --results results.json --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true

			if resultsPath == "" {
				return fmt.Errorf("--results is required")
			}

			data, err := os.ReadFile(resultsPath)
			if err != nil {
				return fmt.Errorf("read results file: %w", err)
			}

			var report bench.BenchReport
			if err := json.Unmarshal(data, &report); err != nil {
				return fmt.Errorf("parse results file: %w", err)
			}

			// Recalculate in case the file was hand-edited
			report.Tally()

			format := ResolveFormat(cmd, "text")
			outputCfg := GetOutputConfig(cmd)
			if outputCfg.Format == OutputFormatJSON {
				format = "json"
			}

			switch format {
			case "json":
				return renderBenchReportJSON(&report)
			default:
				renderBenchReportText(&report)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&resultsPath, "results", "", "Path to JSON results file")

	return cmd
}

func newBenchListCmd() *cobra.Command {
	var datasetsDir string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available benchmark datasets",
		Long:  `Scan the datasets directory for .jsonl files and list them.`,
		Example: `  wave bench list
  wave bench list --datasets-dir ./my-datasets`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true

			if datasetsDir == "" {
				datasetsDir = ".wave/bench/datasets"
			}

			datasets, err := bench.ListDatasets(datasetsDir)
			if err != nil {
				// Directory doesn't exist — not an error, just empty
				if os.IsNotExist(err) {
					fmt.Fprintln(os.Stderr, "No datasets directory found. Create .wave/bench/datasets/ and add .jsonl files.")
					return nil
				}
				return fmt.Errorf("list datasets: %w", err)
			}

			if len(datasets) == 0 {
				fmt.Fprintln(os.Stderr, "No datasets found. Add .jsonl files to .wave/bench/datasets/")
				return nil
			}

			format := ResolveFormat(cmd, "text")
			outputCfg := GetOutputConfig(cmd)
			if outputCfg.Format == OutputFormatJSON {
				format = "json"
			}

			switch format {
			case "json":
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(datasets)
			default:
				fmt.Fprintf(os.Stdout, "%-30s %s\n", "NAME", "PATH")
				for _, ds := range datasets {
					fmt.Fprintf(os.Stdout, "%-30s %s\n", ds.Name, ds.Path)
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&datasetsDir, "datasets-dir", ".wave/bench/datasets", "Directory to search for dataset files")

	return cmd
}

func writeReportFile(report *bench.BenchReport, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func renderBenchReportJSON(report *bench.BenchReport) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}

func renderBenchReportText(report *bench.BenchReport) {
	fmt.Fprintln(os.Stdout)
	fmt.Fprintf(os.Stdout, "Benchmark Report: %s\n", report.Pipeline)
	fmt.Fprintf(os.Stdout, "Dataset: %s\n", report.Dataset)
	fmt.Fprintln(os.Stdout, "─────────────────────────────────────")
	fmt.Fprintf(os.Stdout, "Total:     %d\n", report.Total)
	fmt.Fprintf(os.Stdout, "Passed:    %d\n", report.Passed)
	fmt.Fprintf(os.Stdout, "Failed:    %d\n", report.Failed)
	fmt.Fprintf(os.Stdout, "Errors:    %d\n", report.Errors)
	fmt.Fprintf(os.Stdout, "Pass rate: %.1f%%\n", report.PassRate*100)
	if report.DurationMs > 0 {
		fmt.Fprintf(os.Stdout, "Duration:  %s\n", time.Duration(report.DurationMs)*time.Millisecond)
	}
	fmt.Fprintln(os.Stdout)

	if len(report.Results) > 0 {
		fmt.Fprintf(os.Stdout, "%-40s %-8s %10s\n", "TASK", "STATUS", "DURATION")
		for _, r := range report.Results {
			dur := ""
			if r.DurationMs > 0 {
				dur = (time.Duration(r.DurationMs) * time.Millisecond).String()
			}
			fmt.Fprintf(os.Stdout, "%-40s %-8s %10s\n", r.TaskID, r.Status, dur)
		}
	}
}
