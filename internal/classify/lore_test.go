package classify

import "testing"

func TestNoOpLoreProvider(t *testing.T) {
	p := NoOpLoreProvider{}
	ctx := p.GetTaskContext("fix a bug")
	if len(ctx.Hints) != 0 {
		t.Errorf("NoOpLoreProvider.GetTaskContext() hints = %v, want empty", ctx.Hints)
	}
	if len(ctx.Conventions) != 0 {
		t.Errorf("NoOpLoreProvider.GetTaskContext() conventions = %v, want empty", ctx.Conventions)
	}
	if len(ctx.Memories) != 0 {
		t.Errorf("NoOpLoreProvider.GetTaskContext() memories = %v, want empty", ctx.Memories)
	}
}

func TestRegisterLoreProvider(t *testing.T) {
	orig := activeLoreProvider
	defer func() { activeLoreProvider = orig }()

	// Register a custom provider that returns a security hint.
	RegisterLoreProvider(&stubLoreProvider{
		domain:     DomainSecurity,
		complexity: ComplexityComplex,
		confidence: 0.8,
	})

	ctx := loreProvider().GetTaskContext("anything")
	if len(ctx.Hints) != 1 {
		t.Fatalf("expected 1 hint, got %d", len(ctx.Hints))
	}
	if ctx.Hints[0].Domain != DomainSecurity {
		t.Errorf("hint domain = %q, want %q", ctx.Hints[0].Domain, DomainSecurity)
	}
	if ctx.Hints[0].Confidence != 0.8 {
		t.Errorf("hint confidence = %f, want 0.8", ctx.Hints[0].Confidence)
	}

	// Register nil reverts to no-op.
	RegisterLoreProvider(nil)
	ctx = loreProvider().GetTaskContext("anything")
	if len(ctx.Hints) != 0 {
		t.Errorf("after nil register, Hints = %v, want empty", ctx.Hints)
	}
}

func TestLoreEnrichesClassify(t *testing.T) {
	orig := activeLoreProvider
	defer func() { activeLoreProvider = orig }()

	// Without lore: "do something" → DomainFeature (no keyword match).
	base := Classify("do something", "")
	if base.Domain != DomainFeature {
		t.Fatalf("base domain = %q, want %q", base.Domain, DomainFeature)
	}

	// With lore hint for security at high confidence.
	RegisterLoreProvider(&stubLoreProvider{
		domain:     DomainSecurity,
		complexity: "",
		confidence: 0.9,
	})

	enriched := Classify("do something", "")
	if enriched.Domain != DomainSecurity {
		t.Errorf("enriched domain = %q, want %q (lore should enrich fallback)", enriched.Domain, DomainSecurity)
	}
}

func TestLoreDoesNotOverrideKeywords(t *testing.T) {
	orig := activeLoreProvider
	defer func() { activeLoreProvider = orig }()

	// "fix a bug" → DomainBug via keywords.
	RegisterLoreProvider(&stubLoreProvider{
		domain:     DomainDocs,
		complexity: ComplexitySimple,
		confidence: 0.9,
	})

	profile := Classify("fix a bug", "")
	if profile.Domain != DomainBug {
		t.Errorf("domain = %q, want %q (keywords should win over lore)", profile.Domain, DomainBug)
	}
}

func TestTaskContextStructure(t *testing.T) {
	p := &fullLoreProvider{}
	ctx := p.GetTaskContext("implement auth")

	if len(ctx.Hints) != 1 {
		t.Fatalf("expected 1 hint, got %d", len(ctx.Hints))
	}
	if len(ctx.Conventions) != 1 {
		t.Fatalf("expected 1 convention, got %d", len(ctx.Conventions))
	}
	if ctx.Conventions[0] != "all auth changes require security review" {
		t.Errorf("convention = %q", ctx.Conventions[0])
	}
	if len(ctx.Memories) != 1 {
		t.Fatalf("expected 1 memory, got %d", len(ctx.Memories))
	}
	if ctx.Memories[0].Source != "memory" {
		t.Errorf("memory source = %q, want %q", ctx.Memories[0].Source, "memory")
	}
}

// stubLoreProvider returns a single hint with fixed values.
type stubLoreProvider struct {
	domain     Domain
	complexity Complexity
	confidence float64
}

func (s *stubLoreProvider) GetTaskContext(string) TaskContext {
	return TaskContext{
		Hints: []LoreHint{{
			Domain:     s.domain,
			Complexity: s.complexity,
			Confidence: s.confidence,
			Source:     "test_stub",
		}},
	}
}

// fullLoreProvider returns context with hints, conventions, and memories.
type fullLoreProvider struct{}

func (f *fullLoreProvider) GetTaskContext(string) TaskContext {
	return TaskContext{
		Hints: []LoreHint{{
			Domain:     DomainSecurity,
			Complexity: ComplexityComplex,
			Confidence: 0.85,
			Source:     "memory",
		}},
		Conventions: []string{"all auth changes require security review"},
		Memories: []MemoryResult{{
			Key:    "auth-pattern",
			Value:  "Last auth implementation used JWT with refresh tokens",
			Score:  0.92,
			Source: "memory",
		}},
	}
}
