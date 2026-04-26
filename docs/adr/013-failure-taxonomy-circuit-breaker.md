# ADR-013: Failure Taxonomy and Circuit Breaker

## Status

Accepted

## Date

2026-03-27 (proposed) — 2026-04-26 (accepted; renumbered from ADR-004 to resolve collision with ADR-004 Multi-Adapter Architecture)

## Implementation Status

Substantially landed. Verified 2026-04-26:
- Unified 6-class taxonomy in `internal/pipeline/failure.go` (`FailureClassTransient`, `Deterministic`, `BudgetExhausted`, `ContractFailure`, `TestFailure`, `Canceled`).
- `ClassifyStepFailure()` in `failure.go` collapses prior `adapter.FailureReason`, `recovery.ErrorClass`, `contract.FailureType` into the canonical taxonomy.
- `CircuitBreaker` struct with fingerprinting (`NormalizeFingerprint()` strips timestamps, hex, UUIDs, temp paths) integrated into executor at `executor.go:~1856`.
- SQLite `step_attempt.failure_class` column persists classifications.

Remaining work (not blocking acceptance):
- Stall watchdog (Phase 4 of original plan) not implemented.
- Logic lives in `internal/pipeline/failure.go` rather than a cross-cutting `internal/failure/` package; revisit if ADR-003 layer enforcement requires extraction.

## Context

Wave currently has three overlapping, uncoordinated failure classification systems:

1. **adapter.FailureReason** (4 types: `timeout`, `context_exhaustion`, `rate_limit`, `general_error`) — pattern-matches subprocess stderr content in `internal/adapter/errors.go`.
2. **recovery.ErrorClass** (5 types: `contract_validation`, `security_violation`, `preflight`, `runtime_error`, `unknown`) — maps errors to recovery hints in `internal/recovery/classify.go`.
3. **contract.FailureType** (6 types: `schema_mismatch`, `missing_content`, `format_error`, `quality_gate`, `structure`, `unknown`) — classifies validation failures with confidence scores in `internal/contract/retry_strategy.go`.

These taxonomies produce incompatible output types (`StepError`, `RecoveryBlock`, `ClassifiedFailure`), are consumed by different subsystems, and none map to the categories the executor actually needs for retry and routing decisions.

The retry loop in `executor.go` (lines 757-847) supports per-step `RetryConfig` with `max_attempts`, backoff strategies, and `on_failure` policy routing (`fail`/`skip`/`continue`/`rework`), but has no circuit breaker to detect repeated identical failures and no stall watchdog to catch hung steps. The SQLite `step_attempt` table records `failure_class` as a free-form string with no normalization, making historical failure pattern queries unreliable.

At 3,283 lines, `executor.go` is already a monolith that ADR-002 proposes splitting. The continuous runner (`internal/continuous/`) has its own independent `halt`/`skip` failure policies with no connection to step-level classification.

## Decision

Create a new **`internal/failure/`** package positioned as a **Layer 4 cross-cutting concern** (per ADR-003). This package will own:

1. **Canonical taxonomy** — a `FailureCategory` enum with 6 values: `transient`, `deterministic`, `budget_exhausted`, `contract_failure`, `test_failure`, `canceled`.
2. **Unified classifier** — a `Classifier` that dispatches to existing classification logic in adapter, recovery, and contract packages via a `FailureSource` interface, producing a single `ClassifiedFailure` struct with category, fingerprint, confidence, and remediation fields.
3. **Fingerprinter** — normalizes heterogeneous error messages (stderr patterns, structured `ValidationError` objects, error string content) into stable fingerprint strings suitable for circuit breaker matching.
4. **Circuit breaker** — tracks fingerprint frequencies across attempts and trips when configurable thresholds are exceeded, surfacing a `CircuitOpen` signal to the executor's retry loop.
5. **Stall watchdog** — monitors heartbeat signals from step execution and fires timeout events when progress stalls, complementing the existing per-step timeout system in `internal/timeouts/`.

Existing classification functions (`adapter.ClassifyFailure`, `recovery.ClassifyError`, `contract.FailureClassifier`) will be refactored to implement a `FailureSource` interface defined by the failure package, or to produce raw errors that the failure package classifies.

## Options Considered

### Option A: Inline in Executor

Add the 6-category taxonomy, fingerprinting, circuit breaker, and stall watchdog directly into `executor.go`. Map existing classifiers to the new taxonomy via a local `classify()` method. Circuit breaker state tracked in executor-local maps.

**Pros:**
- Zero new package boundaries; no import graph changes
- Classification logic co-located with the retry loop and `on_failure` routing
- Fastest path to a working prototype

