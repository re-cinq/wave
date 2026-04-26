# Implementation Plan — Reorganize `internal/contract` by contract type

## 1. Objective

Reorganize the `internal/contract` package so each contract type co-locates its
validation, recovery, and judge logic in type-named files, collapsing the
operation-grouped files (`json_recovery.go`, `json_cleaner.go`, `wrapper_detection.go`,
`input_validator.go`) into the contract type that owns them. Public/exported API
must remain unchanged so external callers stay unaffected.

## 2. Approach

The package already has one file per contract type (`jsonschema.go`,
`markdownspec.go`, `typescript.go`, `testsuite.go`, `non_empty_file.go`,
`source_diff.go`, `spec_derived.go`, `event_contains.go`, `agent_review.go`,
`llm_judge.go`, `format_validator.go`). The problem is that JSON-schema-related
operations were factored into separate "operation" files even though they only
serve the JSON-schema contract type. Cross-cutting machinery (retry strategy,
validation error formatting, the dispatcher) is genuinely shared and stays in
its own files.

Steps:

1. Map every symbol in the operation files to the contract type that consumes it.
2. Move type-specific operation code into per-type sub-files using the naming
   pattern `<type>_<operation>.go` (and matching `_test.go`). Use plain `git mv`
   for tests; for `.go` files, prefer `git mv` + rename followed by editing the
   minimal header so commits stay reviewable.
3. Keep cross-cutting modules (`contract.go`, `retry_strategy.go`,
   `validation_error_formatter.go`, `doc.go`, `sync_test.go`) untouched in
   responsibility — they coordinate dispatch and must stay shared.
4. Run `go build ./...`, `go vet ./...`, `go test ./internal/contract/...` and
   the full `go test -race ./...` to prove the refactor is API-preserving.
5. Update `internal/contract/README.md` to describe the new file layout.

The refactor is a pure file/symbol relocation. No exported symbol is renamed,
removed, or has its signature changed. No production behaviour changes.

## 3. File Mapping

### Move / rename (per-type ownership)

| From                                        | To                                          | Rationale |
| ------------------------------------------- | ------------------------------------------- | --------- |
| `internal/contract/json_recovery.go`        | `internal/contract/jsonschema_recovery.go`  | Recovery logic is exclusive to JSON-based contracts. |
| `internal/contract/json_recovery_test.go`   | `internal/contract/jsonschema_recovery_test.go` | Test follows the file. |
| `internal/contract/json_cleaner.go`         | `internal/contract/jsonschema_cleaner.go`   | Helper used only by JSON recovery / jsonschema validation. |
| `internal/contract/json_cleaner_test.go`    | `internal/contract/jsonschema_cleaner_test.go` | Test follows the file. |
| `internal/contract/wrapper_detection.go`    | `internal/contract/jsonschema_wrapper.go`   | Detects error wrappers around JSON payloads — only `jsonschema.go` calls it. |
| `internal/contract/wrapper_detection_test.go` | `internal/contract/jsonschema_wrapper_test.go` | Test follows the file. |
| `internal/contract/input_validator.go`      | `internal/contract/jsonschema_input.go`     | Input artifact validation is JSON-schema-specific (depends on `jsonschema/v6`). |
| `internal/contract/input_validator_test.go` | `internal/contract/jsonschema_input_test.go` | Test follows the file. |
| `internal/contract/format_validator.go`     | `internal/contract/format.go`               | Align with the per-type naming pattern (`format` is a contract type). |
| `internal/contract/format_validator_test.go`| `internal/contract/format_test.go`          | Test follows the file. |

### Stay (genuinely cross-cutting)

