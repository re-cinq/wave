# Implementation Plan: CLI --adapter flag inheritance for sub-pipelines

## Objective

Propagate the CLI `--adapter` override flag from parent to child executors created in `runNamedSubPipeline`, matching the existing behavior of `--model` override propagation.

## Approach

Single-line fix: add `WithAdapterOverride` propagation in `runNamedSubPipeline` alongside the existing `WithModelOverride` propagation. The `WithAdapterOverride` option already exists (`executor.go:167`); it just isn't used when constructing child executors from named sub-pipeline calls.

The existing adapter resolution priority (CLI > step-level > persona-level) is already correctly implemented in `resolveAdapterAndRun` — the fix only ensures the parent's `adapterOverride` value reaches the child executor.

## File Mapping

| File | Action | Change |
|------|--------|--------|
| `internal/pipeline/executor.go` | modify | Add `adapterOverride` propagation in `runNamedSubPipeline` childOpts block (~line 5086) |
| `internal/pipeline/subpipeline_test.go` | modify | Add test for adapter override inheritance |

## Architecture Decisions

- **No signature change**: The fix is entirely internal to `runNamedSubPipeline`; no public API changes.
- **Consistent with model override**: Mirrors the exact pattern used for `modelOverride` propagation.
- **`NewChildExecutor` already correct**: `NewChildExecutor` at line 296 already copies `adapterOverride` — only `runNamedSubPipeline`'s dynamic `childOpts` path is missing it.

## Risks

- **Low risk**: Purely additive — adds a missing option that is already supported everywhere else.
- **Existing tests**: No existing tests assert the broken behavior, so no test rewrites needed.

## Testing Strategy

Add a test in `subpipeline_test.go` that:
1. Creates a parent executor with `WithAdapterOverride("opencode")`
2. Runs a pipeline that includes a sub-pipeline step
3. Verifies the child executor receives the adapter override (via a stub/spy adapter runner)

Alternatively, verify via the `resolvedAdapterName` logic path by checking which adapter binary is invoked in the child.
