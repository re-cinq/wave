package config

// This file holds the per-run configuration snapshot that mirrors the
// `wave run` CLI flag set. RuntimeConfig is consumed by both the foreground
// CLI path (cmd/wave/commands) and the webui launch path (internal/webui)
// so the two surfaces produce identical runtime behaviour.
//
// Lifting the struct into internal/config keeps it next to Env (the process
// environment snapshot) so a single package owns runtime-configuration
// types. A future EffectiveConfig will overlay manifest defaults with
// RuntimeConfig and Env to produce the resolved view used by the executor.

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

// RuntimeConfig captures every CLI-parity input accepted by `wave run`. The
// cmd layer aliases this type as RunOptions for local naming hygiene; the
// webui layer constructs RuntimeConfig directly. Field order is preserved
// from the original cmd RunOptions so the reflection-based exhaustiveness
// test keeps its existing assumptions.
type RuntimeConfig struct {
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
