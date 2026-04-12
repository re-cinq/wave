package classify

import (
	"testing"

	"github.com/recinq/wave/internal/pipeline"
	"github.com/recinq/wave/internal/suggest"
)

func TestModelTierForComplexity(t *testing.T) {
	tests := []struct {
		name       string
		complexity Complexity
		wantImpl   string
		wantNav    string
	}{
		{"simple", ComplexitySimple, pipeline.TierCheapest, pipeline.TierCheapest},
		{"medium", ComplexityMedium, pipeline.TierBalanced, pipeline.TierCheapest},
		{"complex", ComplexityComplex, pipeline.TierStrongest, pipeline.TierCheapest},
		{"architectural", ComplexityArchitectural, pipeline.TierStrongest, pipeline.TierStrongest},
		{"unknown_fallthrough", Complexity("unknown"), pipeline.TierBalanced, pipeline.TierCheapest},
		{"empty_fallthrough", Complexity(""), pipeline.TierBalanced, pipeline.TierCheapest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := modelTierForComplexity(tt.complexity)
			if got.Impl != tt.wantImpl {
				t.Errorf("Impl = %q, want %q", got.Impl, tt.wantImpl)
			}
			if got.Nav != tt.wantNav {
				t.Errorf("Nav = %q, want %q", got.Nav, tt.wantNav)
			}
		})
	}
}

func TestSelectPipeline(t *testing.T) {
	tests := []struct {
		name          string
		profile       TaskProfile
		wantPipeline  string
		wantModelTier ModelTierMap
	}{
		{
			name:          "pr_url_short_circuits",
			profile:       TaskProfile{InputType: suggest.InputTypePRURL, Domain: DomainBug, Complexity: ComplexityComplex},
			wantPipeline:  "ops-pr-review",
			wantModelTier: ModelTierMap{Impl: pipeline.TierStrongest, Nav: pipeline.TierCheapest},
		},
		{
			name:          "security_simple",
			profile:       TaskProfile{Domain: DomainSecurity, Complexity: ComplexitySimple},
			wantPipeline:  "audit-security",
			wantModelTier: ModelTierMap{Impl: pipeline.TierCheapest, Nav: pipeline.TierCheapest},
		},
		{
			name:          "security_complex",
			profile:       TaskProfile{Domain: DomainSecurity, Complexity: ComplexityComplex},
			wantPipeline:  "audit-security",
			wantModelTier: ModelTierMap{Impl: pipeline.TierStrongest, Nav: pipeline.TierCheapest},
		},
		{
			name:          "security_architectural",
			profile:       TaskProfile{Domain: DomainSecurity, Complexity: ComplexityArchitectural},
			wantPipeline:  "audit-security",
			wantModelTier: ModelTierMap{Impl: pipeline.TierStrongest, Nav: pipeline.TierStrongest},
		},
		{
			name:          "research",
			profile:       TaskProfile{Domain: DomainResearch, Complexity: ComplexityMedium},
			wantPipeline:  "impl-research",
			wantModelTier: ModelTierMap{Impl: pipeline.TierBalanced, Nav: pipeline.TierCheapest},
		},
		{
			name:          "docs",
			profile:       TaskProfile{Domain: DomainDocs, Complexity: ComplexitySimple},
			wantPipeline:  "doc-fix",
			wantModelTier: ModelTierMap{Impl: pipeline.TierCheapest, Nav: pipeline.TierCheapest},
		},
		{
			name:          "refactor_complex",
			profile:       TaskProfile{Domain: DomainRefactor, Complexity: ComplexityComplex},
			wantPipeline:  "impl-speckit",
			wantModelTier: ModelTierMap{Impl: pipeline.TierStrongest, Nav: pipeline.TierCheapest},
		},
		{
			name:          "refactor_architectural",
			profile:       TaskProfile{Domain: DomainRefactor, Complexity: ComplexityArchitectural},
			wantPipeline:  "impl-speckit",
			wantModelTier: ModelTierMap{Impl: pipeline.TierStrongest, Nav: pipeline.TierStrongest},
		},
		{
			name:          "refactor_simple",
			profile:       TaskProfile{Domain: DomainRefactor, Complexity: ComplexitySimple},
			wantPipeline:  "impl-issue",
			wantModelTier: ModelTierMap{Impl: pipeline.TierCheapest, Nav: pipeline.TierCheapest},
		},
		{
			name:          "refactor_medium",
			profile:       TaskProfile{Domain: DomainRefactor, Complexity: ComplexityMedium},
			wantPipeline:  "impl-issue",
			wantModelTier: ModelTierMap{Impl: pipeline.TierBalanced, Nav: pipeline.TierCheapest},
		},
		{
			name:          "simple_bug",
			profile:       TaskProfile{Domain: DomainBug, Complexity: ComplexitySimple},
			wantPipeline:  "impl-issue",
			wantModelTier: ModelTierMap{Impl: pipeline.TierCheapest, Nav: pipeline.TierCheapest},
		},
		{
			name:          "medium_feature",
			profile:       TaskProfile{Domain: DomainFeature, Complexity: ComplexityMedium},
			wantPipeline:  "impl-issue",
			wantModelTier: ModelTierMap{Impl: pipeline.TierBalanced, Nav: pipeline.TierCheapest},
		},
		{
			name:          "complex_feature",
			profile:       TaskProfile{Domain: DomainFeature, Complexity: ComplexityComplex},
			wantPipeline:  "impl-speckit",
			wantModelTier: ModelTierMap{Impl: pipeline.TierStrongest, Nav: pipeline.TierCheapest},
		},
		{
			name:          "architectural_feature",
			profile:       TaskProfile{Domain: DomainFeature, Complexity: ComplexityArchitectural},
			wantPipeline:  "impl-speckit",
			wantModelTier: ModelTierMap{Impl: pipeline.TierStrongest, Nav: pipeline.TierStrongest},
		},
		{
			name:          "medium_bug",
			profile:       TaskProfile{Domain: DomainBug, Complexity: ComplexityMedium},
			wantPipeline:  "impl-issue",
			wantModelTier: ModelTierMap{Impl: pipeline.TierBalanced, Nav: pipeline.TierCheapest},
		},
		{
			name:          "complex_bug",
			profile:       TaskProfile{Domain: DomainBug, Complexity: ComplexityComplex},
			wantPipeline:  "impl-speckit",
			wantModelTier: ModelTierMap{Impl: pipeline.TierStrongest, Nav: pipeline.TierCheapest},
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
			if got.ModelTier != tt.wantModelTier {
				t.Errorf("SelectPipeline() ModelTier = %+v, want %+v", got.ModelTier, tt.wantModelTier)
			}
		})
	}
}
