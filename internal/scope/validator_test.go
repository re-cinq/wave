package scope

import (
	"fmt"
	"testing"

	"github.com/recinq/wave/internal/forge"
)

// mockIntrospector returns fixed results for testing.
type mockIntrospector struct {
	results map[string]*TokenInfo
}

func (m *mockIntrospector) Introspect(envVar string) (*TokenInfo, error) {
	if info, ok := m.results[envVar]; ok {
		return info, nil
	}
	return &TokenInfo{
		EnvVar: envVar,
		Error:  nil,
		Scopes: []string{},
	}, nil
}

func TestValidatePersonas_AllScopesSatisfied(t *testing.T) {
	resolver := NewResolver(forge.ForgeGitHub)
	introspector := &mockIntrospector{
		results: map[string]*TokenInfo{
			"GH_TOKEN": {EnvVar: "GH_TOKEN", Scopes: []string{"repo", "read:packages"}, TokenType: "classic"},
		},
	}
	v := NewValidator(resolver, introspector, forge.ForgeInfo{Type: forge.ForgeGitHub}, []string{"GH_TOKEN"})

	personas := map[string][]string{
		"navigator": {"issues:read", "pulls:read"},
	}

	result, err := v.ValidatePersonas(personas)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.HasViolations() {
		t.Errorf("expected no violations, got: %s", result.Error())
	}
}

func TestValidatePersonas_MissingScopes(t *testing.T) {
	resolver := NewResolver(forge.ForgeGitHub)
	introspector := &mockIntrospector{
		results: map[string]*TokenInfo{
			"GH_TOKEN": {EnvVar: "GH_TOKEN", Scopes: []string{"read:packages"}, TokenType: "classic"},
		},
	}
	v := NewValidator(resolver, introspector, forge.ForgeInfo{Type: forge.ForgeGitHub}, []string{"GH_TOKEN"})

	personas := map[string][]string{
		"implementer": {"issues:write", "packages:read"},
	}

	result, err := v.ValidatePersonas(personas)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.HasViolations() {
		t.Fatal("expected violations for missing repo scope")
	}
	// issues:write requires "repo" which is missing
	found := false
	for _, v := range result.Violations {
		if v.MissingScope == "issues:write" && v.PersonaName == "implementer" {
			found = true
		}
	}
	if !found {
		t.Error("expected violation for issues:write on implementer persona")
	}
}

func TestValidatePersonas_NoTokenScopes(t *testing.T) {
	resolver := NewResolver(forge.ForgeGitHub)
	introspector := &mockIntrospector{}
	v := NewValidator(resolver, introspector, forge.ForgeInfo{Type: forge.ForgeGitHub}, nil)

	personas := map[string][]string{
		"navigator": {}, // No token_scopes — skip validation
	}

	result, err := v.ValidatePersonas(personas)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.HasViolations() {
		t.Errorf("expected no violations for persona without token_scopes")
	}
	if len(result.Warnings) > 0 {
		t.Errorf("expected no warnings, got: %v", result.Warnings)
	}
}

func TestValidatePersonas_UnknownForge(t *testing.T) {
	resolver := NewResolver(forge.ForgeUnknown)
	// NewIntrospector returns nil for unknown forge
	v := NewValidator(resolver, nil, forge.ForgeInfo{Type: forge.ForgeUnknown}, nil)

	personas := map[string][]string{
		"navigator": {"issues:read"},
	}

	result, err := v.ValidatePersonas(personas)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.HasViolations() {
		t.Error("expected no violations for unknown forge (should warn and skip)")
	}
	if len(result.Warnings) == 0 {
		t.Error("expected warning for unknown forge type")
	}
}

func TestValidatePersonas_IntrospectionFailure(t *testing.T) {
	resolver := NewResolver(forge.ForgeGitHub)
	introspector := &mockIntrospector{
		results: map[string]*TokenInfo{
			"GH_TOKEN": {
				EnvVar:    "GH_TOKEN",
				TokenType: "unknown",
				Error:     fmt.Errorf("gh api failed"),
			},
		},
	}
	v := NewValidator(resolver, introspector, forge.ForgeInfo{Type: forge.ForgeGitHub}, []string{"GH_TOKEN"})

	personas := map[string][]string{
		"navigator": {"issues:read"},
	}

	result, err := v.ValidatePersonas(personas)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Introspection failure should produce a warning, not a violation
	if result.HasViolations() {
		t.Error("expected no violations when introspection fails (should warn)")
	}
	if len(result.Warnings) == 0 {
		t.Error("expected warning for introspection failure")
	}
}

func TestValidatePersonas_MultiPersonaMixed(t *testing.T) {
	resolver := NewResolver(forge.ForgeGitHub)
	introspector := &mockIntrospector{
		results: map[string]*TokenInfo{
			"GH_TOKEN": {EnvVar: "GH_TOKEN", Scopes: []string{"repo"}, TokenType: "classic"},
		},
	}
	v := NewValidator(resolver, introspector, forge.ForgeInfo{Type: forge.ForgeGitHub}, []string{"GH_TOKEN"})

	personas := map[string][]string{
		"navigator":   {"issues:read"},                    // satisfied by "repo"
		"implementer": {"issues:write", "packages:write"}, // packages:write needs "write:packages"
		"auditor":     {},                                 // no scopes — skip
	}

	result, err := v.ValidatePersonas(personas)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Only implementer's packages:write should fail (requires write:packages)
	if !result.HasViolations() {
		t.Fatal("expected violations for packages:write")
	}
	for _, v := range result.Violations {
		if v.PersonaName != "implementer" || v.MissingScope != "packages:write" {
			// navigator should pass, implementer's issues:write should pass (repo covers it)
			if v.PersonaName == "navigator" {
				t.Errorf("navigator should not have violations")
			}
		}
	}
}

func TestValidatePersonas_EnvPassthroughMissing(t *testing.T) {
	resolver := NewResolver(forge.ForgeGitHub)
	introspector := &mockIntrospector{
		results: map[string]*TokenInfo{
			"GH_TOKEN": {EnvVar: "GH_TOKEN", Scopes: []string{"repo"}, TokenType: "classic"},
		},
	}
	// env_passthrough does NOT include GH_TOKEN
	v := NewValidator(resolver, introspector, forge.ForgeInfo{Type: forge.ForgeGitHub}, []string{"PATH", "HOME"})

	personas := map[string][]string{
		"navigator": {"issues:read"},
	}

	result, err := v.ValidatePersonas(personas)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.HasViolations() {
		t.Fatal("expected violation for missing env_passthrough")
	}
	found := false
	for _, v := range result.Violations {
		if v.PersonaName == "navigator" && v.EnvVar == "GH_TOKEN" {
			found = true
		}
	}
	if !found {
		t.Error("expected violation mentioning GH_TOKEN env_passthrough")
	}
}
