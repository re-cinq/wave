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

	// Rule 4: Debug domain → ops-debug or impl-hotfix.
	if profile.Domain == DomainDebug {
		if profile.Complexity == ComplexitySimple {
			return PipelineConfig{
				Name:              "impl-hotfix",
				Reason:            "simple debug task routed to hotfix pipeline",
				VerificationDepth: profile.VerificationDepth,
				ModelTier:         ModelTierBalanced,
			}
		}
		return PipelineConfig{
			Name:              "ops-debug",
			Reason:            "debug task routed to systematic debugging pipeline",
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

	// Rule 6: Testing domain → test-gen.
	if profile.Domain == DomainTesting {
		return PipelineConfig{
			Name:              "test-gen",
			Reason:            "testing domain routed to test generation pipeline",
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
			Name:              "audit-quality-loop",
			Reason:            "audit task routed to quality loop pipeline",
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

	// Rule 10: Performance domain → benchmark pipeline or impl-improve.
	if profile.Domain == DomainPerformance {
		return PipelineConfig{
			Name:              "impl-improve",
			Reason:            "performance domain routed to improvement pipeline",
			VerificationDepth: profile.VerificationDepth,
			ModelTier:         tier,
		}
	}

	// Rule 11: Research domain → impl-research.
	if profile.Domain == DomainResearch {
		return PipelineConfig{
			Name:              "impl-research",
			Reason:            "research domain routed to research pipeline",
			VerificationDepth: profile.VerificationDepth,
			ModelTier:         tier,
		}
	}

	// Rule 12: Docs domain → route by content type.
	if profile.Domain == DomainDocs {
		return PipelineConfig{
			Name:              "doc-fix",
			Reason:            "documentation domain routed to doc-fix pipeline",
			VerificationDepth: profile.VerificationDepth,
			ModelTier:         ModelTierCheapest,
		}
	}

	// Rule 13: Bug domain → hotfix (simple) or impl-issue (medium+).
	if profile.Domain == DomainBug {
		if profile.Complexity == ComplexitySimple {
			return PipelineConfig{
				Name:              "impl-hotfix",
				Reason:            "simple bug routed to hotfix pipeline",
				VerificationDepth: profile.VerificationDepth,
				ModelTier:         ModelTierBalanced,
			}
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
