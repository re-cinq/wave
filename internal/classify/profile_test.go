package classify

import (
	"testing"

	"github.com/recinq/wave/internal/suggest"
)

func TestComplexityConstants(t *testing.T) {
	tests := []struct {
		name     string
		c        Complexity
		expected string
	}{
		{"simple", ComplexitySimple, "simple"},
		{"medium", ComplexityMedium, "medium"},
		{"complex", ComplexityComplex, "complex"},
		{"architectural", ComplexityArchitectural, "architectural"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.c) != tt.expected {
				t.Errorf("got %q, want %q", tt.c, tt.expected)
			}
		})
	}
}

func TestDomainConstants(t *testing.T) {
	tests := []struct {
		name     string
		d        Domain
		expected string
	}{
		{"security", DomainSecurity, "security"},
		{"performance", DomainPerformance, "performance"},
		{"bug", DomainBug, "bug"},
		{"refactor", DomainRefactor, "refactor"},
		{"feature", DomainFeature, "feature"},
		{"docs", DomainDocs, "docs"},
		{"research", DomainResearch, "research"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.d) != tt.expected {
				t.Errorf("got %q, want %q", tt.d, tt.expected)
			}
		})
	}
}

func TestVerificationDepthConstants(t *testing.T) {
	tests := []struct {
		name     string
		v        VerificationDepth
		expected string
	}{
		{"structural_only", VerificationStructuralOnly, "structural_only"},
		{"behavioral", VerificationBehavioral, "behavioral"},
		{"full_semantic", VerificationFullSemantic, "full_semantic"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.v) != tt.expected {
				t.Errorf("got %q, want %q", tt.v, tt.expected)
			}
		})
	}
}

func TestTaskProfileZeroValue(t *testing.T) {
	var p TaskProfile
	if p.BlastRadius != 0 {
		t.Errorf("zero BlastRadius: got %f, want 0", p.BlastRadius)
	}
	if p.Complexity != "" {
		t.Errorf("zero Complexity: got %q, want empty", p.Complexity)
	}
	if p.Domain != "" {
		t.Errorf("zero Domain: got %q, want empty", p.Domain)
	}
	if p.VerificationDepth != "" {
		t.Errorf("zero VerificationDepth: got %q, want empty", p.VerificationDepth)
	}
	if p.InputType != "" {
		t.Errorf("zero InputType: got %q, want empty", p.InputType)
	}
}

func TestPipelineConfigFields(t *testing.T) {
	cfg := PipelineConfig{
		Name:              "impl-issue",
		Reason:            "simple bug fix",
		VerificationDepth: VerificationBehavioral,
	}
	if cfg.Name != "impl-issue" {
		t.Errorf("Name: got %q, want %q", cfg.Name, "impl-issue")
	}
	if cfg.Reason != "simple bug fix" {
		t.Errorf("Reason: got %q, want %q", cfg.Reason, "simple bug fix")
	}
	if cfg.VerificationDepth != VerificationBehavioral {
		t.Errorf("VerificationDepth: got %q, want %q", cfg.VerificationDepth, VerificationBehavioral)
	}
}

func TestTaskProfileFieldAssignment(t *testing.T) {
	p := TaskProfile{
		BlastRadius:       0.7,
		Complexity:        ComplexityComplex,
		Domain:            DomainSecurity,
		VerificationDepth: VerificationFullSemantic,
		InputType:         suggest.InputTypeIssueURL,
	}
	if p.BlastRadius != 0.7 {
		t.Errorf("BlastRadius: got %f, want 0.7", p.BlastRadius)
	}
	if p.Complexity != ComplexityComplex {
		t.Errorf("Complexity: got %q, want %q", p.Complexity, ComplexityComplex)
	}
	if p.Domain != DomainSecurity {
		t.Errorf("Domain: got %q, want %q", p.Domain, DomainSecurity)
	}
	if p.VerificationDepth != VerificationFullSemantic {
		t.Errorf("VerificationDepth: got %q, want %q", p.VerificationDepth, VerificationFullSemantic)
	}
	if p.InputType != suggest.InputTypeIssueURL {
		t.Errorf("InputType: got %q, want %q", p.InputType, suggest.InputTypeIssueURL)
	}
}
