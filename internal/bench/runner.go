package bench

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/recinq/wave/internal/workspace"
)

// RunMode controls whether tasks run through Wave pipelines or standalone Claude.
const (
	ModeWave   = "wave"
	ModeClaude = "claude"
)

// PatchArtifact is the output artifact name a bench pipeline must declare: a
// unified diff of the solver's changes against the task's base commit.
const PatchArtifact = "patch"

// RunConfig holds the configuration for a benchmark run.
type RunConfig struct {
	// Pipeline is the name of the Wave pipeline to execute per task.
	Pipeline string
	// Mode selects execution mode: "wave" (default) or "claude".
	Mode string
	// Adapter and Model are passed to `wave run --adapter/--model`
	// (Model is forced onto every step). In claude mode Model goes to
	// `claude --model`.
	Adapter string
	Model   string
	// RunLabel is a human-readable label for this run (e.g. "baseline-v1").
	RunLabel string
	// Limit caps the number of tasks to run (0 = no limit).
	Limit int
	// DatasetPath is the path to the JSONL dataset file.
	DatasetPath string
	// WorkDir is the root directory for task workspaces and the repo cache.
	// Defaults to ".agents/bench/workspaces".
	WorkDir string
	// Timeout per task. Zero means no timeout.
	TaskTimeout time.Duration
	// KeepWorkspaces preserves task worktrees after completion.
	KeepWorkspaces bool
	// Concurrency is the number of tasks to run in parallel. Defaults to 1.
	Concurrency int
	// Offset skips the first N tasks in the dataset.
	Offset int
	// WaveBin is the wave binary used for nested runs. Defaults to the
	// running executable.
	WaveBin string
	// ClaudeBin is the claude binary for claude mode. Defaults to "claude".
	ClaudeBin string
}

// pipelineRunner is the interface for executing a single pipeline against a task.
// This enables testing without subprocess execution.
type pipelineRunner interface {
	RunTask(ctx context.Context, task BenchTask, cfg RunConfig) (*BenchResult, error)
}

// subprocessRunner executes pipelines by invoking the wave or claude binary.
type subprocessRunner struct {
	// cache is shared across concurrent tasks to avoid clone races.
	cache *repoCache
}

// NewSubprocessRunner creates a runner with a shared repo cache.
func NewSubprocessRunner(cacheDir string) *subprocessRunner {
	return &subprocessRunner{
		cache: &repoCache{CacheDir: cacheDir},
	}
}

var validTaskID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)

// RunTask runs a single benchmark task. It clones the repo, creates a worktree
// at the base commit, runs the solver (a Wave pipeline or Claude directly),
// and records the produced patch. The task's test patch is never visible to
// the solver.
func (s *subprocessRunner) RunTask(ctx context.Context, task BenchTask, cfg RunConfig) (*BenchResult, error) {
	if !validTaskID.MatchString(task.ID) {
		return nil, fmt.Errorf("invalid task id %q", task.ID)
	}
	workDir := cfg.WorkDir
	if workDir == "" {
		workDir = ".agents/bench/workspaces"
	}
	taskDir, err := filepath.Abs(filepath.Join(workDir, task.ID))
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		return nil, fmt.Errorf("create task workspace %s: %w", taskDir, err)
	}

	result := &BenchResult{
		TaskID:    task.ID,
		RunID:     generateRunID(),
		Pipeline:  cfg.Pipeline,
		StartedAt: time.Now(),
		LogPath:   filepath.Join(taskDir, "run.log"),
	}
	start := time.Now()
	fail := func(status BenchStatus, format string, args ...any) *BenchResult {
		result.DurationMs = time.Since(start).Milliseconds()
		result.Status = status
		result.Error = fmt.Sprintf(format, args...)
		return result
	}

	if task.Repo == "" || task.BaseCommit == "" {
		return fail(StatusError, "task has no repo/base_commit"), nil
	}
	if _, err := s.cache.EnsureCloned(ctx, task.Repo, task.BaseCommit); err != nil {
		return fail(StatusError, "clone repo %s: %v", task.Repo, err), nil
	}
	worktreePath := filepath.Join(taskDir, "src")
	if err := s.cache.PrepareWorktree(ctx, task.Repo, task.BaseCommit, worktreePath); err != nil {
		return fail(StatusError, "prepare worktree: %v", err), nil
	}
	if !cfg.KeepWorkspaces {
		defer func() {
			// Wave keeps step workspaces outside the project checkout, in
			// a per-project dir under the user cache dir; drop it with the
			// task worktree (RemoveWorktree prunes their git entries).
			_ = os.RemoveAll(workspace.DefaultRootFor(worktreePath))
			// Use a fresh context: the task context may already be cancelled.
			_ = s.cache.RemoveWorktree(context.Background(), task.Repo, worktreePath)
		}()
	}

	var patch string
	var runErr error
	if cfg.Mode == ModeClaude {
		patch, runErr = s.runClaudeDirect(ctx, task, cfg, worktreePath, result.LogPath)
	} else {
		var runID string
		runID, patch, runErr = s.runWavePipeline(ctx, task, cfg, worktreePath, result.LogPath)
		if runID != "" {
			result.RunID = runID
		}
	}
	result.PatchDiff = patch
	result.DurationMs = time.Since(start).Milliseconds()

	if runErr != nil {
		return fail(StatusError, "execution failed: %v", runErr), nil
	}
	if patch == "" {
		return fail(StatusError, "solver produced no patch"), nil
	}

	if task.TestCommand == "" {
		result.Status = StatusPatched
		return result, nil
	}

	// Local verification: hidden tests go in only now, after the solve.
	if task.TestPatch != "" {
		if err := applyPatch(ctx, worktreePath, task.TestPatch); err != nil {
			return fail(StatusError, "apply test patch: %v", err), nil
		}
	}
	if out := runTestCommand(ctx, worktreePath, task.TestCommand); out != nil {
		return fail(StatusFail, "test verification failed: %s", out), nil
	}
	result.Status = StatusPass
	return result, nil
}

