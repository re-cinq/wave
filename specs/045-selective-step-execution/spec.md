# feat(cli): add --steps and -x flags for selective step execution

> Issue: [#45](https://github.com/re-cinq/wave/issues/45)
> Author: nextlevelshit
> State: OPEN
> Complexity: medium

## Summary

Add `--steps` and `-x` (`--exclude`) flags to `wave run` for selective step execution. These follow established CLI conventions (Gradle `-x`, Maven `--projects`, Nx `--exclude`) rather than inventing new patterns.

### New Flags

| Flag | Long form | Purpose | Example |
|------|-----------|---------|---------|
| `--steps` | `--steps` | Run only the named steps | `wave run --steps clarify,plan speckit-flow` |
| `-x` | `--exclude` | Skip the named steps | `wave run -x implement,create-pr speckit-flow` |

### Existing Flags (unchanged)

| Flag | Purpose |
|------|---------|
| `--from-step` | Resume from a step (runs it and everything after) |
| `--force` | Skip validation when using `--from-step` |

## Motivation

Long pipelines (e.g. speckit-flow with 8 steps) are expensive to run end-to-end during development. Current `--from-step` only supports "resume from here to end". Common needs:

- **Run a specific step**: `wave run --steps plan speckit-flow`
- **Skip expensive steps**: `wave run -x implement,create-pr speckit-flow`
- **Run a subset**: `wave run --steps clarify,plan,tasks speckit-flow`
- **Combine with resume**: `wave run --from-step clarify -x create-pr speckit-flow`

## Design

### Convention Alignment

| Convention | Precedent | Wave equivalent |
|------------|-----------|-----------------|
| Inclusion filter | Maven `--projects`, Turborepo `--filter` | `--steps` |
| Exclusion filter | Gradle `-x`, Nx `--exclude` | `-x` / `--exclude` |
| Resume-from | Maven `-rf`, Ansible `--start-at-task` | `--from-step` (existing) |

### Combination Rules

- `--steps` and `-x` are **mutually exclusive** — error if both provided
- `--from-step` + `-x` is valid: resume from a step, skip specific later steps
- `--from-step` + `--steps` is invalid: conflicting semantics
- Both accept **comma-separated step names** (no numeric indices)

### Artifact Handling

- When steps are skipped, artifact injection reads from existing workspace outputs (same as `--from-step` behavior)
- If a step depends on a skipped step and no prior workspace artifacts exist, fail with a clear error listing the missing artifacts
- `--dry-run` should show which steps will run and which will be skipped, including artifact availability warnings

## Acceptance Criteria

- [ ] `wave run --steps step1,step2 <pipeline>` runs only the named steps
- [ ] `wave run -x step1,step2 <pipeline>` / `wave run --exclude step1,step2 <pipeline>` skips the named steps
- [ ] `--steps` and `-x` are mutually exclusive (clear error if both provided)
- [ ] `--from-step` + `-x` combine correctly (resume, then exclude specific steps)
- [ ] `--from-step` + `--steps` is rejected with a clear error
- [ ] Invalid step names produce clear errors listing available steps
- [ ] Skipped steps with missing workspace artifacts produce clear errors
- [ ] `--dry-run` shows the execution plan including skip/include status
- [ ] Existing `--from-step` behavior is preserved unchanged
- [ ] Unit tests for all flag combinations and error cases
