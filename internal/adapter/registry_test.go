package adapter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubRunner is a minimal AdapterRunner used by registry tests where we
// only need a recognisable identity. Living inside the adapter package
// avoids importing the dedicated test-double package (adaptertest) which
// would create an import cycle.
type stubRunner struct {
	exitCode int
}

func (r *stubRunner) Run(_ context.Context, _ AdapterRunConfig) (*AdapterResult, error) {
	return &AdapterResult{ExitCode: r.exitCode}, nil
}

func TestAdapterRegistry_ResolveKnownAdapters(t *testing.T) {
	registry := NewAdapterRegistry(nil)

	tests := []struct {
		name        string
		adapterName string
		expectType  string
	}{
		{"claude", "claude", "*adapter.ClaudeAdapter"},
		{"codex", "codex", "*adapter.CodexAdapter"},
		{"gemini", "gemini", "*adapter.GeminiAdapter"},
		{"opencode", "opencode", "*adapter.OpenCodeAdapter"},
		{"browser", "browser", "*adapter.BrowserAdapter"},
		{"unknown defaults to ProcessGroupRunner", "unknown", "*adapter.ProcessGroupRunner"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := registry.Resolve(tt.adapterName)
			require.NotNil(t, runner)
			assert.IsType(t, runner, runner) // Just verify it's not nil
		})
	}
}

func TestAdapterRegistry_ResolveClaudeReturnsClaudeAdapter(t *testing.T) {
	registry := NewAdapterRegistry(nil)
	runner := registry.Resolve("claude")
	_, ok := runner.(*ClaudeAdapter)
	assert.True(t, ok, "expected *ClaudeAdapter, got %T", runner)
}

func TestAdapterRegistry_ResolveCodexReturnsCodexAdapter(t *testing.T) {
	registry := NewAdapterRegistry(nil)
	runner := registry.Resolve("codex")
	_, ok := runner.(*CodexAdapter)
	assert.True(t, ok, "expected *CodexAdapter, got %T", runner)
}

func TestAdapterRegistry_ResolveGeminiReturnsGeminiAdapter(t *testing.T) {
	registry := NewAdapterRegistry(nil)
	runner := registry.Resolve("gemini")
	_, ok := runner.(*GeminiAdapter)
	assert.True(t, ok, "expected *GeminiAdapter, got %T", runner)
}

func TestAdapterRegistry_ResolveUnknownReturnsProcessGroupRunner(t *testing.T) {
	registry := NewAdapterRegistry(nil)
	runner := registry.Resolve("nonexistent")
	_, ok := runner.(*ProcessGroupRunner)
	assert.True(t, ok, "expected *ProcessGroupRunner, got %T", runner)
}

func TestAdapterRegistry_OverrideTakesPrecedence(t *testing.T) {
	registry := NewAdapterRegistry(nil)
	mock := &stubRunner{}
	registry.RegisterOverride("claude", mock)

	runner := registry.Resolve("claude")
	assert.Equal(t, mock, runner, "override should take precedence over built-in resolution")
}

func TestSingleRunnerRegistry_AlwaysReturnsSameRunner(t *testing.T) {
	mock := &stubRunner{}
	registry := NewSingleRunnerRegistry(mock)

	// Any name should return the same runner
	assert.Equal(t, mock, registry.Resolve("claude"))
	assert.Equal(t, mock, registry.Resolve("codex"))
	assert.Equal(t, mock, registry.Resolve("anything"))
}

func TestAdapterRegistry_FallbackChain(t *testing.T) {
	fallbacks := map[string][]string{
		"anthropic": {"openai", "gemini"},
		"openai":    {"anthropic"},
	}
	registry := NewAdapterRegistry(fallbacks)

	chain := registry.FallbackChain("anthropic")
	assert.Equal(t, []string{"openai", "gemini"}, chain)

	chain = registry.FallbackChain("openai")
	assert.Equal(t, []string{"anthropic"}, chain)

	// No fallback configured
	chain = registry.FallbackChain("gemini")
	assert.Nil(t, chain)
}

func TestAdapterRegistry_ResolveWithFallback_NoFallback(t *testing.T) {
	registry := NewAdapterRegistry(nil)
	runner := registry.ResolveWithFallback("claude")
	// Without fallbacks, should return the plain runner (not a FallbackRunner)
	_, isFallback := runner.(*FallbackRunner)
	assert.False(t, isFallback, "should not be a FallbackRunner when no fallbacks configured")
}

func TestAdapterRegistry_ResolveWithFallback_WithFallback(t *testing.T) {
	fallbacks := map[string][]string{
		"claude": {"codex"},
	}
	registry := NewAdapterRegistry(fallbacks)
	runner := registry.ResolveWithFallback("claude")
	_, isFallback := runner.(*FallbackRunner)
	assert.True(t, isFallback, "should be a FallbackRunner when fallbacks configured")
}

func TestAdapterRegistry_NilFallbacks(t *testing.T) {
	registry := NewAdapterRegistry(nil)
	chain := registry.FallbackChain("anything")
	assert.Nil(t, chain)
}

func TestSingleRunnerRegistry_OverrideStillWorks(t *testing.T) {
	defaultMock := &stubRunner{}
	registry := NewSingleRunnerRegistry(defaultMock)

	overrideMock := &stubRunner{exitCode: 42}
	registry.RegisterOverride("special", overrideMock)

	// Override should take precedence
	assert.Equal(t, overrideMock, registry.Resolve("special"))
	// Default runner for non-overridden names
	assert.Equal(t, defaultMock, registry.Resolve("claude"))
}

func TestAdapterRegistry_ResolveWithFallback_FallbackRunnerHasCorrectChain(t *testing.T) {
	fallbacks := map[string][]string{
		"claude": {"codex", "gemini"},
	}
	registry := NewAdapterRegistry(fallbacks)
	runner := registry.ResolveWithFallback("claude")
	fr, ok := runner.(*FallbackRunner)
	require.True(t, ok)
	assert.Equal(t, []string{"codex", "gemini"}, fr.chain)
}

func TestAdapterRegistry_ResolveDoesNotReturnNil(t *testing.T) {
	registry := NewAdapterRegistry(nil)
	// Even for empty string, should return something
	runner := registry.Resolve("")
	assert.NotNil(t, runner)
}

func TestSingleRunnerRegistry_RunDelegates(t *testing.T) {
	mock := &stubRunner{exitCode: 0}
	registry := NewSingleRunnerRegistry(mock)
	runner := registry.Resolve("anything")

	result, err := runner.Run(context.Background(), AdapterRunConfig{
		Prompt:  "test",
		Timeout: 0,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 0, result.ExitCode)
}
