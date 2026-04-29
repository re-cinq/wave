package runner

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/recinq/wave/internal/config"
	"github.com/recinq/wave/internal/state"
)

// detachFlagSpec mirrors a single Options field into the argv of a detached
// `wave run` subprocess. emit appends "--flag" or "--flag value" tokens to
// args when the field warrants forwarding (skipping zero/default values).
// Together DetachFlagSpecs forms the single source of truth for the argv
// rebuilder. TestDetachedArgsExhaustive guards against new Options fields
// being silently dropped — every field must be registered here or in
// DetachFlagSkippedFields.
type detachFlagSpec struct {
	field string // Options struct field name (matched by exhaustiveness test)
	flag  string // CLI flag name without leading dashes
	emit  func(opts Options, args []string) []string
}

// FlagSpecField returns the Options field name registered with a spec.
// Exposed so out-of-package tests can introspect the spec list.
func (s detachFlagSpec) FlagSpecField() string { return s.field }

// FlagSpecFlag returns the CLI flag (without leading dashes) for a spec.
func (s detachFlagSpec) FlagSpecFlag() string { return s.flag }

// DetachFlagSkippedFields lists Options fields that intentionally do NOT
// flow through to the detached subprocess. Update this list (with a reason)
// when adding a new field that should not be mirrored.
var DetachFlagSkippedFields = map[string]string{
	"Pipeline": "always emitted explicitly as --pipeline before spec processing",
	"RunID":    "always emitted explicitly as --run with the freshly created runID",
	"Detach":   "subprocess must not recurse into detached mode",
	"DryRun":   "Detach is unreachable when --dry-run is set (handled in runRun)",
	"Output":   "OutputConfig is a struct — Verbose handled outside the spec list",
}

// boolFlag emits "--<flag>" when get(o) is true.
func boolFlag(field, flag string, get func(Options) bool) detachFlagSpec {
	return detachFlagSpec{field: field, flag: flag, emit: func(o Options, a []string) []string {
		if get(o) {
			return append(a, "--"+flag)
		}
		return a
	}}
}

// strFlag emits "--<flag> <value>" when get(o) is non-empty and not equal to skip.
func strFlag(field, flag, skip string, get func(Options) string) detachFlagSpec {
	return detachFlagSpec{field: field, flag: flag, emit: func(o Options, a []string) []string {
		v := get(o)
		if v != "" && v != skip {
			return append(a, "--"+flag, v)
		}
		return a
	}}
}

// intFlag emits "--<flag> <value>" when get(o) > 0.
func intFlag(field, flag string, get func(Options) int) detachFlagSpec {
	return detachFlagSpec{field: field, flag: flag, emit: func(o Options, a []string) []string {
		if v := get(o); v > 0 {
			return append(a, "--"+flag, fmt.Sprintf("%d", v))
		}
		return a
	}}
}

// DetachFlagSpecs is the single source of truth for argv mirroring.
// Adding a new pass-through flag means adding ONE entry here.
var DetachFlagSpecs = []detachFlagSpec{
	strFlag("Input", "input", "", func(o Options) string { return o.Input }),
	strFlag("FromStep", "from-step", "", func(o Options) string { return o.FromStep }),
	boolFlag("Force", "force", func(o Options) bool { return o.Force }),
	intFlag("Timeout", "timeout", func(o Options) int { return o.Timeout }),
	strFlag("Manifest", "manifest", "wave.yaml", func(o Options) string { return o.Manifest }),
	boolFlag("Mock", "mock", func(o Options) bool { return o.Mock }),
	strFlag("Model", "model", "", func(o Options) string { return o.Model }),
	strFlag("Adapter", "adapter", "", func(o Options) string { return o.Adapter }),
	boolFlag("PreserveWorkspace", "preserve-workspace", func(o Options) bool { return o.PreserveWorkspace }),
	strFlag("Steps", "steps", "", func(o Options) string { return o.Steps }),
	strFlag("Exclude", "exclude", "", func(o Options) string { return o.Exclude }),
	boolFlag("Continuous", "continuous", func(o Options) bool { return o.Continuous }),
	strFlag("Source", "source", "", func(o Options) string { return o.Source }),
	intFlag("MaxIterations", "max-iterations", func(o Options) int { return o.MaxIterations }),
	strFlag("Delay", "delay", "0s", func(o Options) string { return o.Delay }),
	strFlag("OnFailure", "on-failure", "halt", func(o Options) string { return o.OnFailure }),
	boolFlag("AutoApprove", "auto-approve", func(o Options) bool { return o.AutoApprove }),
	boolFlag("NoRetro", "no-retro", func(o Options) bool { return o.NoRetro }),
	boolFlag("ForceModel", "force-model", func(o Options) bool { return o.ForceModel }),
}

