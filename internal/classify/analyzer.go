package classify

import (
	"strings"

	"github.com/recinq/wave/internal/suggest"
)

// domainKeywords maps domains to their detection keywords, ordered by priority.
// More specific domains first to avoid false matches on generic terms.
var domainKeywords = []struct {
	domain   Domain
	keywords []string
}{
	{DomainSecurity, []string{"vulnerability", "cve", "injection", "xss", "csrf", "auth bypass", "security", "owasp", "threat model"}},
	{DomainDebug, []string{"debug", "diagnose", "root cause", "stack trace", "reproduce", "bisect"}},
	{DomainPerformance, []string{"slow", "latency", "performance", "optimize", "benchmark", "memory leak", "profil"}},
	{DomainBug, []string{"bug", "broken", "crash", "error", "null pointer", "panic", "doesn't work", "hotfix", "bugfix"}},
	{DomainReview, []string{"review", "code review", "pr review", "pull request review", "feedback"}},
	{DomainTesting, []string{"test coverage", "test gap", "missing test", "generate test", "test quality", "unit test", "integration test"}},
	{DomainAudit, []string{"audit", "dead code", "unused", "duplicate", "junk", "unwired", "coverage gap", "quality scan", "scan for"}},
	{DomainPlanning, []string{"plan", "scope", "break down", "decompose", "adr", "decision record", "epic", "roadmap", "task breakdown"}},
	{DomainOps, []string{"release", "deploy", "bootstrap", "scaffold", "changelog", "ci/cd", "pipeline run", "batch", "supervise"}},
	{DomainRefactor, []string{"refactor", "restructure", "reorganize", "redesign", "clean up", "technical debt", "simplify"}},
	{DomainResearch, []string{"research", "investigate", "analyze", "compare", "evaluate", "explore"}},
	{DomainDocs, []string{"documentation", "readme", "typo", "comment", "docs", "docstring", "onboard", "explain", "changelog"}},
	{DomainFeature, []string{"add", "implement", "create", "new", "feature", "support", "prototype"}},
}

// complexityKeywords maps complexity levels to their detection keywords.
var complexityKeywords = []struct {
	complexity Complexity
	keywords   []string
}{
	{ComplexityArchitectural, []string{"architecture", "redesign", "system-wide", "entire"}},
	{ComplexityComplex, []string{"multiple", "several", "complex", "integration", "across"}},
	{ComplexitySimple, []string{"typo", "rename", "single", "minor", "small", "trivial"}},
}

// complexityBlastBase maps complexity to base blast radius values.
var complexityBlastBase = map[Complexity]float64{
	ComplexitySimple:        0.1,
	ComplexityMedium:        0.3,
	ComplexityComplex:       0.6,
	ComplexityArchitectural: 0.8,
}

// domainBlastModifier maps domain to blast radius modifiers.
var domainBlastModifier = map[Domain]float64{
	DomainSecurity:    0.2,
	DomainPerformance: 0.1,
	DomainDocs:        -0.1,
}

// Classify analyzes input text and optional issue body to produce a TaskProfile.
func Classify(input string, issueBody string) TaskProfile {
	inputType := suggest.ClassifyInput(input)

	trimmed := strings.TrimSpace(input)
	if trimmed == "" && strings.TrimSpace(issueBody) == "" {
		return TaskProfile{
			BlastRadius:       0.1,
			Complexity:        ComplexitySimple,
			Domain:            DomainFeature,
			VerificationDepth: VerificationStructuralOnly,
			InputType:         inputType,
		}
	}

	text := strings.ToLower(trimmed + " " + issueBody)

	domain := matchDomain(text)
	complexity := matchComplexity(text)

	// Lore enrichment: if keyword matching fell through to defaults,
	// let high-confidence lore hints fill in the gap.
	domain, complexity = applyLoreHints(domain, complexity, input)

	blastRadius := deriveBlastRadius(complexity, domain)
	depth := deriveVerificationDepth(complexity)

	return TaskProfile{
		BlastRadius:       blastRadius,
		Complexity:        complexity,
		Domain:            domain,
		VerificationDepth: depth,
		InputType:         inputType,
	}
}

// matchDomain returns the highest-priority domain whose keywords appear in text.
// Falls back to DomainFeature if no keywords match.
func matchDomain(text string) Domain {
	for _, dk := range domainKeywords {
		for _, kw := range dk.keywords {
			if strings.Contains(text, kw) {
				return dk.domain
			}
		}
	}
	return DomainFeature
}

// matchComplexity returns the highest-priority complexity whose keywords appear in text.
// Falls back to ComplexityMedium if no keywords match.
func matchComplexity(text string) Complexity {
	for _, ck := range complexityKeywords {
		for _, kw := range ck.keywords {
			if strings.Contains(text, kw) {
				return ck.complexity
			}
		}
	}
	return ComplexityMedium
}

// deriveBlastRadius computes blast radius from complexity base + domain modifier,
// clamped to [0.0, 1.0].
func deriveBlastRadius(c Complexity, d Domain) float64 {
	base := complexityBlastBase[c]
	mod := domainBlastModifier[d]
	r := base + mod
	if r < 0 {
		return 0
	}
	if r > 1 {
		return 1
	}
	return r
}

// applyLoreHints enriches domain/complexity with lore provider hints when
// keyword matching fell through to defaults (DomainFeature / ComplexityMedium).
// Lore hints only fill gaps — they never override a positive keyword match.
func applyLoreHints(domain Domain, complexity Complexity, input string) (Domain, Complexity) {
	hints := loreProvider().Hints(input)
	if len(hints) == 0 {
		return domain, complexity
	}
	const minConfidence = 0.5
	for _, h := range hints {
		if h.Confidence < minConfidence {
			continue
		}
		if domain == DomainFeature && h.Domain != "" {
			domain = h.Domain
		}
		if complexity == ComplexityMedium && h.Complexity != "" {
			complexity = h.Complexity
		}
	}
	return domain, complexity
}

// deriveVerificationDepth maps complexity to verification depth.
func deriveVerificationDepth(c Complexity) VerificationDepth {
	switch c {
	case ComplexitySimple:
		return VerificationStructuralOnly
	case ComplexityMedium:
		return VerificationBehavioral
	default:
		return VerificationFullSemantic
	}
}
