package classify

import "github.com/recinq/wave/internal/suggest"

// Complexity enumerates task complexity levels.
type Complexity string

const (
	ComplexitySimple        Complexity = "simple"
	ComplexityMedium        Complexity = "medium"
	ComplexityComplex       Complexity = "complex"
	ComplexityArchitectural Complexity = "architectural"
)

// Domain enumerates task domain categories.
type Domain string

const (
	DomainSecurity    Domain = "security"
	DomainPerformance Domain = "performance"
	DomainBug         Domain = "bug"
	DomainRefactor    Domain = "refactor"
	DomainFeature     Domain = "feature"
	DomainDocs        Domain = "docs"
	DomainResearch    Domain = "research"
)

// VerificationDepth enumerates verification levels.
type VerificationDepth string

const (
	VerificationStructuralOnly VerificationDepth = "structural_only"
	VerificationBehavioral     VerificationDepth = "behavioral"
	VerificationFullSemantic   VerificationDepth = "full_semantic"
)

// TaskProfile is the structured output of task classification.
type TaskProfile struct {
	BlastRadius       float64           // 0.0-1.0, risk/impact score
	Complexity        Complexity        // simple/medium/complex/architectural
	Domain            Domain            // security/performance/bug/refactor/feature/docs/research
	VerificationDepth VerificationDepth // structural_only/behavioral/full_semantic
	InputType         suggest.InputType // reused from internal/suggest
}

// ModelTier enumerates model selection tiers.
type ModelTier string

const (
	ModelTierCheapest  ModelTier = "cheapest"
	ModelTierBalanced  ModelTier = "balanced"
	ModelTierStrongest ModelTier = "strongest"
)

// PipelineConfig is the result of pipeline selection.
type PipelineConfig struct {
	Name              string            // pipeline name, e.g. "impl-issue"
	Reason            string            // human-readable routing explanation
	VerificationDepth VerificationDepth // depth of verification to apply (advisory until wired into executor)
	ModelTier         ModelTier         // recommended model tier based on task complexity
}