func (cfg RunConfig) waveBin() string {
	if cfg.WaveBin != "" {
		return cfg.WaveBin
	}
	if exe, err := os.Executable(); err == nil {
		return exe
	}
	return "wave"
}

// runWavePipeline initialises a Wave project in the task worktree and runs
// `wave run <pipeline> --json -- <problem>`. It returns the run ID and the
// content of the pipeline's patch artifact.
func (s *subprocessRunner) runWavePipeline(ctx context.Context, task BenchTask, cfg RunConfig, dir, logPath string) (string, string, error) {
	logFile, err := os.Create(logPath)
	if err != nil {
		return "", "", err
	}
	defer logFile.Close()

	waveBin := cfg.waveBin()
	initArgs := []string{"init", "--yes", "--force", "--all"}
	if cfg.Adapter != "" {
		initArgs = append(initArgs, "--adapter", cfg.Adapter)
	}
	initCmd := exec.CommandContext(ctx, waveBin, initArgs...)
	initCmd.Dir = dir
	if out, err := initCmd.CombinedOutput(); err != nil {
		_, _ = logFile.Write(out)
		return "", "", fmt.Errorf("wave init: %w\n%s", err, out)
	}

	args := []string{"run", cfg.Pipeline, "--json", "--no-retro"}
	if cfg.Adapter != "" {
		args = append(args, "--adapter", cfg.Adapter)
	}
	if cfg.Model != "" {
		args = append(args, "--model", cfg.Model, "--force-model")
	}
	if cfg.TaskTimeout > 0 {
		minutes := int((cfg.TaskTimeout + time.Minute - 1) / time.Minute)
		args = append(args, "--timeout", strconv.Itoa(minutes))
	}
	args = append(args, "--", task.Problem)

	var stdout bytes.Buffer
	cmd := exec.CommandContext(ctx, waveBin, args...)
	cmd.Dir = dir
	cmd.Stdout = &teeWriter{a: &stdout, b: logFile}
	cmd.Stderr = logFile
	runErr := cmd.Run()

	runID, patchPath := parseWaveEvents(stdout.Bytes())
	if runErr != nil {
		return runID, "", fmt.Errorf("wave run %s: %w (log: %s)", cfg.Pipeline, runErr, logPath)
	}
	if patchPath == "" {
		return runID, "", fmt.Errorf("pipeline %s declared no %q output artifact", cfg.Pipeline, PatchArtifact)
	}
	data, err := os.ReadFile(patchPath)
	if err != nil {
		return runID, "", fmt.Errorf("read patch artifact: %w", err)
	}
	return runID, string(data), nil
}

// parseWaveEvents scans `wave run --json` NDJSON for the run ID and the path
// of the patch deliverable reported in the completion event.
func parseWaveEvents(out []byte) (runID, patchPath string) {
	sc := bufio.NewScanner(bytes.NewReader(out))
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for sc.Scan() {
		var evt struct {
			PipelineID string `json:"pipeline_id"`
			Outcomes   *struct {
				Deliverables []struct {
					Name string `json:"name"`
					Path string `json:"path"`
				} `json:"deliverables"`
			} `json:"outcomes"`
		}
		if json.Unmarshal(sc.Bytes(), &evt) != nil {
			continue
		}
		if runID == "" && evt.PipelineID != "" {
			runID = evt.PipelineID
		}
		if evt.Outcomes == nil {
			continue
		}
		for _, d := range evt.Outcomes.Deliverables {
			if d.Name == PatchArtifact {
				patchPath = d.Path
			}
		}
	}
	return runID, patchPath
}

