package bench

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// RunConfig holds the configuration for a benchmark run.
type RunConfig struct {
	// Pipeline is the name of the Wave pipeline to execute per task.
	Pipeline string
	// Limit caps the number of tasks to run (0 = no limit).
	Limit int
	// DatasetPath is the path to the JSONL dataset file.
	DatasetPath string
	// WaveBinary is the path to the wave binary. Defaults to "wave".
	WaveBinary string
	// WorkDir is the root directory for benchmark workspaces.
	// Defaults to ".wave/bench".
	WorkDir string
	// Timeout per task. Zero means no timeout.
	TaskTimeout time.Duration
}

// PipelineRunner is the interface for executing a single pipeline against a task.
// This enables testing without subprocess execution.
type PipelineRunner interface {
	RunTask(ctx context.Context, task BenchTask, cfg RunConfig) (*BenchResult, error)
}

// SubprocessRunner executes pipelines by invoking the wave binary.
type SubprocessRunner struct{}

// RunTask runs a single benchmark task by invoking `wave run` as a subprocess.
func (s *SubprocessRunner) RunTask(ctx context.Context, task BenchTask, cfg RunConfig) (*BenchResult, error) {
	waveBin := cfg.WaveBinary
	if waveBin == "" {
		waveBin = "wave"
	}

	workDir := cfg.WorkDir
	if workDir == "" {
		workDir = ".wave/bench"
	}

	taskDir := filepath.Join(workDir, task.ID)
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		return nil, fmt.Errorf("create task workspace %s: %w", taskDir, err)
	}

	runID := generateRunID()
	result := &BenchResult{
		TaskID:    task.ID,
		RunID:     runID,
		Pipeline:  cfg.Pipeline,
		StartedAt: time.Now(),
	}

	start := time.Now()

	// Build wave run command
	args := []string{"run", cfg.Pipeline, "--quiet", "--", task.Problem}
	cmd := exec.CommandContext(ctx, waveBin, args...)
	cmd.Dir = taskDir

	output, err := cmd.CombinedOutput()
	result.DurationMs = time.Since(start).Milliseconds()

	if err != nil {
		result.Status = StatusError
		result.Error = fmt.Sprintf("pipeline execution failed: %v\n%s", err, string(output))
		return result, nil
	}

	// If a test command is specified, run it to verify the result
	if task.TestCommand != "" {
		testResult := s.runTestCommand(ctx, taskDir, task.TestCommand)
		if testResult != nil {
			result.Status = StatusFail
			result.Error = fmt.Sprintf("test verification failed: %s", string(testResult))
		} else {
			result.Status = StatusPass
		}
	} else {
		// No test command — mark as pass (pipeline completed without error)
		result.Status = StatusPass
	}

	// Try to capture the generated patch
	result.PatchDiff = captureDiff(taskDir)

	return result, nil
}

// runTestCommand executes a shell command in the task directory.
// Returns nil on success, or the combined output on failure.
func (s *SubprocessRunner) runTestCommand(ctx context.Context, dir, command string) []byte {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return output
	}
	return nil
}

// captureDiff attempts to capture a git diff from the task workspace.
func captureDiff(dir string) string {
	cmd := exec.Command("git", "diff")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(output)
}

// RunBenchmark executes a benchmark suite: loads tasks, runs them sequentially
// through the specified pipeline, and returns an aggregated report.
func RunBenchmark(ctx context.Context, tasks []BenchTask, cfg RunConfig, runner PipelineRunner) (*BenchReport, error) {
	if cfg.Pipeline == "" {
		return nil, fmt.Errorf("pipeline name is required")
	}

	limit := len(tasks)
	if cfg.Limit > 0 && cfg.Limit < limit {
		limit = cfg.Limit
	}
	tasks = tasks[:limit]

	report := &BenchReport{
		Dataset:   cfg.DatasetPath,
		Pipeline:  cfg.Pipeline,
		StartedAt: time.Now(),
		Results:   make([]BenchResult, 0, len(tasks)),
	}

	for i, task := range tasks {
		select {
		case <-ctx.Done():
			report.CompletedAt = time.Now()
			report.DurationMs = time.Since(report.StartedAt).Milliseconds()
			report.Tally()
			return report, ctx.Err()
		default:
		}

		taskCtx := ctx
		var cancel context.CancelFunc
		if cfg.TaskTimeout > 0 {
			taskCtx, cancel = context.WithTimeout(ctx, cfg.TaskTimeout)
		}

		fmt.Fprintf(os.Stderr, "[%d/%d] Running task %s...\n", i+1, len(tasks), task.ID)
		result, err := runner.RunTask(taskCtx, task, cfg)
		if cancel != nil {
			cancel()
		}

		if err != nil {
			// Runner-level error (not task failure)
			report.Results = append(report.Results, BenchResult{
				TaskID:   task.ID,
				Pipeline: cfg.Pipeline,
				Status:   StatusError,
				Error:    err.Error(),
			})
		} else {
			report.Results = append(report.Results, *result)
		}
	}

	report.CompletedAt = time.Now()
	report.DurationMs = time.Since(report.StartedAt).Milliseconds()
	report.Tally()
	return report, nil
}

func generateRunID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return "bench-" + hex.EncodeToString(b)
}
