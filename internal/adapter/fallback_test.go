package adapter

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// failingRunner always returns a result with the given failure reason.
type failingRunner struct {
	failureReason string
	callCount     int
}

func (r *failingRunner) Run(_ context.Context, _ AdapterRunConfig) (*AdapterResult, error) {
	r.callCount++
	return &AdapterResult{
		ExitCode:      1,
		FailureReason: r.failureReason,
		ResultContent: "failed: " + r.failureReason,
	}, nil
}

// successRunner always returns a successful result.
type successRunner struct {
	callCount int
}

func (r *successRunner) Run(_ context.Context, _ AdapterRunConfig) (*AdapterResult, error) {
	r.callCount++
	return &AdapterResult{
		ExitCode:      0,
		ResultContent: "success",
	}, nil
}

// errorRunner returns an error (no result).
type errorRunner struct{}

func (r *errorRunner) Run(_ context.Context, _ AdapterRunConfig) (*AdapterResult, error) {
	return nil, fmt.Errorf("hard error")
}

func TestFallbackRunner_PrimarySucceeds(t *testing.T) {
	primary := &successRunner{}
	registry := NewAdapterRegistry(nil)

	fr := NewFallbackRunner(primary, []string{"codex"}, registry)
	result, err := fr.Run(context.Background(), AdapterRunConfig{})

	assert.NoError(t, err)
	assert.Equal(t, "success", result.ResultContent)
	assert.Equal(t, 1, primary.callCount, "should only call primary once")
}

func TestFallbackRunner_RateLimitTriggersFallback(t *testing.T) {
	primary := &failingRunner{failureReason: "rate_limit"}
	fallback := &successRunner{}

	registry := NewAdapterRegistry(nil)
	registry.RegisterOverride("codex", fallback)

	fr := NewFallbackRunner(primary, []string{"codex"}, registry)
	result, err := fr.Run(context.Background(), AdapterRunConfig{})

	assert.NoError(t, err)
	assert.Equal(t, "success", result.ResultContent)
	assert.Equal(t, 1, primary.callCount)
	assert.Equal(t, 1, fallback.callCount)
}

func TestFallbackRunner_ContextExhaustionTriggersFallback(t *testing.T) {
	primary := &failingRunner{failureReason: "context_exhaustion"}
	fallback := &successRunner{}

	registry := NewAdapterRegistry(nil)
	registry.RegisterOverride("codex", fallback)

	fr := NewFallbackRunner(primary, []string{"codex"}, registry)
	result, err := fr.Run(context.Background(), AdapterRunConfig{})

	assert.NoError(t, err)
	assert.Equal(t, "success", result.ResultContent,
		"context_exhaustion should trigger fallback to a peer with a different context budget")
	assert.Equal(t, 1, fallback.callCount)
}

func TestFallbackRunner_TimeoutTriggersFallback(t *testing.T) {
	primary := &failingRunner{failureReason: "timeout"}
	fallback := &successRunner{}

	registry := NewAdapterRegistry(nil)
	registry.RegisterOverride("codex", fallback)

	fr := NewFallbackRunner(primary, []string{"codex"}, registry)
	result, err := fr.Run(context.Background(), AdapterRunConfig{})

	assert.NoError(t, err)
	assert.Equal(t, "success", result.ResultContent,
		"timeout should trigger fallback so a stalled local model hands off to a peer")
	assert.Equal(t, 1, fallback.callCount)
}

func TestFallbackRunner_GeneralErrorDoesNotTriggerFallback(t *testing.T) {
	primary := &failingRunner{failureReason: "general_error"}
	fallback := &successRunner{}

	registry := NewAdapterRegistry(nil)
	registry.RegisterOverride("codex", fallback)

	fr := NewFallbackRunner(primary, []string{"codex"}, registry)
	result, err := fr.Run(context.Background(), AdapterRunConfig{})

	assert.NoError(t, err)
	assert.Equal(t, "general_error", result.FailureReason)
	assert.Equal(t, 0, fallback.callCount, "should NOT call fallback on general_error")
}

func TestFallbackRunner_AllFallbacksExhausted(t *testing.T) {
	primary := &failingRunner{failureReason: "rate_limit"}
	fb1 := &failingRunner{failureReason: "rate_limit"}
	fb2 := &failingRunner{failureReason: "rate_limit"}

	registry := NewAdapterRegistry(nil)
	registry.RegisterOverride("fb1", fb1)
	registry.RegisterOverride("fb2", fb2)

	fr := NewFallbackRunner(primary, []string{"fb1", "fb2"}, registry)
	result, err := fr.Run(context.Background(), AdapterRunConfig{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "all fallback adapters exhausted")
	assert.NotNil(t, result)
	assert.Equal(t, 1, primary.callCount)
	assert.Equal(t, 1, fb1.callCount)
	assert.Equal(t, 1, fb2.callCount)
}

func TestFallbackRunner_EmptyChainReturnsPrimaryResult(t *testing.T) {
	primary := &failingRunner{failureReason: "rate_limit"}

	registry := NewAdapterRegistry(nil)
	fr := NewFallbackRunner(primary, []string{}, registry)
	result, err := fr.Run(context.Background(), AdapterRunConfig{})

	// Empty chain — the initial rate_limit result won't trigger any fallbacks
	// but the loop doesn't execute, so we get the "all fallback adapters exhausted" error
	assert.Error(t, err)
	assert.NotNil(t, result)
}

func TestFallbackRunner_HardErrorFromPrimary(t *testing.T) {
	primary := &errorRunner{}

	registry := NewAdapterRegistry(nil)
	fr := NewFallbackRunner(primary, []string{"codex"}, registry)
	_, err := fr.Run(context.Background(), AdapterRunConfig{})

	// Hard error with no result — should not fallback
	assert.Error(t, err)
}

func TestFallbackRunner_SecondFallbackSucceeds(t *testing.T) {
	primary := &failingRunner{failureReason: "rate_limit"}
	fb1 := &failingRunner{failureReason: "rate_limit"}
	fb2 := &successRunner{}

	registry := NewAdapterRegistry(nil)
	registry.RegisterOverride("fb1", fb1)
	registry.RegisterOverride("fb2", fb2)

	fr := NewFallbackRunner(primary, []string{"fb1", "fb2"}, registry)
	result, err := fr.Run(context.Background(), AdapterRunConfig{})

	assert.NoError(t, err)
	assert.Equal(t, "success", result.ResultContent)
	assert.Equal(t, 1, primary.callCount)
	assert.Equal(t, 1, fb1.callCount)
	assert.Equal(t, 1, fb2.callCount)
}

func TestFallbackRunner_ContextCancelledDuringFallback(t *testing.T) {
	primary := &failingRunner{failureReason: "rate_limit"}

	registry := NewAdapterRegistry(nil)
	registry.RegisterOverride("codex", &successRunner{})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	fr := NewFallbackRunner(primary, []string{"codex"}, registry)
	_, err := fr.Run(ctx, AdapterRunConfig{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cancel")
}

func TestIsFallbackTrigger(t *testing.T) {
	assert.True(t, isFallbackTrigger(&AdapterResult{FailureReason: "rate_limit"}))
	assert.True(t, isFallbackTrigger(&AdapterResult{FailureReason: "context_exhaustion"}))
	assert.True(t, isFallbackTrigger(&AdapterResult{FailureReason: "timeout"}))
	assert.False(t, isFallbackTrigger(&AdapterResult{FailureReason: "general_error"}))
	assert.False(t, isFallbackTrigger(&AdapterResult{FailureReason: ""}))
	assert.False(t, isFallbackTrigger(nil))
}
