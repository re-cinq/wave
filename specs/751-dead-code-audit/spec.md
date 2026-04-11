# chore: dead code report (28 findings)

**Issue:** [re-cinq/wave#751](https://github.com/re-cinq/wave/issues/751)
**Labels:** code-quality
**Author:** nextlevelshit
**Scan date:** 2026-04-09T10:50:01Z

## Summary

Dead code audit of `internal/pipeline/` identified 28 findings across 4 types:

| Type | Count |
|------|-------|
| dead-code | 21 |
| duplicate | 4 |
| junk-code | 2 |
| dx | 1 |

| Action | Count |
|--------|-------|
| refactor | 15 |
| remove | 11 |
| merge | 1 |
| fix | 1 |

| Severity | Count |
|----------|-------|
| high | 6 |
| medium | 16 |
| low | 6 |

## Findings

### High Severity (6)

- **DC-001** `prototype_spec_test.go` -- 3 broken tests reading non-existent fixture. Remove.
- **DC-002** `prototype_docs_test.go` -- 5 broken tests, hidden behind `//go:build integration`. Remove.
- **DC-003** `prototype_dummy_test.go` -- 6 broken tests, hidden behind integration tag. Remove.
- **DC-004** `prototype_implement_test.go` -- 7 broken tests, hidden behind integration tag. Remove.
- **DC-005** `prototype_e2e_test.go` -- 5 broken tests, swallows errors. Remove.
- **DC-006** `executor.go` -- stdlib reimplementation (custom indexOf vs `strings.Index`). Remove.

### Medium Severity (16)

- **DC-007** `.wave/contracts/mock-analysis.schema.json` -- Orphaned fixture, schema embedded in meta.go. Remove.
- **DC-008** `.wave/contracts/mock-result.schema.json` -- Orphaned fixture, schema embedded in meta.go. Remove.
- **DC-009** 3x `copyFile` implementations across subpipeline.go, skill.go, workspace.go. Merge.
- **DC-010** Duplicated pipeline defaults logic in meta.go vs dag.go. Deduplicate.
- **DC-011** `adhoc.go:10` -- `DefaultNavigatorPersona` exported but internal-only. Unexport.
- **DC-012** `chatworkspace.go:21` -- `ChatModeAnalysis`/`ChatModeManipulate` exported but internal-only. Unexport.
- **DC-013** `types.go:16` -- 7 re-exported state constants (redundant with `state` package). Remove.
- **DC-014** `meta.go:24` -- 2 persona constants exported but internal-only. Unexport.
- **DC-015** `meta.go:20` -- 3 config constants exported but internal-only. Unexport.
- **DC-016** `executor.go:207` -- 2 ExecutorOption funcs exported but internal-only. Unexport.
- **DC-017** `executor.go:189` -- `WithPreserveWorkspace` exported but internal-only. Unexport.
- **DC-018** `dag.go:363` -- 3 validation funcs exported but internal-only. Unexport.
- **DC-019** `executor.go:5562` -- Exported error type, internal-only. Unexport.
- **DC-020** `errors.go:23` -- `GateAbortError` exported, internal-only. Unexport.
- **DC-021** `outcomes.go:13` -- `EmptyArrayError` exported, internal-only. Unexport.
- **DC-022** `matrix.go:1172` -- `StackedBaseBranchFromContext` exported, internal-only. Unexport.

### Low Severity (6)

- **DC-023** Duplicate test bodies across prototype files. Moot after DC-001..005.
- **DC-024** Duplicate `findPipelineStep` helper in prototype tests. Moot after DC-001..005.
- **DC-025** `subpipeline.go:187` -- `SubPipelineDefaultTimeout` exported, internal-only. Unexport.
- **DC-026** `sequence.go:18` -- `ErrParallelStagePartialFailure` exported, internal-only. Unexport.
- **DC-027** 7 near-identical `emit()` methods. Consolidate via embedded helper.
- **DC-028** `go.mod:29` -- 4 indirect deps actually directly imported. Run `go mod tidy`.

## Acceptance Criteria

1. All 5 prototype test files removed
2. Both orphaned schema fixtures removed
3. stdlib reimplementation replaced with `strings.Index`
4. 3 `copyFile` implementations consolidated into one shared utility
5. Pipeline defaults logic deduplicated between meta.go and dag.go
6. All internal-only exported symbols unexported
7. Redundant state constant re-exports removed
8. 7 `emit()` methods consolidated via embedded helper struct
9. `go mod tidy` run to fix dep classifications
10. `go test ./...` passes
11. `golangci-lint run ./...` passes
