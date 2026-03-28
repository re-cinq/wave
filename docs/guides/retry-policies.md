# Retry Policies

Wave provides named retry policies that control how steps handle failure. Policies define the number of attempts, backoff strategy, and delay timing. They replace raw `max_attempts` configuration with well-tested presets.

## Named Policies

| Policy | Attempts | Backoff | Base Delay | Max Delay | Use Case |
|--------|----------|---------|------------|-----------|----------|
| `none` | 1 | fixed | 0s | 0s | No retry -- fail immediately |
| `standard` | 3 | exponential | 1s | 30s | Default for most steps |
| `aggressive` | 5 | exponential | 200ms | 30s | Flaky operations, network calls |
| `patient` | 3 | exponential | 5s | 90s | Rate-limited APIs, heavy operations |

### Using a Named Policy

```yaml
steps:
  - id: fetch-data
    persona: navigator
    exec:
      type: prompt
      source: "Fetch and analyze the issue"
    retry:
      policy: standard
```

This gives the step 3 attempts with exponential backoff starting at 1 second.

### Custom Overrides

Explicit fields override policy defaults. Use this to customize a preset without defining everything from scratch:

```yaml
retry:
  policy: standard
  max_attempts: 5  # Override: 5 attempts instead of 3
  base_delay: "2s" # Override: start at 2 seconds
```

## Backoff Strategies

| Strategy | Formula | Example (base=1s) |
|----------|---------|-------------------|
| `fixed` | `base_delay` every attempt | 1s, 1s, 1s, ... |
| `linear` | `base_delay * attempt` | 1s, 2s, 3s, ... |
| `exponential` | `base_delay * 2^(attempt-1)` | 1s, 2s, 4s, 8s, ... |

All delays are capped at `max_delay`. Exponential backoff with a 1-second base and 30-second cap: 1s, 2s, 4s, 8s, 16s, 30s, 30s, ...

## Custom Policy Fields

For full control, skip the named policy and set fields directly:

```yaml
retry:
  max_attempts: 4
  backoff: exponential
  base_delay: "500ms"
  max_delay: "60s"
```

| Field | Default | Description |
|-------|---------|-------------|
| `policy` | -- | Named preset (`none`, `standard`, `aggressive`, `patient`) |
| `max_attempts` | 1 | Total attempts including the first run |
| `backoff` | `linear` | Backoff strategy (`fixed`, `linear`, `exponential`) |
| `base_delay` | `1s` | Initial delay between attempts |
| `max_delay` | `30s` | Maximum delay cap |

## Failure Handling: `on_failure`

The `on_failure` field controls what happens when all retry attempts are exhausted:

| Mode | Behavior |
|------|----------|
| `fail` | Pipeline stops with an error (default) |
| `skip` | Step is marked as skipped; pipeline continues |
| `continue` | Step is marked as failed but pipeline continues |
| `rework` | Delegates failure to a different step for remediation |
| `retry` | Re-executes the step (used in contract blocks) |

### Skip on Failure

For optional steps where failure should not block the pipeline:

```yaml
  - id: notify
    persona: navigator
    exec:
      type: prompt
      source: "Post a notification to Slack"
    retry:
      policy: standard
      on_failure: skip
```

### Continue on Failure

The step is recorded as failed but does not block downstream steps:

```yaml
  - id: lint
    persona: navigator
    exec:
      type: prompt
      source: "Run linting and report issues"
    retry:
      policy: none
      on_failure: continue
```

### Rework on Failure

Delegates the failure to a dedicated repair step. The rework step receives full failure context (error message, failure class, partial artifacts):

```yaml
  - id: implement
    persona: craftsman
    exec:
      type: prompt
      source: "Implement the feature"
    retry:
      policy: standard
      on_failure: rework
      rework_step: repair

  - id: repair
    persona: craftsman
    rework_only: true  # Only runs when triggered by rework
    exec:
      type: prompt
      source: "Fix the implementation failure"
```

The `rework_step` field is required when `on_failure` is `rework`, and the target step should typically set `rework_only: true` so it does not run during normal DAG scheduling.

## Prompt Adaptation: `adapt_prompt`

When `adapt_prompt: true`, Wave injects failure context from the previous attempt into the step's prompt. This gives the LLM information about what went wrong:

```yaml
retry:
  policy: standard
  adapt_prompt: true
```

The injected context includes:

- **Attempt number** and total allowed attempts
- **Prior error message** from the failed execution
- **Failure classification** (e.g., `compilation_error`, `test_failure`)
- **Last stdout** (up to 2000 characters)
- **Contract validation errors** (if the failure was a contract violation)

This is useful for implementation steps where the LLM can learn from its previous mistakes.

## Migration from `max_attempts`

If you have existing pipelines using raw retry configuration, migrate to named policies for consistency:

```yaml
# Before (raw configuration)
retry:
  max_attempts: 3

# After (named policy -- equivalent behavior)
retry:
  policy: standard
```

```yaml
# Before (aggressive raw configuration)
retry:
  max_attempts: 5
  backoff: exponential
  base_delay: "200ms"
  max_delay: "30s"

# After (named policy -- exact equivalent)
retry:
  policy: aggressive
```

Raw `max_attempts` still works -- named policies are syntactic sugar that fills in defaults. Explicit fields always override policy defaults.

## Combining Retry with Graph Loops

Retry policies and graph loops serve different purposes and compose cleanly:

- **Retry policies** handle within-step failures (transient errors, flaky operations)
- **Graph loops** handle step-to-step cycling (test failures routed to a fix step)

```yaml
  - id: fix
    persona: craftsman
    max_visits: 3  # Graph loop: max 3 fix cycles
    retry:
      policy: standard  # Within-step: 3 attempts per visit
    edges:
      - target: test
```

In this configuration, each visit to `fix` gets 3 retry attempts. If all 3 fail, the graph loop routes back to `test`, which may route back to `fix` up to `max_visits` times.

## Further Reading

- [Graph Loops](/guides/graph-loops) -- Edge-based routing and loop limits
- [Pipeline Configuration](/guides/pipeline-configuration) -- Step configuration and contracts
- [State & Resumption](/guides/state-resumption) -- Resuming after retry exhaustion
