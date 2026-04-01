package pipeline

import (
	"strings"

	"github.com/recinq/wave/internal/manifest"
)

// Complexity tier constants returned by ClassifyStepComplexity.
// These tiers guide model selection: cheapest (cost), fastest (latency), strongest (capability).
const (
	TierCheapest  = "cheapest"
	TierFastest   = "fastest"
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
// The tier is one of TierCheapest, TierFastest, or TierStrongest.
//
// Classification heuristics (evaluated in order):
//   - cheapest: persona name contains a lightweight keyword, OR step type is "command"/"conditional"
//   - strongest: persona name contains a complex keyword, OR step uses sub_pipeline/loop/branch/aggregate
//   - fastest: fallthrough for everything else (balance of cost and capability)
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

	return TierFastest
}
