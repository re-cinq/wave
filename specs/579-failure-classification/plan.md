# Implementation Plan: Failure Classification Taxonomy

## Objective

Add a 6-class failure taxonomy, circuit breaker with fingerprinting, configurable retry policy presets, and stall watchdog to Wave's pipeline executor. This replaces the current flat `general_error` classification with intelligent failure routing that prevents infinite retry loops and enables class-aware retry decisions.

## Approach

The implementation extends existing structures rather than replacing them. The current `adapter.ClassifyFailure()` (4 reasons) becomes the first stage of a two-stage classification pipeline: adapter-level classification feeds into the new 6-class taxonomy in `internal/pipeline/failure.go`. The circuit breaker and stall watchdog are new subsystems that hook into the executor's existing retry loop.

### Key Design Decisions

1. **Two-stage classification**: `adapter.ClassifyFailure()` stays as-is (it maps adapter-specific signals). A new `pipeline.ClassifyFailure()` consumes the adapter reason + error type + contract result to produce the 6-class taxonomy. This avoids coupling adapter code to pipeline semantics.

2. **Retry policy = syntactic sugar**: Policies (`standard`, `aggressive`, `patient`) resolve to concrete `RetryConfig` values at load time. A `Policy` field on `RetryConfig` is sugar — explicit `MaxAttempts`/`Backoff`/`BaseDelay` override it. No new struct needed.

3. **Circuit breaker as a standalone type**: `CircuitBreaker` lives in `internal/pipeline/failure.go`, holds an in-memory map of `fingerprint → count`, and is consulted by the executor before each retry attempt. State is also persisted to SQLite for cross-resume awareness.

4. **Stall watchdog via context cancellation**: The executor already uses `context.WithTimeout` for step timeout. The stall watchdog wraps this with an activity-based deadline that resets on each progress event. Implemented as a `StallWatchdog` goroutine that monitors a channel.

5. **No migration for fingerprints**: The existing `step_attempt` table already has `failure_class` and `error_message` columns. Fingerprints can be computed from these columns at resume time — no new table needed.

## File Mapping

### New Files

| File | Purpose |
|------|---------|
| `internal/pipeline/failure.go` | Failure taxonomy constants, `ClassifyFailure()`, fingerprinting, `CircuitBreaker` type |
| `internal/pipeline/failure_test.go` | Unit tests for classification, fingerprinting, circuit breaker |
| `internal/pipeline/watchdog.go` | `StallWatchdog` type with activity channel + context cancellation |
| `internal/pipeline/watchdog_test.go` | Unit tests for stall watchdog |

### Modified Files

| File | Changes |
|------|---------|
| `internal/pipeline/types.go` | Add `Policy` field to `RetryConfig`, add `ResolvePolicy()` method, add `FailureClass` constants |
| `internal/pipeline/types_test.go` | Tests for policy resolution and validation |
| `internal/pipeline/executor.go` | Wire classification into retry loop, consult circuit breaker before retry, start stall watchdog, pass failure class to `AttemptContext` |
| `internal/manifest/types.go` | Add `CircuitBreaker` and `StallTimeout` to `Runtime` struct |
| `internal/event/emitter.go` | Add `FailureClass` field to `Event` struct (alongside existing `FailureReason`) |
| `internal/recovery/classify.go` | Map new failure classes to recovery `ErrorClass` (bridge layer) |
| `internal/adapter/errors.go` | No changes — existing classification feeds into pipeline-level classifier |

## Architecture Decisions

### AD-1: Failure class as string constants, not iota enum
String constants (`"transient"`, `"deterministic"`, etc.) are used because they serialize directly to YAML/JSON/SQLite without conversion. This matches the existing pattern for `OnFailureFail`, `StatePending`, etc.

### AD-2: Circuit breaker is per-pipeline-run, not global
Each `Executor.Run()` call creates a fresh `CircuitBreaker`. On resume, fingerprint counts are rebuilt from `step_attempt` rows for that `run_id`. This prevents cross-run interference while still detecting repeated failures within a run.

### AD-3: Stall watchdog is opt-in via runtime config
Default `stall_timeout: 0` means disabled. When set, the watchdog runs as a goroutine per step execution. It monitors the event emitter's output channel — any event for the current step resets the timer.

### AD-4: Policy resolution happens at validation time
When a pipeline is loaded, `RetryConfig.ResolvePolicy()` fills in `MaxAttempts`, `Backoff`, `BaseDelay` from the named policy. Explicit values override policy defaults. This means the executor only ever sees concrete values — no runtime policy lookup.

### AD-5: Classification is best-effort, falls back to existing behavior
If the classifier can't determine a specific class (e.g., opaque exit code 1), it falls back to the existing `general_error` → maps to `transient` for retryability purposes. This preserves backward compatibility.

## Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| False classification (e.g., test failure classified as transient) | Unnecessary retries, wasted tokens | Conservative classification — unknown errors default to `transient` which allows retry (safe fallback) |
| Circuit breaker too aggressive | Steps that would eventually succeed get terminated | Default limit of 3 is generous; configurable per manifest |
| Stall watchdog false positives | Steps doing long computations without events get killed | Default disabled (0); when enabled, 30min is generous |
| Retry policy changes break existing pipelines | Existing `max_attempts`/`backoff` settings ignored | Explicit values always override policy — policy is additive sugar |
| `step_attempt` table gets large for long-running pipelines | Slow fingerprint queries | Already indexed by `run_id` and `step_id`; fingerprint computation is in-memory |

## Testing Strategy

### Unit Tests
- `failure_test.go`: Classification logic for all 6 classes with various error types, exit codes, error messages
- `failure_test.go`: Fingerprint normalization (same error with different timestamps produces same fingerprint)
- `failure_test.go`: Circuit breaker tracking and tripping
- `watchdog_test.go`: Activity resets, timeout firing, cancellation propagation
- `types_test.go`: Policy resolution, policy + explicit override, unknown policy validation error

### Integration Tests
- Executor test with circuit breaker: step fails 3 times with same error → pipeline terminated
- Executor test with transient failure: rate limit error → classified as transient → retried
- Executor test with contract failure: schema mismatch → classified as `contract_failure`
- Executor test with stall watchdog: no events for N seconds → step terminated

### Existing Test Compatibility
- All existing executor tests must pass unchanged (classification is additive)
- Existing `RetryConfig` tests must pass (no fields removed)
