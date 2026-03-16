# Tasks: Continuous Pipeline Execution

**Feature**: #201 — Continuous Pipeline Execution
**Branch**: `201-continuous-pipeline`
**Generated**: 2026-03-16

## Phase 1: Setup

- [X] T001 [P1] Create `internal/continuous/` package directory and add doc.go with package comment

## Phase 2: Foundational — Data Types & Source URI Parsing (US3)

- [X] T002 [P1] [P] Define `WorkItem`, `IterationResult`, `IterationStatus`, `FailurePolicy`, `SourceConfig` types in `internal/continuous/types.go`
- [X] T003 [P1] [P] Define `WorkItemSource` interface (`Next(ctx) (*WorkItem, error)`, `Name() string`) in `internal/continuous/source.go`
- [X] T004 [P2] [US3] Implement `ParseSourceURI(uri string) (*SourceConfig, error)` in `internal/continuous/parse.go` — splits on first `:`, validates provider (`github`, `file`), parses comma-separated key=value params
- [X] T005 [P2] [US3] Write table-driven tests for `ParseSourceURI` in `internal/continuous/parse_test.go` — valid URIs, invalid providers, missing params, empty string, missing colon

## Phase 3: Work Item Sources (US3)

- [X] T006 [P2] [US3] Implement `GitHubSource` in `internal/continuous/source_github.go` — on first `Next()` shell out to `gh issue list --json number,url --label <label> --state <state> --limit <limit>`, parse JSON, iterate pre-fetched list on subsequent calls
- [X] T007 [P2] [US3] Implement `FileSource` in `internal/continuous/source_file.go` — load all lines on construction, `Next()` iterates through items, return nil when exhausted
- [X] T008 [P2] [US3] Implement `NewSourceFromConfig(cfg *SourceConfig) (WorkItemSource, error)` factory in `internal/continuous/source.go` dispatching to `GitHubSource` or `FileSource`
- [X] T009 [P2] [US3] Write tests for `GitHubSource` in `internal/continuous/source_github_test.go` — mock `gh` CLI output via exec helper, verify item ordering, empty result, malformed JSON
- [X] T010 [P2] [US3] Write tests for `FileSource` in `internal/continuous/source_file_test.go` — temp file with multiple lines, empty file, missing file
- [X] T011 [P2] [US3] Write tests for `NewSourceFromConfig` in `internal/continuous/source_test.go` — valid github config, valid file config, unknown provider

## Phase 4: Event Extension (US4)

- [X] T012 [P2] [US4] [P] Add iteration metadata fields to `Event` struct in `internal/event/emitter.go`: `Iteration int`, `TotalProcessed int`, `WorkItemID string` (all `json:",omitempty"`)

## Phase 5: Core Loop Controller (US1, US2)

- [X] T013 [P1] [US1] Define `Runner` struct in `internal/continuous/runner.go` with fields: `Source WorkItemSource`, `PipelineName string`, `OnFailure FailurePolicy`, `MaxIterations int`, `Delay time.Duration`, `Emitter event.EventEmitter`, `ExecutorFactory func(input string) ExecutorFunc`
- [X] T014 [P1] [US1] Define `ExecutorFunc` type (`func(ctx context.Context) (string, error)`) and `Summary` struct (`Total, Succeeded, Failed, Skipped int`, `Results []IterationResult`) with `String()` method in `internal/continuous/runner.go`
- [X] T015 [P1] [US1] Implement `Runner.Run(ctx context.Context) (*Summary, error)` in `internal/continuous/runner.go` — main loop: call `source.Next()`, check dedup in `processedIDs map`, invoke executor factory, record `IterationResult`, check failure policy, sleep delay, check `ctx.Err()`
- [X] T016 [P1] [US2] Implement graceful shutdown in `Runner.Run()` — between iterations check `ctx.Err()` for SIGINT/SIGTERM context cancellation, break loop without starting next item
- [X] T017 [P2] [US4] Emit `loop_start`, `loop_iteration_start`, `loop_iteration_complete`, `loop_iteration_failed`, `loop_summary` events from `Runner.Run()` via `Emitter`
- [X] T018 [P1] [US1] Write table-driven tests in `internal/continuous/runner_test.go`: normal completion (all succeed), empty source (immediate exit), max-iterations cap reached
- [X] T019 [P1] [US2] Write signal handling tests in `internal/continuous/runner_test.go`: context cancellation between iterations exits cleanly, context cancellation during executor propagates
- [X] T020 [P1] [US1] Write dedup test in `internal/continuous/runner_test.go`: source returns duplicate item ID → second occurrence skipped
- [X] T021 [P3] [US5] Write failure policy tests in `internal/continuous/runner_test.go`: `on_failure: halt` stops on first failure, `on_failure: skip` continues past failures with correct tally

