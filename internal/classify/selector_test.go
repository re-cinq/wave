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
	}{
		{
			name:         "pr_url_short_circuits",
			profile:      TaskProfile{InputType: suggest.InputTypePRURL, Domain: DomainBug, Complexity: ComplexityComplex},
			wantPipeline: "ops-pr-review",
		},
		{
			name:         "security_simple",
			profile:      TaskProfile{Domain: DomainSecurity, Complexity: ComplexitySimple},
			wantPipeline: "audit-security",
		},
		{
			name:         "security_complex",
			profile:      TaskProfile{Domain: DomainSecurity, Complexity: ComplexityComplex},
			wantPipeline: "audit-security",
		},
		{
			name:         "security_architectural",
			profile:      TaskProfile{Domain: DomainSecurity, Complexity: ComplexityArchitectural},
			wantPipeline: "audit-security",
		},
		{
			name:         "research",
			profile:      TaskProfile{Domain: DomainResearch, Complexity: ComplexityMedium},
			wantPipeline: "impl-research",
		},
		{
			name:         "docs",
			profile:      TaskProfile{Domain: DomainDocs, Complexity: ComplexitySimple},
			wantPipeline: "doc-fix",
		},
		{
			name:         "refactor_complex",
			profile:      TaskProfile{Domain: DomainRefactor, Complexity: ComplexityComplex},
			wantPipeline: "impl-speckit",
		},
		{
			name:         "refactor_architectural",
			profile:      TaskProfile{Domain: DomainRefactor, Complexity: ComplexityArchitectural},
			wantPipeline: "impl-speckit",
		},
		{
			name:         "refactor_simple",
			profile:      TaskProfile{Domain: DomainRefactor, Complexity: ComplexitySimple},
			wantPipeline: "impl-issue",
		},
		{
			name:         "refactor_medium",
			profile:      TaskProfile{Domain: DomainRefactor, Complexity: ComplexityMedium},
			wantPipeline: "impl-issue",
		},
		{
			name:         "simple_bug",
			profile:      TaskProfile{Domain: DomainBug, Complexity: ComplexitySimple},
			wantPipeline: "impl-issue",
		},
		{
			name:         "medium_feature",
			profile:      TaskProfile{Domain: DomainFeature, Complexity: ComplexityMedium},
			wantPipeline: "impl-issue",
		},
		{
			name:         "complex_feature",
			profile:      TaskProfile{Domain: DomainFeature, Complexity: ComplexityComplex},
			wantPipeline: "impl-speckit",
		},
		{
			name:         "architectural_feature",
			profile:      TaskProfile{Domain: DomainFeature, Complexity: ComplexityArchitectural},
			wantPipeline: "impl-speckit",
		},
		{
			name:         "medium_bug",
			profile:      TaskProfile{Domain: DomainBug, Complexity: ComplexityMedium},
			wantPipeline: "impl-issue",
		},
		{
			name:         "complex_bug",
			profile:      TaskProfile{Domain: DomainBug, Complexity: ComplexityComplex},
			wantPipeline: "impl-speckit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SelectPipeline(tt.profile)
			if got.Name != tt.wantPipeline {
				t.Errorf("SelectPipeline() = %q, want %q", got.Name, tt.wantPipeline)
			}
			if got.Reason == "" {
				t.Error("SelectPipeline() Reason is empty, want non-empty")
			}
		})
	}
}
