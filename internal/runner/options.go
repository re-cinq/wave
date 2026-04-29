// options.go centralises ExecutorOption assembly so both the CLI's foreground
// path (cmd/wave/commands.runOnce) and the in-process launcher
// (LaunchInProcess) build the same option list from the same inputs.
//
// Before this lived here, the two paths drifted — most notably webui's
// LaunchInProcess never called pipeline.WithRegistry, so manifest-declared
// adapter binaries (e.g. opencode-patched) were silently ignored when a run
// was launched from the dashboard. BuildExecutorOptions fixes that by always
// attaching a registry seeded from the manifest's adapter binaries.
package runner

import (
	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/audit"
	"github.com/recinq/wave/internal/config"
	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/pipeline"
	"github.com/recinq/wave/internal/relay"
	"github.com/recinq/wave/internal/retro"
	"github.com/recinq/wave/internal/skill"
	"github.com/recinq/wave/internal/state"
	"github.com/recinq/wave/internal/workspace"
)

// ExecutorBuildConfig is the union of inputs needed to construct the executor
// option list for either launch path. Fields are conditionally honoured:
// nil/zero values produce no option, non-nil/non-zero values append the
// corresponding pipeline.With* option in a deterministic order.
//
// Fields fall into three groups:
//   - Identity: RunID, Manifest — always available.
//   - Wiring: Store, Emitter, WorkspaceManager, GateHandler, AuditLogger,
//     DebugTracer, Runner — supplied by both paths but optional individually.
//   - CLI extras: RetroGenerator, RelayMonitor, SkillStore, Debug, plus the
//     fields read from Runtime (model, adapter override, timeout, step filter,
//     preserve workspace, force model, auto approve). The webui path leaves
//     these zero; the CLI populates them from RunOptions.
type ExecutorBuildConfig struct {
	RunID            string
	Manifest         *manifest.Manifest
	Store            state.StateStore
	Emitter          event.EventEmitter
	WorkspaceManager workspace.WorkspaceManager
	GateHandler      pipeline.GateHandler
	AuditLogger      audit.AuditLogger
	DebugTracer      *audit.DebugTracer

	// Runtime carries the merged CLI flag snapshot (model/adapter/timeout/
	// step filters/preserve-workspace/auto-approve/force-model). The webui
	// path supplies a zero-valued struct.
	Runtime config.RuntimeConfig

	// Runner is the resolved adapter runner used as the registry's default.
	// In mock mode the CLI sets this to a MockAdapter and also wants every
	// manifest-declared adapter rerouted through it; pass MockOverride=true
	// to enable that fan-out.
	Runner        adapter.AdapterRunner
	MockOverride  bool

	// CLI extras — webui leaves these nil.
	RetroGenerator *retro.Generator
	RelayMonitor   *relay.RelayMonitor
	SkillStore     skill.Store
	StepFilter     *pipeline.StepFilter

	// Debug toggles WithDebug(true). LaunchInProcess hardcodes this; the CLI
	// threads its --debug flag through.
	Debug bool
}

