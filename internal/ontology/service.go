// Package ontology provides bounded-context ontology services for Wave pipelines:
// staleness detection, per-step context injection, decision lineage tracking,
// audit-trail logging, and health checks. Consumers depend on the Service
// interface; NoOp is returned when the feature is disabled so call sites can
// stay unconditional.
//
// Boundary rule: only this package is permitted to read or write the
// .agents/.ontology-stale sentinel file, own the "wave-ctx-*" skill path
// convention, or emit the [ONTOLOGY_*] audit trace lines. Consumers must
// depend on the Service interface, never on the sentinel path.
//
// Layer classification: Domain (see ADR-003, ADR-009). Cross-package imports
// allowed: manifest (cross-cutting), state (infrastructure), event
// (cross-cutting), audit (cross-cutting).
package ontology

import (
	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/state"
)

// Service is the bounded-context ontology API consumed by Wave domain and
// presentation packages. All methods are safe to call with the NoOp impl
// (the one returned when the feature is disabled) — they return zero values
// and perform no side effects.
type Service interface {
	// Enabled reports whether ontology features are active for this run.
	Enabled() bool

	// CheckStaleness returns a human-readable warning if the ontology may be
	// out of date (post-merge sentinel present, or wave.yaml older than the
	// latest ontology-touching commit), or "" if fresh. Reading the sentinel
	// clears it, so the warning fires once per merge.
	CheckStaleness() string

	// BuildStepSection renders the ontology markdown section for a step.
	// Returns "" when the feature is disabled, when the manifest has no
	// contexts, or when the step references no contexts and the ontology is
	// filtered. Also emits ONTOLOGY_WARN for any undefined context names and
	// ONTOLOGY_INJECT when a non-empty section is produced.
	BuildStepSection(pipelineID, stepID string, stepContexts []string) string

	// RecordUsage records which ontology contexts a step used and the step
	// outcome for decision-lineage analysis. Only targeted (explicitly
	// declared) contexts are recorded — bulk injection is excluded.
	RecordUsage(runID, stepID string, stepContexts []string, hasContract bool, stepStatus string)

	// ValidateManifest returns validation errors for ontology.contexts in m.
	// Pure shape validation (duplicate names, empty names). Present on the
	// Service rather than inline on manifest so the feature gate can skip it.
	ValidateManifest(m *manifest.Manifest) []error

	// InstallStalenessHook writes the post-merge git hook that marks the
	// ontology stale after repo updates. Called during onboarding.
	InstallStalenessHook() error
}

// Config controls Service selection.
type Config struct {
	// Enabled turns the feature on. When false, New returns NoOp.
	// Callers typically derive this from the manifest: a manifest with at
	// least one ontology.contexts entry enables the service.
	Enabled bool
}

// EnabledFromManifest returns true if m declares at least one ontology context.
// This is the canonical feature-gate check — the service is implicitly on when
// the user supplies ontology.contexts in wave.yaml.
func EnabledFromManifest(m *manifest.Manifest) bool {
	return m != nil && m.Ontology != nil && len(m.Ontology.Contexts) > 0
}

// Deps carries the collaborators the real Service needs. All fields are
// optional — the real Service tolerates nil store/emitter/sink and degrades
// to no-op behavior on the affected path.
type Deps struct {
	// Manifest is required by the real Service for context/invariant lookup.
	Manifest *manifest.Manifest
	// Store receives lineage records via RecordOntologyUsage. Nil disables
	// persistent lineage (the warn/inject events still fire).
	Store state.OntologyStore
	// Emitter receives the [ontology_*] progress events. Nil disables them.
	Emitter event.EventEmitter
	// AuditSink receives ONTOLOGY_INJECT / ONTOLOGY_LINEAGE / ONTOLOGY_WARN
	// trace lines. Nil disables the audit trail.
	AuditSink AuditSink
}

// AuditSink is the minimal audit contract the ontology Service depends on.
// Any logger exposing a generic "log a line with a kind and key=value body"
// method can satisfy it. Implemented by audit.TraceLogger.LogEvent.
type AuditSink interface {
	LogEvent(kind, body string) error
}

// New returns a real Service when cfg.Enabled, otherwise NoOp. Safe to call
// with zero-value deps — every real-service method checks its dependencies.
func New(cfg Config, deps Deps) Service {
	if !cfg.Enabled {
		return NoOp{}
	}
	return newRealService(deps)
}
