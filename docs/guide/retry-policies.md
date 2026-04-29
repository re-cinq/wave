# Retry Policies

Named retry policies provide preset configurations for common retry patterns instead of raw `max_attempts` values.

## Policies

| Policy | Max Attempts | Base Delay | Backoff | Max Delay | Use Case |
|--------|-------------|------------|---------|-----------|----------|
| `none` | 1 | — | — | — | Steps that must not retry |
| `standard` | 3 | 1s | 2x exponential | 30s | Default for implementation steps |
| `aggressive` | 5 | 200ms | 2x exponential | 30s | API calls, fetches, publishes |
| `patient` | 3 | 5s | 3x exponential | 90s | Analysis, scanning, exploration |

## Usage

```yaml
steps:
  - id: fetch
    retry:
      policy: aggressive        # 5 attempts, fast backoff

  - id: implement
    retry:
      policy: standard          # 3 attempts, balanced
      max_attempts: 5           # override: more attempts than default

  - id: analyze
    retry:
      policy: patient           # 3 attempts, slow backoff
```

Explicit fields override policy defaults — set `policy` for the base, then override individual fields as needed.

## Model Tier Escalation on Retry

When a step retries, Wave automatically escalates the model one tier stronger
along the cost ladder `cheapest -> balanced -> strongest`. The first retry
moves up one tier, the second another, and once `strongest` is reached
further retries stay there.

```yaml
steps:
  - id: implement
    persona: craftsman
    model: cheapest          # attempt 1: cheapest -> haiku
    retry:
      policy: standard       # attempt 2: balanced -> adapter default
                             # attempt 3: strongest -> opus
```

Escalation only applies when the step's effective model is a recognized
tier name (`cheapest`, `balanced`, `strongest`). Literal model IDs (e.g.
`claude-opus-4`, `gpt-4o-mini`) are user-pinned overrides and are
preserved verbatim across retries.

Set `retry.no_escalate: true` to disable escalation and reuse the same
model across retries:

```yaml
steps:
  - id: scan
    persona: navigator
    model: cheapest
    retry:
      policy: standard
      no_escalate: true      # keep using cheapest on every retry
```

## Failure Classification

The retry system classifies failures into 6 categories:

| Class | Retryable? | Example |
|-------|------------|---------|
| `transient` | Yes (auto-retry) | API 429, timeout |
| `deterministic` | No | Invalid API key, missing binary |
| `budget_exhausted` | No (trigger fallback) | Context window exceeded |
| `contract_failure` | Yes (rework) | JSON schema mismatch |
| `test_failure` | Yes (fix loop) | `go test` exit code 1 |
| `canceled` | No | SIGINT, timeout |

## Circuit Breaker

Repeated identical failures terminate the step, preventing infinite retry loops on persistent issues:

```yaml
runtime:
  circuit_breaker:
    limit: 3
    tracked_classes: [deterministic, contract_failure, test_failure]
```

**Failure fingerprinting**: The circuit breaker tracks identical errors by creating a fingerprint from step ID, failure class, and error message. Only the same error repeated counts—not different errors.

**tracked_classes**: Configure which failure types count toward the limit:
- `deterministic` — Invalid API keys, missing binaries (won't succeed on retry)
- `contract_failure` — Schema mismatches, output validation failures
- `test_failure` — Test suite failures
- `transient` — Network timeouts, rate limits
- `budget_exhausted` — Context window exceeded

**vs max_visits**: max_visits counts any step visit (same or different errors), useful for limiting total attempts. Circuit breaker only trips on repeated identical errors, useful for detecting persistent failures.

## Stall Watchdog

Steps producing no progress events for 30 minutes are terminated:

```yaml
runtime:
  stall_timeout: 1800s
```
