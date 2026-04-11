# Implementation Plan: Ontology Injection Fixes and Diff-Gate Contract

## Objective

Fix four gaps in Wave's ontology injection and contract validation system discovered during a pipeline post-mortem: add undefined-context warnings, implement a `source_diff` contract type, add a schema guard preventing `clarify` from being skipped when `missing_info` is non-empty, and verify/document context inheritance behavior.

## Approach

Four discrete, independent changes targeting three subsystems (audit logger, contract engine, JSON schema). Each change is self-contained with minimal coupling risk.

1. **ONTOLOGY_WARN** — Extend the audit logger interface and add a detection pass in the executor ontology injection path.
2. **source_diff contract** — Add a new validator file following the existing pattern (`internal/contract/*.go`). Register in the switch. Add ContractConfig fields for `glob`, `exclude`, `min_files`.
3. **Schema guard** — Add JSON Schema `if/then` conditional to `issue-assessment.schema.json`.
4. **Finding #1 verification** — Confirm `wave analyze --deep` already writes enriched SKILL.md (code at `analyze.go:717-741`). If stub content is still generated elsewhere, fix the gap.
5. **Documentation** — Add a note about context inheritance to AGENTS.md.

## File Mapping

### Create
- `internal/contract/source_diff.go` — New `sourceDiffValidator` struct + `Validate()` implementation
- `internal/contract/source_diff_test.go` — Unit tests for source_diff validator

### Modify
- `internal/audit/logger.go` — Add `LogOntologyWarn(pipelineID, stepID string, undefinedContexts []string) error` to interface and `TraceLogger` impl
- `internal/pipeline/executor.go` — In ontology injection path (near line 2920), detect undefined context names before calling `RenderMarkdown`; call `LogOntologyWarn` and emit `StateOntologyWarn` event for any missing contexts
- `internal/contract/contract.go` — Register `"source_diff"` in `NewValidator()` switch; add `Glob`, `Exclude`, `MinFiles` fields to `ContractConfig`
- `internal/defaults/contracts/issue-assessment.schema.json` — Add `if/then` constraint: if `missing_info` non-empty, `skip_steps` must not contain `"clarify"`
- `AGENTS.md` — Document context inheritance (no-`contexts:` field = inject all) and explain trace output difference

### Possibly Modify
- `internal/event/event.go` — Add `StateOntologyWarn` event state constant (if not already present)
- `.wave/pipelines/impl-issue.yaml` — Add inline comment explaining context inheritance behavior on the `plan` step

## Architecture Decisions

1. **`source_diff` uses `git diff`**: Run `git diff --name-only HEAD` (or `git diff --name-only --cached && git diff --name-only`) in the workspace dir. Parse output against glob and exclude patterns using `filepath.Match`. This is deterministic and requires only git (already a system dependency).

2. **ONTOLOGY_WARN does not block**: The warning is informational only. The step runs unconstrained (same as today). This matches the issue proposal and avoids breaking existing pipelines with legacy context references.

3. **Schema `if/then` guard**: JSON Schema draft-07 supports `if/then`. The guard: `if missing_info has minItems: 1, then skip_steps items must not equal "clarify"`. This is purely declarative; no Go code changes needed for this finding.

4. **ContractConfig extended minimally**: Add only the fields needed for `source_diff`: `Glob string`, `Exclude []string`, `MinFiles int`. These are optional and backward-compatible.

## Risks

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| `git diff` behavior varies across git versions | Low | Use `git diff --name-only HEAD` which is stable; fall back to `git status --short` if needed |
| JSON Schema `if/then` not supported by project's validator lib | Low | Check which JSON Schema library is used in `internal/contract/jsonschema.go`; draft-07 `if/then` is widely supported |
| Adding fields to `ContractConfig` breaks serialization | Low | All new fields are `omitempty`; existing YAML/JSON unmarshaling is additive |
| `StateOntologyWarn` event constant may need updating in TUI/webui | Medium | Check event consumers; add new state constant following existing pattern |

## Testing Strategy

- **`source_diff` validator**: Unit tests with a temporary git repo, staged/unstaged changes, glob matching, exclude patterns, and min_files thresholds.
- **ONTOLOGY_WARN**: Unit test executor ontology injection path with a mock logger; verify warn is called for undefined context, not called for defined contexts.
- **Schema guard**: Validate the schema against test fixtures: `{missing_info: ["x"], skip_steps: ["clarify"]}` must fail; `{missing_info: [], skip_steps: ["clarify"]}` must pass.
- Existing tests must continue to pass (`go test ./...`).
