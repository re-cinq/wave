# feat: add second concurrent-pattern example pipeline and document concurrency verification

> Issue: [re-cinq/wave#117](https://github.com/re-cinq/wave/issues/117)
> Labels: `documentation`, `enhancement`, `pipeline`
> Author: nextlevelshit
> State: OPEN

## Problem / Motivation

Wave supports DAG-level concurrent step execution (shipped in e14d91b, v0.19.0+). The `code-review` pipeline already demonstrates the fan-out/fan-in pattern with `security-review` and `quality-review` running in parallel after `diff-analysis`.

However, only **one** pipeline showcases explicit concurrency. A second example demonstrating the **independent parallel tracks** pattern is still missing. Additionally, the README's pipeline section only shows a linear dependency example and does not mention concurrent execution, and there is no documented way to verify parallel execution via timestamps or audit logs.

## What's Already Done

- **DAG executor implemented** — The pipeline executor uses a ready-queue batch approach with `errgroup` to run steps concurrently when their dependencies are satisfied (commit `e14d91b`).
- **`code-review.yaml` demonstrates fan-out/fan-in** — `diff-analysis` fans out to `security-review` + `quality-review` (both depend only on `diff-analysis`), then `summary` fans in from both.
- **`docs/concepts/pipelines.md`** documents fan-out, fan-in, and convergence patterns with YAML examples and a Mermaid dependency diagram.
- **`docs/guides/enterprise.md`** mentions parallel step execution as a scaling pattern.

## Remaining Work

### 1. Second example pipeline: independent parallel tracks

Create a pipeline in `.wave/pipelines/` that demonstrates two or more **independent step sequences** running simultaneously and converging at a final step. For example:

```
step-1a (track-A-start)    step-1b (track-B-start)
       │                           │
step-2a (track-A-detail)   step-2b (track-B-detail)
       └──────────┬────────────────┘
              step-3 (merge)
```

This is distinct from the fan-out pattern in `code-review.yaml` because the parallel tracks have **no shared upstream step** — they are fully independent sequences that converge only at the end.

### 2. README update

The README's "Pipelines — DAG Workflows" section (line ~273) shows only a linear `navigate → implement → review` example. Update it to show the concurrent pattern, e.g.:

```yaml
steps:
  - id: analyze
    persona: navigator
  - id: security
    persona: auditor
    dependencies: [analyze]
  - id: quality
    persona: auditor
    dependencies: [analyze]
  - id: summary
    persona: summarizer
    dependencies: [security, quality]
```

### 3. Verification documentation

Document how users can verify that steps executed in parallel. Options:
- Per-step start/end timestamps in `wave status` or audit logs (`.wave/traces/`)
- The display already shows per-step elapsed timers when steps run concurrently
- Add a note in `docs/concepts/pipelines.md` or `docs/guides/pipeline-configuration.md` explaining how to confirm parallel execution

## Acceptance Criteria

- [x] At least one pipeline in `.wave/pipelines/` includes steps with explicit concurrency — `code-review.yaml`
- [ ] A second pipeline demonstrating the **independent parallel tracks** pattern
- [ ] Documentation entry for the second (independent tracks) pipeline pattern
- [ ] Running `wave run <pipeline>` executes concurrent steps in parallel — verification method documented
- [x] Existing sequential pipelines continue to pass their tests
- [ ] README updated to showcase the concurrent execution pattern

## Additional Notes

- Prioritise patterns that real users are likely to need (e.g. parallel analysis, parallel code generation).
- Keep examples minimal and well-commented so they serve as templates.
- The core engine work is done — this issue is now primarily about **examples and documentation**.
