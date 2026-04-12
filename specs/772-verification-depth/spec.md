# WS1.3 Verification Depth Configuration

**Issue**: [re-cinq/wave#772](https://github.com/re-cinq/wave/issues/772)
**Labels**: epic
**Author**: nextlevelshit
**State**: CLOSED

## Description

Add `verification_depth` field to the existing `PipelineConfig` in `internal/classify/selector.go`. Three levels:

- **structural_only**: `json_schema` contracts only
- **behavioral**: `test_suite` + contracts
- **full_semantic**: `llm_judge` + `agent_review` + `test_suite` + contracts

The classifier at `internal/classify/analyzer.go` should set `verification_depth` based on complexity:
- simple → structural_only
- medium → behavioral
- complex/architectural → full_semantic

Add `VerificationDepth` type to `internal/classify/profile.go`. Update selector tests. Reference `internal/contract/` for available contract types.

## Current State

The foundation already exists:
- `VerificationDepth` type and constants defined in `profile.go`
- `deriveVerificationDepth()` implemented in `analyzer.go` with correct mapping
- `TaskProfile.VerificationDepth` field already populated by `Classify()`

**Missing**: `PipelineConfig` does not include `VerificationDepth`, so `SelectPipeline()` discards the depth information from the profile.

## Acceptance Criteria

- [ ] `PipelineConfig` has a `VerificationDepth` field
- [ ] `SelectPipeline()` propagates `VerificationDepth` from the input `TaskProfile` to the returned `PipelineConfig`
- [ ] Selector tests assert correct `VerificationDepth` values on returned configs
- [ ] All existing tests continue to pass
