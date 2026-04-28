// Package runner is the single source of truth for launching pipeline runs —
// either in-process (LaunchInProcess) or as a fully-detached subprocess
// (Detach). It is shared by cmd/wave/commands and internal/webui so the two
// paths produce identical runtime behaviour.
//
// The package intentionally exposes a small surface:
//
//	Options       — every CLI-parity input (mirror of `wave run` flags)
//	OutputConfig  — verbose/format selection for the run
//	Detach        — spawn a `wave run` subprocess (Setsid + Process.Release)
//	LaunchInProcess — wire up DefaultPipelineExecutor and run a goroutine
//
// The detach flag-spec table (DetachFlagSpecs / DetachFlagSkippedFields) lives
// in detach.go and is exercised by TestDetachedArgsExhaustive — adding a new
// Options field requires registering it (or explicitly skipping it) in that
// table, otherwise the test fails.
package runner

// Output format constants — the canonical set used by `wave run` and shared
// with the cmd layer via type aliases.
const (
	OutputFormatAuto  = "auto"
	OutputFormatJSON  = "json"
	OutputFormatText  = "text"
	OutputFormatQuiet = "quiet"
)

// OutputConfig holds the resolved output formatting flags for a run. It is
// the single source of truth for both the foreground CLI path and the webui
// launch path; cmd/wave/commands re-exports it as commands.OutputConfig.
type OutputConfig struct {
	Format  string
	Verbose bool
	NoColor bool
	Debug   bool
}

// Options captures every CLI-parity input accepted by `wave run`. The cmd
// layer aliases this type as RunOptions for backwards compatibility; the
// webui layer constructs Options directly. Field order is preserved from the
// original cmd RunOptions so the reflection-based exhaustiveness test keeps
// its existing assumptions.
type Options struct {
	Pipeline          string
	Input             string
	DryRun            bool
	FromStep          string
	Force             bool
	Timeout           int
	Manifest          string
	Mock              bool
	RunID             string
	Output            OutputConfig
	Model             string
	Adapter           string
	PreserveWorkspace bool
	Steps             string // Comma-separated step names to include (--steps)
	Exclude           string // Comma-separated step names to exclude (-x/--exclude)
	Continuous        bool   // --continuous flag
	Source            string // --source URI for work item discovery
	MaxIterations     int    // --max-iterations cap
	Delay             string // --delay between iterations
	OnFailure         string // --on-failure halt|skip
	Detach            bool   // --detach flag for background execution
	AutoApprove       bool   // --auto-approve flag for skipping approval gates
	NoRetro           bool   // --no-retro flag to skip retrospective generation
	ForceModel        bool   // --force-model overrides all step/persona model tiers
}