**Cons:**
- Adds 400-600 lines to an already 3,283-line file, directly conflicting with ADR-002's extraction plan
- Failure classification cannot be tested in isolation
- Creates a fourth overlapping system rather than unifying the existing three
- Circuit breaker state trapped inside the executor is invisible to the continuous runner's `halt`/`skip` policies
- Violates ADR-003: cross-cutting concern buried in Domain Layer 2

**Risk:** High. **Reversibility:** Difficult.

### Option B: Separate `internal/failure` Package (Cross-cutting) — RECOMMENDED

Create a new cross-cutting package owning the canonical taxonomy, unified classifier, fingerprinter, circuit breaker, and stall watchdog. Existing classifiers refactored to implement `FailureSource` interface.

**Pros:**
- Unifies three overlapping systems (4 + 5 + 6 failure types) into one canonical taxonomy
- Aligns with ADR-002: the extracted `StepExecutor` receives `failure.Classifier` as an injected dependency
- Aligns with ADR-003: Layer 4 cross-cutting packages can be imported by all layers without violating dependency rules
- Independently testable — unit tests need no executor, adapter, or contract infrastructure
- Centralized fingerprinting means all consumers (executor, continuous runner, state store, TUI/WebUI) see the same normalized failure identity
- Circuit breaker state accessible to both per-step retry logic and per-iteration halt/skip logic
- Normalized `failure_class` values make SQLite circuit breaker queries reliable

**Cons:**
- Largest implementation effort — new package, new interfaces, migration of three classifiers
- Circular dependency risk if the failure package imports adapter, contract, or pipeline (must define interfaces those packages implement instead)
- Broad blast radius: touches adapter, recovery, contract, pipeline, and state packages
- Possible over-engineering for current prototype-phase usage
- Stall watchdog scope (per-step vs. per-pipeline) requires careful boundary design

**Risk:** Medium. **Reversibility:** Easy (clean package boundary means it can be inlined or restructured without cascading changes).

### Option C: Hybrid — Shared Types + Executor-owned Behavior

Create a thin `internal/failure/` package owning only types (`FailureCategory`, `ClassifiedFailure`, `Fingerprint`) and the fingerprinter. Keep circuit breaker, stall watchdog, and classification dispatch in the executor. Existing classifiers remain as-is, wrapped via a mapping table.

**Pros:**
- Shared types prevent taxonomy fragmentation without full migration
- Existing classifiers continue working unmodified
- Smaller initial effort (~250 lines for types + fingerprinter)
- Compatible with ADR-002: mapping table and circuit breaker move naturally with StepExecutor extraction

**Cons:**
- Three classification systems remain separate; mapping table is a translation layer that can drift
- Circuit breaker in executor still invisible to continuous runner
- Split ownership (fingerprinter in failure, circuit breaker in executor) for related logic
- Thin types packages tend to grow into full packages over time, making this a temporary compromise
- Mapping table requires manual updates whenever an underlying classifier changes

**Risk:** Medium. **Reversibility:** Easy.

### Option D: Extend Existing Packages In-place

Expand `recovery/` to own the 6-category taxonomy (closest existing enum), add fingerprinting to recovery, add circuit breaker queries to `state/`, add stall watchdog to `timeouts/`. No new packages.

**Pros:**
- Zero import graph changes
- Each extension is a natural growth of its host package
- Smallest disruption; incremental delivery possible
- Prototype-friendly: avoids premature abstraction

**Cons:**
- Overloads `recovery` beyond its original scope (user-facing hints vs. infrastructure classification)
- Couples `state` (Infrastructure Layer 3) to Domain Layer 2 failure concepts, violating ADR-003
- Only unifies one of three systems — adapter and contract classifiers remain separate
- Fingerprinting ownership scattered across packages with no single circuit breaker owner
- Stall watchdog in `timeouts` has no clean path to emit pipeline-level events without circular imports
- Scattered logic becomes technical debt when ADR-002 extracts StepExecutor

**Risk:** Medium. **Reversibility:** Moderate.

## Consequences

### Positive

- **Single source of truth** for failure classification — eliminates the current confusion of three overlapping taxonomies with 15 total failure type values
- **Circuit breaker prevents waste** — stops retrying failures that will never succeed (deterministic errors, budget exhaustion), saving API tokens and wall-clock time
- **Stall watchdog catches hung pipelines** — detects cases where a step appears to be running but makes no progress, complementing the existing hard timeout
- **Normalized fingerprints enable pattern analysis** — historical failure data in SQLite becomes queryable for trend detection and reliability reporting
- **Clean dependency boundary** supports ADR-002's StepExecutor extraction — `failure.Classifier` is a natural injected dependency
- **Consistent failure reporting** across TUI, WebUI, audit logs, and recovery hints — all consumers see the same `ClassifiedFailure` struct

