# Tasks

## Phase 1: Rename init --output to --manifest-path
- [X] Task 1.1: In `cmd/wave/commands/init.go`, rename the `--output` flag to `--manifest-path`
- [X] Task 1.2: In `cmd/wave/commands/init_test.go`, update `TestInitOutputPath` to use `--manifest-path`

## Phase 2: Rename agent export --output to --export-path
- [X] Task 2.1: In `cmd/wave/commands/agent.go`, rename `--output`/`-o` to `--export-path` (no short form) [P]
- [X] Task 2.2: In `cmd/wave/commands/agent_test.go`, update test to use `--export-path` [P]

## Phase 3: Rename bench run --output to --results-path
- [X] Task 3.1: In `cmd/wave/commands/bench.go`, rename `--output` to `--results-path` [P]

## Phase 4: Validation
- [X] Task 4.1: Run `go test ./cmd/wave/commands/...` to verify all command tests pass
- [X] Task 4.2: Run `go test ./...` for full regression
- [X] Task 4.3: Run `go vet ./...` to catch any issues
