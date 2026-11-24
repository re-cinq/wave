package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
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
		dataset         string
		pipeline        string
		mode            string
		adapterName     string
		model           string
		label           string
		limit           int
		timeout         int
		concurrency     int
		offset          int
		outputPath      string
		predictionsPath string
		modelName       string
		datasetsDir     string
		workDir         string
		keepWorkspaces  bool
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run a pipeline against benchmark tasks",
		Long: `Execute a Wave pipeline against each task in a JSONL benchmark dataset
(SWE-bench format). Each task gets a fresh worktree of its repository at the
base commit; the pipeline must declare an output artifact named "patch" with
the diff of its changes. The task's test patch is never shown to the solver.

Tasks without a test_cmd (all SWE-bench tasks) end as "patched", not "pass":
write --predictions-path, evaluate it with the official SWE-bench harness and
import the verdicts with 'wave bench report --harness-report'.

Use --mode to select execution mode:
  wave    Run tasks through a Wave pipeline (default)
  claude  Run tasks through standalone Claude Code (baseline)`,
		Example: `  wave bench run --dataset swe-bench-lite.jsonl --pipeline bench-solve --limit 10
  wave bench run --dataset tasks.jsonl --pipeline bench-solve --adapter codex --model gpt-6.2-sol
  wave bench run --dataset tasks.jsonl --pipeline bench-solve --adapter claude --model sonnet \
    --results-path results.json --predictions-path predictions.jsonl
  wave bench run --dataset tasks.jsonl --mode claude --label baseline-v1`,
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
			if mode == bench.ModeClaude && adapterName != "" && adapterName != "claude" {
				return NewCLIError(CodeInvalidArgs, "--adapter cannot be combined with --mode=claude", "Drop --adapter or use --mode wave")
			}

			// Resolve dataset path
			datasetPath := dataset
			if !filepath.IsAbs(datasetPath) {
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
				Adapter:        adapterName,
				Model:          model,
				RunLabel:       label,
				Limit:          limit,
				DatasetPath:    datasetPath,
				WorkDir:        workDir,
				KeepWorkspaces: keepWorkspaces,
				Concurrency:    concurrency,
				Offset:         offset,
			}
			if timeout > 0 {
				cfg.TaskTimeout = time.Duration(timeout) * time.Second
			}

			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
			defer stop()

			runner := bench.NewSubprocessRunner(filepath.Join(workDir, "repos"))
			report, err := bench.RunBenchmark(ctx, tasks, cfg, runner)
			if err != nil && report == nil {
				return NewCLIError(CodeInternalError, fmt.Sprintf("benchmark failed: %s", err), "Check adapter availability and task configuration").WithCause(err)
			}

			format := benchFormat(cmd)

			if outputPath != "" {
				if err := writeReportFile(report, outputPath); err != nil {
					return NewCLIError(CodeInternalError, fmt.Sprintf("write results: %s", err), "Check write permissions for the output path").WithCause(err)
				}
				fmt.Fprintf(os.Stderr, "Results written to %s\n", outputPath)
			}
			if predictionsPath != "" {
				name := modelName
				if name == "" {
					name = defaultModelName(report)
				}
				if err := bench.WritePredictions(predictionsPath, bench.Predictions(report, name)); err != nil {
					return NewCLIError(CodeInternalError, fmt.Sprintf("write predictions: %s", err), "Check write permissions for the predictions path").WithCause(err)
				}
				fmt.Fprintf(os.Stderr, "SWE-bench predictions written to %s\n", predictionsPath)
			}

			if format == "json" {
				if err := renderBenchReportJSON(report); err != nil {
					return err
				}
			} else {
				renderBenchReportText(report)
			}

			if report.Errors > 0 {
				return NewCLIError(CodeInternalError, fmt.Sprintf("%d of %d task(s) errored", report.Errors, report.Total), "Inspect each task's log_path in the results file")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&dataset, "dataset", "", "Path to JSONL dataset file")
	cmd.Flags().StringVar(&pipeline, "pipeline", "", "Pipeline name to execute per task")
	cmd.Flags().StringVar(&mode, "mode", "", "Execution mode: wave (default) or claude")
	cmd.Flags().StringVar(&adapterName, "adapter", "", "Adapter passed to 'wave init' and 'wave run' (e.g. claude, codex)")
	cmd.Flags().StringVar(&model, "model", "", "Model forced onto every step (claude mode: passed to claude --model)")
	cmd.Flags().StringVar(&label, "label", "", "Human-readable label for this run")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of tasks to run (0 = all)")
	cmd.Flags().IntVar(&timeout, "timeout", 0, "Per-task timeout in seconds (0 = no limit)")
	cmd.Flags().StringVar(&outputPath, "results-path", "", "Path to write JSON results file")
	cmd.Flags().StringVar(&predictionsPath, "predictions-path", "", "Path to write SWE-bench predictions (JSONL) for the official harness")
	cmd.Flags().StringVar(&modelName, "model-name", "", "model_name_or_path for predictions (default: label or wave-<pipeline>-<adapter>-<model>)")
	cmd.Flags().StringVar(&datasetsDir, "datasets-dir", ".agents/bench/datasets", "Directory to search for dataset files")
	cmd.Flags().StringVar(&workDir, "work-dir", ".agents/bench/workspaces", "Directory for task worktrees, logs and the repo cache")
	cmd.Flags().BoolVar(&keepWorkspaces, "keep-workspaces", false, "Preserve task worktrees after completion")
	cmd.Flags().IntVar(&concurrency, "concurrency", 1, "Number of tasks to run in parallel")
	cmd.Flags().IntVar(&offset, "offset", 0, "Skip the first N tasks in the dataset")

	return cmd
}

