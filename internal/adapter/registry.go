package adapter

import (
	"strings"
)

// AdapterRegistry resolves adapter names to AdapterRunner implementations.
// It replaces the single-runner model with per-step adapter resolution,
// supporting fallback chains for provider resilience.
type AdapterRegistry struct {
	fallbacks     map[string][]string      // provider → fallback providers
	overrides     map[string]AdapterRunner // test-injected runners
	defaultRunner AdapterRunner            // fallback for all names when set
}

// NewAdapterRegistry creates a registry with optional fallback chain configuration.
func NewAdapterRegistry(fallbacks map[string][]string) *AdapterRegistry {
	return &AdapterRegistry{
		fallbacks: fallbacks,
		overrides: make(map[string]AdapterRunner),
	}
}

// NewSingleRunnerRegistry creates a registry that always returns the given runner.
// Used for backward compatibility in tests and simple configurations.
func NewSingleRunnerRegistry(runner AdapterRunner) *AdapterRegistry {
	return &AdapterRegistry{
		overrides:     make(map[string]AdapterRunner),
		defaultRunner: runner,
	}
}

// Resolve returns the AdapterRunner for the given adapter name.
// Resolution order: overrides → defaultRunner → built-in adapter mapping.
func (r *AdapterRegistry) Resolve(adapterName string) AdapterRunner {
	if runner, ok := r.overrides[adapterName]; ok {
		return runner
	}
	if r.defaultRunner != nil {
		return r.defaultRunner
	}
	return ResolveAdapter(adapterName)
}

// ResolveWithFallback returns the primary runner wrapped in a FallbackRunner
// when fallbacks are configured for the resolved adapter. If no fallbacks
// are configured, returns the plain runner.
func (r *AdapterRegistry) ResolveWithFallback(adapterName string) AdapterRunner {
	primary := r.Resolve(adapterName)
	chain := r.FallbackChain(adapterName)
	if len(chain) == 0 {
		return primary
	}
	return NewFallbackRunner(primary, chain, r)
}

// FallbackChain returns the ordered fallback adapter names for the given
// primary adapter. Returns nil if no fallbacks configured.
func (r *AdapterRegistry) FallbackChain(primary string) []string {
	if r.fallbacks == nil {
		return nil
	}
	return r.fallbacks[strings.ToLower(primary)]
}

// RegisterOverride injects a custom runner for the given adapter name.
// Used for testing.
func (r *AdapterRegistry) RegisterOverride(name string, runner AdapterRunner) {
	r.overrides[name] = runner
}
