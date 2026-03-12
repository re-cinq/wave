# Tasks

## Phase 1: Fix Core Bugs

- [X] Task 1.1: Replace unsafe `Read(buf)` with `io.ReadAll()` in `invokePhilosopherWithSchemas` (`internal/pipeline/meta.go:283-285`)
- [X] Task 1.2: Add `checkOnboarding()` call at the top of `runMeta()` in `cmd/wave/commands/meta.go`

## Phase 2: Mock Adapter Support

- [X] Task 2.1: Add `generateMetaPhilosopherOutput()` function to `internal/adapter/mock.go` that produces `--- PIPELINE ---` / `--- SCHEMAS ---` delimited output with a valid 2-step pipeline (navigate → implement) and matching JSON schema files
- [X] Task 2.2: Add workspace-path routing in `generateRealisticOutput()` to detect `meta-philosopher` workspace path and call `generateMetaPhilosopherOutput()` [P]

## Phase 3: Testing

- [X] Task 3.1: Add test in `internal/pipeline/meta_test.go` to verify `extractPipelineAndSchemas` correctly parses the new mock output format
- [X] Task 3.2: Add `TestMetaCommand_MockDryRun` in `cmd/wave/commands/meta_test.go` that exercises `runMeta` with `Mock: true, DryRun: true` end-to-end [P]
- [X] Task 3.3: Run `go test -race ./...` to verify no regressions

## Phase 4: Validation

- [X] Task 4.1: Verify all 19 existing meta tests still pass
- [X] Task 4.2: Verify `go build ./...` succeeds
- [X] Task 4.3: Manual verification that the generated mock pipeline YAML is valid and can be loaded by `YAMLPipelineLoader`
