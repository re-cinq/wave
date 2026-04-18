package commands

import (
	"context"
	"encoding/json"
	"errors"
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
  list     List available benchmark datasets
  compare  Compare two benchmark result files`,
		Example: `  wave bench run --dataset swe-bench-lite.jsonl --pipeline bench-solve
  wave bench report --results results.json
  wave bench compare --base baseline.json --compare wave-run.json
  wave bench list`,
	}

	cmd.AddCommand(newBenchRunCmd())
	cmd.AddCommand(newBenchReportCmd())
	cmd.AddCommand(newBenchListCmd())
	cmd.AddCommand(newBenchCompareCmd())

	return cmd
}

func newBenchRunCmd() *cobra.Command {
	var (
		dataset        string
		pipeline       string
		mode           string
		label          string
		limit          int
		timeout        int
		concurrency    int
		offset         int
		outputPath     string
		datasetsDir    string
		keepWorkspaces bool
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run a pipeline against benchmark tasks",
		Long: `Execute a Wave pipeline against each task in a JSONL benchmark dataset.
Results are written to a JSON file. Use --concurrency to run tasks in parallel.

Use --mode to select execution mode:
  wave    Run tasks through a Wave pipeline (default)
  claude  Run tasks through standalone Claude Code`,
		Example: `  wave bench run --dataset swe-bench-lite.jsonl --pipeline bench-solve
  wave bench run --dataset tasks.jsonl --pipeline bench-solve --limit 10
  wave bench run --dataset tasks.jsonl --mode claude --label baseline-v1
  wave bench run --dataset tasks.jsonl --pipeline bench-solve --results-path results.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true

			if dataset == "" {
				return NewCLIError(CodeInvalidArgs, "--dataset is required", "Specify a dataset file with --dataset <path>")
			}
			if mode == "" {
				mode = bench.ModeWave
			}
			if mode != bench.ModeWave && mode != bench.ModeClaude {
				return NewCLIError(CodeInvalidArgs, fmt.Sprintf("--mode must be %q or %q", bench.ModeWave, bench.ModeClaude), "Use --mode wave or --mode claude")
			}
			if pipeline == "" && mode != bench.ModeClaude {
				return NewCLIError(CodeInvalidArgs, "--pipeline is required (unless --mode=claude)", "Specify a pipeline with --pipeline <name>")
			}

			// Resolve dataset path
			datasetPath := dataset
			if !filepath.IsAbs(datasetPath) {
				// Check in datasets directory first
				if datasetsDir == "" {
					datasetsDir = ".agents/bench/datasets"
				}
				candidate := filepath.Join(datasetsDir, datasetPath)
				if _, err := os.Stat(candidate); err == nil {
					datasetPath = candidate
				}
			}

			tasks, err := bench.LoadDataset(datasetPath)
			if err != nil {
				return NewCLIError(CodeDatasetError, fmt.Sprintf("load dataset: %s", err), "Check that the dataset file exists and is valid JSONL").WithCause(err)
			}

			fmt.Fprintf(os.Stderr, "Loaded %d tasks from %s\n", len(tasks), datasetPath)

			cfg := bench.RunConfig{
				Pipeline:       pipeline,
				Mode:           mode,
				RunLabel:       label,
				Limit:          limit,
				DatasetPath:    datasetPath,
				WorkDir:        ".agents/bench/workspaces",
				KeepWorkspaces: keepWorkspaces,
				Concurrency:    concurrency,
				Offset:         offset,
			}
			if timeout > 0 {
				cfg.TaskTimeout = time.Duration(timeout) * time.Second
			}

			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
			defer stop()

			cacheDir := filepath.Join(".agents/bench/workspaces", "repos")
			runner := bench.NewSubprocessRunner(cacheDir)
			report, err := bench.RunBenchmark(ctx, tasks, cfg, runner)
			if err != nil && report == nil {
				return NewCLIError(CodeInternalError, fmt.Sprintf("benchmark failed: %s", err), "Check adapter availability and task configuration").WithCause(err)
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
					return NewCLIError(CodeInternalError, fmt.Sprintf("write results: %s", err), "Check write permissions for the output path").WithCause(err)
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
	cmd.Flags().StringVar(&mode, "mode", "", "Execution mode: wave (default) or claude")
	cmd.Flags().StringVar(&label, "label", "", "Human-readable label for this run")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of tasks to run (0 = all)")
	cmd.Flags().IntVar(&timeout, "timeout", 0, "Per-task timeout in seconds (0 = no limit)")
	cmd.Flags().StringVar(&outputPath, "results-path", "", "Path to write JSON results file")
	cmd.Flags().StringVar(&datasetsDir, "datasets-dir", ".agents/bench/datasets", "Directory to search for dataset files")
	cmd.Flags().BoolVar(&keepWorkspaces, "keep-workspaces", false, "Preserve task worktrees after completion")
	cmd.Flags().IntVar(&concurrency, "concurrency", 1, "Number of tasks to run in parallel")
	cmd.Flags().IntVar(&offset, "offset", 0, "Skip the first N tasks in the dataset")

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
				return NewCLIError(CodeInvalidArgs, "--results is required", "Specify a results file with --results <path>")
			}

			data, err := os.ReadFile(resultsPath)
			if err != nil {
				return NewCLIError(CodeDatasetError, fmt.Sprintf("read results file: %s", err), "Check that the results file exists and is readable").WithCause(err)
			}

			var report bench.BenchReport
			if err := json.Unmarshal(data, &report); err != nil {
				return NewCLIError(CodeDatasetError, fmt.Sprintf("parse results file: %s", err), "The results file is not valid JSON").WithCause(err)
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
				datasetsDir = ".agents/bench/datasets"
			}

			datasets, err := bench.ListDatasets(datasetsDir)
			if err != nil {
				// Directory doesn't exist — not an error, just empty
				if errors.Is(err, os.ErrNotExist) {
					fmt.Fprintln(os.Stderr, "No datasets directory found. Create .agents/bench/datasets/ and add .jsonl files.")
					return nil
				}
				return NewCLIError(CodeInternalError, fmt.Sprintf("list datasets: %s", err), "Check datasets directory permissions").WithCause(err)
			}

			if len(datasets) == 0 {
				fmt.Fprintln(os.Stderr, "No datasets found. Add .jsonl files to .agents/bench/datasets/")
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

	cmd.Flags().StringVar(&datasetsDir, "datasets-dir", ".agents/bench/datasets", "Directory to search for dataset files")

	return cmd
}

func newBenchCompareCmd() *cobra.Command {
	var (
		basePath    string
		comparePath string
	)

	cmd := &cobra.Command{
		Use:   "compare",
		Short: "Compare two benchmark result files",
		Long: `Load two benchmark result files and show per-task differences.
Identifies tasks that improved, regressed, or stayed the same.`,
		Example: `  wave bench compare --base baseline.json --compare wave-run.json
  wave bench compare --base baseline.json --compare wave-run.json --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true

			if basePath == "" {
				return NewCLIError(CodeInvalidArgs, "--base is required", "Specify a base results file with --base <path>")
			}
			if comparePath == "" {
				return NewCLIError(CodeInvalidArgs, "--compare is required", "Specify a comparison results file with --compare <path>")
			}

			baseReport, err := loadReportFile(basePath)
			if err != nil {
				return NewCLIError(CodeDatasetError, fmt.Sprintf("load base report: %s", err), "Check that the base results file exists and is valid JSON").WithCause(err)
			}

			compReport, err := loadReportFile(comparePath)
			if err != nil {
				return NewCLIError(CodeDatasetError, fmt.Sprintf("load compare report: %s", err), "Check that the comparison results file exists and is valid JSON").WithCause(err)
			}

			cr := bench.Compare(baseReport, compReport)

			format := ResolveFormat(cmd, "text")
			outputCfg := GetOutputConfig(cmd)
			if outputCfg.Format == OutputFormatJSON {
				format = "json"
			}

			switch format {
			case "json":
				return renderCompareReportJSON(cr)
			default:
				renderCompareReportText(cr)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&basePath, "base", "", "Path to base/baseline results JSON")
	cmd.Flags().StringVar(&comparePath, "compare", "", "Path to comparison results JSON")

	return cmd
}

func loadReportFile(path string) (*bench.BenchReport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var report bench.BenchReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	report.Tally()
	return &report, nil
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
	if report.Mode != "" {
		fmt.Fprintf(os.Stdout, "Mode:    %s\n", report.Mode)
	}
	if report.RunLabel != "" {
		fmt.Fprintf(os.Stdout, "Label:   %s\n", report.RunLabel)
	}
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

func renderCompareReportJSON(cr *bench.CompareReport) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(cr)
}

func renderCompareReportText(cr *bench.CompareReport) {
	fmt.Fprintln(os.Stdout)
	fmt.Fprintf(os.Stdout, "Comparison: %s vs %s\n", describeRef(cr.Base), describeRef(cr.Compare))
	fmt.Fprintln(os.Stdout, "─────────────────────────────────────")
	fmt.Fprintf(os.Stdout, "Base pass rate:    %.1f%% (%d/%d)\n", cr.Base.PassRate*100, cr.Base.Passed, cr.Base.Total)
	fmt.Fprintf(os.Stdout, "Compare pass rate: %.1f%% (%d/%d)\n", cr.Compare.PassRate*100, cr.Compare.Passed, cr.Compare.Total)
	sign := "+"
	if cr.Summary.DeltaRate < 0 {
		sign = ""
	}
	fmt.Fprintf(os.Stdout, "Delta:             %s%.1f%%\n", sign, cr.Summary.DeltaRate*100)
	fmt.Fprintln(os.Stdout)
	fmt.Fprintf(os.Stdout, "Improved:   %d\n", cr.Summary.Improved)
	fmt.Fprintf(os.Stdout, "Regressed:  %d\n", cr.Summary.Regressed)
	fmt.Fprintf(os.Stdout, "Unchanged:  %d\n", cr.Summary.Unchanged)
	if cr.Summary.OnlyInBase > 0 {
		fmt.Fprintf(os.Stdout, "Only base:  %d\n", cr.Summary.OnlyInBase)
	}
	if cr.Summary.OnlyInComp > 0 {
		fmt.Fprintf(os.Stdout, "Only comp:  %d\n", cr.Summary.OnlyInComp)
	}
	fmt.Fprintln(os.Stdout)

	// Show per-task changes (only non-unchanged).
	hasChanges := false
	for _, d := range cr.Diffs {
		if d.Change == "unchanged" {
			continue
		}
		if !hasChanges {
			fmt.Fprintf(os.Stdout, "%-40s %-12s %-8s → %-8s\n", "TASK", "CHANGE", "BASE", "COMPARE")
			hasChanges = true
		}
		fmt.Fprintf(os.Stdout, "%-40s %-12s %-8s → %-8s\n", d.TaskID, d.Change, d.BaseStatus, d.CompStatus)
	}
}

func describeRef(ref bench.ReportRef) string {
	if ref.RunLabel != "" {
		return ref.RunLabel
	}
	if ref.Mode != "" {
		return ref.Mode + "/" + ref.Pipeline
	}
	return ref.Pipeline
}
