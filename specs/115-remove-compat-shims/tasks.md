# Tasks

## Phase 1: Contract Package — StrictMode Removal
- [X] Task 1.1: Rename `StrictMode` to `MustPass` in `internal/contract/contract.go` and remove deprecated comment
- [X] Task 1.2: Update all `StrictMode` references in `internal/contract/jsonschema.go` to use `MustPass`
- [X] Task 1.3: Update `StrictMode` references in `internal/contract/typescript.go` to use `MustPass`
- [X] Task 1.4: Remove `IsTypeScriptAvailable()` wrapper in `internal/contract/typescript.go`; update callers to use `CheckTypeScriptAvailability()`
- [X] Task 1.5: Update all `StrictMode` references in `internal/contract/typescript_test.go` and `contract_test.go`
- [X] Task 1.6: Update `StrictMode` reference in `internal/pipeline/executor.go:1142` and `executor.go:1173`
- [X] Task 1.7: Update `StrictMode` in `internal/pipeline/executor_schema_test.go`
- [X] Task 1.8: Update `StrictMode` reference in `internal/security/sanitize.go` and `internal/security/config.go`
- [X] Task 1.9: Run `go test ./internal/contract/... ./internal/pipeline/... ./internal/security/...`

## Phase 2: Pipeline Retry Config Migration
- [X] Task 2.1: Migrate all `.wave/pipelines/*.yaml` files from `handover.contract.max_retries: N` to `retry: { max_attempts: N }` [P]
- [X] Task 2.2: Migrate all `internal/defaults/pipelines/*.yaml` files from `handover.contract.max_retries: N` to `retry: { max_attempts: N }` [P]
- [X] Task 2.3: Migrate `docs/examples/*.yaml` files similarly [P]
- [X] Task 2.4: Remove legacy handover retry fallback in `internal/pipeline/executor.go:569-576`
- [X] Task 2.5: Remove `MaxRetries` field from `HandoverConfig` in `internal/pipeline/types.go:245`
- [X] Task 2.6: Evaluate `MaxRetries` field in `ContractConfig` (types.go:258) — keep if used for contract-level retries, remove if only used for step retry compat
- [X] Task 2.7: Remove `TestExecuteStep_RetryConfig_BackwardCompat` test in `executor_test.go:3423-3454`
- [X] Task 2.8: Update any remaining tests that use `Handover.MaxRetries` or `Handover.Contract.MaxRetries` for retry config
- [X] Task 2.9: Run `go test ./internal/pipeline/...`

## Phase 3: Meta-Pipeline Legacy Format Removal
- [X] Task 3.1: Remove `extractYAMLLegacy()` function from `internal/pipeline/meta.go:604+`
- [X] Task 3.2: Update fallback in `internal/pipeline/meta.go:578` to return error instead of calling legacy function
- [X] Task 3.3: Remove `extractYAMLLegacy` tests from `internal/pipeline/meta_test.go`
- [X] Task 3.4: Run `go test ./internal/pipeline/...`

## Phase 4: JSON Cleaner Legacy Fallback Removal
- [X] Task 4.1: Remove `extractJSONFromTextLegacy()` from `internal/contract/json_cleaner.go:83+`
- [X] Task 4.2: Update `extractJSONFromText` to return error instead of falling back to legacy extraction
- [X] Task 4.3: Run `go test ./internal/contract/...`

## Phase 5: State Store Compat Path Removal
- [X] Task 5.1: Remove old `schema.sql` fallback path in `internal/state/store.go:164-174`; always use migration system
- [X] Task 5.2: Remove `collectRunsFromPipelineState()` and its fallback call in `cmd/wave/commands/list.go:782-783,825+`
- [X] Task 5.3: Remove migration `Down` SQL from all migrations in `internal/state/migration_definitions.go`
- [X] Task 5.4: Run `go test ./internal/state/... ./cmd/wave/commands/...`

## Phase 6: Output Config Fallback Removal
- [X] Task 6.1: Remove direct flag reading fallback in `cmd/wave/commands/output.go:107-118`; require `ResolvedFlags` in context
- [X] Task 6.2: Run `go test ./cmd/wave/commands/...`

## Phase 7: Comment and Label Cleanup [P]
- [X] Task 7.1: Remove "backward compat" comment from `internal/display/types.go:271`
- [X] Task 7.2: Remove "backward compat" comment from `internal/display/bubbletea_progress.go:383`
- [X] Task 7.3: Remove "legacy" label from template variable comment in `internal/pipeline/context.go:103`
- [X] Task 7.4: Remove "legacy" comment from workspace lookup in `internal/pipeline/resume.go:254`
- [X] Task 7.5: Remove compat comment from `itemKindAvailable` in `internal/tui/pipeline_list.go:28`
- [X] Task 7.6: Remove "legacy behavior" comment from `internal/worktree/worktree.go:92`
- [X] Task 7.7: Change "Wave project detected (legacy)" to "Wave project detected" in `internal/doctor/doctor.go:190`
- [X] Task 7.8: Remove "legacy directory" from comment in `internal/pipeline/types.go:199`
- [X] Task 7.9: Remove "legacy workspace" comment from `cmd/wave/commands/list.go:991`
- [X] Task 7.10: Remove "backward compatible" from comment in `internal/contract/testsuite.go:101`
- [X] Task 7.11: Remove "backward compatibility" from comment in `internal/contract/jsonschema.go:170`

## Phase 8: Final Validation
- [X] Task 8.1: Run `go vet ./...`
- [X] Task 8.2: Run `go test -race ./...`
- [X] Task 8.3: Grep codebase for remaining "backward compat", "backwards compat", "legacy" in Go source (verify only documentation references remain)
- [X] Task 8.4: Verify build: `go build ./cmd/wave/`
