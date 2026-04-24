package classify

import "github.com/recinq/wave/internal/suggest"

// SelectPipeline maps a TaskProfile to a PipelineConfig using priority-ordered
// routing rules derived from the AGENTS.md pipeline selection table.
// modelTierForComplexity maps task complexity to recommended model tier.
func modelTierForComplexity(c Complexity) ModelTier {
	switch c {
	case ComplexitySimple:
		return ModelTierCheapest
	case ComplexityMedium:
		return ModelTierBalanced
	case ComplexityComplex, ComplexityArchitectural:
		return ModelTierStrongest
	default:
		return ModelTierBalanced
	}
}

// SelectPipeline maps a TaskProfile to a PipelineConfig using priority-ordered
// routing rules. Covers the full pipeline catalog.
func SelectPipeline(profile TaskProfile) PipelineConfig {
	tier := modelTierForComplexity(profile.Complexity)

	// Rule 1: PR URLs always route to ops-pr-review regardless of content.
	if profile.InputType == suggest.InputTypePRURL {
		return PipelineConfig{
			Name:              "ops-pr-review",
			Reason:            "pull request URL routed to PR review pipeline",
			VerificationDepth: profile.VerificationDepth,
			ModelTier:         ModelTierCheapest,
		}
	}

	// Rule 2: Issue URLs route by domain — the issue content determines the pipeline.
	// (InputType detection happens upstream in suggest.ClassifyInput)

	// Rule 3: Security domain → security audit.
	if profile.Domain == DomainSecurity {
		return PipelineConfig{
			Name:              "audit-security",
			Reason:            "security domain routed to security audit pipeline",
			VerificationDepth: profile.VerificationDepth,
			ModelTier:         tier,
		}
	}

	// Rule 5: Review domain → ops-pr-review.
	if profile.Domain == DomainReview {
		return PipelineConfig{
			Name:              "ops-pr-review",
			Reason:            "review domain routed to PR review pipeline",
			VerificationDepth: profile.VerificationDepth,
			ModelTier:         ModelTierCheapest,
		}
	}

	// Rule 6: Testing domain → audit-tests.
	if profile.Domain == DomainTesting {
		return PipelineConfig{
			Name:              "audit-tests",
			Reason:            "testing domain routed to test audit pipeline",
			VerificationDepth: profile.VerificationDepth,
			ModelTier:         tier,
		}
	}

	// Rule 7: Audit domain → route by complexity.
	if profile.Domain == DomainAudit {
		if profile.Complexity == ComplexityComplex || profile.Complexity == ComplexityArchitectural {
			return PipelineConfig{
				Name:              "ops-parallel-audit",
				Reason:            "complex audit routed to parallel audit pipeline",
				VerificationDepth: profile.VerificationDepth,
				ModelTier:         tier,
			}
		}
		return PipelineConfig{
			Name:              "audit-architecture",
			Reason:            "audit task routed to architecture audit pipeline",
			VerificationDepth: profile.VerificationDepth,
			ModelTier:         tier,
		}
	}

	// Rule 8: Planning domain → plan-task or plan-scope.
	if profile.Domain == DomainPlanning {
		if profile.Complexity == ComplexityComplex || profile.Complexity == ComplexityArchitectural {
			return PipelineConfig{
				Name:              "plan-scope",
				Reason:            "complex planning routed to epic scoping pipeline",
				VerificationDepth: profile.VerificationDepth,
				ModelTier:         tier,
			}
		}
		return PipelineConfig{
			Name:              "plan-task",
			Reason:            "planning task routed to task breakdown pipeline",
			VerificationDepth: profile.VerificationDepth,
			ModelTier:         ModelTierBalanced,
		}
	}

	// Rule 9: Ops domain → route by complexity.
	if profile.Domain == DomainOps {
		if profile.Complexity == ComplexityComplex || profile.Complexity == ComplexityArchitectural {
			return PipelineConfig{
				Name:              "ops-epic-runner",
				Reason:            "complex ops routed to epic runner pipeline",
				VerificationDepth: profile.VerificationDepth,
				ModelTier:         tier,
			}
		}
		return PipelineConfig{
			Name:              "ops-bootstrap",
			Reason:            "ops task routed to bootstrap pipeline",
			VerificationDepth: profile.VerificationDepth,
			ModelTier:         tier,
		}
	}

	// Rule 11: Research domain → plan-research.
	if profile.Domain == DomainResearch {
		return PipelineConfig{
			Name:              "plan-research",
			Reason:            "research domain routed to research planning pipeline",
			VerificationDepth: profile.VerificationDepth,
			ModelTier:         tier,
		}
	}

	// Rule 12: Docs domain → doc-explain.
	if profile.Domain == DomainDocs {
		return PipelineConfig{
			Name:              "doc-explain",
			Reason:            "documentation domain routed to doc-explain pipeline",
			VerificationDepth: profile.VerificationDepth,
			ModelTier:         ModelTierCheapest,
		}
	}

	// Rule 14: Complex/architectural refactors → spec-driven.
	if profile.Domain == DomainRefactor &&
		(profile.Complexity == ComplexityComplex || profile.Complexity == ComplexityArchitectural) {
		return PipelineConfig{
			Name:              "impl-speckit",
			Reason:            "complex refactor routed to spec-driven implementation pipeline",
			VerificationDepth: profile.VerificationDepth,
			ModelTier:         tier,
		}
	}

	// Rule 15: Simple and medium tasks → direct implementation.
	if profile.Complexity == ComplexitySimple || profile.Complexity == ComplexityMedium {
		return PipelineConfig{
			Name:              "impl-issue",
			Reason:            "simple/medium task routed to direct implementation pipeline",
			VerificationDepth: profile.VerificationDepth,
			ModelTier:         tier,
		}
	}

	// Rule 16: Complex and architectural tasks → spec-driven implementation.
	return PipelineConfig{
		Name:              "impl-speckit",
		Reason:            "complex/architectural task routed to spec-driven implementation pipeline",
		VerificationDepth: profile.VerificationDepth,
		ModelTier:         tier,
	}
}
