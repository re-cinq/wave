# ADR-009: Ontology as a Bounded-Context Package

## Status
Accepted

## Date
2026-04-18

## Context

Wave's ontology feature — telos declaration, bounded-context injection into
persona prompts, post-merge staleness detection, and decision-lineage
tracking — was initially scoped off the default build with a `//go:build
ontology` tag. As the feature grew, the tag bled into six separate
packages:

| Package         | Enabled file                          | Disabled stub                          |
|-----------------|----------------------------------------|----------------------------------------|
| `pipeline`      | `ontology_enabled.go`                 | `ontology_disabled.go`                 |
| `doctor`        | `checks_ontology_enabled.go`          | `checks_ontology_disabled.go`          |
| `manifest`      | `parser_ontology_enabled.go`          | `parser_ontology_disabled.go`          |
| `audit`         | `logger_ontology_enabled.go`          | `logger_ontology_disabled.go`          |
| `onboarding`    | `ontology_step_enabled.go`            | `ontology_step_disabled.go`            |
| `tui`           | `ontology_list.go` + friends          | `ontology_stubs_disabled.go`           |

Each pair duplicated type declarations, method signatures, and the no-op
shape. Three specific problems resulted:

1. **Scattered ownership.** The `.agents/.ontology-stale` sentinel path
   appeared as a literal in at least four files (pipeline, doctor, tui,
   onboarding). Changing the sentinel location required edits in every
   package that knew about ontology at all.
2. **Interface pollution.** The `audit.AuditLogger` interface grew three
   ontology-specific methods (`LogOntologyInject/Lineage/Warn`). Domain
   concerns leaked into a cross-cutting package, and every future bounded
   context would have had to follow the same (bad) pattern.
3. **Runtime gate, not compile-time gate.** The `ontology` tag was never
   used by any CI or release workflow — every build compiled it. So the
   tag protected nothing; it only cost us source-of-truth drift and forced
   duplicate type declarations (e.g. `OntologyInfo` had to be re-declared
   in `tui/ontology_stubs_disabled.go` so untagged `content.go` could
   compile).

This ADR builds on [ADR-003: Layered Architecture](003-layered-architecture.md),
which classifies `pipeline`, `doctor`, and `onboarding` as Domain packages
and `manifest`, `audit`, and `event` as Cross-cutting. Under that model,
the correct home for behavioral ontology logic is a Domain-layer package
consumed through an interface.

## Decision

Consolidate all behavioral ontology logic into a new `internal/ontology`
package (Domain layer) exposing a single `Service` interface:

```go
type Service interface {
    Enabled() bool
    CheckStaleness() string
    BuildStepSection(pipelineID, stepID string, stepContexts []string) string
    RecordUsage(runID, stepID string, stepContexts []string, hasContract bool, stepStatus string)
    ValidateManifest(m *manifest.Manifest) []error
    InstallStalenessHook() error
}
```

`ontology.New(cfg, deps)` returns a real implementation when
`cfg.Enabled`, otherwise `ontology.NoOp{}`. The feature gate is resolved
at runtime from the manifest (`ontology.EnabledFromManifest(m)` = true iff
at least one context is declared) — no build tags.

Consumer rules:

- `internal/pipeline` holds an `ontology.Service` field and calls the
  interface. Auto-wiring happens once at pipeline start; injected callers
  can override via `WithOntologyService`.
- `internal/doctor` calls plain functions (`checkOntology` is untagged)
  and reads the staleness sentinel through `ontology.IsStaleInDir` rather
  than the literal path.
- `internal/onboarding` calls `ontology.InstallStalenessHookAt` instead of
  owning the hook body inline.
- `internal/tui` consumes `ontology.IsStaleInDir` for the sidebar
  staleness indicator; the three ontology view files lose their build
  tag.
- `internal/manifest` keeps `validateOntology` as plain shape validation —
  it is not behavioral, does not need a feature gate, and lives next to
  the other validators.
- `internal/audit` drops the three `LogOntology*` methods and gains a
  generic `LogEvent(kind, body)` that any bounded context can reuse.

The `internal/webui` ontology files keep their `//go:build ontology` tag —
they are the cosmetic dashboard UI, a legitimate feature-gated UI, and do
not participate in pipeline execution.

## Options Considered

### Option 1: Keep the build-tag split (status quo)

Leave the six enabled/disabled pairs in place.

**Pros:**
- No refactoring cost.
- Existing code paths are well-understood by contributors who already
  wrote them.

**Cons:**
- Every new behavioral surface (e.g. adding ontology events to the
  cost ledger) multiplies the number of tagged files.
- Sentinel path ownership remains scattered — a rename is a six-file
  change.
- `AuditLogger` keeps ontology-specific methods that no other bounded
  context would ever use, establishing a bad precedent.
- The tag is a runtime nop anyway (we build everything), so the cost
  buys nothing.

### Option 2: Interface in `pipeline`, inline everywhere else

Extract only the `pipeline` usage behind an interface but keep
doctor/onboarding/tui/manifest/audit talking directly to the feature.

**Pros:**
- Smallest blast radius.
- Keeps the type declarations close to their only consumer.

**Cons:**
- Solves only the pipeline-layer violation. The sentinel-path sprawl,
  audit-interface pollution, and duplicated type declarations persist.
