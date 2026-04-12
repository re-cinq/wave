# Implementation Plan: Verification Depth Configuration

## Objective

Wire `VerificationDepth` from `TaskProfile` through `SelectPipeline()` into `PipelineConfig` so downstream consumers know which contract types to enforce for a given pipeline run.

## Approach

Minimal, additive change. The type system and derivation logic already exist. The only gap is the `PipelineConfig` struct and `SelectPipeline()` function not carrying the depth forward.

1. Add `VerificationDepth` field to `PipelineConfig`
2. Set `VerificationDepth` from the input profile in every return path of `SelectPipeline()`
3. Update selector tests to assert the depth value
4. Update profile tests to cover the new `PipelineConfig` field

## File Mapping

| File | Action | Change |
|------|--------|--------|
| `internal/classify/profile.go` | modify | Add `VerificationDepth VerificationDepth` field to `PipelineConfig` struct |
| `internal/classify/selector.go` | modify | Set `VerificationDepth: profile.VerificationDepth` in every returned `PipelineConfig` |
| `internal/classify/selector_test.go` | modify | Add `wantDepth` field to test cases; assert depth on each result |
| `internal/classify/profile_test.go` | modify | Update `TestPipelineConfigFields` to cover new field |

## Architecture Decisions

- **Pass-through, not re-derive**: `SelectPipeline` receives a fully-populated `TaskProfile` that already has `VerificationDepth` set by the analyzer. The selector passes it through rather than re-computing it. This keeps derivation logic in one place (`deriveVerificationDepth`).
- **No runtime behavior change yet**: This PR adds the field to the config struct. How the executor uses it (enabling/disabling specific contract types) is a separate concern for a follow-up issue.

## Risks

| Risk | Mitigation |
|------|-----------|
| Callers of `SelectPipeline` may not expect the new field | Field is additive with zero value `""`, existing callers unaffected |
| Consumers might assume depth implies runtime enforcement | Doc comment on the field clarifies it's advisory until wired into executor |

## Testing Strategy

- Update all existing selector test cases to include expected `VerificationDepth`
- Verify profile test covers the new `PipelineConfig` field
- Run full `go test ./internal/classify/...` to confirm no regressions
