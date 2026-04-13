package pipeline

import (
	"strings"

	"github.com/recinq/wave/internal/manifest"
)

// Complexity tier constants returned by ClassifyStepComplexity.
// Three tiers: cheapest (cost), balanced (quality/cost), strongest (capability).
const (
	TierCheapest  = "cheapest"
	TierBalanced  = "balanced"
	TierStrongest = "strongest"
)

// cheapestPersonaKeywords identifies personas whose work is typically lightweight.
// These route to cost-optimized models.
var cheapestPersonaKeywords = []string{
	"navigator",
	"summarizer",
	"auditor",
	"planner",
}

// strongestPersonaKeywords identifies personas whose work is typically complex.
// These route to capability-optimized models.
var strongestPersonaKeywords = []string{
	"craftsman",
	"implementer",
	"debugger",
	"researcher",
	"supervisor",
	"philosopher",
	"provocateur",
}

// ClassifyStepComplexity returns a complexity tier for the given step and persona.
// The tier is one of TierCheapest, TierBalanced, or TierStrongest.
//
// Classification heuristics (evaluated in order):
//   - cheapest: persona name contains a lightweight keyword, OR step type is "command"/"conditional"
//   - strongest: persona name contains a complex keyword, OR step uses sub_pipeline/loop/branch/aggregate
//   - balanced: fallthrough for everything else (balance of cost and capability)
func ClassifyStepComplexity(step *Step, persona *manifest.Persona, personaName string) string {
	// Normalize persona name for keyword matching.
	lowerName := strings.ToLower(personaName)

	// Check cheapest signals — lightweight operations route to cheaper models.
	if step != nil && (step.Type == StepTypeCommand || step.Type == StepTypeConditional) {
		return TierCheapest
	}
	for _, kw := range cheapestPersonaKeywords {
		if strings.Contains(lowerName, kw) {
			return TierCheapest
		}
	}

	// Check strongest signals — heavy operations route to more capable models.
	if step != nil && (step.SubPipeline != "" || step.Loop != nil || step.Branch != nil || step.Aggregate != nil) {
		return TierStrongest
	}
	for _, kw := range strongestPersonaKeywords {
		if strings.Contains(lowerName, kw) {
			return TierStrongest
		}
	}

	return TierBalanced
}

// TierRank returns the cost rank of a tier (lower = cheaper).
// Used to resolve conflicts: when multiple tiers apply, the cheaper one wins.
func TierRank(tier string) int {
	switch tier {
	case TierCheapest:
		return 0
	case TierBalanced:
		return 1
	case TierStrongest:
		return 2
	default:
		return -1 // not a tier (literal model name)
	}
}

// AdjustTierForTaskComplexity adjusts a step-level tier based on task-level complexity.
// Simple tasks cap at balanced (even for strongest personas) to save tokens.
// Complex/architectural tasks floor at balanced (even for cheapest personas).
// Empty taskComplexity means no adjustment.
func AdjustTierForTaskComplexity(stepTier, taskComplexity string) string {
	switch taskComplexity {
	case "simple":
		if stepTier == TierStrongest {
			return TierBalanced
		}
	case "complex", "architectural":
		if stepTier == TierCheapest {
			return TierBalanced
		}
	}
	return stepTier
}

// CheaperTier returns the cheaper of two tier names.
// If either is not a recognized tier, it returns the other.
func CheaperTier(a, b string) string {
	ra, rb := TierRank(a), TierRank(b)
	if ra < 0 {
		return b
	}
	if rb < 0 {
		return a
	}
	if ra <= rb {
		return a
	}
	return b
}
