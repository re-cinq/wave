package bench

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// RunMode controls whether tasks run through Wave pipelines or standalone Claude.
const (
	ModeWave   = "wave"
	ModeClaude = "claude"
)

// RunConfig holds the configuration for a benchmark run.
type RunConfig struct {
	// Pipeline is the name of the Wave pipeline to execute per task.
	Pipeline string
	// Mode selects execution mode: "wave" (default) or "claude".
	Mode string
	// RunLabel is a human-readable label for this run (e.g. "baseline-v1").
	RunLabel string
	// Limit caps the number of tasks to run (0 = no limit).
	Limit int
	// DatasetPath is the path to the JSONL dataset file.
	DatasetPath string
	// WaveBinary is the path to the wave binary. Defaults to "wave".
	WaveBinary string
	// ClaudeBinary is the path to the claude binary. Defaults to "claude".
	ClaudeBinary string
	// WorkDir is the root directory for benchmark workspaces.
	// Defaults to ".wave/bench".
	WorkDir string
	// Timeout per task. Zero means no timeout.
	TaskTimeout time.Duration
	// KeepWorkspaces preserves task worktrees after completion.
	KeepWorkspaces bool
	// Concurrency is the number of tasks to run in parallel. Defaults to 1.
	Concurrency int
	// Offset skips the first N tasks in the dataset.
	Offset int
}

// PipelineRunner is the interface for executing a single pipeline against a task.
// This enables testing without subprocess execution.
type PipelineRunner interface {
	RunTask(ctx context.Context, task BenchTask, cfg RunConfig) (*BenchResult, error)
}

// SubprocessRunner executes pipelines by invoking the wave or claude binary.
type SubprocessRunner struct {
	// repoCache is shared across concurrent tasks to avoid clone races.
	repoCache *RepoCache
}

// NewSubprocessRunner creates a runner with a shared repo cache.
func NewSubprocessRunner(cacheDir string) *SubprocessRunner {
	return &SubprocessRunner{
		repoCache: &RepoCache{CacheDir: cacheDir},
	}
}

// RunTask runs a single benchmark task. It clones the repo, creates a worktree
// at the base commit, and then either runs a Wave pipeline or invokes Claude
// directly depending on the configured mode.
func (s *SubprocessRunner) RunTask(ctx context.Context, task BenchTask, cfg RunConfig) (*BenchResult, error) {
	workDir := cfg.WorkDir
	if workDir == "" {
		workDir = ".wave/bench"
	}

	taskDir := filepath.Join(workDir, "workspaces", task.ID)
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

	// Set up repository worktree if repo info is available.
	worktreePath := taskDir
	if task.Repo != "" && task.BaseCommit != "" {
		rc := s.repoCache

		if _, err := rc.EnsureCloned(ctx, task.Repo); err != nil {
			result.DurationMs = time.Since(start).Milliseconds()
			result.Status = StatusError
			result.Error = fmt.Sprintf("clone repo %s: %v", task.Repo, err)
			return result, nil
		}

		wtPath := filepath.Join(taskDir, "src")
		if err := rc.PrepareWorktree(ctx, task.Repo, task.BaseCommit, wtPath, task.TestPatch); err != nil {
			result.DurationMs = time.Since(start).Milliseconds()
			result.Status = StatusError
			result.Error = fmt.Sprintf("prepare worktree: %v", err)
			return result, nil
		}
		worktreePath = wtPath

		if !cfg.KeepWorkspaces {
			defer func() {
				_ = rc.RemoveWorktree(ctx, task.Repo, wtPath)
			}()
		}
	}

	// Execute based on mode.
	mode := cfg.Mode
	if mode == "" {
		mode = ModeWave
	}

	var runErr error
	switch mode {
	case ModeClaude:
		runErr = s.runClaudeDirect(ctx, task, cfg, worktreePath)
	default:
		runErr = s.runWavePipeline(ctx, task, cfg, worktreePath)
	}

	result.DurationMs = time.Since(start).Milliseconds()

	if runErr != nil {
		result.Status = StatusError
		result.Error = fmt.Sprintf("execution failed: %v", runErr)
		return result, nil
	}

	// If a test command is specified, run it to verify the result.
	if task.TestCommand != "" {
		testResult := s.runTestCommand(ctx, worktreePath, task.TestCommand)
		if testResult != nil {
			result.Status = StatusFail
			result.Error = fmt.Sprintf("test verification failed: %s", string(testResult))
		} else {
			result.Status = StatusPass
		}
	} else {
		result.Status = StatusPass
	}

	// Try to capture the generated patch.
	result.PatchDiff = captureDiff(worktreePath)

	return result, nil
}

