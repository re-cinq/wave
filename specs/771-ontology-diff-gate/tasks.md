# Tasks

## Phase 1: Verification

- [X] Task 1.1: Verify Finding #1 — Run `wave analyze --deep` (or review analyze.go:717-741) to confirm SKILL.md files are already populated with enriched content; if stub text persists, identify the gap and plan a fix

## Phase 2: Schema Guard (Finding #4)

- [X] Task 2.1: Update `internal/defaults/contracts/issue-assessment.schema.json` — Add JSON Schema draft-07 `if/then` constraint: if `assessment.missing_info` has `minItems: 1`, then `assessment.skip_steps` must not contain `"clarify"` [P]

## Phase 3: Audit Logger Extension (Finding #2)

- [X] Task 3.1: Add `LogOntologyWarn(pipelineID, stepID string, undefinedContexts []string) error` to `AuditLogger` interface in `internal/audit/logger.go` [P]
- [X] Task 3.2: Implement `LogOntologyWarn` on `TraceLogger` in `internal/audit/logger.go` — emit `[ONTOLOGY_WARN]` log line with step/pipeline/undefined-contexts fields [P]
- [X] Task 3.3: Add `StateOntologyWarn` event state constant in `internal/event/emitter.go` (or wherever event states are defined) [P]

## Phase 4: Executor Warning Logic (Finding #2, depends on Phase 3)

- [X] Task 4.1: In `internal/pipeline/executor.go` ontology injection path (near line 2920), before calling `RenderMarkdown`, build a set of defined context names from `execution.Manifest.Ontology.Contexts`
- [X] Task 4.2: Detect any `step.Contexts` entries not in the defined set; call `e.logger.LogOntologyWarn` and `e.emit(StateOntologyWarn)` for each undefined context name

## Phase 5: source_diff Contract (Finding #3)

- [X] Task 5.1: Add `Glob`, `Exclude []string`, and `MinFiles int` fields to `ContractConfig` in `internal/contract/contract.go` (with `omitempty` tags) [P]
- [X] Task 5.2: Create `internal/contract/source_diff.go` — implement `sourceDiffValidator` struct and `Validate(cfg ContractConfig, workspacePath string) error` using `git diff --name-only HEAD` piped through glob/exclude matching
- [X] Task 5.3: Register `"source_diff"` case in `NewValidator()` switch in `internal/contract/contract.go` (depends on 5.2)
- [X] Task 5.4: Create `internal/contract/source_diff_test.go` — unit tests covering: no diff (should fail with min_files=1), diff with matching file (should pass), diff with only excluded files (should fail), glob non-match (should fail)

## Phase 6: Documentation (Finding #5, Finding #6)

- [X] Task 6.1: Update `AGENTS.md` — add section explaining that a step with no `contexts:` field receives ALL ontology contexts (inherit-all behavior), with note on how this differs from explicit context injection in trace logs [P]
- [X] Task 6.2: Add inline comment to `.wave/pipelines/impl-issue.yaml` on the `plan` step explaining why it has no `contexts:` field (inherits all) [P]

## Phase 7: Testing and Validation

- [X] Task 7.1: Run `go test ./internal/contract/...` — all existing and new tests pass
- [X] Task 7.2: Run `go test ./internal/pipeline/...` — executor tests pass including new ONTOLOGY_WARN behavior
- [X] Task 7.3: Run `go test ./...` — full test suite passes
- [X] Task 7.4: Validate updated `issue-assessment.schema.json` against passing and failing fixture JSON to confirm the guard works
