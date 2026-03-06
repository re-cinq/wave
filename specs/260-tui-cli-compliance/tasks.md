# Tasks: CLI Compliance Polish

**Feature Branch**: `260-tui-cli-compliance`
**Generated**: 2026-03-07
**Spec**: `specs/260-tui-cli-compliance/spec.md`
**Plan**: `specs/260-tui-cli-compliance/plan.md`

## Phase 1: Setup & Foundation

- [X] T001 [P1] [Setup] Create `cmd/wave/commands/errors.go` with `CLIError` type implementing `error` interface, JSON tags (`error`, `code`, `suggestion`, `debug`), constructor `NewCLIError(code, message, suggestion)`, and error code constants (`pipeline_not_found`, `manifest_missing`, `manifest_invalid`, `contract_violation`, `adapter_not_found`, `flag_conflict`, `onboarding_required`, `step_not_found`, `run_not_found`, `preflight_failed`, `timeout`, `cancelled`, `internal_error`) per data-model.md
- [X] T002 [P1] [Setup] Create `cmd/wave/commands/errors_test.go` with table-driven tests: `CLIError.Error()` returns message, JSON marshaling produces `{"error":..,"code":..,"suggestion":..}`, `debug` field omitted when empty, included when set

## Phase 2: Root Flag Registration & Conflict Detection (US1 — Consistent Flag Surface)

- [X] T003 [P1] [US1] Register `--json` (bool), `-q`/`--quiet` (bool), and `--no-color` (bool) as persistent root flags in `cmd/wave/main.go` `init()` function, alongside existing `--output`, `--debug`, `--verbose`, `--no-tui`
- [X] T004 [P1] [US1] Add `ResolvedFlags` struct and `ResolveOutputConfig(cmd *cobra.Command) (*ResolvedFlags, error)` function to `cmd/wave/commands/output.go` that: reads all root flag states via `.Changed()`, detects conflicts (`--json` + `--output` non-json → `CLIError{Code: "flag_conflict"}`; `--quiet` + `--output` non-quiet → error; `--quiet` + `--verbose` → warn stderr, quiet wins), resolves final `OutputConfig{Format, Verbose, NoColor, Debug}`, and returns `ResolvedFlags{Output, NoTUI}`
- [X] T005 [P1] [US1] Extend `OutputConfig` struct in `cmd/wave/commands/output.go` with `NoColor bool` and `Debug bool` fields per data-model.md
- [X] T006 [P1] [US1] Add `PersistentPreRunE` to `rootCmd` in `cmd/wave/main.go` that calls `commands.ResolveOutputConfig(cmd)`, stores result in cmd context via `cmd.SetContext()`, sets `os.Setenv("NO_COLOR", "1")` when `--no-color` is true, and sets `noTUI` override when `--json` or `--quiet`
- [X] T007 [P1] [US1] Update `GetOutputConfig()` in `cmd/wave/commands/output.go` to first check cmd context for `ResolvedFlags`, falling back to existing flag reading for backward compatibility
- [X] T008 [P1] [US1] Add conflict detection and resolution tests to `cmd/wave/commands/output_test.go`: `--json` alone → Format="json"; `--quiet` alone → Format="quiet"; `--json`+`--output text` → error "flag_conflict"; `--quiet`+`--output json` → error; `--json`+`--quiet` → no error, Format="json"; `--no-color` → NoColor=true; `--quiet`+`--verbose` → quiet wins with warning

## Phase 3: JSON Error Rendering (US2 — Machine-Readable Output, US5 — Actionable Errors)

- [X] T009 [P1] [US2] Add `RenderJSONError(w io.Writer, err error, debug bool)` function to `cmd/wave/commands/errors.go` that marshals `*CLIError` as JSON to writer, or wraps plain `error` as `CLIError{Code: "internal_error"}` before marshaling
- [X] T010 [P1] [US2] Update `main()` error handler in `cmd/wave/main.go` to: read resolved output format from root flags, if JSON mode call `RenderJSONError(os.Stderr, err, debug)`, else render text error with suggestion if `*CLIError`
- [X] T011 [P1] [US5] Add `RenderTextError(w io.Writer, err error, debug bool)` function to `cmd/wave/commands/errors.go` that formats `*CLIError` as `"Error: <message>\n  Suggestion: <suggestion>"` and includes debug chain only when debug=true
- [X] T012 [P1] [US2,US5] Add tests to `cmd/wave/commands/errors_test.go` for: `RenderJSONError` produces valid JSON with error/code/suggestion fields on stderr, plain error wraps as `CLIError{Code: "internal_error"}`, debug details included only when debug=true, `RenderTextError` shows suggestion line

