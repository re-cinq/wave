package pipeline

import (
	"encoding/json"
	"time"

	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/state"
)

func (e *DefaultPipelineExecutor) resolveModel(step *Step, persona *manifest.Persona, routing *manifest.RoutingConfig, personaName string, adapterTierModels map[string]string) string {
	// Force override — bypasses all tier logic
	if e.forceModel {
		if e.modelOverride != "" {
			return e.modelOverride
		}
	}

	// Determine step-level tier (if any)
	stepTier := ""
	if step != nil && step.Model != "" {
		stepTier = step.Model
	} else if persona.Model != "" {
		stepTier = persona.Model
	}

	if e.modelOverride != "" {
		overrideRank := TierRank(e.modelOverride)
		if overrideRank >= 0 && stepTier != "" {
			// Both are tiers — use the cheaper one
			effectiveTier := CheaperTier(e.modelOverride, stepTier)
			if resolved, isTier := resolveTierModel(effectiveTier, routing, adapterTierModels); isTier {
				return resolved
			}
			return effectiveTier
		}
		// CLI is a literal model name — use it directly
		return e.modelOverride
	}

	// No CLI override — use step, persona, auto-route
	if step != nil && step.Model != "" {
		if resolved, isTier := resolveTierModel(step.Model, routing, adapterTierModels); isTier {
			return resolved
		}
		return step.Model
	}
	if persona.Model != "" {
		if resolved, isTier := resolveTierModel(persona.Model, routing, adapterTierModels); isTier {
			return resolved
		}
		return persona.Model
	}
	if routing != nil && routing.AutoRoute {
		tier := ClassifyStepComplexity(step, persona, personaName)
		if e.taskComplexity != "" {
			tier = AdjustTierForTaskComplexity(tier, e.taskComplexity)
		}
		if resolved, isTier := resolveTierModel(tier, routing, adapterTierModels); isTier {
			return resolved
		}
	}
	return ""
}

// resolveTierModel checks if a model string is a tier name (cheapest/balanced/strongest)
// and resolves it to an actual model via:
//  1. Adapter-specific tier_models (highest priority)
//  2. Global routing complexity_map
//
// Returns (resolved model, true) if input is a tier name, or ("", false) if it's a literal model.
func resolveTierModel(model string, routing *manifest.RoutingConfig, adapterTierModels map[string]string) (string, bool) {
	switch model {
	case TierCheapest, TierBalanced, TierStrongest:
		// Priority 1: adapter-specific tier_models
		if adapterTierModels != nil {
			if m, ok := adapterTierModels[model]; ok && m != "" {
				return m, true
			}
		}
		// Priority 2: global routing complexity_map
		return routing.ResolveComplexityModel(model), true
	default:
		return "", false
	}
}

// recordDecision records a structured decision to the state store.
// It is a no-op if the store is nil.
func (e *DefaultPipelineExecutor) recordDecision(runID, stepID, category, decision, rationale string, ctx map[string]interface{}) {
	if e.store == nil {
		return
	}
	contextJSON := "{}"
	if ctx != nil {
		if data, err := json.Marshal(ctx); err == nil {
			contextJSON = string(data)
		}
	}
	_ = e.store.RecordDecision(&state.DecisionRecord{
		RunID:     runID,
		StepID:    stepID,
		Timestamp: time.Now(),
		Category:  category,
		Decision:  decision,
		Rationale: rationale,
		Context:   contextJSON,
	})
}
