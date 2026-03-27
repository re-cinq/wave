# feat: failure classification taxonomy with circuit breaker and intelligent retry

**Issue**: [re-cinq/wave#579](https://github.com/re-cinq/wave/issues/579)
**Labels**: enhancement
**Author**: nextlevelshit
**Complexity**: complex

## Context

Fabro classifies failures into 6 categories and uses **normalized failure fingerprints** to detect repeated failures — automatically terminating when the same failure signature repeats N times (circuit breaker). They also have 3-layer retry (LLM call, turn, node) with different policies per failure class.

Wave currently has basic `on_failure` actions (`fail`/`skip`/`continue`/`rework`/`retry`) without failure classification or intelligent circuit breaking.

## Design Goals — Best of Both Worlds

Combine Fabro's failure intelligence with Wave's contract validation to create a more resilient execution model.

### Failure Classification

Classify every step failure into one of these categories:

| Class | Meaning | Retryable? | Example |
|-------|---------|------------|---------|
| `transient` | Rate limits, timeouts, network | Yes (auto-retry) | API 429, connection timeout |
| `deterministic` | Auth errors, bad config | No | Invalid API key, missing binary |
| `budget_exhausted` | Context/token limits | No (trigger fallback) | Context window exceeded |
| `contract_failure` | Output doesn't match schema | Yes (rework) | JSON schema mismatch |
| `test_failure` | Tests/validation failed | Yes (fix loop) | `go test` exit code 1 |
| `canceled` | User/system cancellation | No | SIGINT, timeout |

The executor should attempt to classify failures based on:
1. Exit codes from adapter/commands
2. Error message patterns (regex matching)
3. Contract validation results
4. Adapter-specific error parsing

### Circuit Breaker

After each failure, create a **normalized fingerprint**: `step_name + failure_class + normalized_error`. When the same fingerprint repeats `circuit_breaker_limit` times (default: 3), terminate the step/pipeline.

Only `deterministic` and `contract_failure` failures are tracked — transient failures are excluded from fingerprinting.

```yaml
runtime:
  circuit_breaker:
    limit: 3                    # same failure 3x = terminate
    tracked_classes: [deterministic, contract_failure, test_failure]
```

### Retry Policies

Per-step retry policy configuration:

```yaml
steps:
  - name: implement
    retry:
      policy: standard          # none | standard | aggressive | patient
      max_attempts: 5
      backoff: exponential      # exponential | linear | constant
      initial_delay_ms: 200
      max_delay_ms: 30000
```

Built-in policies:
- `none`: 1 attempt, immediate failure
- `standard`: 3 attempts, 1s initial, 2x exponential (default)
- `aggressive`: 5 attempts, 200ms initial, 2x exponential
- `patient`: 3 attempts, 5s initial, 3x exponential

### Stall Watchdog

If a step produces no progress events for `stall_timeout` (default: 30min), terminate it:

```yaml
runtime:
  stall_timeout: 1800s
```

## What Wave Keeps

- Contract validation (enriches failure classification — contract failures are a distinct class)
- SQLite state (failure signatures stored for circuit breaker)
- Existing `on_failure` actions (extended with classification awareness)

## What Wave Gains

- **Intelligent retry** — don't retry deterministic failures, do retry transient ones
- **Circuit breaker** — prevents infinite loops on repeated identical failures
- **Failure analytics** — classification feeds into retrospectives
- **Stall detection** — catches hung adapters

## Acceptance Criteria

1. All step failures are classified into one of the 6 failure classes
2. Retry policies (none/standard/aggressive/patient) can be configured per-step and resolve correctly
3. Circuit breaker terminates execution when same failure fingerprint repeats N times
4. Stall watchdog terminates steps with no progress events after configurable timeout
5. Failure class is stored in `step_attempt.failure_class` and emitted in events
6. Existing `on_failure` behavior is preserved — classification is additive
7. All new code has comprehensive unit tests
8. `go test -race ./...` passes

## Research Sources

- Fabro failure handling: https://docs.fabro.sh/execution/failures
- Fabro 6 failure classes: transient_infra, deterministic, budget_exhausted, compilation_loop, canceled, structural
- Fabro circuit breaker: normalized fingerprint + 3-repeat threshold
