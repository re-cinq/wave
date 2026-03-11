# docs: documentation consistency report

**Issue**: [#303](https://github.com/re-cinq/wave/issues/303)
**Labels**: documentation
**Author**: nextlevelshit
**State**: OPEN

## Summary

A documentation consistency audit identified 12 inconsistencies across 8+ documentation files. Each item has been assigned a severity (Critical, High, Medium, Low) and includes exact file paths, line numbers, and explicit fix instructions.

| Severity | Count |
|----------|-------|
| Critical | 2 |
| High     | 5 |
| Medium   | 3 |
| Low      | 2 |

## Inconsistencies

### Critical

- **[DOC-001]** Three CLI commands exist in code but are missing from docs — `docs/reference/cli.md`
  - Verified: `wave compose`, `wave doctor`, `wave suggest` are registered in `cmd/wave/main.go:150-152` but absent from `docs/reference/cli.md`
  - Fix: Add documentation sections for all three commands including usage, flags, examples, and output. Add entries to the Quick Reference table.

- **[DOC-002]** `non_empty_file` contract type used in 16+ pipelines but not implemented — `internal/contract/contract.go`, `internal/defaults/pipelines/*.yaml`
  - Verified: `NewValidator()` in `internal/contract/contract.go:71-86` has no `non_empty_file` case — returns `nil` for unknown types. 32 pipeline YAML files reference it.
  - Fix: Either implement a `non_empty_file` validator or replace all usages with an implemented type. Document if implemented.

### High

- **[DOC-003]** Global flags `--json`, `--quiet`/`-q`, `--no-color` missing from CLI docs — `docs/reference/cli.md:637-650`
  - Verified: Flags registered in `cmd/wave/main.go:132-134` but not in the Global Options table in cli.md
  - Fix: Add `--json`, `--quiet`/`-q`, and `--no-color` to the Global Options table.

- **[DOC-004]** `--model` flag on `run`, `do`, and `meta` commands not documented — `docs/reference/cli.md`
  - Verified: `--model` registered in `do.go:58`, `meta.go:64`, and `run.go` but absent from docs Options sections
  - Fix: Add `--model` flag to the Options sections for `wave run`, `wave do`, and `wave meta`.

- **[DOC-005]** Architecture doc shows wrong workspace path and outdated directory layout — `docs/concepts/architecture.md:58-64`
  - Verified: Shows `/tmp/wave/<pipeline-id>/<step-id>/` but actual path is `.wave/workspaces/<pipeline>/<step>/`. Directory layout shows `src/` and `artifacts/` but actual layout uses `.wave/artifacts/`, `.wave/output/`, `CLAUDE.md`, `.claude/`
  - Fix: Update workspace path and directory layout to match actual structure.

- **[DOC-006]** Architecture doc says OpenCode adapter is "future" but it already exists — `docs/concepts/architecture.md:167`
  - Verified: `internal/adapter/opencode.go` exists. Line 167 says "OpenCode (future)".
  - Fix: Change to "OpenCode (implemented)" or just "OpenCode".

- **[DOC-007]** Architecture doc lists only 3 of 5 implemented contract types — `docs/concepts/architecture.md:94-97`
  - Verified: Lists JSON Schema, TypeScript Interface, Test Suite. Missing: Markdown Spec and Format.
  - Fix: Add "Markdown Spec" and "Format (experimental)" to the contract types list.

### Medium

- **[DOC-008]** CLAUDE.md file structure missing three internal packages — **ALREADY RESOLVED**
  - Verified: Commit `f5987f2` ("docs: add doctor, forge, and suggest packages to file structure") already added `doctor/`, `forge/`, `suggest/` to CLAUDE.md.
  - No action needed.

- **[DOC-009]** `max_retries` default value inconsistent across documentation
  - Verified: Code in `contract.go:109-110` defaults to `1` (single attempt). `docs/concepts/contracts.md:98` says default `0`. `docs/reference/contract-types.md:51` says default `2`.
  - Fix: Align all docs to match the code behavior (default: 1 for `Validate`, 3 for `ValidateWithRetries`).

- **[DOC-010]** `wave chat` listed twice in CLI quick reference table — `docs/reference/cli.md:15,21`
  - Verified: Line 15 says "Interactive analysis of pipeline runs", Line 21 says "Interactive chat session with a persona".
  - Fix: Remove the duplicate entry. Keep the more complete description.

### Low

- **[DOC-011]** `CLAUDE_CODE_MODEL` env var documented but not used by Wave's Claude adapter — `docs/reference/environment.md:160`
  - Verified: Listed in adapter environment variables section. Wave manages model selection via `--model` flag, not this env var.
  - Fix: Clarify that `CLAUDE_CODE_MODEL` must be in `runtime.sandbox.env_passthrough` to take effect, or note that Wave's `--model` flag is the preferred mechanism.

- **[DOC-012]** Quick start guide shows inaccurate `wave init` output — `docs/guide/quick-start.md:30-45`
  - Verified: Quick start shows simple "Created wave.yaml / Created .wave/personas/..." but actual output shows ASCII art banner, file counts, and next steps.
  - Fix: Update the example output to reflect the actual `printInitSuccess` output format.

## Acceptance Criteria

- [ ] All 11 active inconsistencies resolved (DOC-008 already fixed)
- [ ] Documentation accurately reflects code behavior
- [ ] No new inconsistencies introduced
- [ ] `go test ./...` passes (if DOC-002 involves code changes)
