# Gates

Gate steps pause pipeline execution. Wave supports four gate types:

- **approval** — Wait for human decision with choices (approve, revise, abort)
- **timer** — Pause for a fixed duration
- **pr_merge** — Poll until a GitHub PR is merged
- **ci_pass** — Wait for CI checks to pass on a branch

For approval gates, reviewers can approve, revise, or abort — with optional freeform text feedback.

## Basic Gate

```yaml
steps:
  - id: plan
    persona: navigator

  - id: approve
    gate:
      prompt: "Review the implementation plan"
      choices:
        - label: "Approve"
          key: "a"
          target: implement
        - label: "Revise"
          key: "r"
          target: plan          # loops back
        - label: "Abort"
          key: "q"
          target: _fail         # fails the pipeline
      freeform: true            # allow text input
      default: "a"              # used on timeout
      timeout: "1h"
    dependencies: [plan]

  - id: implement
    persona: craftsman
    dependencies: [approve]
```

## Timer Gate

Pause pipeline execution for a fixed duration:

```yaml
- id: cooldown
  gate:
    type: timer
    timeout: 5m
    message: "Cooling down before next phase"
```

## PR Merge Gate

Poll GitHub PR status until merged or closed:

```yaml
- id: wait-merge
  gate:
    type: pr_merge
    pr_number: 123
    # repo: owner/repo  # optional, auto-detected from git
    interval: 30s      # optional, default 30s
    timeout: 10m        # optional, default 10m
```

The gate resolves when the PR is merged. If the PR is closed without merging, the step fails.

## CI Pass Gate

Wait for CI checks to pass on a branch:

```yaml
- id: wait-ci
  gate:
    type: ci_pass
    branch: main        # optional, auto-detected from git
    # repo: owner/repo  # optional, auto-detected
    interval: 30s       # optional, default 30s
    timeout: 15m        # optional, default 10m
```

Polls the most recent CI run for the branch. Resolves when all checks pass or are skipped. Fails if any check fails or is cancelled.

## Interaction Channels

Gates work across all Wave interfaces:
- **CLI**: Keyboard shortcuts (`[A] Approve / [R] Revise / [Q] Abort`)
- **TUI**: Bubble Tea modal with selection
- **WebUI**: Button panel with freeform text input
- **API**: `POST /api/runs/:id/gates/:step/approve`

## Auto-Approve for CI

Skip gates in automated environments:

```bash
wave run --auto-approve impl-issue -- "..."
```

Uses the `default` choice for each gate.

## Gate Context

After a decision, downstream steps can access:
- `{{ gate.<step>.choice }}` — selected choice label
- `{{ gate.<step>.text }}` — freeform text (if provided)

```yaml
- id: implement
  exec:
    source: |
      Implement based on the plan.
      Reviewer feedback: {{ gate.approve.text }}
```

## Example Pipeline

See `.wave/pipelines/plan-approve-implement.yaml` for a complete working example.