// defaultModelName builds a model_name_or_path for SWE-bench predictions.
func defaultModelName(r *bench.BenchReport) string {
	if r.RunLabel != "" {
		return r.RunLabel
	}
	parts := []string{r.Mode}
	for _, p := range []string{r.Pipeline, r.Adapter, r.Model} {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return strings.Join(parts, "-")
}

func benchFormat(cmd *cobra.Command) string {
	format := ResolveFormat(cmd, "text")
	if GetOutputConfig(cmd).Format == OutputFormatJSON {
		format = "json"
	}
	return format
}

func newBenchReportCmd() *cobra.Command {
	var resultsPath, harnessReport, outPath string

	cmd := &cobra.Command{
		Use:   "report",
		Short: "Generate a summary from benchmark results",
		Long: `Read a JSON results file from a previous bench run and display a summary.

With --harness-report, per-task verdicts from the official SWE-bench harness
summary (<model>.<run_id>.json) are applied first: resolved -> pass,
unresolved or empty patch -> fail, harness error or not evaluated -> error.
Use --out to save the evaluated results for 'wave bench compare'.`,
		Example: `  wave bench report --results results.json
  wave bench report --results results.json --harness-report wave.r1.json --out evaluated.json
  wave bench report --results results.json --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true

			if resultsPath == "" {
				return NewCLIError(CodeInvalidArgs, "--results is required", "Specify a results file with --results <path>")
			}

			report, err := loadReportFile(resultsPath)
			if err != nil {
				return NewCLIError(CodeDatasetError, err.Error(), "Check that the results file exists and is valid JSON").WithCause(err)
			}

			if harnessReport != "" {
				hr, err := bench.LoadHarnessReport(harnessReport)
				if err != nil {
					return NewCLIError(CodeDatasetError, err.Error(), "Pass the summary JSON written by swebench.harness.run_evaluation").WithCause(err)
				}
				bench.ApplyHarnessReport(report, hr, filepath.Base(harnessReport))
			}
			if outPath != "" {
				if err := writeReportFile(report, outPath); err != nil {
					return NewCLIError(CodeInternalError, fmt.Sprintf("write results: %s", err), "Check write permissions for the output path").WithCause(err)
				}
			}

			if benchFormat(cmd) == "json" {
				return renderBenchReportJSON(report)
			}
			renderBenchReportText(report)
			return nil
		},
	}

	cmd.Flags().StringVar(&resultsPath, "results", "", "Path to JSON results file")
	cmd.Flags().StringVar(&harnessReport, "harness-report", "", "SWE-bench harness summary report to take verdicts from")
	cmd.Flags().StringVar(&outPath, "out", "", "Write the (evaluated) results to this path")

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
				// Directory doesn't exist - not an error, just empty
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

			switch benchFormat(cmd) {
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

			switch benchFormat(cmd) {
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
	if report.Adapter != "" || report.Model != "" {
		fmt.Fprintf(os.Stdout, "Adapter: %s %s\n", report.Adapter, report.Model)
	}
	if report.RunLabel != "" {
		fmt.Fprintf(os.Stdout, "Label:   %s\n", report.RunLabel)
	}
	evaluation := report.Evaluation
	if evaluation == "" {
		evaluation = "none (pass rate counts verified tasks only)"
	}
	fmt.Fprintf(os.Stdout, "Verdicts: %s\n", evaluation)
	fmt.Fprintln(os.Stdout, "-------------------------------------")
	fmt.Fprintf(os.Stdout, "Total:     %d\n", report.Total)
	fmt.Fprintf(os.Stdout, "Passed:    %d\n", report.Passed)
	fmt.Fprintf(os.Stdout, "Failed:    %d\n", report.Failed)
	fmt.Fprintf(os.Stdout, "Errors:    %d\n", report.Errors)
	if report.Patched > 0 {
		fmt.Fprintf(os.Stdout, "Patched:   %d (not yet evaluated)\n", report.Patched)
	}
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
	fmt.Fprintln(os.Stdout, "-------------------------------------")
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
			fmt.Fprintf(os.Stdout, "%-40s %-12s %-8s -> %-8s\n", "TASK", "CHANGE", "BASE", "COMPARE")
			hasChanges = true
		}
		fmt.Fprintf(os.Stdout, "%-40s %-12s %-8s -> %-8s\n", d.TaskID, d.Change, d.BaseStatus, d.CompStatus)
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
