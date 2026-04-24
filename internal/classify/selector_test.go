package classify

import (
	"testing"

	"github.com/recinq/wave/internal/suggest"
)

func TestSelectPipeline(t *testing.T) {
	tests := []struct {
		name         string
		profile      TaskProfile
		wantPipeline string
		wantDepth    VerificationDepth
	}{
		{
			name:         "pr_url_short_circuits",
			profile:      TaskProfile{InputType: suggest.InputTypePRURL, Domain: DomainBug, Complexity: ComplexityComplex, VerificationDepth: VerificationFullSemantic},
			wantPipeline: "ops-pr-review",
			wantDepth:    VerificationFullSemantic,
		},
		{
			name:         "security_simple",
			profile:      TaskProfile{Domain: DomainSecurity, Complexity: ComplexitySimple, VerificationDepth: VerificationStructuralOnly},
			wantPipeline: "audit-security",
			wantDepth:    VerificationStructuralOnly,
		},
		{
			name:         "security_complex",
			profile:      TaskProfile{Domain: DomainSecurity, Complexity: ComplexityComplex, VerificationDepth: VerificationFullSemantic},
			wantPipeline: "audit-security",
			wantDepth:    VerificationFullSemantic,
		},
		{
			name:         "security_architectural",
			profile:      TaskProfile{Domain: DomainSecurity, Complexity: ComplexityArchitectural, VerificationDepth: VerificationFullSemantic},
			wantPipeline: "audit-security",
			wantDepth:    VerificationFullSemantic,
		},
		{
			name:         "research",
			profile:      TaskProfile{Domain: DomainResearch, Complexity: ComplexityMedium, VerificationDepth: VerificationBehavioral},
			wantPipeline: "plan-research",
			wantDepth:    VerificationBehavioral,
		},
		{
			name:         "docs",
			profile:      TaskProfile{Domain: DomainDocs, Complexity: ComplexitySimple, VerificationDepth: VerificationStructuralOnly},
			wantPipeline: "doc-explain",
			wantDepth:    VerificationStructuralOnly,
		},
		{
			name:         "refactor_complex",
			profile:      TaskProfile{Domain: DomainRefactor, Complexity: ComplexityComplex, VerificationDepth: VerificationFullSemantic},
			wantPipeline: "impl-speckit",
			wantDepth:    VerificationFullSemantic,
		},
		{
			name:         "refactor_architectural",
			profile:      TaskProfile{Domain: DomainRefactor, Complexity: ComplexityArchitectural, VerificationDepth: VerificationFullSemantic},
			wantPipeline: "impl-speckit",
			wantDepth:    VerificationFullSemantic,
		},
		{
			name:         "refactor_simple",
			profile:      TaskProfile{Domain: DomainRefactor, Complexity: ComplexitySimple, VerificationDepth: VerificationStructuralOnly},
			wantPipeline: "impl-issue",
			wantDepth:    VerificationStructuralOnly,
		},
		{
			name:         "refactor_medium",
			profile:      TaskProfile{Domain: DomainRefactor, Complexity: ComplexityMedium, VerificationDepth: VerificationBehavioral},
			wantPipeline: "impl-issue",
			wantDepth:    VerificationBehavioral,
		},
		{
			name:         "simple_bug",
			profile:      TaskProfile{Domain: DomainBug, Complexity: ComplexitySimple, VerificationDepth: VerificationStructuralOnly},
			wantPipeline: "impl-issue",
			wantDepth:    VerificationStructuralOnly,
		},
		{
			name:         "medium_feature",
			profile:      TaskProfile{Domain: DomainFeature, Complexity: ComplexityMedium, VerificationDepth: VerificationBehavioral},
			wantPipeline: "impl-issue",
			wantDepth:    VerificationBehavioral,
		},
		{
			name:         "complex_feature",
			profile:      TaskProfile{Domain: DomainFeature, Complexity: ComplexityComplex, VerificationDepth: VerificationFullSemantic},
			wantPipeline: "impl-speckit",
			wantDepth:    VerificationFullSemantic,
		},
		{
			name:         "architectural_feature",
			profile:      TaskProfile{Domain: DomainFeature, Complexity: ComplexityArchitectural, VerificationDepth: VerificationFullSemantic},
			wantPipeline: "impl-speckit",
			wantDepth:    VerificationFullSemantic,
		},
		{
			name:         "medium_bug",
			profile:      TaskProfile{Domain: DomainBug, Complexity: ComplexityMedium, VerificationDepth: VerificationBehavioral},
			wantPipeline: "impl-issue",
			wantDepth:    VerificationBehavioral,
		},
		{
			name:         "complex_bug",
			profile:      TaskProfile{Domain: DomainBug, Complexity: ComplexityComplex, VerificationDepth: VerificationFullSemantic},
			wantPipeline: "impl-speckit",
			wantDepth:    VerificationFullSemantic,
		},
		// New domain routing tests
		{
			name:         "simple_bug_hotfix",
			profile:      TaskProfile{Domain: DomainBug, Complexity: ComplexitySimple, VerificationDepth: VerificationStructuralOnly},
			wantPipeline: "impl-issue",
			wantDepth:    VerificationStructuralOnly,
		},
		{
			name:         "debug_simple",
			profile:      TaskProfile{Domain: DomainDebug, Complexity: ComplexitySimple, VerificationDepth: VerificationStructuralOnly},
			wantPipeline: "impl-issue",
			wantDepth:    VerificationStructuralOnly,
		},
		{
			name:         "debug_complex",
			profile:      TaskProfile{Domain: DomainDebug, Complexity: ComplexityComplex, VerificationDepth: VerificationFullSemantic},
			wantPipeline: "impl-speckit",
			wantDepth:    VerificationFullSemantic,
		},
		{
			name:         "review",
			profile:      TaskProfile{Domain: DomainReview, Complexity: ComplexityMedium, VerificationDepth: VerificationBehavioral},
			wantPipeline: "ops-pr-review",
			wantDepth:    VerificationBehavioral,
		},
		{
			name:         "testing",
			profile:      TaskProfile{Domain: DomainTesting, Complexity: ComplexityMedium, VerificationDepth: VerificationBehavioral},
			wantPipeline: "audit-tests",
			wantDepth:    VerificationBehavioral,
		},
		{
			name:         "audit_simple",
			profile:      TaskProfile{Domain: DomainAudit, Complexity: ComplexitySimple, VerificationDepth: VerificationStructuralOnly},
			wantPipeline: "audit-architecture",
			wantDepth:    VerificationStructuralOnly,
		},
		{
			name:         "audit_complex",
			profile:      TaskProfile{Domain: DomainAudit, Complexity: ComplexityComplex, VerificationDepth: VerificationFullSemantic},
			wantPipeline: "ops-parallel-audit",
			wantDepth:    VerificationFullSemantic,
		},
		{
			name:         "planning_simple",
			profile:      TaskProfile{Domain: DomainPlanning, Complexity: ComplexitySimple, VerificationDepth: VerificationStructuralOnly},
			wantPipeline: "plan-task",
			wantDepth:    VerificationStructuralOnly,
		},
		{
			name:         "planning_complex",
			profile:      TaskProfile{Domain: DomainPlanning, Complexity: ComplexityComplex, VerificationDepth: VerificationFullSemantic},
			wantPipeline: "plan-scope",
			wantDepth:    VerificationFullSemantic,
		},
		{
			name:         "ops_simple",
			profile:      TaskProfile{Domain: DomainOps, Complexity: ComplexitySimple, VerificationDepth: VerificationStructuralOnly},
			wantPipeline: "ops-bootstrap",
			wantDepth:    VerificationStructuralOnly,
		},
		{
			name:         "ops_complex",
			profile:      TaskProfile{Domain: DomainOps, Complexity: ComplexityComplex, VerificationDepth: VerificationFullSemantic},
			wantPipeline: "ops-epic-runner",
			wantDepth:    VerificationFullSemantic,
		},
		{
			name:         "performance",
			profile:      TaskProfile{Domain: DomainPerformance, Complexity: ComplexityMedium, VerificationDepth: VerificationBehavioral},
			wantPipeline: "impl-issue",
			wantDepth:    VerificationBehavioral,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SelectPipeline(tt.profile)
			if got.Name != tt.wantPipeline {
				t.Errorf("SelectPipeline() Name = %q, want %q", got.Name, tt.wantPipeline)
			}
			if got.Reason == "" {
				t.Error("SelectPipeline() Reason is empty, want non-empty")
			}
			if got.VerificationDepth != tt.wantDepth {
				t.Errorf("SelectPipeline() VerificationDepth = %q, want %q", got.VerificationDepth, tt.wantDepth)
			}
		})
	}
}