| File                                         | Reason |
| -------------------------------------------- | ------ |
| `internal/contract/contract.go`              | Public types, dispatcher (`NewValidator`, `Validate`, `ValidateWithRetries`, `ValidateWithAdaptiveRetry`). |
| `internal/contract/doc.go`                   | Package doc. |
| `internal/contract/retry_strategy.go`        | Retry strategy + failure classifier are shared by all contract types. |
| `internal/contract/retry_strategy_test.go`   | — |
| `internal/contract/adaptive_retry_test.go`   | Tests cross-cutting retry behaviour. |
| `internal/contract/validation_error_formatter.go` | Error formatting shared by every validator. |
| `internal/contract/sync_test.go`             | Cross-cutting integration tests. |
| `internal/contract/false_positive_test.go`   | Cross-cutting regression suite. |
| `internal/contract/contract_test.go`         | Tests dispatcher and shared types. |

### Per-type files (already correct, no move needed)

`jsonschema.go`, `markdownspec.go`, `typescript.go`, `testsuite.go`,
`non_empty_file.go`, `source_diff.go`, `spec_derived.go`, `event_contains.go`,
`agent_review.go`, `llm_judge.go` (each with their `_test.go`).

### Documentation

- `internal/contract/README.md` — refresh the "File layout" section to describe
  per-type ownership and call out which files are cross-cutting.

## 4. Architecture Decisions

1. **Type-prefix naming.** Per-type sub-files are named `<type>_<operation>.go`
   (e.g. `jsonschema_recovery.go`). This makes the contract type's full footprint
   discoverable with `ls jsonschema_*` and keeps related logic alphabetically
   adjacent.
2. **Cross-cutting code stays shared.** `retry_strategy.go`,
   `validation_error_formatter.go`, and the dispatcher in `contract.go` operate
   over every contract type and are not duplicated per type.
3. **No symbol renames.** Function and type names (`JSONRecoveryParser`,
   `CleanJSON`, `DetectErrorWrapper`, `ValidateInputArtifacts`,
   `FormatValidator`) are preserved. Only the file each lives in changes.
4. **JSON helpers belong to `jsonschema`.** `json_cleaner` and `json_recovery`
   are exclusively consumed by `jsonschema.go` (and the `jsonschema` recovery
   pathway), so they belong under that type — not as a generic JSON utility.
5. **Tests track their source.** Test files move alongside the file they cover,
   keeping git history aligned per-type.

## 5. Risks and Mitigations

| Risk | Mitigation |
| ---- | ---------- |
| Hidden imports across the codebase (e.g. via `contract.JSONRecoveryParser`) break because the symbol is now in a renamed file. | Symbols stay package-level and unchanged. `go build ./...` + full test suite confirms nothing broke. |
| `git mv` followed by edits losing rename detection. | Do `git mv` first in its own commit (or single-file mv) so git records it as a rename. Avoid simultaneous large content changes. |
| Test files declaring internal helpers with name collisions after move. | Run `go test ./internal/contract/...` after each move; resolve duplicate symbol errors before continuing. |
| README drift. | Update `README.md` "File layout" section in the same change. |
| External consumers (other packages) referencing internal helpers. | `internal/` import path means only repo-local consumers exist. `grep` confirms callers use only exported `contract.*` symbols. No callers reference filenames. |
| `format_validator.go` rename collides with `format` package or symbol elsewhere. | Confirmed `FormatValidator` is unique; renaming the file is safe. |

## 6. Testing Strategy

1. **Unit tests:** No new tests needed — the existing per-file tests already
   exercise every moved symbol. They move with their source.
2. **Build verification:** `go build ./...` after every batch of moves.
3. **Vet:** `go vet ./...` to catch any stale references.
4. **Full test suite:** `go test -race ./...` (the project's standard pre-PR
   gate per AGENTS.md / CLAUDE.md).
5. **Targeted contract suite:** `go test ./internal/contract/... -count=1 -race`
   to surface contract-specific regressions early.
6. **Linter:** `golangci-lint run ./internal/contract/...` (project default).
7. **API surface check:** Run `go doc ./internal/contract` before and after;
   diff must be empty (modulo file location annotations).