// runWavePipeline invokes `wave run <pipeline> --quiet -- <problem>`.
// It first ensures the worktree has a wave project via `wave init`.
func (s *SubprocessRunner) runWavePipeline(ctx context.Context, task BenchTask, cfg RunConfig, dir string) error {
	waveBin := cfg.WaveBinary
	if waveBin == "" {
		waveBin = "wave"
	}

	// Initialize a wave project in the worktree so the manifest and
	// pipeline definitions are available.
	initCmd := exec.CommandContext(ctx, waveBin, "init", "--yes", "--force", "--all")
	initCmd.Dir = dir
	if out, err := initCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("wave init: %w\n%s", err, string(out))
	}

	args := []string{"run", cfg.Pipeline, "--quiet", "--force", "--", task.Problem}
	cmd := exec.CommandContext(ctx, waveBin, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v\n%s", err, string(output))
	}
	return nil
}

// runClaudeDirect invokes `claude -p` with the problem statement directly.
func (s *SubprocessRunner) runClaudeDirect(ctx context.Context, task BenchTask, cfg RunConfig, dir string) error {
	claudeBin := cfg.ClaudeBinary
	if claudeBin == "" {
		claudeBin = "claude"
	}

	args := []string{
		"-p",
		"--dangerously-skip-permissions",
		"--allowedTools", "Bash,Read,Write,Edit,Glob,Grep",
		task.Problem,
	}
	cmd := exec.CommandContext(ctx, claudeBin, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v\n%s", err, string(output))
	}
	return nil
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
	if cfg.Pipeline == "" && cfg.Mode != ModeClaude {
		return nil, fmt.Errorf("pipeline name is required (unless mode is %q)", ModeClaude)
	}

	if cfg.Offset > 0 && cfg.Offset < len(tasks) {
		tasks = tasks[cfg.Offset:]
	}
	limit := len(tasks)
	if cfg.Limit > 0 && cfg.Limit < limit {
		limit = cfg.Limit
	}
	tasks = tasks[:limit]

	mode := cfg.Mode
	if mode == "" {
		mode = ModeWave
	}

	report := &BenchReport{
		Dataset:   cfg.DatasetPath,
		Pipeline:  cfg.Pipeline,
		Mode:      mode,
		RunLabel:  cfg.RunLabel,
		StartedAt: time.Now(),
		Results:   make([]BenchResult, len(tasks)),
	}

	concurrency := cfg.Concurrency
	if concurrency < 1 {
		concurrency = 1
	}

	// Use a semaphore to limit concurrency.
	sem := make(chan struct{}, concurrency)
	var mu sync.Mutex
	var completed int
	var wg sync.WaitGroup

	for i, task := range tasks {
		if ctx.Err() != nil {
			break
		}

		wg.Add(1)
		go func(idx int, t BenchTask) {
			defer wg.Done()

			sem <- struct{}{}        // acquire
			defer func() { <-sem }() // release

			select {
			case <-ctx.Done():
				report.Results[idx] = BenchResult{
					TaskID:   t.ID,
					Pipeline: cfg.Pipeline,
					Status:   StatusError,
					Error:    "cancelled",
				}
				return
			default:
			}

			mu.Lock()
			completed++
			n := completed
			mu.Unlock()
			fmt.Fprintf(os.Stderr, "[%d/%d] Running task %s (%s)...\n", n, len(tasks), t.ID, mode)

			taskCtx := ctx
			var cancel context.CancelFunc
			if cfg.TaskTimeout > 0 {
				taskCtx, cancel = context.WithTimeout(ctx, cfg.TaskTimeout)
			}

			result, err := runner.RunTask(taskCtx, t, cfg)
			if cancel != nil {
				cancel()
			}

			if err != nil {
				report.Results[idx] = BenchResult{
					TaskID:   t.ID,
					Pipeline: cfg.Pipeline,
					Status:   StatusError,
					Error:    err.Error(),
				}
			} else {
				report.Results[idx] = *result
			}
		}(i, task)
	}

	wg.Wait()

	report.CompletedAt = time.Now()
	report.DurationMs = time.Since(report.StartedAt).Milliseconds()
	report.Tally()
	return report, ctx.Err()
}

func generateRunID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return "bench-" + hex.EncodeToString(b)
}
