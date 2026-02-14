# Implementation Plan: Restore `wave meta` Dynamic Pipeline Generation

## Objective

Fix the broken `wave meta` command so it correctly reads adapter output, generates valid pipelines via the philosopher persona, and executes them end-to-end. The primary bug is that `invokePhilosopherWithSchemas()` reads raw NDJSON from `result.Stdout` instead of the parsed `result.ResultContent`.

## Approach

The fix is surgical: change how the meta executor reads adapter output, add a proper mock output generator for the philosopher persona in meta-pipeline context, and add tests to prevent regression.

### Strategy

1. **Fix the output reading** — Use `result.ResultContent` as primary source, fall back to `io.ReadAll(result.Stdout)` when empty
2. **Fix mock adapter** — Add a `generateMetaPhilosopherOutput()` function that returns valid `--- PIPELINE ---` / `--- SCHEMAS ---` formatted output
3. **Add integration tests** — Test the full `--mock` flow from command through execution
4. **Verify existing tests pass** — No regressions

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `internal/pipeline/meta.go` | **modify** | Fix `invokePhilosopherWithSchemas()` to use `ResultContent` instead of raw `Stdout` |
| `internal/adapter/mock.go` | **modify** | Add `generateMetaPhilosopherOutput()` for valid meta-pipeline mock output |
| `internal/pipeline/meta_test.go` | **modify** | Add test verifying `ResultContent` is preferred over `Stdout`; update `mockMetaRunner` to set `ResultContent` |
| `cmd/wave/commands/meta_test.go` | **modify** | Add integration test for `--mock --dry-run` flow |

## Architecture Decisions

### AD-1: Use `ResultContent` as primary output source

**Decision**: Read from `result.ResultContent` first, fall back to `io.ReadAll(result.Stdout)`.

**Rationale**: The `ClaudeAdapter` already parses the NDJSON stream and extracts the final result content into `ResultContent`. The meta executor should consume this parsed content rather than re-reading the raw stream. The fallback ensures backward compatibility with adapters that don't set `ResultContent`.

### AD-2: Detect meta-pipeline context in mock adapter by workspace path

**Decision**: In `generateRealisticOutput()`, check for `meta-philosopher` in the workspace path to trigger meta-specific output.

**Rationale**: The meta executor already sets `WorkspacePath` to `.wave/workspaces/meta-philosopher`, making it a reliable signal. This follows the existing pattern in the mock adapter (e.g., `github-issue-impl` detection).

### AD-3: Keep mock pipeline output minimal and valid

**Decision**: The mock philosopher output will generate a 2-step pipeline (navigator + implementer) with proper `--- PIPELINE ---` / `--- SCHEMAS ---` sections.

**Rationale**: The mock output needs to pass `ValidateGeneratedPipeline()` which requires: first step is navigator, all steps have fresh memory, all steps have handover contracts. A minimal 2-step pipeline is sufficient for testing.

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Existing tests break due to `ResultContent` change | Low | Medium | The `mockMetaRunner` in tests returns responses via `Stdout` (as `io.NopCloser`). The fallback to `io.ReadAll(Stdout)` when `ResultContent` is empty preserves this behavior. |
| Mock pipeline output doesn't match what real philosopher generates | Medium | Low | The mock output follows the exact same format that `extractPipelineAndSchemas()` expects. Tests verify parsing. |
| Schema file validation fails because mock schemas aren't written to disk | Medium | Medium | The mock output includes schema paths, and `saveSchemaFiles()` writes them. The `ValidateGeneratedPipeline()` checks schema files exist on disk. We need to ensure the mock flow writes schemas before validation. |

## Testing Strategy

### Unit Tests (`internal/pipeline/meta_test.go`)
- Test that `invokePhilosopherWithSchemas()` prefers `ResultContent` over `Stdout`
- Test that fallback to `Stdout` works when `ResultContent` is empty
- Update existing `mockMetaRunner` tests to remain compatible

### Integration Tests (`cmd/wave/commands/meta_test.go`)
- Test `runMeta()` with `--mock --dry-run` produces valid pipeline output
- Test `runMeta()` with `--mock` executes without error (may need to mock the child executor)

### Existing Tests
- All existing tests in `internal/pipeline/meta_test.go` must continue to pass
- All existing tests in `cmd/wave/commands/meta_test.go` must continue to pass
- Run full test suite: `go test ./internal/pipeline/... ./cmd/wave/commands/... ./internal/adapter/...`
