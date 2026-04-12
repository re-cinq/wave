# Tasks

## Phase 1: Core Type Change
- [X] Task 1.1: Add `VerificationDepth VerificationDepth` field to `PipelineConfig` struct in `internal/classify/profile.go`

## Phase 2: Selector Wiring
- [X] Task 2.1: Set `VerificationDepth: profile.VerificationDepth` in every `PipelineConfig` return in `SelectPipeline()` (`internal/classify/selector.go`)

## Phase 3: Testing
- [X] Task 3.1: Add `wantDepth VerificationDepth` to selector test table and assert on each case (`internal/classify/selector_test.go`) [P]
- [X] Task 3.2: Update `TestPipelineConfigFields` to include `VerificationDepth` (`internal/classify/profile_test.go`) [P]
- [X] Task 3.3: Run `go test ./internal/classify/...` to confirm all tests pass

## Phase 4: Validation
- [X] Task 4.1: Run `go vet ./internal/classify/...` for static analysis
