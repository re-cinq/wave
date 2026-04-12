package classify

import "github.com/recinq/wave/internal/suggest"

// SelectPipeline maps a TaskProfile to a PipelineConfig using priority-ordered
// routing rules derived from the AGENTS.md pipeline selection table.
func SelectPipeline(profile TaskProfile) PipelineConfig {
	// Rule 1: PR URLs always route to ops-pr-review regardless of content.
	if profile.InputType == suggest.InputTypePRURL {
		return PipelineConfig{
			Name:              "ops-pr-review",
			Reason:            "pull request URL routed to PR review pipeline",
			VerificationDepth: profile.VerificationDepth,
		}
	}

	// Rule 2: Security domain overrides complexity-based routing.
	if profile.Domain == DomainSecurity {
		return PipelineConfig{
			Name:              "audit-security",
			Reason:            "security domain routed to security audit pipeline",
			VerificationDepth: profile.VerificationDepth,
		}
	}

	// Rule 3: Research domain routes to research pipeline.
	if profile.Domain == DomainResearch {
		return PipelineConfig{
			Name:              "impl-research",
			Reason:            "research domain routed to research pipeline",
			VerificationDepth: profile.VerificationDepth,
		}
	}

	// Rule 4: Docs domain routes to doc-fix pipeline.
	if profile.Domain == DomainDocs {
		return PipelineConfig{
			Name:              "doc-fix",
			Reason:            "documentation domain routed to doc-fix pipeline",
			VerificationDepth: profile.VerificationDepth,
		}
	}

	// Rule 5: Complex/architectural refactors need spec-driven implementation.
	if profile.Domain == DomainRefactor &&
		(profile.Complexity == ComplexityComplex || profile.Complexity == ComplexityArchitectural) {
		return PipelineConfig{
			Name:              "impl-speckit",
			Reason:            "complex refactor routed to spec-driven implementation pipeline",
			VerificationDepth: profile.VerificationDepth,
		}
	}

	// Rule 6: Simple and medium tasks use direct implementation.
	if profile.Complexity == ComplexitySimple || profile.Complexity == ComplexityMedium {
		return PipelineConfig{
			Name:              "impl-issue",
			Reason:            "simple/medium task routed to direct implementation pipeline",
			VerificationDepth: profile.VerificationDepth,
		}
	}

	// Rule 7: Complex and architectural tasks use spec-driven implementation.
	return PipelineConfig{
		Name:              "impl-speckit",
		Reason:            "complex/architectural task routed to spec-driven implementation pipeline",
		VerificationDepth: profile.VerificationDepth,
	}
}
