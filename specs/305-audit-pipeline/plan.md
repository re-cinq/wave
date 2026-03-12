# Implementation Plan: Closed-Issue/PR Audit Pipeline (#305)

**Branch**: `305-audit-pipeline` | **Date**: 2026-03-11 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/305-audit-pipeline/spec.md`

## Summary

Build the `wave-audit` pipeline — a 4-step DAG that inventories all closed issues and merged PRs from a GitHub repository, audits each item against the current codebase via static analysis, produces a prioritized triage report, and optionally creates GitHub issues for fixable gaps. The pipeline uses existing personas (`github-analyst`, `navigator`, `craftsman`) with no Go code changes — it is entirely defined via YAML pipeline definition and JSON Schema contracts.

## Technical Context

**Language/Version**: Go 1.25+ (project language, but no Go code changes needed for this feature)
**Primary Dependencies**: `gh` CLI (GitHub CLI for API access), Wave pipeline executor (existing)
**Storage**: Filesystem artifacts (JSON files in `.wave/output/`)
**Testing**: `go test -race ./...` — validates pipeline YAML loads correctly via existing test infrastructure
**Target Platform**: Linux/macOS with `gh` CLI installed and authenticated
**Project Type**: Pipeline definition (YAML + JSON Schema contracts)
**Performance Goals**: Complete audit of 500 issues + 300 PRs within 90-minute step timeout (SC-001, SC-007)
**Constraints**: Read-only analysis (FR-015), static verification only (no test execution), existing personas only
**Scale/Scope**: 5 new files (1 pipeline YAML + 4 contract schemas), ~0 Go LOC

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

| Principle | Status | Notes |
|-----------|--------|-------|
| P1: Single Binary | PASS | No new runtime dependencies — uses existing `gh` CLI adapter prerequisite |
| P2: Manifest as Truth | PASS | Pipeline definition in `.wave/pipelines/`; personas declared in `wave.yaml` |
| P3: Persona-Scoped | PASS | Each step bound to exactly one persona |
| P4: Fresh Memory | PASS | No cross-step chat history; artifacts flow via `inject_artifacts` |
| P5: Navigator-First | DEVIATION | First step uses `github-analyst` instead of `navigator` — see Complexity Tracking |
| P6: Contracts at Handover | PASS | All 4 steps have `json_schema` contract validation |
| P7: Relay via Summarizer | PASS | Standard Wave relay applies if steps exceed context threshold |
| P8: Ephemeral Workspaces | PASS | All steps use `workspace.type: worktree` |
| P9: Credentials Never Disk | PASS | `gh` CLI uses OS keychain/env vars; no credentials in pipeline files |
| P10: Observable Progress | PASS | Standard Wave progress events emitted per step |
| P11: Bounded Recursion | N/A | Not a meta-pipeline |
| P12: Minimal State Machine | PASS | Standard 5-state transitions |
| P13: Test Ownership | PASS | `go test ./...` must pass after adding pipeline YAML |

## Project Structure

### Documentation (this feature)

```
specs/305-audit-pipeline/
├── plan.md              # This file
├── research.md          # Phase 0 research output
├── data-model.md        # Phase 1 data model output
├── contracts/           # Phase 1 contract schemas
│   ├── audit-inventory.schema.json
│   ├── audit-findings.schema.json
│   ├── audit-triage-report.schema.json
│   └── audit-publish-result.schema.json
└── tasks.md             # Phase 2 output (not created by /speckit.plan)
```

### Source Code (repository root)

```
.wave/
├── pipelines/
│   └── wave-audit.yaml                # NEW: Pipeline definition (4 steps)
├── contracts/
│   ├── audit-inventory.schema.json     # NEW: collect-inventory step contract
│   ├── audit-findings.schema.json      # NEW: audit-items step contract
│   ├── audit-triage-report.schema.json # NEW: compose-triage step contract
│   └── audit-publish-result.schema.json # NEW: publish step contract
```

**Structure Decision**: This is a pipeline-only feature. All deliverables are Wave pipeline primitives (YAML + JSON Schema) — no Go source code changes. The pipeline YAML goes in `.wave/pipelines/` following established convention. Contract schemas go in `.wave/contracts/` following the pattern of `doc-consistency-report.schema.json` and `doc-issue-result.schema.json`.

## Implementation Phases

### Phase A: Contract Schemas
**Files**: `.wave/contracts/audit-*.schema.json`

Copy the 4 contract schemas from `specs/305-audit-pipeline/contracts/` to `.wave/contracts/`:
1. `audit-inventory.schema.json` — validates inventory output (scope, items array, timestamp)
2. `audit-findings.schema.json` — validates audit findings (per-item classification, evidence, summary)
3. `audit-triage-report.schema.json` — validates triage report (metadata, summary counts, findings, prioritized actions)
4. `audit-publish-result.schema.json` — validates publish result (success, issues created, errors)

### Phase B: Pipeline Definition
**Files**: `.wave/pipelines/wave-audit.yaml`

Create the 4-step pipeline following the `doc-audit.yaml` pattern:

**Step 1: `collect-inventory`** (github-analyst)
- Parses CLI input for scope (time range, label filter, or full)
- Fetches closed issues via `gh issue list --state closed --json ...`
- Fetches merged PRs via `gh pr list --state merged --json ...`
- Filters out `not_planned` issues (FR-011)
- Extracts acceptance criteria from issue bodies
- Handles pagination for large repos (FR-005)
- Outputs `inventory.json` validated against `audit-inventory.schema.json`

**Step 2: `audit-items`** (navigator)
- Receives inventory artifact
- For each item, verifies against codebase at HEAD:
  - Uses Glob to check file existence
  - Uses Grep to find key functions/types referenced in issue
  - Uses Read to verify logic matches description
  - Uses `git log` to detect reverts and post-implementation changes
- Classifies each item into one of 5 fidelity categories (FR-007)
- Includes evidence and remediation for non-verified items (FR-009)
- Outputs `audit-findings.json` validated against `audit-findings.schema.json`

**Step 3: `compose-triage`** (navigator)
- Receives findings artifact
- Groups findings by fidelity category (FR-008)
- Generates summary statistics
- Builds prioritized action list (regressed > partial > unverifiable)
- Outputs `triage-report.json` validated against `audit-triage-report.schema.json`

**Step 4: `publish`** (craftsman)
- Receives triage report artifact
- For each partial/regressed finding with remediation:
  - Creates a GitHub issue via `gh issue create`
  - Links back to the original issue/PR
- Skips if all items are verified/obsolete
- Outputs `publish-result.json` validated against `audit-publish-result.schema.json`

### Phase C: Validation
- Run `go test ./...` to ensure pipeline YAML loads correctly
- Verify pipeline appears in `wave list` output
- Verify `wave validate` passes with the new pipeline

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|-----------|--------------------------------------|
| P5: First step uses `github-analyst` instead of `navigator` | The `collect-inventory` step must run `gh issue list` and `gh pr list` — the `navigator` persona only allows `Bash(git log*)` and `Bash(git status*)`, which cannot execute `gh` CLI commands | Adding a navigator step before collect-inventory would produce a generic codebase map artifact that provides minimal value — each inventory item references different files, so the audit step performs targeted codebase exploration per-item. The 5-step alternative adds latency and token cost for negligible benefit. The `github-analyst` persona is read-only (denies issue edit/create/close), satisfying FR-015's read-only requirement for analysis steps. |
