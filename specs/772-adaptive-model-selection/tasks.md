# Tasks

## Phase 1: Type Definitions
- [X] Task 1.1: Add `ModelTierMap` struct to `internal/classify/profile.go` with `Impl string` and `Nav string` fields
- [X] Task 1.2: Add `ModelTier ModelTierMap` field to `PipelineConfig` struct in `internal/classify/profile.go`

## Phase 2: Core Implementation
- [X] Task 2.1: Add `modelTierForComplexity(Complexity) ModelTierMap` helper in `internal/classify/selector.go` encoding the mapping: simple={cheapest,cheapest}, medium={balanced,cheapest}, complex={strongest,cheapest}, architectural={strongest,strongest}
- [X] Task 2.2: Update every return path in `SelectPipeline` to call `modelTierForComplexity(profile.Complexity)` and set the `ModelTier` field

## Phase 3: Testing
- [X] Task 3.1: Add table-driven `TestModelTierForComplexity` in `internal/classify/selector_test.go` covering all four complexity levels plus unknown/fallthrough [P]
- [X] Task 3.2: Extend existing `TestSelectPipeline` test struct with `wantModelTier` field and assert on all cases [P]

## Phase 4: Validation
- [X] Task 4.1: Run `go test ./internal/classify/...` and `go vet ./internal/classify/...` to confirm green