// runClaudeDirect invokes `claude -p` with the problem statement directly in
// the task worktree and captures the resulting diff.
func (s *subprocessRunner) runClaudeDirect(ctx context.Context, task BenchTask, cfg RunConfig, dir, logPath string) (string, error) {
	claudeBin := cfg.ClaudeBin
	if claudeBin == "" {
		claudeBin = "claude"
	}
	args := []string{
		"-p",
		"--dangerously-skip-permissions",
		"--allowedTools", "Bash,Read,Write,Edit,Glob,Grep",
	}
	if cfg.Model != "" {
		args = append(args, "--model", cfg.Model)
	}
	args = append(args, task.Problem)
	cmd := exec.CommandContext(ctx, claudeBin, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	_ = os.WriteFile(logPath, output, 0o644)
	if err != nil {
		return "", fmt.Errorf("%v\n%s", err, string(output))
	}
	return capturePatch(ctx, dir, task.BaseCommit)
}

// runTestCommand executes a shell command in the task directory.
// Returns nil on success, or the combined output on failure.
func runTestCommand(ctx context.Context, dir, command string) []byte {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		if len(output) == 0 {
			output = []byte(err.Error())
		}
		return output
	}
	return nil
}

type teeWriter struct {
	a, b interface{ Write([]byte) (int, error) }
}

func (t *teeWriter) Write(p []byte) (int, error) {
	_, _ = t.b.Write(p)
	return t.a.Write(p)
}

// RunBenchmark executes a benchmark suite: runs the tasks through the
// configured runner with bounded concurrency and returns an aggregated report.
func RunBenchmark(ctx context.Context, tasks []BenchTask, cfg RunConfig, runner pipelineRunner) (*BenchReport, error) {
	if cfg.Mode == "" {
		cfg.Mode = ModeWave
	}
	if cfg.Pipeline == "" && cfg.Mode != ModeClaude {
		return nil, fmt.Errorf("pipeline name is required (unless mode is %q)", ModeClaude)
	}

	if cfg.Offset > 0 {
		if cfg.Offset >= len(tasks) {
			tasks = nil
		} else {
			tasks = tasks[cfg.Offset:]
		}
	}
	if cfg.Limit > 0 && cfg.Limit < len(tasks) {
		tasks = tasks[:cfg.Limit]
	}

	report := &BenchReport{
		Dataset:   cfg.DatasetPath,
		Pipeline:  cfg.Pipeline,
		Mode:      cfg.Mode,
		Adapter:   cfg.Adapter,
		Model:     cfg.Model,
		RunLabel:  cfg.RunLabel,
		StartedAt: time.Now(),
		Results:   make([]BenchResult, len(tasks)),
	}
	if cfg.Mode == ModeClaude {
		report.Adapter = "claude"
	}

	concurrency := cfg.Concurrency
	if concurrency < 1 {
		concurrency = 1
	}

	sem := make(chan struct{}, concurrency)
	var mu sync.Mutex
	var started int
	var wg sync.WaitGroup

	for i, task := range tasks {
		wg.Add(1)
		go func(idx int, t BenchTask) {
			defer wg.Done()

			cancelled := BenchResult{TaskID: t.ID, Pipeline: cfg.Pipeline, Status: StatusError, Error: "cancelled"}
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				report.Results[idx] = cancelled
				return
			}
			defer func() { <-sem }()
			if ctx.Err() != nil {
				report.Results[idx] = cancelled
				return
			}

			mu.Lock()
			started++
			n := started
			mu.Unlock()
			fmt.Fprintf(os.Stderr, "[%d/%d] Running task %s (%s)...\n", n, len(tasks), t.ID, cfg.Mode)

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
				report.Results[idx] = BenchResult{TaskID: t.ID, Pipeline: cfg.Pipeline, Status: StatusError, Error: err.Error()}
				return
			}
			report.Results[idx] = *result
			fmt.Fprintf(os.Stderr, "Task %s: %s\n", t.ID, result.Status)
		}(i, task)
	}

	wg.Wait()

	report.CompletedAt = time.Now()
	report.DurationMs = time.Since(report.StartedAt).Milliseconds()
	report.Tally()
	for _, r := range report.Results {
		if r.Status == StatusPass || r.Status == StatusFail {
			report.Evaluation = "test_cmd"
			break
		}
	}
	return report, ctx.Err()
}

func generateRunID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return "bench-" + hex.EncodeToString(b)
}
