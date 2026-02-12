# Implementation Plan: Pipeline ID Hash Suffix

## Objective

Add a unique hash suffix to pipeline runtime IDs to prevent state, workspace, and audit log collisions when the same pipeline runs concurrently or is re-run.

## Approach

Introduce a `GenerateRunID(pipelineName string, hashLength int) string` function that generates unique runtime IDs in the format `{name}-{hex_suffix}`. Modify the executor to use this generated ID instead of the raw `Metadata.Name` for all runtime operations, while preserving `Metadata.Name` as the logical pipeline name for display and filtering.

The key change is replacing the pattern `pipelineID := p.Metadata.Name` (used in `executor.go:130`, `executor_enhanced.go:46`) with `pipelineID := GenerateRunID(p.Metadata.Name, hashLength)`, and then threading this generated ID through all downstream code. Currently, many internal methods re-derive the pipeline ID via `execution.Pipeline.Metadata.Name` (e.g., `executeStep:264`, `executeMatrixStep:338`, `runStepExecution:363`, `createStepWorkspace:626`, `injectArtifacts:850`, `checkRelayCompaction:981`). These must be updated to use a stored runtime ID instead.

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `internal/pipeline/runid.go` | **create** | New file: `GenerateRunID` function and helpers |
| `internal/pipeline/runid_test.go` | **create** | Unit tests for run ID generation |
| `internal/pipeline/executor.go` | **modify** | Use `GenerateRunID` in `Execute()`; store runtime ID in `PipelineExecution`; update `executeStep`, `executeMatrixStep`, `runStepExecution`, `createStepWorkspace`, `injectArtifacts`, `checkRelayCompaction` to use stored runtime ID instead of `execution.Pipeline.Metadata.Name` |
| `internal/pipeline/executor_enhanced.go` | **modify** | Use `GenerateRunID` in `ExecuteWithValidation()`; update event emissions to use runtime ID |
| `internal/pipeline/resume.go` | **modify** | Update `ResumeFromStep`, `loadResumeState`, `executeResumedPipeline` to use runtime ID; accept suffixed ID for resume operations |
| `internal/pipeline/context.go` | **modify** | Add `PipelineName` field alongside existing `PipelineID`; update `ToTemplateVars` and `ResolvePlaceholders` |
| `internal/pipeline/types.go` | **modify** | Add `PipelineName` field to `PipelineStatus` |
| `internal/manifest/types.go` | **modify** | Add `PipelineIDHashLength int` to `Runtime` struct |
| `internal/pipeline/matrix.go` | **modify** | Update `executeWorker`, `createWorkerWorkspace`, `Execute` to use stored runtime ID from execution instead of `execution.Pipeline.Metadata.Name` |
| `internal/pipeline/executor_test.go` | **modify** | Update test assertions that compare exact pipeline IDs (e.g., `TestGetStatus`, `TestMemoryCleanupAfterCompletion`) to match suffixed format |
| `internal/pipeline/contract_integration_test.go` | **modify** | Update if it references exact pipeline IDs |

## Architecture Decisions

### AD-1: Random-only hash (not deterministic)

The issue suggests the hash could be "deterministic based on execution context". However, deterministic hashing based on input + timestamp would add complexity and still not guarantee uniqueness in concurrent scenarios with identical inputs. Using `crypto/rand` alone is simpler and provides strong uniqueness guarantees.

**Decision**: Use `crypto/rand` for the hash suffix with `time.Now().UnixNano()` as fallback.

### AD-2: Store runtime ID in PipelineExecution

Currently, `PipelineExecution` does not have a dedicated runtime ID field. Internal methods re-derive the pipeline ID from `execution.Pipeline.Metadata.Name` in at least 8 locations. Rather than adding a `RuntimeID` field to `PipelineExecution`, we will:

1. Generate the runtime ID at the `Execute()` entry point
2. Store it in `PipelineExecution.Status.ID` (already exists)
3. Update all internal methods that currently use `execution.Pipeline.Metadata.Name` to use `execution.Status.ID` instead

