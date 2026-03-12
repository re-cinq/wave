# feat: Add --preserve-workspace flag for debugging

**Issue**: [#27](https://github.com/re-cinq/wave/issues/27)
**Labels**: enhancement, good first issue, priority: low
**Author**: nextlevelshit

## Context

From Copilot review on PR #26: workspace cleaning at pipeline start could break debugging workflows. When debugging a failed pipeline step, the workspace is destroyed before the developer can inspect intermediate artifacts, logs, or agent outputs.

## Current Behavior

Pipeline workspace is cleaned (`os.RemoveAll`) at the start of each run to ensure fresh state. This happens in the workspace setup phase before step execution begins.

## Proposed Change

Add `--preserve-workspace` flag to `wave run` command to skip workspace cleaning for debugging purposes.

```bash
wave run --preserve-workspace my-pipeline
```

When set, the workspace directory from the previous run is kept intact. The pipeline still runs but reuses the existing workspace rather than starting from a clean state.

## Implementation Notes

- **Flag registration**: Add `--preserve-workspace` bool flag in `cmd/wave/commands/run.go` via Cobra
- **Workspace cleanup skip**: The `RemoveAll` call in `internal/pipeline/executor.go` (line ~318) should be gated on this flag
- **Interaction with `--from-step`**: These flags are complementary — `--from-step` resumes from a specific step, `--preserve-workspace` keeps the filesystem state. Both can be used together for debugging a specific step with its prior outputs intact
- **Warning output**: When `--preserve-workspace` is active, emit a warning that stale workspace state may cause non-reproducible results

## Acceptance Criteria

- [ ] Add `--preserve-workspace` bool flag to `wave run` command
- [ ] Skip `RemoveAll` when flag is set
- [ ] Emit warning when flag is active about potential stale state
- [ ] Document flag in `--help` text with usage example
- [ ] Works correctly in combination with `--from-step`
- [ ] Unit test for flag parsing and workspace preservation logic
