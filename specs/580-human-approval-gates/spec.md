# feat: human approval gates for plan-approve-implement workflows

**Issue**: [re-cinq/wave#580](https://github.com/re-cinq/wave/issues/580)
**Labels**: enhancement
**Author**: nextlevelshit
**Complexity**: complex

## Context

Wave currently has no human-in-the-loop mechanism. Pipelines run unattended. For Wave's use as a development tool (not just CI), human gates enable supervised workflows where the developer stays in control at critical decision points.

The existing `GateConfig` (`internal/pipeline/types.go`) supports four gate types: `approval`, `timer`, `pr_merge`, `ci_pass`. The `approval` type only supports auto-approve or timeout-based expiry. There is no interactive prompt, no choice routing, no freeform text, and no multi-channel interaction.

## Design Goals

Evolve the existing gate system into a full human-in-the-loop mechanism with:

### Gate Step Type (Enhanced)

```yaml
steps:
  - name: plan
    persona: navigator

  - name: approve-plan
    type: gate
    depends_on: [plan]
    prompt: "Review the implementation plan"
    choices:
      - label: "Approve"
        key: "a"
        target: implement
      - label: "Revise"
        key: "r"
        target: plan
      - label: "Abort"
        key: "q"
        target: _fail     # special: fail the pipeline
    timeout: 3600s         # 1 hour before auto-timeout
    default: approve       # on timeout, use this choice

  - name: implement
    persona: craftsman
    depends_on: [approve-plan]
```

### Interaction Channels

1. **CLI** -- prompt in terminal with keyboard shortcuts
2. **TUI** -- Bubble Tea modal with choice buttons
3. **Web UI** -- browser notification + button panel
4. **API** -- REST endpoint for programmatic approval (enables Slack/webhook integration)

### Freeform Input

Gates can accept freeform text (revision notes, additional instructions):

```yaml
  - name: approve-plan
    type: gate
    freeform: true          # allow text input alongside choices
```

The freeform text becomes available as an artifact for downstream steps.

### Auto-Approve Mode

For CI/automated runs: `wave run --auto-approve impl-issue -- "..."` skips all gates, using default choices.

### Context Population

After a gate decision, the following are set in run context:
- `gate.<step_name>.choice` -- selected choice label
- `gate.<step_name>.text` -- freeform text (if any)
- `gate.<step_name>.timestamp` -- when decision was made

## What Wave Keeps

- Persona system (gates don't need personas -- they're human interaction points)
- Contract validation (can still validate before presenting to human)
- Workspace isolation (gate doesn't modify workspace)

## What Wave Gains

- **Supervised workflows** -- human stays in control at critical points
- **Plan review** -- navigator creates plan, human approves, craftsman implements
- **Quality gates** -- human sign-off before PR creation
- **Revision loops** -- rejected work loops back with human feedback

## Acceptance Criteria

1. `type: gate` step type with `choices`, `freeform`, `default`, `timeout` fields in manifest schema
2. Gate executor pauses execution and waits for human input via an interaction channel
3. CLI handler prompts in terminal with keyboard shortcuts for each choice
4. TUI handler renders Bubble Tea modal with choice buttons
5. WebUI handler exposes REST endpoint for gate approval
6. `--auto-approve` flag on `wave run` skips all gates using default choices
7. Gate context (`gate.<step>.choice`, `.text`, `.timestamp`) available to downstream steps
8. Freeform text captured as artifact for downstream injection
9. Choice routing: `target: _fail` fails pipeline, `target: <step>` re-queues that step
10. Existing gate types (`timer`, `pr_merge`, `ci_pass`) continue working unchanged
