# Human Approval Gates

Gate steps pause pipeline execution for human decisions. Reviewers can approve, revise, or abort — with optional freeform text feedback.

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