## Phase 4: Actionable Error Messages (US5)

- [X] T013 [P3] [US5] Wrap `loadPipeline()` error in `cmd/wave/commands/run.go` with `CLIError{Code: "pipeline_not_found", Suggestion: "Run 'wave list pipelines' to see available pipelines"}`
- [X] T014 [P3] [US5] Wrap manifest read error in `cmd/wave/commands/run.go` `runRun()` with `CLIError{Code: "manifest_missing", Suggestion: "Run 'wave init' to create a manifest"}`
- [X] T015 [P3] [US5] Wrap manifest parse error in `cmd/wave/commands/run.go` `runRun()` with `CLIError{Code: "manifest_invalid", Suggestion: "Check wave.yaml syntax — run 'wave validate' to diagnose"}`
- [X] T016 [P3] [US5] Wrap onboarding check error in `cmd/wave/commands/run.go` `checkOnboarding()` return with `CLIError{Code: "onboarding_required", Suggestion: "Run 'wave init'"}`
- [X] T017 [P] [P3] [US5] Add integration between `recovery.ErrorClass` and `CLIError.Code` — add `ErrorClassToCode(class recovery.ErrorClass) string` mapper function to `cmd/wave/commands/errors.go` that maps `ClassContractValidation` → `"contract_violation"`, `ClassPreflight` → `"preflight_failed"`, `ClassSecurityViolation` → `"security_violation"`, etc.

## Phase 5: Subcommand Format Resolution (US1, US2)

- [X] T018 [P1] [US1] Add `ResolveFormat(cmd *cobra.Command, localFormat string) string` function to `cmd/wave/commands/output.go` that returns root `--json`→"json" / `--quiet`→"quiet" / `--output` value if explicitly set, otherwise returns `localFormat` unchanged
- [X] T019 [P] [P1] [US1] Update `cmd/wave/commands/status.go` `NewStatusCmd` RunE to call `ResolveFormat(cmd, opts.Format)` before passing format to `runStatus`
- [X] T020 [P] [P1] [US1] Update `cmd/wave/commands/list.go` — in each list subcommand's RunE (`pipelines`, `personas`, `adapters`, `runs`, `contracts`, `skills`), call `ResolveFormat(cmd, format)` before format-conditional logic
- [X] T021 [P] [P1] [US1] Update `cmd/wave/commands/logs.go` `NewLogsCmd` RunE to call `ResolveFormat(cmd, opts.Format)` before passing format to `runLogs`
- [X] T022 [P] [P1] [US1] Update `cmd/wave/commands/artifacts.go` `NewArtifactsCmd` RunE to call `ResolveFormat(cmd, opts.Format)` before passing format to `runArtifacts`
- [X] T023 [P] [P1] [US1] Update `cmd/wave/commands/cancel.go` `NewCancelCmd` RunE to call `ResolveFormat(cmd, opts.Format)` before passing format to `runCancel`
- [X] T024 [P1] [US1] Add `ResolveFormat` tests to `cmd/wave/commands/output_test.go`: root `--json` overrides local "table", root `--quiet` overrides local "json", default root → local preserved

## Phase 6: Color Control (US3)

- [X] T025 [P2] [US3] Replace hardcoded ANSI color constants in `cmd/wave/commands/status.go` (`colorReset`, `colorRed`, `colorGreen`, `colorYellow`, `colorGray`) with a `conditionalColor(code string) string` helper that returns empty string when `os.Getenv("NO_COLOR") != ""`
- [X] T026 [P2] [US3] Add `--no-color` test to `cmd/wave/commands/status_test.go` that sets `NO_COLOR=1`, runs status output, and verifies zero ANSI escape sequences in output

## Phase 7: Output Stream Discipline (US6)

