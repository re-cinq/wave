# bug: CLI --adapter flag not inherited by sub-pipelines

**Issue**: re-cinq/wave#768
**URL**: https://github.com/re-cinq/wave/issues/768
**Author**: nextlevelshit
**State**: OPEN
**Labels**: bug, priority:medium

## Description

When running a pipeline with `wave run --adapter opencode`, the adapter override only applies to the parent pipeline. Sub-pipelines (referenced via `pipeline:` in the YAML) still use the adapter defined in their persona definitions in wave.yaml (default: claude).

## Steps to Reproduce

1. Run a composition pipeline: `wave run full-impl-cycle --adapter opencode`
2. Observe that impl-issue-core, test-gen, audit-* sub-pipelines still use claude adapter
3. The parent uses opencode, but all sub-pipelines use their wave.yaml-defined adapter (claude)

## Expected Behavior

The `--adapter` flag from the CLI should be inherited by all sub-pipelines unless explicitly overridden at the step level in the pipeline YAML.

## Root Cause

In `executor.go`, `runNamedSubPipeline` doesn't receive or use the parent's adapter override. It resolves adapters via persona definitions from wave.yaml, not from the CLI context.

Specifically, in `runNamedSubPipeline` (line ~5077), when building `childOpts`, `modelOverride` is propagated (line ~5086) but `adapterOverride` is not.

## Workaround

Manually add `adapter: opencode` to each step in the pipeline YAML.

## Acceptance Criteria

- `--adapter` CLI flag propagates to all sub-pipelines spawned via `runNamedSubPipeline`
- Step-level `adapter:` in pipeline YAML still takes precedence over CLI flag (existing resolution order preserved)
- Existing tests pass; new test covers the inheritance behavior