// BuildExecutorOptions returns the ExecutorOption slice both launch paths
// feed into pipeline.NewDefaultPipelineExecutor.
//
// Ordering matches the historical CLI list so behavioural diffs versus the
// pre-extraction implementation stay limited to the registry-and-binaries
// fix described in the package comment.
func BuildExecutorOptions(cfg ExecutorBuildConfig) []pipeline.ExecutorOption {
	// Single resolve so model/timeout pickup runtime > manifest > env in one
	// place. Both webui and CLI now share these merged values.
	eff := config.Resolve(manifestDefaultsFromManifest(cfg.Manifest), config.FromEnv(), cfg.Runtime)

	opts := []pipeline.ExecutorOption{
		pipeline.WithRunID(cfg.RunID),
		pipeline.WithDebug(cfg.Debug),
	}
	if cfg.Emitter != nil {
		opts = append(opts, pipeline.WithEmitter(cfg.Emitter))
	}
	if cfg.Store != nil {
		opts = append(opts, pipeline.WithStateStore(cfg.Store))
	}
	if cfg.WorkspaceManager != nil {
		opts = append(opts, pipeline.WithWorkspaceManager(cfg.WorkspaceManager))
	}
	if cfg.AuditLogger != nil {
		opts = append(opts, pipeline.WithAuditLogger(cfg.AuditLogger))
	}
	if cfg.DebugTracer != nil {
		opts = append(opts, pipeline.WithDebugTracer(cfg.DebugTracer))
	}
	if cfg.GateHandler != nil {
		opts = append(opts, pipeline.WithGateHandler(cfg.GateHandler))
	}

	if eff.Model != "" {
		opts = append(opts, pipeline.WithModelOverride(eff.Model))
	}
	if cfg.Runtime.ForceModel {
		opts = append(opts, pipeline.WithForceModel(true))
	}
	// WithAdapterOverride forces the named adapter onto every step. Only
	// apply when the runtime supplied an explicit --adapter — the manifest-
	// first / "claude" fallbacks in eff.Adapter are for runner resolution,
	// not per-step dispatch.
	if cfg.Runtime.Adapter != "" {
		opts = append(opts, pipeline.WithAdapterOverride(cfg.Runtime.Adapter))
	}
	if cfg.Runtime.Timeout > 0 {
		// Honour the merged step timeout only when a runtime override was
		// actually supplied — manifest defaults are applied per-step inside
		// the executor, and overriding here would short-circuit them.
		opts = append(opts, pipeline.WithStepTimeout(eff.StepTimeout))
	}
	if cfg.Runtime.PreserveWorkspace {
		opts = append(opts, pipeline.WithPreserveWorkspace(true))
	}
	if cfg.Runtime.AutoApprove {
		opts = append(opts, pipeline.WithAutoApprove(true))
	}

	// Step filter: prefer an explicitly-supplied filter (CLI parses + validates
	// before calling), otherwise derive one from Runtime.Steps/Exclude.
	if cfg.StepFilter != nil {
		opts = append(opts, pipeline.WithStepFilter(cfg.StepFilter))
	} else if cfg.Runtime.Steps != "" || cfg.Runtime.Exclude != "" {
		opts = append(opts, pipeline.WithStepFilter(pipeline.ParseStepFilter(cfg.Runtime.Steps, cfg.Runtime.Exclude)))
	}

	// Adapter registry: always attached, seeded from the manifest's adapter
	// binaries so manifests declaring forks (e.g. opencode-patched) resolve
	// correctly. Before this fix, LaunchInProcess silently dropped these.
	registry := adapter.NewAdapterRegistry(nil)
	if cfg.Manifest != nil {
		for name, a := range cfg.Manifest.Adapters {
			if a.Binary != "" {
				registry.SetBinary(name, a.Binary)
			}
		}
	}
	if cfg.MockOverride && cfg.Runner != nil && cfg.Manifest != nil {
		// Route every adapter declared in the manifest through the mock runner.
		// "mock" itself is always registered so pipelines that pin adapter: mock
		// resolve correctly even when the manifest does not enumerate it.
		registry.RegisterOverride("mock", cfg.Runner)
		for name := range cfg.Manifest.Adapters {
			registry.RegisterOverride(name, cfg.Runner)
		}
	}
	opts = append(opts, pipeline.WithRegistry(registry))

	if cfg.SkillStore != nil {
		opts = append(opts, pipeline.WithSkillStore(cfg.SkillStore))
	}
	if cfg.RetroGenerator != nil {
		opts = append(opts, pipeline.WithRetroGenerator(cfg.RetroGenerator))
	}
	if cfg.RelayMonitor != nil {
		opts = append(opts, pipeline.WithRelayMonitor(cfg.RelayMonitor))
	}

	return opts
}
