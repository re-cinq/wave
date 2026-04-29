package runner

import (
	"context"
	"log"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/audit"
	"github.com/recinq/wave/internal/config"
	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/pipeline"
	"github.com/recinq/wave/internal/skill"
	"github.com/recinq/wave/internal/state"
	"github.com/recinq/wave/internal/workspace"
)

// InProcessConfig collects all dependencies required to launch a pipeline run
// in-process. The webui server populates this from its own state store, SSE
// broker, workspace manager, and gate registry; the cmd layer is free to use
// the same primitive but currently keeps its richer foreground path.
type InProcessConfig struct {
	// RunID is the canonical run identifier (already created in the state DB).
	RunID string
	// PipelineName + Input are recorded for diagnostic logs; the actual
	// Pipeline definition is supplied separately via the Pipeline field
	// because callers (webui) construct it from on-disk YAML.
	PipelineName string
	Input        string
	// Pipeline is the loaded pipeline definition. May be nil for legacy
	// callers that only stash a placeholder; the executor accepts an empty
	// struct in that case.
	Pipeline *pipeline.Pipeline
	// Manifest is the run's manifest (may be nil — an empty value is used).
	Manifest *manifest.Manifest

	// Store is the read-write state store the executor records progress to.
	Store state.StateStore
	// Emitter receives all execution events. Callers wrap it with their own
	// logging/persistence layer (see webui.loggingEmitter, cmd.dbLoggingEmitter).
	Emitter event.EventEmitter
	// WorkspaceManager is optional — when nil the executor falls back to its
	// own default workspace plumbing.
	WorkspaceManager workspace.WorkspaceManager
	// GateHandler optionally intercepts approval gates. The webui passes its
	// WebUIGateHandler so HTTP clients can resolve gates over the API.
	GateHandler pipeline.GateHandler

	// FromStep, when non-empty, calls ResumeWithValidation instead of Execute.
	FromStep string

	// Options carries the CLI-parity flags (model/adapter/timeout/filters etc.).
	Options config.RuntimeConfig

	// OnComplete is invoked from the launched goroutine after the run finishes
	// (success or failure). It runs after the run status update so callers can
	// rely on the DB row reflecting the final state. May be nil.
	OnComplete func(runID string, execErr error)

	// Skills configures the on-disk skill store. When zero-valued the runner
	// falls back to the default ("skills" + ".agents/skills") layout.
	Skills SkillStoreConfig
}

// SkillStoreConfig overrides the directory layout used to discover skills.
// Either field may be left empty to disable a layer.
type SkillStoreConfig struct {
	// PrimaryRoot is the higher-precedence skill directory (default "skills").
	PrimaryRoot string
	// FallbackRoot is the lower-precedence directory (default ".agents/skills").
	FallbackRoot string
}

func (s SkillStoreConfig) resolved() (skill.SkillSource, skill.SkillSource) {
	primary := s.PrimaryRoot
	if primary == "" {
		primary = "skills"
	}
	fallback := s.FallbackRoot
	if fallback == "" {
		fallback = ".agents/skills"
	}
	return skill.SkillSource{Root: primary, Precedence: 2},
		skill.SkillSource{Root: fallback, Precedence: 1}
}