### Negative

- **Broad migration scope** — adapter, recovery, contract, pipeline, and state packages all require changes to adopt the new interfaces
- **Interface design is load-bearing** — the `FailureSource` interface must be correct from the start; a wrong abstraction boundary is harder to fix than no abstraction
- **Temporary complexity increase** during migration — both old and new classification paths may coexist briefly until migration completes
- **Circuit breaker tuning** requires empirical data — default thresholds (trip count, window duration, reset policy) will need adjustment based on real pipeline execution patterns

### Neutral

- SQLite `step_attempt` schema requires migration to use normalized `failure_class` values (no backward-compatibility constraint in prototype phase, but existing test fixtures need updating)
- The `on_failure` policy system (`fail`/`skip`/`continue`/`rework`) is unchanged — the failure taxonomy informs which policy to invoke but does not replace the policy mechanism
- Pipeline YAML authors gain an optional `circuit_breaker` configuration block per step but are not required to use it (sensible defaults apply)
- The continuous runner's `halt`/`skip` policies can now query the circuit breaker for informed decisions but are not required to do so immediately

## Implementation Notes

### Phase 1: Types and Classifier

1. Create `internal/failure/` with `category.go` (FailureCategory enum, ClassifiedFailure struct, Fingerprint type), `classifier.go` (Classifier interface and dispatch implementation), and `fingerprint.go` (normalization logic).
2. Define the `FailureSource` interface that adapter, recovery, and contract packages will implement.
3. Add comprehensive table-driven tests for fingerprint normalization and category mapping.

### Phase 2: Migrate Existing Classifiers

4. Refactor `adapter.ClassifyFailure` to implement `FailureSource` — return structured data the failure package can classify.
5. Refactor `recovery.ClassifyError` to implement `FailureSource` — map existing 5 `ErrorClass` values to the 6 canonical categories.
6. Refactor `contract.FailureClassifier` to implement `FailureSource` — preserve confidence scores in the `ClassifiedFailure` struct.
7. Update `state.Store` to normalize `failure_class` using the canonical taxonomy. Migrate the `step_attempt` schema.

### Phase 3: Circuit Breaker

8. Implement `CircuitBreaker` in `internal/failure/breaker.go` with configurable trip thresholds, time windows, and half-open reset policy.
9. Integrate circuit breaker checks into the executor retry loop (`executor.go:757-847`) — check before each attempt, record fingerprints after each failure.
10. Expose circuit breaker state to `internal/continuous/` for iteration-level `halt` decisions.

### Phase 4: Stall Watchdog

11. Implement `StallWatchdog` in `internal/failure/watchdog.go` — monitors heartbeat signals, fires events via the event emitter.
12. Instrument the executor to emit heartbeats during step execution (adapter invocation, contract validation, artifact collection).
13. Connect watchdog timeout events to the existing `on_failure` policy routing.

### Key Files Affected

| Package | Files | Changes |
|---------|-------|---------|
| `internal/failure/` (new) | `category.go`, `classifier.go`, `fingerprint.go`, `breaker.go`, `watchdog.go` | New package |
| `internal/adapter/` | `errors.go` | Implement `FailureSource`; refactor `ClassifyFailure` |
| `internal/recovery/` | `classify.go` | Implement `FailureSource`; map `ErrorClass` → `FailureCategory` |
| `internal/contract/` | `retry_strategy.go` | Implement `FailureSource`; preserve confidence scores |
| `internal/pipeline/` | `executor.go`, `types.go` | Integrate `Classifier` and `CircuitBreaker` into retry loop |
| `internal/state/` | `store.go`, `schema.sql` | Normalize `failure_class`; add circuit breaker query methods |
| `internal/continuous/` | `runner.go` | Query circuit breaker for halt/skip decisions |
| `internal/event/` | `emitter.go` | Add failure and watchdog event types |

### Dependency Direction

```
Layer 4 (Cross-cutting):  internal/failure/  ←── defines FailureSource interface
                                ↑
Layer 2 (Domain):     adapter, recovery, contract  ←── implement FailureSource
                      pipeline/executor             ←── consumes Classifier, CircuitBreaker
                                ↑
Layer 3 (Infrastructure):  state/               ←── persists ClassifiedFailure
```

The failure package **must not** import adapter, contract, pipeline, or recovery. It defines interfaces; those packages implement them. This is enforced by Go's import cycle detection.