// BuildDetachedArgs constructs argv for a detached `wave run` subprocess from
// the parent Options plus the freshly created runID. Pipeline and run id
// are always emitted; all other fields flow through DetachFlagSpecs so adding
// a new pass-through flag requires editing exactly one spec list.
func BuildDetachedArgs(opts Options, runID string) []string {
	args := []string{"run", "--pipeline", opts.Pipeline, "--run", runID}
	for _, spec := range DetachFlagSpecs {
		args = spec.emit(opts, args)
	}
	// OutputConfig is a struct — only Verbose flows through to the subprocess.
	if opts.Output.Verbose {
		args = append(args, "--verbose")
	}
	return args
}

// BuildDetachEnv constructs a minimal environment for detached subprocesses.
// It guarantees PATH includes $HOME/.local/bin (where uv, pip, cargo install
// binaries) and forwards the env vars adapters and tool managers need.
//
// extraVars is an optional list of additional environment variable names to
// pass through — used by the webui server (it forwards GH_TOKEN/GITHUB_TOKEN
// in addition to the base set).
func BuildDetachEnv(extraVars ...string) []string {
	// Read inherited PATH/HOME via internal/config so the detach contract is
	// centralised with the other env-reader sites. Semantics are identical to
	// a direct os.Getenv: empty string when the variable is unset.
	home, path := config.SubprocessHomePath()
	if home != "" {
		toolBin := filepath.Join(home, ".local", "bin")
		if !strings.Contains(path, toolBin) {
			path = toolBin + string(os.PathListSeparator) + path
		}
	}

	env := []string{
		"HOME=" + home,
		"PATH=" + path,
	}
	// Pass through common env vars needed by adapters and tool managers.
	keys := []string{
		"ANTHROPIC_API_KEY", "CLAUDE_CODE_USE_BEDROCK", "AWS_PROFILE", "AWS_REGION",
		"TERM", "USER", "SHELL",
		// XDG dirs used by uv, pip, and other tool managers for locating data/config
		"XDG_DATA_HOME", "XDG_CONFIG_HOME", "XDG_CACHE_HOME",
	}
	keys = append(keys, extraVars...)
	for _, key := range keys {
		if val, ok := os.LookupEnv(key); ok {
			env = append(env, key+"="+val)
		}
	}
	return env
}

// DetachConfig parameterises a Detach call. workDir is the current working
// directory for the subprocess (typically the repo root). LogsDir is where
// the per-run log file is created. extraEnv lists additional env variable
// names to forward beyond the BuildDetachEnv defaults.
type DetachConfig struct {
	WorkDir  string
	LogsDir  string
	ExtraEnv []string
}

// PIDRecorder is the optional surface SpawnDetached uses to persist a
// freshly-spawned subprocess PID for a run. The TUI and webui both supply a
// state-store-backed implementation; foreground CLI callers can pass nil.
type PIDRecorder interface {
	UpdateRunPID(runID string, pid int) error
}

