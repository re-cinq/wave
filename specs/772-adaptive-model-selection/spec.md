# WS1.4 Adaptive Model Selection

**Issue**: [re-cinq/wave#772](https://github.com/re-cinq/wave/issues/772)
**Labels**: epic
**Author**: nextlevelshit
**State**: CLOSED

## Description

Extend `internal/classify/selector.go` `PipelineConfig` with `ModelTier` field. Map task complexity to model tier:

| Complexity | Implementation Steps | Navigation Steps |
|---|---|---|
| simple | cheapest | cheapest |
| medium | balanced | cheapest |
| complex | strongest | cheapest |
| architectural | strongest | strongest |

Reference `internal/pipeline/routing.go` tier system (`TierCheapest`, `TierBalanced`, `TierStrongest`).

## Acceptance Criteria

1. `PipelineConfig` in `internal/classify/profile.go` has a `ModelTier` field that encodes the per-role tier mapping
2. `SelectPipeline` in `internal/classify/selector.go` sets `ModelTier` based on the complexity of the `TaskProfile`
3. The mapping matches the table above exactly
4. Unit tests cover all four complexity levels and verify both impl and nav tier values