## Phase 6: CLI Integration (US1)

- [X] T022 [P1] [US1] Add `Continuous bool`, `Source string`, `MaxIterations int`, `Delay string`, `OnFailure string` fields to `RunOptions` in `cmd/wave/commands/run.go`
- [X] T023 [P1] [US1] Register cobra flags on `wave run`: `--continuous`, `--source`, `--max-iterations`, `--delay`, `--on-failure` in `NewRunCmd()` in `cmd/wave/commands/run.go`
- [X] T024 [P1] [US1] Add mutual exclusion validation in `runRun()`: reject `--continuous` + `--from-step` combination (FR-010) in `cmd/wave/commands/run.go`
- [X] T025 [P1] [US1] Wire continuous mode dispatch in `runRun()`: when `opts.Continuous` is true, parse source URI, construct `continuous.Runner` with executor factory wrapping existing pipeline execution, call `runner.Run(ctx)`, print summary, set exit code in `cmd/wave/commands/run.go`

## Phase 7: Failure Policy (US5)

- [X] T026 [P3] [US5] Implement failure policy branch in `Runner.Run()` — after executor returns error, check `OnFailure`: if `halt` break loop, if `skip` record failure and continue in `internal/continuous/runner.go`
- [X] T027 [P3] [US5] Implement exit code semantics: return exit code 0 when all succeed, exit code 1 when any failed (per clarification C4) in `cmd/wave/commands/run.go`

## Phase 8: Polish & Cross-Cutting

- [X] T028 [P1] [P] Implement `Summary.String()` method with human-readable output: total iterations, succeeded, failed, skipped, duration per item in `internal/continuous/runner.go`
- [X] T029 [P1] Ensure `--continuous` without `--source` produces a clear error message in `cmd/wave/commands/run.go`
- [X] T030 [P1] Run `go test ./...` and `go vet ./...` to verify all new code compiles and tests pass
- [X] T031 [P1] Run `go test -race ./...` to verify no data races in concurrent paths

## Dependency Graph

```
T001 → T002, T003 (package must exist)
T002, T003 → T004, T005 (types needed for parser)
T004 → T006, T007, T008 (parser needed for source config)
T006, T007, T008 → T009, T010, T011 (implementations before tests)
T002 → T012 (types inform event fields)
T003, T004, T008 → T013, T014 (sources needed for runner)
T013, T014 → T015, T016 (runner struct before logic)
T015, T016 → T017 (loop logic before event emission)
T015 → T018, T019, T020, T021 (implementation before tests)
T013 → T022, T023 (runner exists for CLI wiring)
T022, T023 → T024, T025 (flags registered before validation/dispatch)
T015 → T026 (runner loop before failure policy branch)
T026 → T027 (failure policy before exit code)
T015, T028 → T025 (summary format before CLI print)
T025 → T029 (dispatch before validation message)
All → T030, T031 (final validation)
```

## Parallelization Notes

Tasks marked `[P]` can run concurrently with other tasks in the same phase:
- T002 and T003 are independent type/interface definitions
- T012 (event extension) is independent of source/runner work
- T028 (summary formatting) is independent of CLI wiring
