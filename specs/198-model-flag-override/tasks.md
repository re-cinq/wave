# Tasks

## Phase 1: Core Plumbing

- [X] Task 1.1: Add `WithModelOverride` executor option to `internal/pipeline/executor.go` — add `modelOverride string` field to `DefaultPipelineExecutor`, create `WithModelOverride(model string) ExecutorOption`, propagate in `NewChildExecutor`
- [X] Task 1.2: Apply model override in `runStepExecution` — in the `AdapterRunConfig` construction block (~line 675), apply `e.modelOverride` when `persona.Model == ""`, preserving per-persona pinning precedence

## Phase 2: CLI Flag Wiring

- [X] Task 2.1: Add `--model` flag to `wave run` — add `Model string` to `RunOptions`, register flag in `NewRunCmd`, pass via `pipeline.WithModelOverride(opts.Model)` in `runRun` when non-empty [P]
- [X] Task 2.2: Add `--model` flag to `wave do` — add `Model string` to `DoOptions`, register flag in `NewDoCmd`, pass via executor option [P]
- [X] Task 2.3: Add `--model` flag to `wave meta` — add `Model string` to `MetaOptions`, register flag in `NewMetaCmd`, pass via executor option to both child and meta executors [P]

## Phase 3: Testing

- [X] Task 3.1: Add unit tests for flag registration — verify `--model` flag exists on `run`, `do`, `meta` commands in `cmd/wave/commands/run_test.go` (extend `TestNewRunCmdFlags`), add analogous tests for `do` and `meta`
- [X] Task 3.2: Add unit tests for model override precedence in `internal/pipeline/executor_test.go` — test three cases: override applied when persona has no model, override skipped when persona has model pinned, no override when flag not provided [P]
- [X] Task 3.3: Add integration test verifying model reaches `AdapterRunConfig` — use mock adapter to capture and assert `cfg.Model` value [P]

## Phase 4: Polish

- [X] Task 4.1: Add help text documenting `--model` flag — include usage example and precedence note in `Long` description of `wave run`
- [X] Task 4.2: Run `go test ./...` and fix any regressions
