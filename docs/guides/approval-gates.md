# Human Approval Gates

Gates are blocking steps that pause pipeline execution until an external condition is met. They enable human review checkpoints, timed delays, CI verification, and PR merge awaits.

## Gate Types

| Type | Description | Blocks until |
|------|-------------|-------------|
| `approval` | Human review with choices | User selects a choice or timeout fires |
| `timer` | Timed delay | Duration elapses |
| `pr_merge` | PR merge check | PR is merged or closed |
| `ci_pass` | CI status check | CI run completes with success |

## Approval Gates with Choices

The most common gate type presents choices to a human reviewer. Each choice can route to a different step:

<div v-pre>

```yaml
steps:
  - id: plan
    persona: navigator
    exec:
      type: prompt
      source: "Create an implementation plan for: {{ input }}"
    output_artifacts:
      - name: plan
        path: .wave/output/plan.md
        type: markdown

  - id: approve-plan
    gate:
      type: approval
      prompt: "Review the implementation plan before proceeding"
      choices:
        - label: "Approve"
          key: "a"
          target: implement
        - label: "Revise"
          key: "r"
          target: plan
        - label: "Abort"
          key: "q"
          target: _fail
      freeform: true
      default: "a"
      timeout: "1h"
    dependencies: [plan]

  - id: implement
    persona: craftsman
    dependencies: [approve-plan]
    exec:
      type: prompt
      source: "Implement based on the approved plan"
```

</div>

### Choice Fields

| Field | Required | Description |
|-------|----------|-------------|
| `label` | Yes | Human-readable label displayed in the prompt |
| `key` | Yes | Keyboard shortcut (single character, must be unique) |
| `target` | No | Step ID to route to on selection. `_fail` aborts the pipeline |

### Gate Configuration

| Field | Default | Description |
|-------|---------|-------------|
| `prompt` | -- | Text displayed to the reviewer |
| `choices` | -- | Array of choice options |
| `freeform` | `false` | Allow optional text input after choice selection |
| `default` | -- | Choice key used on timeout or auto-approve |
| `timeout` | Manifest default | Duration before timeout fires (e.g., `"30m"`, `"2h"`) |
| `auto` | `false` | Skip human interaction (uses default choice) |

## Freeform Text Input

When `freeform: true` is set, the gate prompts for optional text after the choice is made. This text is available to downstream steps via template variables.

```
  Review the implementation plan before proceeding

  [a] Approve
  [r] Revise
  [q] Abort (abort)

  Choice: a
  Additional notes (press Enter to skip): Looks good, but add error handling for edge cases
```

## Gate Template Variables

After a gate resolves, its decision is available to downstream steps through template variables:

<div v-pre>

| Variable | Description |
|----------|-------------|
| `{{ gate.STEP_ID.choice }}` | Human-readable label of the selected choice |
| `{{ gate.STEP_ID.key }}` | Key of the selected choice |
| `{{ gate.STEP_ID.text }}` | Freeform text input (empty if not provided) |
| `{{ gate.STEP_ID.timestamp }}` | RFC 3339 timestamp of the decision |

</div>

Use these in downstream prompts to incorporate reviewer feedback:

<div v-pre>

```yaml
  - id: implement
    persona: craftsman
    dependencies: [approve-plan]
    exec:
      type: prompt
      source: |
        Implement the plan. The reviewer provided this feedback:
        {{ gate.approve-plan.text }}
```

</div>

## Timeout and Default Behavior

When a gate times out and a `default` choice is configured, the pipeline proceeds with that choice automatically. Without a default, timeout causes a pipeline failure.

```yaml
gate:
  type: approval
  prompt: "Approve deployment?"
  choices:
    - label: "Deploy"
      key: "d"
      target: deploy
    - label: "Cancel"
      key: "c"
      target: _fail
  default: "d"
  timeout: "30m"  # Auto-deploys after 30 minutes
```

## Timer Gates

Timer gates pause execution for a fixed duration. Use them for cooldown periods or rate limiting between steps:

```yaml
  - id: cooldown
    gate:
      type: timer
      timeout: "5m"
      message: "Waiting 5 minutes before proceeding..."
    dependencies: [deploy-staging]
```

## PR Merge Gates

PR merge gates poll the forge (GitHub, GitLab, Bitbucket) until a PR is merged:

```yaml
  - id: wait-for-merge
    gate:
      type: pr_merge
      pr_number: 42
      repo: "owner/repo"     # Optional -- detected from git remotes
      interval: "30s"        # Poll interval (default: 30s)
      timeout: "2h"          # Give up after 2 hours
      message: "Waiting for PR #42 to be merged..."
    dependencies: [create-pr]
```

If the PR is closed without merging, the gate fails immediately.

## CI Pass Gates

CI pass gates poll until the latest CI run on a branch succeeds:

```yaml
  - id: wait-for-ci
    gate:
      type: ci_pass
      branch: "feature/my-branch"  # Optional -- detected from git
      repo: "owner/repo"           # Optional -- detected from git remotes
      interval: "30s"
      timeout: "30m"
      message: "Waiting for CI to pass..."
    dependencies: [push]
```

The gate resolves on `success` or `skipped` conclusions and fails on `failure`, `cancelled`, or `timed_out`.

## Gate Handlers

Wave supports multiple interaction channels for approval gates:

| Handler | Context | Description |
|---------|---------|-------------|
| CLI | Terminal | Reads choices from stdin, writes to stderr |
| TUI | Bubble Tea UI | Modal dialog with keyboard navigation |
| WebUI | Browser dashboard | Web form with real-time SSE updates |
| Auto-approve | CI / `--auto-approve` | Uses default choice without interaction |

The handler is selected automatically based on the execution environment. Use `--auto-approve` for non-interactive environments like CI.

## Further Reading

- [Graph Loops](/guides/graph-loops) -- Edge-based routing for conditional flows
- [Pipeline Configuration](/guides/pipeline-configuration) -- Step configuration basics
- [Web Dashboard](/guides/web-dashboard) -- WebUI gate interaction
