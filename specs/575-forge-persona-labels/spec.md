# fix(display): forge template variables not resolved in TUI persona labels

**Issue**: [#575](https://github.com/re-cinq/wave/issues/575)
**Labels**: bug, display
**Author**: nextlevelshit

## Bug Description

Forge template variables (e.g. `{{ forge.type }}`) are not resolved in the TUI/verbose progress display when showing persona names for pipeline steps. The raw template syntax `{{ forge.type }}-analyst` appears instead of the resolved value like `github-analyst`.

## Steps to Reproduce

1. Run any pipeline that uses forge-templated persona references:
   ```bash
   wave run ops-refresh "<issue-url>" --verbose
   ```
2. Observe the progress display output

## Actual Behavior

Persona labels in the progress display show unresolved template variables:

```
 ⠦ gather-context ({{ forge.type }}-analyst) [opus via claude] (15s)
 ○ draft-update ({{ forge.type }}-analyst)
 ○ apply-update ({{ forge.type }}-enhancer)
```

## Expected Behavior

Persona labels should show the resolved forge type:

```
 ⠦ gather-context (github-analyst) [opus via claude] (15s)
 ○ draft-update (github-analyst)
 ○ apply-update (github-enhancer)
```

## Root Cause Analysis

The bug has two contributing factors:

1. **Registration-time**: `CreateEmitter` in `cmd/wave/commands/output.go` calls `btpd.AddStep(step.ID, step.ID, step.Persona)` with the raw (unresolved) `step.Persona` from the pipeline definition. At this point, `{{ forge.type }}` has not yet been resolved. This causes "not started" steps to show unresolved template variables immediately.

2. **Event-time**: When the executor emits "started"/"running" events (in `executor.go:1213-1224`), it correctly includes the resolved persona name in `ev.Persona`. However, the display event handlers (`ProgressDisplay.EmitProgress`, `BubbleTeaProgressDisplay.updateFromEvent`) do not update the stored step's persona from these events. They only update state and message.

3. **WebUI**: The webui handlers (`handlers_runs.go`, `handlers_compose.go`) also read `step.Persona` directly from the pipeline definition without resolving template variables.

## Affected Pipelines

Many pipelines use `{{ forge.type }}` in persona references:
- `ops-refresh.yaml` — `{{ forge.type }}-analyst`, `{{ forge.type }}-enhancer`
- `ops-rewrite.yaml` — `{{ forge.type }}-analyst`, `{{ forge.type }}-enhancer`
- `ops-pr-review.yaml` — `{{ forge.type }}-commenter`
- `plan-scope.yaml` — `{{ forge.type }}-analyst`, `{{ forge.type }}-scoper`
- `plan-research.yaml` — `{{ forge.type }}-analyst`, `{{ forge.type }}-commenter`
- `impl-issue.yaml` — `{{ forge.type }}-commenter`
- And more

## Acceptance Criteria

- [ ] Forge template variables in persona names are resolved before being passed to the display layer
- [ ] All `{{ forge.* }}` variables render correctly in both `--verbose` CLI output and TUI mode
- [ ] Existing pipelines with hardcoded persona names are unaffected
- [ ] WebUI persona labels also show resolved values
