# Tasks

## Phase 1: Contract Package Cleanup

- [X] Task 1.1: Remove `StrictMode` field from `ContractConfig` in `internal/contract/contract.go`; update all references in `jsonschema.go`, `typescript.go` to use `MustPass`
- [X] Task 1.2: Remove `extractJSONFromTextLegacy` function from `internal/contract/json_cleaner.go`; remove fallback call — if `RecoverJSON` fails, return error directly
- [X] Task 1.3: Remove `IsTypeScriptAvailable()` wrapper from `internal/contract/typescript.go`; replace all callers with `CheckTypeScriptAvailability()`
- [X] Task 1.4: Update `internal/contract/testsuite.go:101` comment — replace "backward compatible" with neutral description
- [X] Task 1.5: Update contract test files — replace `StrictMode` with `MustPass`, remove `TestIsTypeScriptAvailable`, update `json_cleaner` tests [P]

## Phase 2: Pipeline Package Cleanup

- [X] Task 2.1: Remove `MaxRetries` field from `HandoverConfig` in `internal/pipeline/types.go`; remove legacy fallback in `executor.go:468-476`
- [X] Task 2.2: Remove legacy template variables (`pipeline_id`, `pipeline_name`, `step_id`) from `internal/pipeline/context.go:103-107`; remove `legacy_variables` test case
- [X] Task 2.3: Remove `extractYAMLLegacy` function from `internal/pipeline/meta.go`; update `parseOutput` to return error if `--- PIPELINE ---` marker not found; remove test references in `meta_test.go`
- [X] Task 2.4: Remove exact-name directory fallback (legacy workspace) from `internal/pipeline/resume.go:254-257`
- [X] Task 2.5: Remove `StrictMode` mapping in `executor.go:1024`; clean up "legacy state store" comment at `executor.go:2404`
- [X] Task 2.6: Update `WorkspaceConfig.Type` comment in `types.go:192` — remove "legacy directory" wording
- [X] Task 2.7: Remove `TestExecuteStep_RetryConfig_BackwardCompat` test from `executor_test.go` [P]

## Phase 3: State / Migration Cleanup

- [X] Task 3.1: Remove all migration `Down` SQL from `internal/state/migration_definitions.go` — set all `Down` fields to empty strings
- [X] Task 3.2: Verify migration tests still pass with empty `Down` fields [P]

## Phase 4: Display / Doctor / Worktree Comment Cleanup

- [X] Task 4.1: Remove "backward compat" comment from `internal/display/types.go:270` and `bubbletea_progress.go:377` [P]
- [X] Task 4.2: Update "legacy" wording in `internal/doctor/doctor.go:190` [P]
- [X] Task 4.3: Update "legacy behavior" comment in `internal/worktree/worktree.go:92` [P]

## Phase 5: CLI Commands Cleanup

- [X] Task 5.1: Remove "backward compatibility" comment from `cmd/wave/commands/output.go:107`
- [X] Task 5.2: Evaluate and remove `collectRunsFromPipelineState` legacy table reader from `cmd/wave/commands/list.go:825-885`; update callers
- [X] Task 5.3: Update "legacy workspace" comment in `cmd/wave/commands/list.go:991`

## Phase 6: Validation

- [X] Task 6.1: Run `go build ./...` to verify compilation
- [X] Task 6.2: Run `go test ./...` to verify all tests pass
- [X] Task 6.3: Run `go test -race ./...` for race detection
- [X] Task 6.4: Run `go vet ./...` for static analysis
- [X] Task 6.5: Grep codebase for remaining "backward compat" / "legacy" references in `.go` files; evaluate each
