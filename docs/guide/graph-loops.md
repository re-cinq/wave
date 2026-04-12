# Graph Loops and Conditional Routing

Wave pipelines support cycles, conditional routing, and command steps — enabling implement-test-fix loops that self-correct without predefined retry counts.

## Basic Loop: Implement → Test → Fix

```yaml
steps:
  - id: implement
    persona: craftsman
    thread: impl

  - id: run-tests
    type: command
    dependencies: [implement]
    script: "{{ project.contract_test_command }}"

  - id: gate
    type: conditional
    dependencies: [run-tests]
    edges:
      - target: finalize
        condition: "outcome=success"
      - target: implement    # loops back on failure

  - id: finalize
    persona: navigator
    dependencies: [gate]
```

## Step Types

| Type | Purpose | Needs Persona? |
|------|---------|----------------|
| *(default)* | LLM persona execution | Yes |
| `command` | Shell script execution | No |
| `conditional` | Route based on outcome | No |
| `gate` | Pause for human approval | No |
| `pipeline` | Invoke sub-pipeline | No |

## Conditional Edges

Edges route execution based on conditions:

```yaml
edges:
  - target: success-step
    condition: "outcome=success"
  - target: fix-step        # fallback (no condition = default)
```

Supported conditions:
- `outcome=success` — previous step succeeded
- `outcome=failure` — previous step failed
- `context.key=value` — check a context variable

### Context Conditions

Context conditions check a variable set by the previous step. Use `context.key=value` to route based on values your steps produce:

```yaml
- id: run-tests
  type: command
  script: "go test ./..."
  output:
    context:
      tests_passed: "{{ .ExitCode == 0 }}"

- id: gate
  type: conditional
  dependencies: [run-tests]
  edges:
    - target: deploy
      condition: "context.tests_passed=true"
    - target: fix
      condition: "context.tests_passed=false"
```

## Safety: max_visits

Every step has a `max_visits` limit (default: 10). When reached, the pipeline fails:

```yaml
- id: fix
  persona: craftsman
  max_visits: 3           # fail after 3 fix attempts
  thread: impl            # keep conversation context
```

## Thread Continuity

Steps sharing a `thread:` name share conversation history. This is critical for fix loops — the fixer sees what was implemented and what failed:

```yaml
- id: implement
  thread: impl            # starts the thread

- id: fix
  thread: impl            # continues the conversation
  max_visits: 3
```

## Command Steps

Run shell commands without an LLM adapter:

```yaml
- id: run-tests
  type: command
  script: "go test ./... 2>&1 | tail -20"
```

Command steps are fast (milliseconds), deterministic, and don't consume tokens.

## Safety Mechanisms

Beyond per-step `max_visits` (documented above), Wave provides two additional safeguards for loops:

### Circuit Breaker

If a step fails with the same error 3 consecutive times, the loop is automatically terminated. This prevents infinite retries of unfixable errors (e.g., a missing dependency that no amount of code changes will resolve).

Error messages are normalized before comparison — variable parts like timestamps and line numbers are stripped so that semantically identical errors are recognized as repeats.

### Graph-Level `max_step_visits`

In addition to per-step `max_visits`, the pipeline enforces a graph-level `max_step_visits` aggregate limit across all steps. This prevents pathological cases where many steps each stay under their individual limits but the total execution count grows unboundedly.

Configure it at the pipeline level:

```yaml
max_step_visits: 50   # total visits across all steps
```

The effective limit is resolved via `EffectiveMaxStepVisits()`, which applies a default if not explicitly set.
