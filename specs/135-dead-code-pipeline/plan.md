# Implementation Plan: Improved Dead-Code Detection Pipeline

## Objective

Rewrite the dead-code detection pipeline to produce structured, actionable JSON output with expanded detection categories, and support three output modes: PR comment, GitHub issue creation, and local JSON report.

## Approach

The implementation follows a **pipeline-first, schema-driven** strategy:

1. **Expand the contract schema** to support new detection categories and fields
2. **Rewrite the scan step** with enhanced detection prompts covering all required categories
3. **Add a format step** that transforms raw scan results into human-readable reports
4. **Add conditional output steps** for PR comment and GitHub issue modes
5. **Split into multiple pipeline YAML files** — one per output mode — sharing the scan step via a common base pattern
6. **Preserve the existing clean/verify/create-pr flow** as the `dead-code-heal.yaml` pipeline (separate concern)

### Architecture Decision: Multiple Pipelines vs. Single Pipeline with Conditionals

Wave pipelines do not currently support conditional step execution based on runtime parameters. Rather than adding conditional logic to the executor, the cleaner approach is:

- `dead-code.yaml` — Core scan + local JSON output (default)
- `dead-code-pr.yaml` — Scan + format + post PR comment
- `dead-code-issue.yaml` — Scan + format + create GitHub issue

All three share the same scan step definition and contract schema. The existing `dead-code.yaml` clean/verify/create-pr steps move to a separate `dead-code-heal.yaml` pipeline that consumes the scan output as input.

## File Mapping

### Files to Modify

| File | Action | Description |
|------|--------|-------------|
| `.wave/contracts/dead-code-scan.schema.json` | modify | Add new categories, `line_range`, `suggested_action` fields |
| `.wave/pipelines/dead-code.yaml` | modify | Rewrite scan step with enhanced prompts; remove clean/verify/create-pr steps |

### Files to Create

| File | Action | Description |
|------|--------|-------------|
| `.wave/pipelines/dead-code-pr.yaml` | create | Pipeline: scan + format + post PR comment |
| `.wave/pipelines/dead-code-issue.yaml` | create | Pipeline: scan + format + create GitHub issue |
| `.wave/pipelines/dead-code-heal.yaml` | create | Pipeline: consume scan results + clean + verify + create PR (moved from current dead-code.yaml) |
| `.wave/contracts/dead-code-report.schema.json` | create | Schema for the formatted markdown report output |
| `.wave/contracts/dead-code-pr-result.schema.json` | create | Schema for PR comment result |
| `.wave/contracts/dead-code-issue-result.schema.json` | create | Schema for GitHub issue creation result |

### No Go Code Changes Required

The issue is entirely about pipeline YAML definitions and contract schemas. All detection logic runs inside the AI persona prompts, and the output modes use existing personas (`navigator`, `summarizer`, `github-commenter`) with `gh` CLI for GitHub interactions. The `internal/github` Go client is not directly invoked by pipelines — pipelines use `gh` CLI via Bash tool permissions.

## Architecture Decisions

### AD-1: Separate pipelines per output mode
**Decision**: Create three pipeline files instead of one with conditionals.
**Rationale**: Wave has no conditional step execution. Separate pipelines are cleaner, more composable, and each can be invoked independently. Users pick the mode by choosing which pipeline to run.

### AD-2: Preserve heal flow separately
**Decision**: Move clean/verify/create-pr to `dead-code-heal.yaml`.
**Rationale**: The issue explicitly states "heal-code pipeline is out of scope" but the existing steps should not be lost. Moving them to a separate pipeline maintains the capability while delineating scope.

### AD-3: Enhanced schema with backward compatibility
**Decision**: Add new enum values and optional fields to the existing schema rather than creating a new one.
**Rationale**: The scan step always produces output matching `dead-code-scan.schema.json`. Adding new enum values to `type` and new optional fields preserves backward compatibility with any existing consumers.

### AD-4: Use gh CLI for GitHub interactions
**Decision**: Pipeline steps use `Bash(gh ...)` for posting PR comments and creating issues.
**Rationale**: This is the established pattern in the codebase (see `code-review.yaml` publish step). The `github-commenter` persona already has `Bash(gh issue comment*)` permissions.

## Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| Scan prompt too broad → low-quality detections | Medium | Structured prompt with verification steps; confidence scoring |
| `gh` CLI not available in pipeline execution environment | High | Pipeline `requires.tools` declaration; preflight check |
| Schema changes break existing consumers | Low | Only additive changes (new enum values, new optional fields) |
| PR comment too long for GitHub API limits | Low | Format step truncates and summarizes; link to full report |
| GitHub-commenter persona lacks permissions for `gh pr comment` | Medium | Extend persona permissions or create new persona variant |

## Testing Strategy

### Contract Schema Tests
- Validate the updated `dead-code-scan.schema.json` with sample payloads covering all new categories
- Validate new schemas (`dead-code-report`, `dead-code-pr-result`, `dead-code-issue-result`)
- Ensure backward compatibility: existing valid payloads still validate

### Pipeline YAML Validation
- Parse all new/modified pipeline YAML files with the existing pipeline loader
- Verify step dependencies resolve correctly
- Verify persona references exist in `wave.yaml`

### Integration Test Considerations
- The scan step is AI-driven, so deterministic testing is not possible for detection quality
- Test the format step with mock scan output to verify report generation
- Test PR comment and issue creation steps with mock `gh` output

### Acceptance Test Mapping
- AC1 (valid JSON output) → Schema validation test
- AC2 (category detection) → Covered by scan prompt enhancement + schema enum expansion
- AC3 (PR trigger) → `dead-code-pr.yaml` pipeline + github-commenter step
- AC4 (issue creation) → `dead-code-issue.yaml` pipeline + github-commenter step
- AC5 (pipeline definition) → File existence + YAML parse test
- AC6 (contract schema) → Schema validation test
- AC7 (unit tests) → Schema and pipeline loader tests
