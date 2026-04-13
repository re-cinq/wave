package classify

import "testing"

func TestNoOpLoreProvider(t *testing.T) {
	p := NoOpLoreProvider{}
	hints := p.Hints("fix a bug")
	if hints != nil {
		t.Errorf("NoOpLoreProvider.Hints() = %v, want nil", hints)
	}
}

func TestRegisterLoreProvider(t *testing.T) {
	// Save and restore original provider.
	orig := activeLoreProvider
	defer func() { activeLoreProvider = orig }()

	// Register a custom provider that always returns a security hint.
	RegisterLoreProvider(&stubLoreProvider{
		domain:     DomainSecurity,
		complexity: ComplexityComplex,
		confidence: 0.8,
	})

	hints := loreProvider().Hints("anything")
	if len(hints) != 1 {
		t.Fatalf("expected 1 hint, got %d", len(hints))
	}
	if hints[0].Domain != DomainSecurity {
		t.Errorf("hint domain = %q, want %q", hints[0].Domain, DomainSecurity)
	}
	if hints[0].Confidence != 0.8 {
		t.Errorf("hint confidence = %f, want 0.8", hints[0].Confidence)
	}

	// Register nil reverts to no-op.
	RegisterLoreProvider(nil)
	hints = loreProvider().Hints("anything")
	if hints != nil {
		t.Errorf("after nil register, Hints() = %v, want nil", hints)
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

// stubLoreProvider returns a single hint with fixed values.
type stubLoreProvider struct {
	domain     Domain
	complexity Complexity
	confidence float64
}

func (s *stubLoreProvider) Hints(string) []LoreHint {
	return []LoreHint{{
		Domain:     s.domain,
		Complexity: s.complexity,
		Confidence: s.confidence,
		Source:     "test_stub",
	}}
}
