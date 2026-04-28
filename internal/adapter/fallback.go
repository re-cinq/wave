package adapter

import (
	"context"
	"fmt"
)

// FallbackRunner wraps a primary AdapterRunner with a fallback chain.
// When the primary fails with a rate_limit failure, it tries each
// fallback adapter in order. Max attempts equals len(chain) + 1.
type FallbackRunner struct {
	primary  AdapterRunner
	chain    []string         // fallback adapter names in order
	registry *AdapterRegistry // for resolving fallback adapter names
}

// NewFallbackRunner creates a FallbackRunner wrapping the primary runner
// with the given fallback chain.
func NewFallbackRunner(primary AdapterRunner, chain []string, registry *AdapterRegistry) *FallbackRunner {
	return &FallbackRunner{
		primary:  primary,
		chain:    chain,
		registry: registry,
	}
}

// Run executes the primary adapter first. On rate_limit failure, tries
// each fallback adapter in chain order. Returns the first successful
// result or the last error if all attempts fail.
func (f *FallbackRunner) Run(ctx context.Context, cfg AdapterRunConfig) (*AdapterResult, error) {
	// Try primary adapter first
	result, err := f.primary.Run(ctx, cfg)
	if err == nil && !isFallbackTrigger(result) {
		return result, nil
	}

	// Only fallback on rate_limit
	if err != nil && result == nil {
		// Hard error with no result — check if context cancelled
		return nil, err
	}
	if result != nil && !isFallbackTrigger(result) {
		return result, err
	}

	// Try fallback chain
	var lastErr error
	var lastResult *AdapterResult
	if err != nil {
		lastErr = err
	}
	if result != nil {
		lastResult = result
	}

	for _, fallbackName := range f.chain {
		select {
		case <-ctx.Done():
			return lastResult, fmt.Errorf("fallback chain cancelled: %w", ctx.Err())
		default:
		}

		runner := f.registry.Resolve(fallbackName)
		result, err = runner.Run(ctx, cfg)
		if err == nil && !isFallbackTrigger(result) {
			return result, nil
		}

		if err != nil {
			lastErr = err
		}
		if result != nil {
			lastResult = result
		}
	}

	if lastErr != nil {
		return lastResult, fmt.Errorf("all fallback adapters exhausted: %w", lastErr)
	}
	return lastResult, fmt.Errorf("all fallback adapters exhausted")
}

// isFallbackTrigger returns true when a failure has a real chance of
// succeeding on a different provider — i.e. the failure is upstream-capacity
// (rate limit), wall-clock (the model stalled past the timeout), or
// context-budget (the model couldn't fit the prompt). All three commonly
// resolve when retried on a peer with different limits, model architecture,
// or context window.
//
// Other classifications (general_error, validation, etc.) are intentionally
// excluded — they typically indicate a bug or schema mismatch that will fail
// the same way on any provider.
func isFallbackTrigger(result *AdapterResult) bool {
	if result == nil {
		return false
	}
	switch result.FailureReason {
	case FailureReasonRateLimit, FailureReasonTimeout, FailureReasonContextExhaustion:
		return true
	}
	return false
}
