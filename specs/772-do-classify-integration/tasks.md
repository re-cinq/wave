# Tasks

## Phase 1: Setup
- [X] Task 1.1: Add `NoClassify` field to `DoOptions` struct and register `--no-classify` flag in `NewDoCmd()`
- [X] Task 1.2: Add `classify` package import to `do.go`

## Phase 2: Core Implementation
- [X] Task 2.1: Add classification logic to `runDo()` — call `classify.Classify()` and `classify.SelectPipeline()` when `--no-classify` is not set and `--persona` is not explicitly provided [P]
- [X] Task 2.2: Add manifest pipeline lookup — check if classified pipeline name exists in manifest, fall back to ad-hoc if not found [P]
- [X] Task 2.3: Wire classified pipeline execution through the existing executor path (load pipeline YAML, execute via `pipeline.NewDefaultPipelineExecutor`)
- [X] Task 2.4: Enhance `--dry-run` output to show classification details (domain, complexity, blast radius, selected pipeline, reason) when classification is active

## Phase 3: Testing
- [X] Task 3.1: Add test for `--no-classify` flag registration and default value
- [X] Task 3.2: Add test for dry-run with classification output (verify domain, complexity, pipeline name appear) [P]
- [X] Task 3.3: Add test for `--no-classify` producing original ad-hoc pipeline output [P]
- [X] Task 3.4: Add test for fallback to ad-hoc when classified pipeline not in manifest [P]
- [X] Task 3.5: Verify all existing `do_test.go` tests still pass unchanged
- [X] Task 3.6: Run `go test ./cmd/wave/commands/... ./internal/classify/...` to validate

## Phase 4: Polish
- [X] Task 4.1: Run `go vet ./cmd/wave/commands/` and `gofmt` check
- [X] Task 4.2: Final review of backward compatibility — verify no existing flag behavior changed
