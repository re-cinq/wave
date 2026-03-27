# Tasks

## Phase 1: Failure Taxonomy Foundation

- [X] Task 1.1: Add failure class constants to `internal/pipeline/failure.go`
  - Define `FailureClassTransient`, `FailureClassDeterministic`, `FailureClassBudgetExhausted`, `FailureClassContractFailure`, `FailureClassTestFailure`, `FailureClassCanceled`
  - Add `IsRetryable(class string) bool` helper
  - Add `ClassifyStepFailure()` that takes adapter error, contract result, context error and returns failure class

- [X] Task 1.2: Add retry policy presets to `internal/pipeline/types.go`
  - Add `Policy` field to `RetryConfig` struct
  - Implement `ResolvePolicy()` method with 4 presets (none/standard/aggressive/patient)
  - Explicit `MaxAttempts`/`Backoff`/`BaseDelay` override policy defaults
  - Add validation: unknown policy name returns error

- [X] Task 1.3: Add `CircuitBreaker` and `StallTimeout` config to `internal/manifest/types.go`
  - Add `CircuitBreakerConfig` struct with `Limit int` and `TrackedClasses []string`
  - Add `StallTimeout string` to `Runtime` struct
  - Add `CircuitBreaker CircuitBreakerConfig` to `Runtime` struct

## Phase 2: Core Implementation

- [X] Task 2.1: Implement failure classification logic in `internal/pipeline/failure.go` [P]
  - Map `adapter.FailureReasonTimeout` → `transient`
  - Map `adapter.FailureReasonRateLimit` → `transient`
  - Map `adapter.FailureReasonContextExhaustion` → `budget_exhausted`
  - Map `contract.ValidationError` → `contract_failure`
  - Map `context.Canceled` → `canceled`
  - Pattern-match error messages for auth/config errors → `deterministic`
  - Pattern-match test failure indicators → `test_failure`
  - Default unknown → `transient` (safe: allows retry)

- [X] Task 2.2: Implement fingerprinting and circuit breaker in `internal/pipeline/failure.go` [P]
  - `NormalizeFingerprint(stepID, failureClass, errorMsg string) string` — strips timestamps, line numbers, variable content
  - `CircuitBreaker` struct with `counts map[string]int`, `limit int`, `trackedClasses map[string]bool`
  - `CircuitBreaker.Record(fingerprint, failureClass string) bool` — returns true if tripped
  - `CircuitBreaker.LoadFromAttempts(attempts []state.StepAttempt)` — rebuild counts on resume

- [X] Task 2.3: Implement stall watchdog in `internal/pipeline/watchdog.go` [P]
  - `StallWatchdog` struct with `timeout time.Duration`, `activity chan struct{}`, `cancel context.CancelFunc`
  - `NewStallWatchdog(timeout time.Duration) *StallWatchdog`
  - `Start(ctx context.Context) context.Context` — returns a derived context that cancels on stall
  - `NotifyActivity()` — resets the stall timer
  - `Stop()` — clean shutdown

## Phase 3: Executor Integration

- [X] Task 3.1: Wire failure classification into executor retry loop
  - In `executeStep()`: after `runStepExecution()` fails, call `ClassifyStepFailure()` to get failure class
  - Store failure class in `AttemptContext.FailureClass`
  - Pass failure class to `RecordStepAttempt()` (already has `failure_class` column)
  - If `!IsRetryable(class)` and not on final attempt, skip remaining retries immediately

- [X] Task 3.2: Wire circuit breaker into executor
  - Create `CircuitBreaker` in `Executor.Run()` from `manifest.Runtime.CircuitBreaker` config
  - Before each retry attempt: compute fingerprint, call `cb.Record()` — if tripped, terminate step
  - On resume: load prior attempts from state DB, rebuild circuit breaker counts
  - Emit event with `FailureClass` field when circuit breaker trips

- [X] Task 3.3: Wire stall watchdog into step execution
  - If `manifest.Runtime.StallTimeout` is set: create `StallWatchdog`
  - Wrap step execution context with watchdog's derived context
  - Forward progress events to `watchdog.NotifyActivity()`
  - On stall timeout: classify as `canceled`, emit event with remediation message

- [X] Task 3.4: Wire retry policy resolution into pipeline loading
  - Call `RetryConfig.ResolvePolicy()` during pipeline validation (in `validation.go` or loader)
  - Ensure policy is resolved before executor sees the config

## Phase 4: Event and Recovery Updates

- [X] Task 4.1: Add `FailureClass` to event emission [P]
  - Add `FailureClass string` field to `event.Event` struct
  - Populate `FailureClass` in retry, failure, and circuit breaker events
  - Existing `FailureReason` field (adapter-level) remains unchanged

- [X] Task 4.2: Bridge failure classes to recovery hints [P]
  - In `recovery/classify.go`: map new pipeline failure classes to existing `ErrorClass`
  - `transient` → `ClassRuntimeError` (with retry hint)
  - `deterministic` → `ClassRuntimeError` (with config fix hint)
  - `budget_exhausted` → `ClassRuntimeError` (with context reduction hint)
  - `contract_failure` → `ClassContractValidation` (already handled)
  - `test_failure` → `ClassRuntimeError` (with fix hint)
  - `canceled` → `ClassUnknown` (no recovery)

## Phase 5: Testing

- [X] Task 5.1: Unit tests for failure classification (`failure_test.go`)
  - Table-driven tests: each failure class with representative error inputs
  - Edge cases: nil error, empty strings, ambiguous messages
  - `IsRetryable()` for each class

- [X] Task 5.2: Unit tests for fingerprinting and circuit breaker (`failure_test.go`)
  - Fingerprint normalization: strip variable content, deterministic output
  - Circuit breaker: recording, tripping at limit, tracked vs untracked classes
  - Circuit breaker: resume from prior attempts

- [X] Task 5.3: Unit tests for stall watchdog (`watchdog_test.go`)
  - Activity resets extend the deadline
  - Timeout fires context cancellation
  - Stop prevents timeout after shutdown
  - Concurrent activity notifications are safe

- [X] Task 5.4: Unit tests for retry policy resolution (`types_test.go`)
  - Each named policy resolves to expected values
  - Explicit values override policy
  - Unknown policy returns validation error
  - Empty policy preserves existing behavior

- [X] Task 5.5: Integration tests for executor with classification
  - Transient failure → retried automatically
  - Deterministic failure → retries skipped, immediate on_failure
  - Circuit breaker trips after N identical failures
  - Stall timeout terminates step
  - Contract failure correctly classified

## Phase 6: Validation

- [X] Task 6.1: Run full test suite
  - `go test ./...` passes
  - `go test -race ./...` passes
  - `golangci-lint run ./...` passes

- [X] Task 6.2: Verify backward compatibility
  - Existing pipelines with `retry.max_attempts` still work
  - Existing `on_failure` policies still work
  - No manifest changes required for existing users