- [X] T027 [P] [P3] [US6] Update `cmd/wave/commands/clean.go` — change all `fmt.Printf` progress/informational messages to `fmt.Fprintf(os.Stderr, ...)`: "Nothing to clean", "Removed %s", "Failed to remove %s", "Cleaned %d item(s)", "Progress: %d/%d", confirmation prompts, and dry-run output
- [X] T028 [P] [P3] [US6] Update `cmd/wave/commands/logs.go` `renderPerformanceSummary()` — change all `fmt.Println`/`fmt.Printf` to `fmt.Fprintf(os.Stderr, ...)` for the "--- Performance Summary ---" block
- [X] T029 [P] [P3] [US6] Update `cmd/wave/commands/artifacts.go` `outputArtifactsTable()` — move "Artifacts for run:" header and "No artifacts found" message to stderr; keep table data rows on stdout
- [X] T030 [P] [P3] [US6] Update `cmd/wave/commands/status.go` `showRunningRuns()` and `showAllRuns()` — move "No running pipelines" and "No pipelines found" informational messages to stderr; keep table data on stdout
- [X] T031 [P] [P3] [US6] Update `cmd/wave/commands/run.go` `performDryRun()` — route dry-run output to stderr since it's informational, not data

## Phase 8: Quiet Mode & TUI Integration (US4)

- [X] T032 [P2] [US4] Update `shouldLaunchTUI()` in `cmd/wave/main.go` — add checks: if `--json` flag set → return false; if `--quiet` flag set → return false (before existing TTY check)
- [X] T033 [P2] [US4] Handle `clean --quiet` coexistence with root `--quiet` in `cmd/wave/commands/clean.go` — ensure both `opts.Quiet` (local flag) and root `--quiet` produce suppressed output without conflict

## Phase 9: Edge Cases & Polish

- [X] T034 [P2] [US1] Ensure `TERM=dumb` triggers `--no-color` equivalence in `cmd/wave/main.go` `PersistentPreRunE` — when `os.Getenv("TERM") == "dumb"`, set `NO_COLOR=1` and `noTUI=true`
- [X] T035 [P] [P3] [US2] Verify empty-list JSON output in subcommands — ensure `wave list pipelines --json` outputs `[]`, `wave status --json` outputs `{"runs":[]}`, not empty string, when no data exists (audit existing code in `list.go`, `status.go`)

## Phase 10: Test Verification & Cross-Cutting

- [X] T036 [P1] [SC-007] Run `go test -race ./...` and fix any regressions introduced by flag changes, ensuring all existing tests pass
- [X] T037 [P1] [SC-001] Add test that verifies `wave <subcommand> --help` for ALL subcommands shows standard persistent flags (`--json`, `-q`/`--quiet`, `--no-color`, `--debug`, `--verbose`, `--no-tui`, `--output`) — can be implemented as table-driven test in `cmd/wave/commands/output_test.go` or a new `cmd/wave/main_test.go`

---

## Task Summary

| Phase | Tasks | Priority | User Story |
|-------|-------|----------|------------|
| 1: Setup | T001-T002 | P1 | Foundation |
| 2: Root Flags | T003-T008 | P1 | US1 |
| 3: JSON Errors | T009-T012 | P1 | US2, US5 |
| 4: Actionable Errors | T013-T017 | P3 | US5 |
| 5: Format Resolution | T018-T024 | P1 | US1, US2 |
| 6: Color Control | T025-T026 | P2 | US3 |
| 7: Stream Discipline | T027-T031 | P3 | US6 |
| 8: Quiet/TUI | T032-T033 | P2 | US4 |
| 9: Edge Cases | T034-T035 | P2-P3 | Cross-cutting |
| 10: Verification | T036-T037 | P1 | SC-001, SC-007 |

**Total Tasks**: 37
**Parallelizable**: T019-T023, T027-T031, T035 (marked with [P])

## Dependency Graph

```
T001 ──┬──> T004 ──> T006 ──> T007
       │              ↑
T005 ──┘              │
       ├──> T009 ──> T010
       └──> T011      │
                       │
T003 ──────────────> T006
                       │
T002, T008, T012 ──> (tests, parallel with implementation)
                       │
T006 ──> T018 ──> T019,T020,T021,T022,T023 (parallel)
                       │
T001 ──> T013,T014,T015,T016,T017 (parallel after T001)
                       │
T006 ──> T025 ──> T026
T006 ──> T027,T028,T029,T030,T031 (parallel)
T006 ──> T032,T033,T034
                       │
T036,T037 ──> (final gate, all other tasks complete)
```
