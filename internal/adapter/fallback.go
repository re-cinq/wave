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

		if f.registry == nil {
			return lastResult, fmt.Errorf("fallback registry is nil; cannot resolve %q", fallbackName)
		}
		runner, resolveErr := f.registry.ResolveStrict(fallbackName)
		if resolveErr != nil {
			lastErr = resolveErr
			continue
		}
		if runner == nil {
			lastErr = fmt.Errorf("%w: %q (registry returned nil)", ErrUnknownAdapter, fallbackName)
			continue
		}
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

// isFallbackTrigger returns true if the result indicates a rate limit
// failure that should trigger fallback to the next provider.
func isFallbackTrigger(result *AdapterResult) bool {
	if result == nil {
		return false
	}
	return result.FailureReason == "rate_limit"
}
