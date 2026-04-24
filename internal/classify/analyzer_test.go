package classify

import (
	"testing"
	"time"

	"github.com/recinq/wave/internal/suggest"
)

func TestClassify(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		issueBody  string
		wantDomain Domain
		wantComplx Complexity
		wantDepth  VerificationDepth
		blastMin   float64
		blastMax   float64
		wantInput  suggest.InputType
	}{
		{
			name:       "empty_input_defaults",
			input:      "",
			issueBody:  "",
			wantDomain: DomainFeature,
			wantComplx: ComplexitySimple,
			wantDepth:  VerificationStructuralOnly,
			blastMin:   0.0,
			blastMax:   0.15,
			wantInput:  suggest.InputTypeFreeText,
		},
		{
			name:       "whitespace_only_defaults",
			input:      "   \t\n  ",
			issueBody:  "   ",
			wantDomain: DomainFeature,
			wantComplx: ComplexitySimple,
			wantDepth:  VerificationStructuralOnly,
			blastMin:   0.0,
			blastMax:   0.15,
			wantInput:  suggest.InputTypeFreeText,
		},
		{
			name:       "security_domain",
			input:      "fix SQL injection vulnerability in user endpoint",
			issueBody:  "",
			wantDomain: DomainSecurity,
			wantComplx: ComplexityMedium,
			wantDepth:  VerificationBehavioral,
			blastMin:   0.4,
			blastMax:   0.6,
			wantInput:  suggest.InputTypeFreeText,
		},
		{
			name:       "performance_domain",
			input:      "optimize slow database query latency",
			issueBody:  "",
			wantDomain: DomainPerformance,
			wantComplx: ComplexityMedium,
			wantDepth:  VerificationBehavioral,
			blastMin:   0.3,
			blastMax:   0.5,
			wantInput:  suggest.InputTypeFreeText,
		},
		{
			name:       "bug_domain",
			input:      "login button doesn't work on mobile",
			issueBody:  "",
			wantDomain: DomainBug,
			wantComplx: ComplexityMedium,
			wantDepth:  VerificationBehavioral,
			blastMin:   0.2,
			blastMax:   0.4,
			wantInput:  suggest.InputTypeFreeText,
		},
		{
			name:       "refactor_domain",
			input:      "refactor the persistence layer",
			issueBody:  "",
			wantDomain: DomainRefactor,
			wantComplx: ComplexityMedium,
			wantDepth:  VerificationBehavioral,
			blastMin:   0.2,
			blastMax:   0.4,
			wantInput:  suggest.InputTypeFreeText,
		},
		{
			name:       "research_domain",
			input:      "investigate and compare caching strategies",
			issueBody:  "",
			wantDomain: DomainResearch,
			wantComplx: ComplexityMedium,
			wantDepth:  VerificationBehavioral,
			blastMin:   0.2,
			blastMax:   0.4,
			wantInput:  suggest.InputTypeFreeText,
		},
		{
			name:       "docs_domain",
			input:      "fix typo in README",
			issueBody:  "",
			wantDomain: DomainDocs,
			wantComplx: ComplexitySimple,
			wantDepth:  VerificationStructuralOnly,
			blastMin:   0.0,
			blastMax:   0.1,
			wantInput:  suggest.InputTypeFreeText,
		},
		{
			name:       "feature_domain",
			input:      "add new user registration feature",
			issueBody:  "",
			wantDomain: DomainFeature,
			wantComplx: ComplexityMedium,
			wantDepth:  VerificationBehavioral,
			blastMin:   0.2,
			blastMax:   0.4,
			wantInput:  suggest.InputTypeFreeText,
		},
		{
			name:       "architectural_complexity",
			input:      "redesign the entire pipeline architecture",
			issueBody:  "",
			wantDomain: DomainRefactor,
			wantComplx: ComplexityArchitectural,
			wantDepth:  VerificationFullSemantic,
			blastMin:   0.7,
			blastMax:   1.0,
			wantInput:  suggest.InputTypeFreeText,
		},
		{
			name:       "complex_complexity",
			input:      "implement integration across multiple services",
			issueBody:  "",
			wantDomain: DomainFeature,
			wantComplx: ComplexityComplex,
			wantDepth:  VerificationFullSemantic,
			blastMin:   0.5,
			blastMax:   0.7,
			wantInput:  suggest.InputTypeFreeText,
		},
		{
			name:       "mixed_domain_security_wins",
			input:      "fix security bug in docs",
			issueBody:  "",
			wantDomain: DomainSecurity,
			wantComplx: ComplexityMedium,
			wantDepth:  VerificationBehavioral,
			blastMin:   0.4,
			blastMax:   0.6,
			wantInput:  suggest.InputTypeFreeText,
		},
		{
			name:       "pr_url_input",
			input:      "https://github.com/org/repo/pull/99",
			issueBody:  "",
			wantDomain: DomainFeature,
			wantComplx: ComplexityMedium,
			wantDepth:  VerificationBehavioral,
			blastMin:   0.2,
			blastMax:   0.4,
			wantInput:  suggest.InputTypePRURL,
		},
		{
			name:       "issue_url_with_body",
			input:      "https://github.com/org/repo/issues/42",
			issueBody:  "login button doesn't work on mobile",
			wantDomain: DomainBug,
			wantComplx: ComplexityMedium,
			wantDepth:  VerificationBehavioral,
			blastMin:   0.2,
			blastMax:   0.4,
			wantInput:  suggest.InputTypeIssueURL,
		},
		{
			name:       "no_keywords_fallback",
			input:      "do something with the widgets",
			issueBody:  "",
			wantDomain: DomainFeature,
			wantComplx: ComplexityMedium,
			wantDepth:  VerificationBehavioral,
			blastMin:   0.2,
			blastMax:   0.4,
			wantInput:  suggest.InputTypeFreeText,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Classify(tt.input, tt.issueBody)

			if got.Domain != tt.wantDomain {
				t.Errorf("Domain = %q, want %q", got.Domain, tt.wantDomain)
			}
			if got.Complexity != tt.wantComplx {
				t.Errorf("Complexity = %q, want %q", got.Complexity, tt.wantComplx)
			}
			if got.VerificationDepth != tt.wantDepth {
				t.Errorf("VerificationDepth = %q, want %q", got.VerificationDepth, tt.wantDepth)
			}
			if got.BlastRadius < tt.blastMin || got.BlastRadius > tt.blastMax {
				t.Errorf("BlastRadius = %f, want [%f, %f]", got.BlastRadius, tt.blastMin, tt.blastMax)
			}
			if got.InputType != tt.wantInput {
				t.Errorf("InputType = %q, want %q", got.InputType, tt.wantInput)
			}
		})
	}
}

