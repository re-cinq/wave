# Implementation Plan: Adaptive Model Selection

## Objective

Add a `ModelTier` field to `PipelineConfig` so that pipeline selection also determines which model tier each step role (implementation vs navigation) should use, based on task complexity.

## Approach

1. Define a `ModelTierMap` struct in `profile.go` with two fields: `Impl` and `Nav` (both `string`, using tier constants from `internal/pipeline/routing.go`).
2. Add `ModelTier ModelTierMap` to `PipelineConfig`.
3. Create a helper `modelTierForComplexity(Complexity) ModelTierMap` in `selector.go` that encodes the mapping table.
4. Call the helper from every return path in `SelectPipeline`.
5. Add unit tests for the helper and extend existing `TestSelectPipeline` cases to assert `ModelTier` values.

## File Mapping

| File | Action | What changes |
|---|---|---|
| `internal/classify/profile.go` | modify | Add `ModelTierMap` type and `ModelTier` field to `PipelineConfig` |
| `internal/classify/selector.go` | modify | Add `modelTierForComplexity` helper; set `ModelTier` on every returned `PipelineConfig` |
| `internal/classify/selector_test.go` | modify | Assert `ModelTier` on all existing cases; add dedicated tests for `modelTierForComplexity` |

## Architecture Decisions

- **Struct over map**: Use a typed struct (`ModelTierMap{Impl, Nav}`) rather than `map[string]string` for compile-time safety and self-documentation.
- **No import of pipeline package**: Use string literals (`"cheapest"`, `"balanced"`, `"strongest"`) matching the constants in `routing.go` to avoid a circular dependency between `classify` and `pipeline`. The tier values are stable string constants.
- **Helper function**: Centralise the complexity-to-tier mapping in one place rather than duplicating it across every return path.

## Risks

| Risk | Mitigation |
|---|---|
| Circular import if importing `pipeline` package | Use string literals matching tier constants |
| Existing callers ignore the new field | No breakage: zero-value `ModelTierMap{}` is safe, callers opt-in |

## Testing Strategy

- **Unit tests for `modelTierForComplexity`**: Table-driven, one case per complexity level, assert both `Impl` and `Nav`.
- **Extended `TestSelectPipeline`**: Add `wantModelTier` to existing test struct, verify on all cases.
- **Edge case**: Unknown complexity defaults (fallthrough) should produce balanced/cheapest as a safe default.
