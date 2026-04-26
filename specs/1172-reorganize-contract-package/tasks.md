# Work Items

## Phase 1: Setup

- [X] 1.1: Confirm feature branch `1172-reorganize-contract-package` is checked out and clean.
- [X] 1.2: Capture pre-refactor `go doc ./internal/contract` output for API-surface diffing.
- [X] 1.3: Run `go test ./internal/contract/... -count=1 -race` to confirm green baseline.

## Phase 2: Core Implementation (per-type relocations)

- [X] 2.1: `git mv internal/contract/json_recovery.go internal/contract/jsonschema_recovery.go` and matching `_test.go`. [P]
- [X] 2.2: `git mv internal/contract/json_cleaner.go internal/contract/jsonschema_cleaner.go` and matching `_test.go`. [P]
- [X] 2.3: `git mv internal/contract/wrapper_detection.go internal/contract/jsonschema_wrapper.go` and matching `_test.go`. [P]
- [X] 2.4: `git mv internal/contract/input_validator.go internal/contract/jsonschema_input.go` and matching `_test.go`. [P]
- [X] 2.5: `git mv internal/contract/format_validator.go internal/contract/format.go` and matching `_test.go`. [P]
- [X] 2.6: After each batch, run `go build ./...` to fail fast on any breakage.

## Phase 3: Testing

- [X] 3.1: `go vet ./...`
- [X] 3.2: `go test ./internal/contract/... -count=1 -race`
- [X] 3.3: `go test -race ./...` (full project suite)
- [ ] 3.4: `golangci-lint run ./internal/contract/...` — skipped, golangci-lint binary not available in this workspace; vet + race tests pass.
- [X] 3.5: Diff `go doc ./internal/contract` against the pre-refactor capture; confirm zero exported-API delta.

## Phase 4: Polish

- [X] 4.1: Update `internal/contract/README.md` "File layout" section to reflect type-owned files and cross-cutting files. [P]
- [X] 4.2: Verify no stale references to old filenames in comments or docs (`grep -R "json_recovery\|json_cleaner\|wrapper_detection\|input_validator\|format_validator" --include='*.{go,md}'`). Live skill `.agents/skills/wave-ctx-validation/SKILL.md` updated; historical changelog/spec files left as-is. [P]
- [X] 4.3: Final `go build ./...`, commit, push, open PR linking back to #1172.
