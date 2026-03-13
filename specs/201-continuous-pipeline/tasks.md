# Tasks

## Phase 1: Foundation — State Store and Event Extensions

- [X] Task 1.1: Add `continuous_processed_items` table migration
  - Add migration definition to `internal/state/migration_definitions.go`
  - Table schema: `id INTEGER PRIMARY KEY, pipeline_name TEXT, item_key TEXT, run_id TEXT, status TEXT, processed_at TIMESTAMP, UNIQUE(pipeline_name, item_key)`
  - Register migration in `internal/state/migrations.go`

- [X] Task 1.2: Add processed-item tracking methods to `StateStore` interface
  - `MarkItemProcessed(pipelineName, itemKey, runID string) error`
  - `IsItemProcessed(pipelineName, itemKey string) (bool, error)`
  - `ListProcessedItems(pipelineName string, limit int) ([]ProcessedItemRecord, error)`
  - Add `ProcessedItemRecord` type to `internal/state/types.go`
  - Implement in `internal/state/store.go`

- [X] Task 1.3: Add continuous-mode event state constants to `internal/event/emitter.go`
  - `StateContinuousStarted = "continuous_started"` — continuous run begins
  - `StateContinuousIterationStarted = "continuous_iteration_started"` — single iteration begins
  - `StateContinuousIterationCompleted = "continuous_iteration_completed"` — iteration succeeded
  - `StateContinuousIterationFailed = "continuous_iteration_failed"` — iteration failed
  - `StateContinuousExhausted = "continuous_exhausted"` — no more work items
  - `StateContinuousStopped = "continuous_stopped"` — graceful shutdown

## Phase 2: Core Implementation — Continuous Package

- [X] Task 2.1: Create `internal/continuous/provider.go` — WorkItemProvider interface [P]
  - Define `WorkItem` struct: `Key string`, `Input string`, `Labels []string`, `URL string`
  - Define `WorkItemProvider` interface: `Next(ctx context.Context) (*WorkItem, error)` — returns nil when exhausted
  - Implement `GitHubProvider` struct with fields: `repo string`, `labelFilter string`, `stateStore state.StateStore`, `pipelineName string`
  - `GitHubProvider.Next()` runs `gh issue list --repo <repo> --state open --json number,title,url --limit 50` and filters against state store
  - Support optional `--label` filter passed through from CLI/manifest

- [X] Task 2.2: Create `internal/continuous/runner.go` — core execution loop [P]
  - Define `Runner` struct with fields: `provider WorkItemProvider`, `pipelineFactory func(input string) error`, `emitter event.EventEmitter`, `delay time.Duration`, `haltOnError bool`, `store state.StateStore`, `pipelineName string`
  - Define `RunnerConfig` struct for configuration
  - `Runner.Run(ctx context.Context) error` — main loop:
    1. Emit `continuous_started` event
    2. Call `provider.Next(ctx)` to get next item
    3. If nil, emit `continuous_exhausted` and return nil
    4. Emit `continuous_iteration_started` with item key
    5. Call `pipelineFactory(item.Input)` to execute pipeline
    6. On success: call `store.MarkItemProcessed()`, emit `continuous_iteration_completed`
    7. On failure: if `haltOnError`, return error; else emit `continuous_iteration_failed` and continue
    8. Sleep for `delay` duration (respecting context cancellation)
    9. Goto step 2
  - On context cancellation: emit `continuous_stopped` and return nil

- [X] Task 2.3: Create `internal/continuous/state.go` — processed-item helpers
  - Wrapper functions for state store interactions specific to continuous mode
  - `BuildItemKey(issueURL string) string` — normalize issue URLs to stable keys
  - `ClearProcessedItems(store state.StateStore, pipelineName string) error` — reset for re-processing

## Phase 3: CLI Integration

- [X] Task 3.1: Add `--continuous` flags to `cmd/wave/commands/run.go`
  - Add to `RunOptions`: `Continuous bool`, `ContinuousDelay time.Duration`, `ContinuousHaltOnError bool`
  - Register flags: `--continuous` (bool), `--continuous-delay` (duration, default "10s"), `--continuous-halt-on-error` (bool)
  - In `runRun()`, when `opts.Continuous` is true:
    1. Detect the forge from `internal/forge` to select the right provider
    2. Create a `GitHubProvider` with repo from manifest metadata or `gh repo view --json nameWithOwner`
    3. Build a pipeline factory closure that calls existing `runRun` logic for a single iteration
    4. Create and run a `continuous.Runner`
  - Ensure SIGINT cancels the context (already handled), which triggers graceful shutdown

- [X] Task 3.2: Add `ContinuousConfig` to pipeline `InputConfig` in `internal/pipeline/types.go`
  - Add `ContinuousConfig` struct: `Enabled bool`, `LabelFilter string`, `Delay string`, `HaltOnError bool`, `MaxIterations int`
  - Add `Continuous *ContinuousConfig` field to `InputConfig`
  - CLI flags override manifest defaults when both are specified

## Phase 4: Testing

- [X] Task 4.1: Write unit tests for `internal/continuous/runner_test.go` [P]
  - Mock provider that returns N items then nil
  - Mock pipeline factory that records calls
  - Test: processes all items in order
  - Test: context cancellation stops loop
  - Test: halt-on-error stops on failure
  - Test: skip-and-continue skips failures
  - Test: delay between iterations is respected
  - Test: empty provider returns immediately

- [X] Task 4.2: Write unit tests for `internal/continuous/provider_test.go` [P]
  - Mock `gh issue list` output
  - Test: parses JSON output correctly
  - Test: filters already-processed items
  - Test: returns nil when all processed
  - Test: label filtering works
  - Test: handles `gh` CLI errors gracefully

- [X] Task 4.3: Write state store tests for processed-item tracking [P]
  - Test: MarkItemProcessed + IsItemProcessed roundtrip
  - Test: duplicate marking is idempotent
  - Test: ListProcessedItems returns correct records
  - Test: ClearProcessedItems resets state

## Phase 5: Polish

- [X] Task 5.1: Add continuous mode to pipeline dry-run output
  - In `performDryRun()`, show continuous configuration if present
  - Display label filter, delay, halt behavior

- [X] Task 5.2: Final validation
  - Run `go test -race ./...` — all tests pass
  - Run `go vet ./...` — no issues
  - Run `golangci-lint run ./...` — clean
  - Verify mock mode: `wave run --continuous --mock gh-implement`
