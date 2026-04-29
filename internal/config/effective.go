package config

// This file holds the EffectiveConfig merge layer — a single resolved view
// over manifest defaults, the process env snapshot, and the per-run runtime
// overrides supplied by `wave run` flags or the webui launch struct.
//
// Precedence (lowest -> highest):
//
//   1. Manifest defaults  — values pulled from wave.yaml's runtime block
//   2. Process env        — config.Env snapshot (NO_COLOR etc.)
//   3. Runtime overrides  — config.RuntimeConfig (CLI flags / launch struct)
//
// Empty / zero-value fields in a higher-precedence source do NOT clobber
// lower-precedence non-empty values: an unset CLI flag defers to env, which
// in turn defers to manifest. This rule exists so callers can hand a single
// resolved view down to executors and adapters without each layer re-applying
// its own "if non-empty, override" boilerplate.
//
// Scope (deliberately narrow): only the fields where merge logic currently
// lives in multiple call-sites (or where bundling them on Effective lets
// callers stop threading RuntimeConfig down for the sake of one bool):
//
//   - Adapter           runtime > manifest first declared adapter > "claude"
//   - Model             runtime > manifest tier table fallback handled
//                       per-step inside the executor; Effective only carries
//                       the run-wide override
//   - StepTimeout       runtime int (minutes) > manifest GetDefaultTimeout()
//                       > timeouts.StepDefault
//   - AutoApprove       runtime-only bool, bundled for caller ergonomics
//   - NoRetro           runtime-only bool, bundled for caller ergonomics
//   - ForceModel        runtime-only bool, bundled for caller ergonomics
//   - NoColor           env (NO_COLOR set) > runtime Output.NoColor flag
//
// Anything else (workspace_root, output format, verbosity) is read from
// exactly one source today and does not benefit from a merge layer; those
// callers are intentionally left alone per PR-2 scope.

import (
	"time"

	"github.com/recinq/wave/internal/timeouts"
)

// ManifestDefaults is the slice of *manifest.Manifest that Resolve consults.
// Declaring it as an interface here keeps the config package free of any
// dependency on internal/manifest (which transitively imports internal/config
// via hooks/scope, so a direct import would form a cycle).
//
// *manifest.Manifest satisfies this interface directly through its existing
// methods and field access — no adapter struct is required at the call site;
// callers wrap raw access in a small helper (see ManifestDefaultsFunc).
type ManifestDefaults interface {
	// FirstAdapter returns the first adapter name declared on the manifest,
	// or empty when the manifest declares none. Used as the fallback adapter
	// when no runtime override is supplied.
	FirstAdapter() string

	// DefaultTimeout returns the manifest-resolved per-step timeout. The
	// underlying manifest type already collapses the legacy
	// DefaultTimeoutMin field and the structured Timeouts.StepDefaultMin
	// field into a single duration.
	DefaultTimeout() time.Duration
}

// Effective is the resolved per-run configuration view. It collapses the
// three input layers (manifest, env, runtime) into one struct so consumers
// (executor wiring, webui launcher, cmd run) stop reaching back into all
// three sources separately.
//
// Zero-valued fields on Effective mean "no opinion" — the executor's own
// per-step defaults still apply. Notably, an empty Adapter still gets the
// hard-coded "claude" fallback applied by Resolve, because that fallback
// already lives in two places today and consolidating it is part of the
// motivation for this layer.
type Effective struct {
	// Adapter is the resolved adapter name. Never empty after Resolve:
	// runtime override wins, else first manifest adapter, else "claude".
	Adapter string

	// Model is the run-wide model override. Empty means "use the per-step
	// or per-persona model"; the executor applies its own tier-table logic
	// from the manifest in that case.
	Model string

	// StepTimeout is the per-step timeout. Always non-zero after Resolve:
	// runtime > manifest > timeouts.StepDefault.
	StepTimeout time.Duration

	// AutoApprove auto-resolves approval gates with their default choice.
	AutoApprove bool

	// NoRetro skips retrospective generation for the run.
	NoRetro bool

	// ForceModel forces the resolved Model onto every step regardless of
	// per-step or per-persona model pinning.
	ForceModel bool

	// NoColor disables ANSI color in CLI/TUI output. True when either the
	// runtime Output.NoColor flag is set or the NO_COLOR env var is
	// non-empty (per the no-color.org convention).
	NoColor bool
}

// Resolve merges manifest defaults, the env snapshot, and runtime overrides
// into a single Effective view. Higher-precedence non-zero values win;
// zero-valued higher-precedence fields defer to the lower layers.
//
// A nil ManifestDefaults is tolerated: it represents the standalone webui
// bootstrap path where the server is launched without a manifest on disk.
// In that case manifest-layer fields fall through to the hard-coded
// fallbacks ("claude" adapter, timeouts.StepDefault).
func Resolve(m ManifestDefaults, env Env, rt RuntimeConfig) Effective {
	out := Effective{
		AutoApprove: rt.AutoApprove,
		NoRetro:     rt.NoRetro,
		ForceModel:  rt.ForceModel,
		Model:       rt.Model,
	}

	// Adapter precedence: runtime > manifest first adapter > "claude".
	switch {
	case rt.Adapter != "":
		out.Adapter = rt.Adapter
	case m != nil && m.FirstAdapter() != "":
		out.Adapter = m.FirstAdapter()
	default:
		out.Adapter = "claude"
	}

	// StepTimeout precedence: runtime (minutes, > 0) > manifest default >
	// hard-coded fallback. Zero on runtime is "unset", not "no timeout".
	switch {
	case rt.Timeout > 0:
		out.StepTimeout = time.Duration(rt.Timeout) * time.Minute
	case m != nil:
		out.StepTimeout = m.DefaultTimeout()
	default:
		out.StepTimeout = timeouts.StepDefault
	}

	// NoColor precedence: env NO_COLOR (any non-empty value) OR runtime
	// flag. The env var is the no-color.org convention; the runtime flag
	// covers cmd's --no-color path.
	out.NoColor = env.NoColor != "" || rt.Output.NoColor

	return out
}

// ManifestDefaultsFunc adapts plain functions into a ManifestDefaults
// implementation. The cmd and webui callers use this to bridge their
// loaded *manifest.Manifest into Resolve without a separate wrapper type.
type ManifestDefaultsFunc struct {
	FirstAdapterFn   func() string
	DefaultTimeoutFn func() time.Duration
}

// FirstAdapter implements ManifestDefaults.
func (f ManifestDefaultsFunc) FirstAdapter() string {
	if f.FirstAdapterFn == nil {
		return ""
	}
	return f.FirstAdapterFn()
}

// DefaultTimeout implements ManifestDefaults.
func (f ManifestDefaultsFunc) DefaultTimeout() time.Duration {
	if f.DefaultTimeoutFn == nil {
		return timeouts.StepDefault
	}
	return f.DefaultTimeoutFn()
}
