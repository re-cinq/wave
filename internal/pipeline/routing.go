package pipeline

import (
	"strings"

	"github.com/recinq/wave/internal/manifest"
)

// Complexity tier constants returned by ClassifyStepComplexity.
const (
	TierSimple   = "simple"
	TierStandard = "standard"
	TierComplex  = "complex"
)

// simplePersonaKeywords identifies personas whose work is typically low-complexity.
var simplePersonaKeywords = []string{
	"navigator",
	"summarizer",
	"auditor",
	"planner",
}

// complexPersonaKeywords identifies personas whose work is typically high-complexity.
var complexPersonaKeywords = []string{
	"craftsman",
	"implementer",
	"debugger",
	"researcher",
	"supervisor",
	"philosopher",
	"provocateur",
}

// ClassifyStepComplexity returns a complexity tier for the given step and persona.
// The tier is one of TierSimple, TierStandard, or TierComplex.
//
// Classification heuristics (evaluated in order):
//   - simple: persona name contains a simple keyword, OR step type is "command"/"conditional"
//   - complex: persona name contains a complex keyword, OR step uses sub_pipeline/loop/branch/aggregate
//   - standard: fallthrough for everything else
func ClassifyStepComplexity(step *Step, persona *manifest.Persona, personaName string) string {
	// Normalize persona name for keyword matching.
	lowerName := strings.ToLower(personaName)

	// Check simple signals first — cheap operations route to cheaper models.
	if step != nil && (step.Type == StepTypeCommand || step.Type == StepTypeConditional) {
		return TierSimple
	}
	for _, kw := range simplePersonaKeywords {
		if strings.Contains(lowerName, kw) {
			return TierSimple
		}
	}

	// Check complex signals — heavy operations route to stronger models.
	if step != nil && (step.SubPipeline != "" || step.Loop != nil || step.Branch != nil || step.Aggregate != nil) {
		return TierComplex
	}
	for _, kw := range complexPersonaKeywords {
		if strings.Contains(lowerName, kw) {
			return TierComplex
		}
	}

	return TierStandard
}
