# Tasks

## Phase 1: Extract examples from Long to Example field

- [X] Task 1.1: `run.go` — Remove argument pattern lines (lines 69-72) from `Long`, they duplicate what's already in `Example` [P]
- [X] Task 1.2: `resume.go` — Extract 4 example lines from `Long`, create `Example` field [P]
- [X] Task 1.3: `cancel.go` — Extract 4 example lines from `Long`, create `Example` field [P]
- [X] Task 1.4: `logs.go` — Extract 9 example lines from `Long`, create `Example` field [P]

## Phase 2: Add Example field where missing

- [X] Task 2.1: `list.go` — Add `Example` field with representative usage patterns (list runs, list pipelines, etc.) [P]
- [X] Task 2.2: `bench.go` — Add `Example` field to parent bench command showing subcommand usage [P]

## Phase 3: Verification

- [X] Task 3.1: Verify `doctor.go` already has correct `Example` field (no changes needed)
- [X] Task 3.2: Confirm `pause` command does not exist (no changes needed)

## Phase 4: Validation

- [X] Task 4.1: Run `go build ./...` to verify compilation
- [X] Task 4.2: Run `go test ./cmd/wave/commands/...` to verify no test regressions