// Integration tests: exercise full classify-then-select flow.
func TestClassifyThenSelect(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		issueBody    string
		wantPipeline string
		wantDomain   Domain
	}{
		{
			name:         "simple_bug_to_impl_issue",
			input:        "fix null pointer in logger",
			issueBody:    "",
			wantPipeline: "impl-issue",
			wantDomain:   DomainBug,
		},
		{
			name:         "complex_feature_to_impl_speckit",
			input:        "implement complex new GraphQL API layer with auth, rate limiting, and caching across multiple services",
			issueBody:    "",
			wantPipeline: "impl-speckit",
			wantDomain:   DomainFeature,
		},
		{
			name:         "docs_typo_to_doc_explain",
			input:        "fix typo in README",
			issueBody:    "",
			wantPipeline: "doc-explain",
			wantDomain:   DomainDocs,
		},
		{
			name:         "pr_url_to_ops_pr_review",
			input:        "https://github.com/org/repo/pull/42",
			issueBody:    "",
			wantPipeline: "ops-pr-review",
			wantDomain:   DomainFeature,
		},
		{
			name:         "security_to_audit_security",
			input:        "fix SQL injection vulnerability",
			issueBody:    "",
			wantPipeline: "audit-security",
			wantDomain:   DomainSecurity,
		},
		{
			name:         "research_to_plan_research",
			input:        "investigate caching strategies and compare options",
			issueBody:    "",
			wantPipeline: "plan-research",
			wantDomain:   DomainResearch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := Classify(tt.input, tt.issueBody)
			if profile.Domain != tt.wantDomain {
				t.Errorf("Classify().Domain = %q, want %q", profile.Domain, tt.wantDomain)
			}

			cfg := SelectPipeline(profile)
			if cfg.Name != tt.wantPipeline {
				t.Errorf("SelectPipeline() = %q, want %q (profile: %+v)", cfg.Name, tt.wantPipeline, profile)
			}
			if cfg.Reason == "" {
				t.Error("SelectPipeline() Reason is empty")
			}
		})
	}
}

func BenchmarkClassify(b *testing.B) {
	inputs := []struct {
		input     string
		issueBody string
	}{
		{"fix null pointer in logger", ""},
		{"redesign the entire pipeline architecture across multiple packages", ""},
		{"fix SQL injection vulnerability in user endpoint", "security audit required"},
		{"", ""},
		{"https://github.com/org/repo/issues/42", "login button doesn't work"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		inp := inputs[i%len(inputs)]
		Classify(inp.input, inp.issueBody)
	}
}

func TestClassifyPerformance(t *testing.T) {
	start := time.Now()
	for i := 0; i < 1000; i++ {
		Classify("fix SQL injection vulnerability in user endpoint", "security audit")
	}
	elapsed := time.Since(start)
	perOp := elapsed / 1000
	if perOp > time.Millisecond {
		t.Errorf("classification took %v per op, want < 1ms", perOp)
	}
}
