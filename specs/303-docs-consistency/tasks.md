# Tasks

## Phase 1: Critical — non_empty_file Validator (DOC-002)

- [X] Task 1.1: Implement `non_empty_file` validator in `internal/contract/non_empty_file.go` — struct implementing `ContractValidator` interface that checks source file exists and is non-empty
- [X] Task 1.2: Add `"non_empty_file"` case to `NewValidator()` switch in `internal/contract/contract.go`
- [X] Task 1.3: Write table-driven tests in `internal/contract/non_empty_file_test.go` — test missing file, empty file, non-empty file, must_pass behavior
- [X] Task 1.4: Add `non_empty_file` section to `docs/reference/contract-types.md` — Quick Reference table entry + full section with fields, examples
- [X] Task 1.5: Run `go test ./internal/contract/...` to verify validator works

## Phase 2: Critical — Missing CLI Commands (DOC-001)

- [X] Task 2.1: Add `wave compose` to Quick Reference table and add full documentation section in `docs/reference/cli.md` — usage, options, examples, output [P]
- [X] Task 2.2: Add `wave doctor` to Quick Reference table and add full documentation section in `docs/reference/cli.md` — usage, options (--fix, --optimize, --dry-run, --json, --skip-ai, --yes, --skip-codebase), examples, output [P]
- [X] Task 2.3: Add `wave suggest` to Quick Reference table and add full documentation section in `docs/reference/cli.md` — usage, options (--limit, --dry-run, --json), examples, output [P]

## Phase 3: High — CLI Docs Fixes (DOC-003, DOC-004, DOC-010)

- [X] Task 3.1: Add `--json`, `--quiet`/`-q`, `--no-color` to Global Options table in `docs/reference/cli.md` (DOC-003)
- [X] Task 3.2: Add `--model` flag to Options sections for `wave run`, `wave do`, and `wave meta` in `docs/reference/cli.md` (DOC-004)
- [X] Task 3.3: Remove duplicate `wave chat` entry from Quick Reference table in `docs/reference/cli.md` — keep line 15, remove line 21 (DOC-010)

## Phase 4: High — Architecture Doc Fixes (DOC-005, DOC-006, DOC-007)

- [X] Task 4.1: Fix workspace path from `/tmp/wave/<pipeline-id>/<step-id>/` to `.wave/workspaces/<pipeline>/<step>/` and update directory layout in `docs/concepts/architecture.md` (DOC-005) [P]
- [X] Task 4.2: Change "OpenCode (future)" to "OpenCode" in `docs/concepts/architecture.md:167` (DOC-006) [P]
- [X] Task 4.3: Add "Markdown Spec" and "Format (experimental)" to contract types list in `docs/concepts/architecture.md:94-97` (DOC-007) [P]

## Phase 5: Medium — Default Value Alignment (DOC-009)

- [X] Task 5.1: Update `max_retries` default in `docs/concepts/contracts.md:98` to say `2` instead of `0`
- [X] Task 5.2: Verify `max_retries` default in `docs/reference/contract-types.md:51` already says `2` (no change if correct)

## Phase 6: Low — Environment and Quick Start (DOC-011, DOC-012)

- [X] Task 6.1: Add clarification to `CLAUDE_CODE_MODEL` entry in `docs/reference/environment.md:160` noting it must be in `runtime.sandbox.env_passthrough` and that Wave's `--model` flag is the preferred mechanism (DOC-011) [P]
- [X] Task 6.2: Update `wave init` output example in `docs/guide/quick-start.md` to match actual `printInitSuccess` output format from `cmd/wave/commands/init.go:790-824` (DOC-012) [P]

## Phase 7: Validation

- [X] Task 7.1: Run `go test ./...` to verify no regressions
- [X] Task 7.2: Review all modified files for accuracy and consistency