This requires changes at these call sites:
- `executor.go:264` (`executeStep`)
- `executor.go:338` (`executeMatrixStep`)
- `executor.go:363` (`runStepExecution`)
- `executor.go:626` (`createStepWorkspace`)
- `executor.go:850` (`injectArtifacts`)
- `executor.go:981` (`checkRelayCompaction`)
- `matrix.go:47,326,406` (MatrixExecutor methods)
- `resume.go` (multiple locations)

### AD-3: PipelineName in PipelineStatus

Add a `PipelineName` field to `PipelineStatus` to preserve the logical name for display:
- `PipelineStatus.ID` = runtime ID (e.g., `my-pipeline-a3b2c1d4`)
- `PipelineStatus.PipelineName` = logical name (e.g., `my-pipeline`)

### AD-4: Hash length configuration via manifest

The hash length will be configurable via `runtime.pipeline_id_hash_length` in the manifest's `Runtime` struct. Default is 8 hex chars (4 bytes), matching the issue specification.

### AD-5: No database migration needed

The `pipeline_state` table uses `pipeline_id TEXT PRIMARY KEY` with an UPSERT pattern. Since each run now produces a unique ID, there are no conflicts. No schema change is required.

### AD-6: Backward compatibility for resume

Resume operations already accept a `pipelineID` parameter. With suffixed IDs, users must pass the full suffixed ID (e.g., `my-pipeline-a3b2c1d4`) to resume. The `ListRecentPipelines` command already shows pipeline IDs, so users can copy-paste the correct one.

### AD-7: Executor option for hash length

Add a `WithPipelineIDHashLength(n int)` executor option to thread the configuration from the manifest through to `GenerateRunID`. This avoids passing the manifest config deep into the executor internals.

## Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| Many internal methods hardcode `execution.Pipeline.Metadata.Name` | High | Systematic replacement using `execution.Status.ID`; compiler will catch missing changes |
| Existing tests hardcode exact pipeline IDs (e.g., `TestGetStatus`, `TestProgressEventFields`) | Medium | Update test assertions to match on prefix or use regex; tests like `assert.Equal(t, "event-fields-test", completedEvent.PipelineID)` need updating |
| Resume requires full suffixed ID | Low | Document behavior; `list` command already shows full IDs |
| Workspace cleanup targets old ID format | Low | Cleanup now targets unique workspace per run — no collision |
| More rows in `pipeline_state` table | Low | Acceptable; runs are already expected to accumulate |
| Matrix executor copies `execution.Pipeline.Metadata.Name` in multiple places | Medium | Update `MatrixExecutor.Execute`, `executeWorker`, `createWorkerWorkspace` consistently |

## Testing Strategy

### Unit Tests (`internal/pipeline/runid_test.go`)
- `TestGenerateRunID_Format`: Verify output format `{name}-{hex}`
- `TestGenerateRunID_Length`: Verify configurable hash lengths (4, 8, 12, 16)
- `TestGenerateRunID_Uniqueness`: Generate 1000 IDs and verify all unique
- `TestGenerateRunID_DefaultLength`: Verify default of 8 when 0 passed
- `TestGenerateRunID_FallbackOnRandFailure`: Test fallback behavior

### Existing Test Updates
Key tests that assert exact pipeline IDs and need updating:
- `executor_test.go:598` - `assert.Equal(t, "event-fields-test", completedEvent.PipelineID)` → use `strings.HasPrefix`
- `executor_test.go:664` - `assert.Equal(t, "status-test", status.ID)` → use `strings.HasPrefix`
- `executor_test.go:900-908` - memory cleanup tests use exact IDs → use prefix matching
- `contract_integration_test.go` - review for exact ID assertions

### Integration Tests
- `TestExecutor_UniqueIDsPerRun`: Run same pipeline twice, verify different IDs in state store
- `TestExecutor_WorkspaceIsolation`: Verify each run creates its own workspace directory

### Regression Tests
- Run full `go test ./...` to catch regressions
- Run `go test -race ./...` for race condition validation