// SpawnDetached starts a fully-detached `wave <subcommand>` subprocess from
// the supplied argv. It is the shared spawn primitive used by Detach (for
// `wave run`) and by callers that drive other subcommands such as the TUI's
// `wave compose` orchestration path.
//
// Behaviour mirrored from Detach:
//   - argv[0] is the wave binary resolved via os.Executable() (with argv[0]
//     fallback for tests and unusual environments).
//   - Setsid is set so the child becomes its own session leader and survives
//     the parent process exit.
//   - cfg.WorkDir, cfg.LogsDir, and cfg.ExtraEnv are honoured exactly the
//     same way as in Detach.
//   - cmd.Stdout / cmd.Stderr are redirected to <logsDir>/<runID>.log.
//   - On success the child PID is recorded via recorder (when non-nil) and
//     cmd.Process.Release is called so the OS fully reparents the child.
//
// Callers are responsible for any pre-spawn coordination (run-id reservation,
// status transitions) — SpawnDetached only owns the fork/exec dance.
func SpawnDetached(args []string, runID string, recorder PIDRecorder, cfg DetachConfig) error {
	waveBin, exeErr := os.Executable()
	if exeErr != nil {
		// Fall back to argv[0] for compatibility with tests / unusual envs.
		waveBin = os.Args[0]
	}

	cmd := exec.Command(waveBin, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Env = BuildDetachEnv(cfg.ExtraEnv...)
	if cfg.WorkDir != "" {
		cmd.Dir = cfg.WorkDir
	}

	logsDir := cfg.LogsDir
	if logsDir == "" {
		logsDir = filepath.Join(".agents", "logs")
	}
	if mkErr := os.MkdirAll(logsDir, 0o755); mkErr != nil {
		return fmt.Errorf("failed to create logs directory: %w", mkErr)
	}
	logPath := filepath.Join(logsDir, runID+".log")
	logFile, logErr := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if logErr != nil {
		return fmt.Errorf("failed to create log file: %w", logErr)
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if startErr := cmd.Start(); startErr != nil {
		logFile.Close()
		return fmt.Errorf("failed to start detached subprocess: %w", startErr)
	}

	// The subprocess inherited the fd via fork; close our copy.
	logFile.Close()

	if recorder != nil {
		_ = recorder.UpdateRunPID(runID, cmd.Process.Pid)
	}
	_ = cmd.Process.Release()
	return nil
}

// Detach spawns a fully-detached `wave run` subprocess that survives the
// parent process exit. The subprocess writes to the shared state DB so
// `wave status`, `wave logs`, and the webui dashboard can all observe it.
//
// When opts.RunID is empty Detach reserves a fresh run ID via the supplied
// state store, respecting maxConcurrentWorkers via CreateRunWithLimit. When
// opts.RunID is non-empty it is reused (the resume-in-place path used by
// `--from-step`, see issue #1452).
//
// detachStore is the narrow run-lifecycle surface Detach needs: pre-create
// or reuse a run row, mark it running, and stamp the spawned PID.
type detachStore interface {
	GetRun(runID string) (*state.RunRecord, error)
	CreateRunWithLimit(pipelineName string, input string, maxConcurrent int) (string, error)
	UpdateRunStatus(runID string, status string, currentStep string, tokens int) error
	UpdateRunPID(runID string, pid int) error
}

// On success the returned runID is the canonical ID for the spawned run and
// the subprocess has already been fully released (cmd.Process.Release).
func Detach(opts Options, store detachStore, maxConcurrentWorkers int, cfg DetachConfig) (runID string, err error) {
	if store == nil {
		return "", fmt.Errorf("Detach: state store is required")
	}
	if maxConcurrentWorkers <= 0 {
		maxConcurrentWorkers = 5
	}

	// Reuse a pre-existing run row when the caller supplies opts.RunID and
	// the row exists. Two callers rely on this:
	//
	//  - cmd/wave/commands runDetached with --from-step <step> --run <prior>:
	//    the auto-injector (#1452) needs the original run id preserved so
	//    artifact paths resolve from the original workspace.
	//  - internal/webui handlers: every HTTP launch path pre-creates the run
	//    row via rwStore.CreateRun before calling Detach. Without this
	//    branch we would spawn a duplicate row for every webui-initiated
	//    detach.
	//
	// Falling through to CreateRunWithLimit when the lookup fails preserves
	// the legacy "create on demand" behaviour used by foreground CLI runs.
	if opts.RunID != "" {
		if _, err := store.GetRun(opts.RunID); err == nil {
			runID = opts.RunID
		}
	}
	if runID == "" {
		notified := false
		for {
			var createErr error
			runID, createErr = store.CreateRunWithLimit(opts.Pipeline, opts.Input, maxConcurrentWorkers)
			if createErr == nil {
				break
			}
			if !errors.Is(createErr, state.ErrConcurrencyLimit) {
				return "", fmt.Errorf("failed to create run record: %w", createErr)
			}
			if !notified {
				fmt.Fprintf(os.Stderr, "  Queued: %d/%d workers busy, waiting for a slot...\n", maxConcurrentWorkers, maxConcurrentWorkers)
				notified = true
			}
			time.Sleep(5 * time.Second)
		}
	}
	// Mark as running so wave status picks it up immediately.
	_ = store.UpdateRunStatus(runID, "running", "", 0)

	// Build subprocess args: same flags minus --detach/-d, plus --run <runID>.
	args := BuildDetachedArgs(opts, runID)

	if err := SpawnDetached(args, runID, store, cfg); err != nil {
		return "", err
	}
	return runID, nil
}