// LaunchInProcess starts a pipeline run inside the calling process. It builds
// a DefaultPipelineExecutor with all wiring, then spawns a goroutine that
// drives Execute (or ResumeWithValidation, when cfg.FromStep is set) and
// updates run-status rows on completion.
//
// The returned cancel function aborts the run. Callers typically remember it
// in an activeRuns map keyed by run ID so HTTP cancel requests can fire it.
//
// LaunchInProcess returns immediately — the goroutine owns the executor's
// lifetime. If cfg.OnComplete is non-nil it is invoked after the final
// status row is written.
func LaunchInProcess(cfg InProcessConfig) context.CancelFunc {
	// Resolve manifest + env + runtime into a single effective view. The
	// merge layer owns the "runtime > manifest > fallback" precedence for
	// adapter and timeout, so the option-list assembly below stops doing
	// it inline.
	eff := config.Resolve(manifestDefaultsFromManifest(cfg.Manifest), config.FromEnv(), cfg.Options)

	runner := adapter.ResolveAdapter(eff.Adapter)

	traceLogger, traceErr := audit.NewTraceLogger()
	if traceErr != nil {
		log.Printf("Warning: failed to create trace logger: %v", traceErr)
	}

	execOpts := []pipeline.ExecutorOption{
		pipeline.WithRunID(cfg.RunID),
		pipeline.WithStateStore(cfg.Store),
		pipeline.WithEmitter(cfg.Emitter),
		pipeline.WithDebug(true),
	}
	if cfg.WorkspaceManager != nil {
		execOpts = append(execOpts, pipeline.WithWorkspaceManager(cfg.WorkspaceManager))
	}
	if traceLogger != nil {
		execOpts = append(execOpts, pipeline.WithAuditLogger(traceLogger))
	}
	if cfg.GateHandler != nil {
		execOpts = append(execOpts, pipeline.WithGateHandler(cfg.GateHandler))
	}
	if eff.Model != "" {
		execOpts = append(execOpts, pipeline.WithModelOverride(eff.Model))
	}
	// WithAdapterOverride forces the named adapter onto every step,
	// overriding step.Adapter and persona.Adapter. Only apply it when the
	// runtime supplied an explicit --adapter — the manifest-first / "claude"
	// fallbacks captured by eff.Adapter are for runner resolution, not
	// per-step dispatch.
	if cfg.Options.Adapter != "" {
		execOpts = append(execOpts, pipeline.WithAdapterOverride(cfg.Options.Adapter))
	}
	if cfg.Options.Timeout > 0 {
		// Only honour the merged timeout when it actually came from a
		// runtime override — manifest defaults are applied per-step inside
		// the executor and overriding here would short-circuit them.
		execOpts = append(execOpts, pipeline.WithStepTimeout(eff.StepTimeout))
	}
	if cfg.Options.Steps != "" || cfg.Options.Exclude != "" {
		execOpts = append(execOpts, pipeline.WithStepFilter(pipeline.ParseStepFilter(cfg.Options.Steps, cfg.Options.Exclude)))
	}

	primary, fallback := cfg.Skills.resolved()
	execOpts = append(execOpts, pipeline.WithSkillStore(skill.NewDirectoryStore(primary, fallback)))

	executor := pipeline.NewDefaultPipelineExecutor(runner, execOpts...)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		defer func() {
			if traceLogger != nil {
				traceLogger.Close()
			}
			cancel()
		}()

		if err := cfg.Store.UpdateRunStatus(cfg.RunID, "running", "", 0); err != nil {
			log.Printf("Warning: failed to update run %s to running: %v", cfg.RunID, err)
		}

		m := cfg.Manifest
		if m == nil {
			m = &manifest.Manifest{}
		}
		p := cfg.Pipeline
		if p == nil {
			p = &pipeline.Pipeline{}
		}

		var execErr error
		if cfg.FromStep != "" {
			execErr = executor.ResumeWithValidation(ctx, p, m, cfg.Input, cfg.FromStep, false, cfg.RunID)
		} else {
			execErr = executor.Execute(ctx, p, m, cfg.Input)
		}

		tokens := executor.GetTotalTokens()
		if execErr != nil {
			log.Printf("Pipeline %s (%s) failed: %v", cfg.PipelineName, cfg.RunID, execErr)
			if err := cfg.Store.UpdateRunStatus(cfg.RunID, "failed", execErr.Error(), tokens); err != nil {
				log.Printf("Warning: failed to update run %s to failed: %v", cfg.RunID, err)
			}
		} else {
			if err := cfg.Store.UpdateRunStatus(cfg.RunID, "completed", "", tokens); err != nil {
				log.Printf("Warning: failed to update run %s to completed: %v", cfg.RunID, err)
			}
		}

		if cfg.OnComplete != nil {
			cfg.OnComplete(cfg.RunID, execErr)
		}
	}()

	return cancel
}

// manifestDefaultsFromManifest adapts a *manifest.Manifest into the
// config.ManifestDefaults interface used by config.Resolve. The interface
// indirection exists because internal/config cannot import internal/manifest
// (manifest transitively imports config via hooks/scope, which would cycle).
//
// A nil manifest yields a ManifestDefaults that returns zero values; Resolve
// treats that the same as a zero-valued manifest, falling through to the
// hard-coded adapter and timeout fallbacks.
func manifestDefaultsFromManifest(m *manifest.Manifest) config.ManifestDefaults {
	if m == nil {
		return nil
	}
	return config.ManifestDefaultsFunc{
		FirstAdapterFn: func() string {
			// Map iteration order is non-deterministic; the previous
			// implementation relied on this. Preserve semantics by
			// returning the first key Go yields — callers that pin a
			// specific adapter must use --adapter explicitly.
			for adapterName := range m.Adapters {
				return adapterName
			}
			return ""
		},
		DefaultTimeoutFn: m.Runtime.GetDefaultTimeout,
	}
}
