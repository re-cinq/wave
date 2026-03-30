# Dead Code Report

Generated: 2026-03-30

## Summary

25 findings from automated scan + manual verification. 12 items cleaned, 6 remaining low-priority, 7 classified as "investigate" (not truly dead — unwired features that were subsequently wired).

## Cleaned (this session)

| Package | Item | Action |
|---------|------|--------|
| `continuous` | `GitHubSource.Sort`, `.Direction` | Removed — written but never read |
| `continuous` | `IterationResult.RunID`, `.Error`, `.Status` | Removed — write-only, events provide this |
| `continuous` | `IterationStatus` type + constants | Removed — only used by removed fields |
| `continuous` | `SourceConfig.RawURI` | Removed — write-only |
| `bench` | `LoadSWEBenchLite` | Removed — trivial wrapper around `LoadDataset` |
| `bench` | `RunConfig.CacheDir`, `.WaveBinary`, `.ClaudeBinary` | Removed — never configured |
| `bench` | `BenchTask.Version`, `.ExpectedPatch` | Removed — JSON-populated but never read |
| `bench` | `BenchResult.TokensUsed` | Removed — always zero |

## Remaining (low priority)

| Package | Item | Lines | Recommendation |
|---------|------|-------|----------------|
| `relay` | `ErrInvalidCheckpoint` | 1 | Remove — never returned |
| `relay` | `ErrWriteCheckpointFailed` | 1 | Fix — `Compact` wraps OS error directly instead of this sentinel |
| `relay` | `Checkpoint.Context` field | 1 | Remove — initialized but never populated |
| `relay` | `validateConfig` | 15 | Remove — only called from tests |
| `pipeline` | `composition_state.go` (entire file) | 80 | Remove — file-based state persistence unused; executor uses SQLite |
| `pipeline` | `CompositionExecutor` | 200 | Remove — legacy parallel implementation, superseded by `DefaultPipelineExecutor` methods |

## False Positives (PRs #680, #681 — closed)

Two automated PRs incorrectly identified active code as dead:
- `injectArtifacts` — called in `executor.go:2221`, `executor.go:3319`, `matrix.go:421`
- `MatrixWorkerContext` — used in `matrix.go:634`
- `ValidatePipelineSkills` — defined but potentially useful (validated skill refs)
- `DetectSubPipelineCycles` — defined but potentially useful (cycle detection)

These were correctly rejected to prevent build breakage.

## Wired (previously unwired)

| Feature | Package | Fix |
|---------|---------|-----|
| Relay context compaction | `relay` | `WithRelayMonitor` now called from `run.go` |
| Composition primitives | `pipeline` | iterate/aggregate/branch/loop dispatched in executor |
| Child run records | `pipeline` | `createRunID` creates `pipeline_run` records for sub-pipelines |
