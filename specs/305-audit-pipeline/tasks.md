# Tasks: Closed-Issue/PR Audit Pipeline (#305)

**Branch**: `305-audit-pipeline` | **Date**: 2026-03-11 | **Spec**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md)

## Phase 1: Setup — Contract Schemas

These tasks create the JSON Schema contracts that validate each pipeline step's output. All four schemas are independent and can be created in parallel.

- [X] T001 [P] [Setup] Copy `specs/305-audit-pipeline/contracts/audit-inventory.schema.json` to `.wave/contracts/audit-inventory.schema.json`
- [X] T002 [P] [Setup] Copy `specs/305-audit-pipeline/contracts/audit-findings.schema.json` to `.wave/contracts/audit-findings.schema.json`
- [X] T003 [P] [Setup] Copy `specs/305-audit-pipeline/contracts/audit-triage-report.schema.json` to `.wave/contracts/audit-triage-report.schema.json`
- [X] T004 [P] [Setup] Copy `specs/305-audit-pipeline/contracts/audit-publish-result.schema.json` to `.wave/contracts/audit-publish-result.schema.json`

## Phase 2: Foundational — Pipeline Definition

The pipeline YAML is the single deliverable for this feature. It depends on all contract schemas being in place (Phase 1).

- [X] T005 [US1] Create `.wave/pipelines/wave-audit.yaml` with pipeline metadata (kind, name, description, input schema) following the `doc-audit.yaml` pattern
- [X] T006 [US1] Add `collect-inventory` step to `wave-audit.yaml` — uses `github-analyst` persona, worktree workspace, prompt that fetches closed issues and merged PRs via `gh` CLI, filters out `not_planned` issues (FR-002, FR-003, FR-004, FR-005, FR-011), outputs `inventory.json`, contract validates against `audit-inventory.schema.json`
- [X] T007 [US1] Add `audit-items` step to `wave-audit.yaml` — uses `navigator` persona, depends on `collect-inventory`, injects inventory artifact, prompt instructs static analysis verification using Glob/Grep/Read and `git log` for revert detection (FR-006, FR-007), classifies items into 5 fidelity categories, outputs `audit-findings.json`, contract validates against `audit-findings.schema.json`
- [X] T008 [US1] Add `compose-triage` step to `wave-audit.yaml` — uses `navigator` persona, depends on `audit-items`, injects findings artifact, prompt instructs grouping by category, summary statistics, and prioritized action list (FR-008, FR-009), outputs `triage-report.json`, contract validates against `audit-triage-report.schema.json`
- [X] T009 [US3] Add `publish` step to `wave-audit.yaml` — uses `craftsman` persona, depends on `compose-triage`, injects triage-report artifact, prompt instructs creating GitHub issues for partial/regressed findings via `gh issue create` (FR-015), outputs `publish-result.json`, contract validates against `audit-publish-result.schema.json`, includes outcomes section for issue URL extraction

## Phase 3: User Story 2 — Scoped Audit Support (P2)

Scope parsing is handled entirely within the `collect-inventory` step prompt (per spec C3). These tasks refine the prompt to handle time-range and label filters.

- [X] T010 [US2] Enhance `collect-inventory` prompt in `wave-audit.yaml` to parse CLI input for time-range expressions ("last N days", "since YYYY-MM-DD") and translate to `gh` search queries using `closed:>YYYY-MM-DD` / `merged:>YYYY-MM-DD` syntax (FR-010)
- [X] T011 [US2] Enhance `collect-inventory` prompt in `wave-audit.yaml` to parse CLI input for label filters ("label:X") and translate to `gh --label X` flag (FR-010)
- [X] T012 [US2] Ensure `collect-inventory` prompt handles empty scope gracefully — when no matching items are found, output a valid inventory JSON with an empty items array and a summary indicating zero results

## Phase 4: User Story 3 — Actionable Remediation (P2)

These tasks refine the audit and triage prompts to produce evidence-based remediation details.

- [X] T013 [US3] Enhance `audit-items` prompt to include specific unmet acceptance criteria in findings for partial items — each finding must list which criteria passed and which did not (FR-009)
- [X] T014 [US3] Enhance `audit-items` prompt to include revert commit SHAs and affected file paths in findings for regressed items — use `git log --grep="Revert"` and `git log --all -- <file>` output as evidence (FR-009)
- [X] T015 [US3] Enhance `compose-triage` prompt to generate actionable `prioritized_actions` — each action must reference the original issue URL, describe the specific remediation, and rank by severity (regressed > partial with many unmet criteria > partial with few > unverifiable)

## Phase 5: User Story 4 — Resume Support (P3)

Resume leverages Wave's built-in `--from-step` capability. No special task needed beyond ensuring artifacts persist correctly.

- [X] T016 [US4] Verify that all 4 steps in `wave-audit.yaml` use `output_artifacts` with explicit paths so that artifacts are persisted for resume via `wave run wave-audit --from-step <step>` (FR-013)

## Phase 6: Polish & Cross-Cutting Concerns

- [X] T017 [P] Verify pipeline loads correctly by running `go test ./...` — confirms YAML parsing, persona references, and contract schema paths are valid
- [X] T018 [P] Review all 4 step prompts for edge case handling: issues with no linked PRs (unverifiable), issues referencing deleted files (obsolete), single issues spanning many files (summarize rather than exhaustively list), rate limit awareness in collect-inventory prompt (FR-012)
- [X] T019 Verify `wave-audit.yaml` pipeline step contracts all reference correct schema paths in `.wave/contracts/` and use `on_failure: retry` with `max_retries: 2` consistent with `doc-audit.yaml` pattern