- Future bounded contexts (e.g. the pending "budget" context) would
  repeat the same mistakes.

### Option 3: `internal/ontology` package with a `Service` interface (chosen)

Consolidate all behavioral logic in one package and expose a
`Service` + `NoOp` pair.

**Pros:**
- Single owner for the sentinel path, hook body, skill-path convention,
  and audit-line format.
- `AuditLogger` loses three domain-specific methods and gains one
  generic `LogEvent` that future bounded contexts can reuse.
- Untagged `tui` files no longer require duplicate type declarations in
  a stub file — the Service interface lets the UI handle "disabled" via
  the `NoOp` implementation.
- Sets a template for future bounded contexts (budget, workspace,
  thread) to follow the same shape.

**Cons:**
- One-time refactor touching seven packages.
- The Service method set is stable enough to be load-bearing now —
  adding a new behavior later costs one interface extension.

### Option 4: Extract each concern into its own package

`internal/ontology/staleness`, `internal/ontology/injection`,
`internal/ontology/lineage`, ...

**Pros:**
- Maximally decoupled.

**Cons:**
- Premature. All four concerns share the same manifest, same sentinel
  path, and the same audit sink. Splitting them dilutes the bounded
  context across packages.
- Go prefers cohesive packages; 4 one-file packages is more fragmentation
  cost than value.

## Consequences

### Positive
- `.agents/.ontology-stale` is written/read by `internal/ontology` only
  (plus the existing `cmd/wave/commands/analyze.go:745` cleanup site,
  which removes the sentinel after a successful `wave analyze` — noted
  and intentional). Moving the sentinel becomes a one-file change.
- `audit.AuditLogger` is three methods slimmer and closer to a generic
  trace sink.
- `internal/pipeline/executor.go` no longer reaches directly into
  manifest ontology fields — it calls `e.ontology.BuildStepSection(...)`
  and `e.ontology.RecordUsage(...)`, which matches ADR-003's separation
  of domain concerns.
- Future bounded contexts can follow the same pattern: package with
  `Service` interface + `NoOp` + `New(cfg, deps) Service` constructor.
- TUI ontology views no longer need a duplicate-type stub file — the
  NoOp service + nil checks in `content.go` handle the disabled case
  cleanly.

### Negative
- One-time refactor churn across seven packages.
- The `ontology` build tag is now meaningless outside `internal/webui`.
  We keep it there because the webui ontology dashboard is a cosmetic
  feature and feature-gating the handlers + templates saves binary size
  when the tag is off.

### Neutral
- Database schema, migrations, and `internal/state/` ontology types are
  unchanged. The `manifest.Ontology` struct is unchanged. Public CLI
  surface is unchanged.
- Depguard cannot express the sentinel-path-ownership rule (it operates
  on import paths, not string literals), so the boundary is enforced by
  convention and the `internal/ontology/service.go` package doc.

## Implementation Notes

Files created:
- `internal/ontology/service.go` — `Service`, `Config`, `Deps`, `New`,
  `EnabledFromManifest`, `AuditSink`.
- `internal/ontology/noop.go` — `NoOp` implementation.
- `internal/ontology/real.go` — staleness / injection / lineage.
- `internal/ontology/validate.go` — `ValidateManifest` delegating to
  `manifest.ValidateOntology`.
- `internal/ontology/hook.go` — `InstallStalenessHook*`, `IsStale*`.
- `internal/ontology/service_test.go` — NoOp, New, staleness, hook,
  validation.
- `internal/manifest/parser_ontology.go` — untagged `validateOntology` +
  exported `ValidateOntology`.
- `internal/doctor/checks_ontology.go` — untagged `checkOntology`.
- `internal/onboarding/ontology_step.go` — untagged wizard step.
- `docs/adr/009-ontology-bounded-context.md` — this ADR.

Files deleted:
- `internal/pipeline/ontology_{enabled,disabled}.go`
- `internal/doctor/checks_ontology_{enabled,disabled}.go`
- `internal/manifest/parser_ontology_{enabled,disabled}.go`
- `internal/audit/logger_ontology_{enabled,disabled}.go`
- `internal/onboarding/ontology_step_{enabled,disabled}.go`
- `internal/tui/ontology_stubs_disabled.go`

Files modified:
- `internal/pipeline/executor.go` — add `ontology` field, option,
  auto-wire, replace three helper calls.
- `internal/audit/logger.go` — drop three `LogOntology*` methods, add
  `LogEvent`.
- `internal/manifest/parser.go` — drop the stale comment block.
- `internal/tui/{ontology_list,ontology_detail,ontology_provider}.go`
  and their tests — remove `//go:build ontology` tag.
- `internal/tui/ontology_provider.go` — read staleness via
  `ontology.IsStaleInDir`.
- `internal/doctor/checks_config.go` — update the header comment.
- `.golangci.yml` — ADR-009 comment marker (no enforceable rule
  available).

Related work:
- ADR-003 layer classification is preserved — `internal/ontology` is
  Domain, consuming only Cross-cutting and Infrastructure.
- `cmd/wave/commands/analyze.go:745` still removes the sentinel after a
  successful refresh. Left as-is because moving it into the Service
  would require the analyze command to construct one mid-flow for a
  single side effect; the sentinel file lifecycle crosses the command
  boundary intentionally (produce: git hook, consume: pipeline + doctor,
  clear: analyze command).
