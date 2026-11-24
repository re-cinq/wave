package pipeline

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/state"
)

// resolveModel is a thin wrapper preserved for test ergonomics; production
// dispatch goes through resolveModelForAttempt so the retry tier escalation
// path is exercised. The nolint:unparam directive silences the linter
// noticing that adapterTierModels is always nil in test callers — the
// parameter is part of the wider tier-resolution API and must stay.
//
//nolint:unparam // test-only helper; production callers use resolveModelForAttempt directly.
func (e *DefaultPipelineExecutor) resolveModel(step *Step, persona *manifest.Persona, routing *manifest.RoutingConfig, personaName string, adapterTierModels map[string]string) string {
	return e.resolveModelForAttempt(step, persona, routing, personaName, adapterTierModels, 1)
}

// resolveModelForAttempt is the retry-aware variant of resolveModel. When
// attempt > 1 and the step's RetryConfig does not set NoEscalate, any
// tier-named source (cheapest/balanced/strongest) is escalated by
// (attempt - 1) tiers along the cost ladder before being resolved to a
// concrete model. Literal model names (user-pinned exact IDs) are
// preserved verbatim — the auto-escalation only walks the recognized
// tier ladder so explicit overrides are never overridden.
func (e *DefaultPipelineExecutor) resolveModelForAttempt(step *Step, persona *manifest.Persona, routing *manifest.RoutingConfig, personaName string, adapterTierModels map[string]string, attempt int) string {
	// Force override — bypasses all tier logic
	if e.forceModel {
		if e.modelOverride != "" {
			return e.modelOverride
		}
	}

	retries := attempt - 1
	if retries < 0 {
		retries = 0
	}
	if step != nil && step.Retry.NoEscalate {
		retries = 0
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
			effectiveTier = EscalateTier(effectiveTier, retries)
			if resolved, isTier := resolveTierModel(effectiveTier, routing, adapterTierModels); isTier {
				return resolved
			}
			return effectiveTier
		}
		// CLI is a tier name with no step tier — escalate it on retry
		if overrideRank >= 0 {
			tier := EscalateTier(e.modelOverride, retries)
			if resolved, isTier := resolveTierModel(tier, routing, adapterTierModels); isTier {
				return resolved
			}
			return tier
		}
		// CLI is a literal model name — use it directly (no escalation)
		return e.modelOverride
	}

	// No CLI override — use step, persona, auto-route
	if step != nil && step.Model != "" {
		tier := EscalateTier(step.Model, retries)
		if resolved, isTier := resolveTierModel(tier, routing, adapterTierModels); isTier {
			return resolved
		}
		return tier
	}
	if persona.Model != "" {
		tier := EscalateTier(persona.Model, retries)
		if resolved, isTier := resolveTierModel(tier, routing, adapterTierModels); isTier {
			return resolved
		}
		return tier
	}
	if routing != nil && routing.AutoRoute {
		tier := ClassifyStepComplexity(step, persona, personaName)
		if e.taskComplexity != "" {
			tier = AdjustTierForTaskComplexity(tier, e.taskComplexity)
		}
		tier = EscalateTier(tier, retries)
		if resolved, isTier := resolveTierModel(tier, routing, adapterTierModels); isTier {
			return resolved
		}
	}
	return ""
}

// resolveTierModel checks if a model string is a tier name (cheapest/balanced/strongest)
// and resolves it to an actual model via:
//  1. Adapter-specific tier_models (highest priority). A tier present with
//     an empty value resolves to "" so the adapter's default_model applies.
//  2. Global routing complexity_map
//
// Returns (resolved model, true) if input is a tier name, or ("", false) if it's a literal model.
func resolveTierModel(model string, routing *manifest.RoutingConfig, adapterTierModels map[string]string) (string, bool) {
	switch model {
	case TierCheapest, TierBalanced, TierStrongest:
		// Priority 1: adapter-specific tier_models
		if m, ok := adapterTierModels[model]; ok {
			return m, true
		}
		// Priority 2: global routing complexity_map
		return routing.ResolveComplexityModel(model), true
	default:
		return "", false
	}
}

// effectiveTierModels returns the tier-to-model map used for the resolved
// adapter. Explicit adapter tier_models always win. The built-in tier
// defaults (manifest.DefaultComplexityMap) are Claude model IDs, so for any
// other adapter without tier_models each tier maps to the user's
// routing.complexity_map entry, or to "" so the adapter's default_model is
// used instead of handing e.g. "claude-haiku-4-5" to opencode.
func effectiveTierModels(adapterName string, adapterDef *manifest.Adapter, routing *manifest.RoutingConfig) map[string]string {
	if adapterDef != nil && len(adapterDef.TierModels) > 0 {
		return adapterDef.TierModels
	}
	if strings.EqualFold(adapterName, "claude") {
		return nil
	}
	tiers := make(map[string]string, 3)
	for _, tier := range []string{TierCheapest, TierBalanced, TierStrongest} {
		model := ""
		if routing != nil {
			model = routing.ComplexityMap[tier]
		}
		tiers[tier] = model
	}
	return tiers
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
