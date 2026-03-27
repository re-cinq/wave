package pipeline

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/contract"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		class    string
		expected bool
	}{
		{"transient is retryable", FailureClassTransient, true},
		{"contract_failure is retryable", FailureClassContractFailure, true},
		{"test_failure is retryable", FailureClassTestFailure, true},
		{"deterministic is not retryable", FailureClassDeterministic, false},
		{"budget_exhausted is not retryable", FailureClassBudgetExhausted, false},
		{"canceled is not retryable", FailureClassCanceled, false},
		{"unknown class is not retryable", "unknown_class", false},
		{"empty string is not retryable", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsRetryable(tt.class))
		})
	}
}

func TestClassifyStepFailure_ContextErrors(t *testing.T) {
	tests := []struct {
		name     string
		ctxErr   error
		expected string
	}{
		{
			name:     "context.Canceled returns canceled",
			ctxErr:   context.Canceled,
			expected: FailureClassCanceled,
		},
		{
			name:     "context.DeadlineExceeded returns transient",
			ctxErr:   context.DeadlineExceeded,
			expected: FailureClassTransient,
		},
		{
			name:     "wrapped context.Canceled returns canceled",
			ctxErr:   fmt.Errorf("wrapped: %w", context.Canceled),
			expected: FailureClassCanceled,
		},
		{
			name:     "wrapped context.DeadlineExceeded returns transient",
			ctxErr:   fmt.Errorf("wrapped: %w", context.DeadlineExceeded),
			expected: FailureClassTransient,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyStepFailure(errors.New("some error"), nil, tt.ctxErr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClassifyStepFailure_AdapterStepError(t *testing.T) {
	tests := []struct {
		name     string
		reason   string
		expected string
	}{
		{
			name:     "timeout returns transient",
			reason:   adapter.FailureReasonTimeout,
			expected: FailureClassTransient,
		},
		{
			name:     "rate_limit returns transient",
			reason:   adapter.FailureReasonRateLimit,
			expected: FailureClassTransient,
		},
		{
			name:     "context_exhaustion returns budget_exhausted",
			reason:   adapter.FailureReasonContextExhaustion,
			expected: FailureClassBudgetExhausted,
		},
		{
			name:     "general_error falls through to message matching",
			reason:   adapter.FailureReasonGeneralError,
			expected: FailureClassTransient, // general_error not matched, falls to message pattern, default transient
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stepErr := adapter.NewStepError(tt.reason, errors.New("cause"), 0, "")
			result := ClassifyStepFailure(stepErr, nil, nil)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClassifyStepFailure_ContractValidationError(t *testing.T) {
	t.Run("via errors.As on err param", func(t *testing.T) {
		valErr := &contract.ValidationError{
			ContractType: "test_suite",
			Message:      "tests failed",
		}
		result := ClassifyStepFailure(valErr, nil, nil)
		assert.Equal(t, FailureClassContractFailure, result)
	})

	t.Run("wrapped ValidationError via errors.As", func(t *testing.T) {
		valErr := &contract.ValidationError{
			ContractType: "json_schema",
			Message:      "schema mismatch",
		}
		wrapped := fmt.Errorf("contract check: %w", valErr)
		result := ClassifyStepFailure(wrapped, nil, nil)
		assert.Equal(t, FailureClassContractFailure, result)
	})

	t.Run("via contractErr param", func(t *testing.T) {
		result := ClassifyStepFailure(errors.New("some other error"), errors.New("contract failed"), nil)
		assert.Equal(t, FailureClassContractFailure, result)
	})

	t.Run("contractErr takes precedence over message patterns", func(t *testing.T) {
		// Even though the err message matches auth patterns, contractErr wins
		result := ClassifyStepFailure(errors.New("permission denied"), errors.New("contract issue"), nil)
		// StepError check happens first, then ValidationError, then contractErr
		// But "permission denied" would match deterministic via message — contractErr should override
		// Actually: errors.As for StepError fails, errors.As for ValidationError fails,
		// then contractErr != nil returns contract_failure before message matching
		assert.Equal(t, FailureClassContractFailure, result)
	})
}

func TestClassifyStepFailure_MessagePatterns(t *testing.T) {
	tests := []struct {
		name     string
		errMsg   string
		expected string
	}{
		// Auth / deterministic patterns
		{"invalid api key", "Invalid API Key provided", FailureClassDeterministic},
		{"authentication failed", "authentication failed for user", FailureClassDeterministic},
		{"missing binary", "missing binary: claude", FailureClassDeterministic},
		{"not found in path", "claude not found in path", FailureClassDeterministic},
		{"permission denied", "permission denied: /etc/secret", FailureClassDeterministic},
		{"access denied", "Access Denied to resource", FailureClassDeterministic},
		{"unauthorized", "401 Unauthorized", FailureClassDeterministic},
		{"forbidden", "403 Forbidden", FailureClassDeterministic},

		// Test failure patterns (tightened to avoid false positives)
		{"test failed", "test failed: TestFoo", FailureClassTestFailure},
		{"tests failed", "3 tests failed", FailureClassTestFailure},
		{"--- FAIL:", "--- FAIL: TestBar", FailureClassTestFailure},
		{"go test ./", "go test ./... exited with code 1", FailureClassTestFailure},
		{"npm test with trailing space", "npm test exited with code 1", FailureClassTestFailure},
		{"pytest with trailing space", "pytest returned exit code 1", FailureClassTestFailure},

		// Rate limit patterns (tightened: "429" alone won't match)
		{"rate limit", "rate limit exceeded", FailureClassTransient},
		{"too many requests", "too many requests, slow down", FailureClassTransient},
		{"status 429", "status 429 response", FailureClassTransient},
		{"http 429", "HTTP 429 too many requests", FailureClassTransient},
		{"error 429", "error 429: rate limited", FailureClassTransient},

		// Budget exhaustion patterns
		{"context window", "context window full", FailureClassBudgetExhausted},
		{"token limit", "token limit reached", FailureClassBudgetExhausted},
		{"prompt is too long", "prompt is too long for model", FailureClassBudgetExhausted},

		// Default / unrecognized
		{"unknown error defaults to transient", "something unexpected happened", FailureClassTransient},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyStepFailure(errors.New(tt.errMsg), nil, nil)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClassifyStepFailure_NilError(t *testing.T) {
	result := ClassifyStepFailure(nil, nil, nil)
	assert.Equal(t, "", result, "all-nil inputs should return empty string")
}

func TestClassifyStepFailure_EmptyMessage(t *testing.T) {
	result := ClassifyStepFailure(errors.New(""), nil, nil)
	assert.Equal(t, FailureClassTransient, result, "empty error message should default to transient")
}

func TestNormalizeFingerprint_StripTimestamps(t *testing.T) {
	msg := "error at 2026-03-27T14:30:00Z: connection reset"
	fp := NormalizeFingerprint("step1", FailureClassTransient, msg)

	assert.NotContains(t, fp, "2026-03-27")
	assert.NotContains(t, fp, "14:30:00")
	assert.Contains(t, fp, "connection reset")
	assert.True(t, strings.HasPrefix(fp, "step1:transient:"))
}

func TestNormalizeFingerprint_StripLineNumbers(t *testing.T) {
	msg := "error in file.go:123: something broke"
	fp := NormalizeFingerprint("compile", FailureClassDeterministic, msg)

	// :123: should be replaced with :
	assert.NotContains(t, fp, ":123:")
	assert.Contains(t, fp, "something broke")
}

func TestNormalizeFingerprint_StripHexAddresses(t *testing.T) {
	msg := "nil pointer dereference at 0x7fff5fbff8c0"
	fp := NormalizeFingerprint("step1", FailureClassTransient, msg)

	assert.NotContains(t, fp, "0x7fff5fbff8c0")
	assert.Contains(t, fp, "nil pointer dereference")
}

func TestNormalizeFingerprint_StripUUIDs(t *testing.T) {
	msg := "failed to process request a1b2c3d4-e5f6-7890-abcd-ef1234567890"
	fp := NormalizeFingerprint("step1", FailureClassTransient, msg)

	assert.NotContains(t, fp, "a1b2c3d4-e5f6-7890-abcd-ef1234567890")
	assert.Contains(t, fp, "failed to process request")
}

func TestNormalizeFingerprint_StripTempPaths(t *testing.T) {
	msg := "cannot read /tmp/wave-abc123/output.json"
	fp := NormalizeFingerprint("step1", FailureClassTransient, msg)

	assert.NotContains(t, fp, "/tmp/wave-abc123/output.json")
	assert.Contains(t, fp, "cannot read")
}

func TestNormalizeFingerprint_Truncation(t *testing.T) {
	// Build a message longer than 200 characters (after normalization).
	longMsg := strings.Repeat("a", 300)
	fp := NormalizeFingerprint("step1", FailureClassTransient, longMsg)

	// The format is "step1:transient:<normalized>" — the normalized part is capped at 200
	prefix := "step1:transient:"
	assert.True(t, strings.HasPrefix(fp, prefix))
	normalizedPart := strings.TrimPrefix(fp, prefix)
	assert.LessOrEqual(t, len(normalizedPart), 200)
}

func TestNormalizeFingerprint_Deterministic(t *testing.T) {
	msg := "error at 2026-03-27T10:00:00Z in /tmp/wave-xyz/foo.go:42: panic"

	fp1 := NormalizeFingerprint("step1", FailureClassTransient, msg)
	fp2 := NormalizeFingerprint("step1", FailureClassTransient, msg)

	assert.Equal(t, fp1, fp2, "same input must produce the same fingerprint")
}

func TestNormalizeFingerprint_Lowercased(t *testing.T) {
	msg := "ERROR: Something BROKE"
	fp := NormalizeFingerprint("step1", FailureClassTransient, msg)

	normalizedPart := strings.TrimPrefix(fp, "step1:transient:")
	assert.Equal(t, strings.ToLower(normalizedPart), normalizedPart, "normalized part should be lowercased")
}

func TestCircuitBreaker_Limit(t *testing.T) {
	cb := NewCircuitBreaker(5, nil)
	assert.Equal(t, 5, cb.Limit())

	cb2 := NewCircuitBreaker(0, nil)
	assert.Equal(t, 3, cb2.Limit(), "zero limit should default to 3")
}

func TestCircuitBreaker_Basic(t *testing.T) {
	cb := NewCircuitBreaker(2, []string{FailureClassDeterministic})

	fp := "step1:deterministic:auth failed"

	// First record — not tripped yet
	tripped := cb.Record(fp, FailureClassDeterministic)
	assert.False(t, tripped)
	assert.Equal(t, 1, cb.Count(fp))

	// Second record — trips at limit
	tripped = cb.Record(fp, FailureClassDeterministic)
	assert.True(t, tripped)
	assert.Equal(t, 2, cb.Count(fp))

	// Third record — stays tripped
	tripped = cb.Record(fp, FailureClassDeterministic)
	assert.True(t, tripped)
	assert.Equal(t, 3, cb.Count(fp))
}

func TestCircuitBreaker_TrackedClasses(t *testing.T) {
	cb := NewCircuitBreaker(2, []string{FailureClassDeterministic})

	fp := "step1:transient:timeout"

	// Transient is not tracked, so recording it should not increment count
	tripped := cb.Record(fp, FailureClassTransient)
	assert.False(t, tripped)
	assert.Equal(t, 0, cb.Count(fp))

	// Deterministic is tracked
	fpDet := "step1:deterministic:auth failed"
	tripped = cb.Record(fpDet, FailureClassDeterministic)
	assert.False(t, tripped)
	assert.Equal(t, 1, cb.Count(fpDet))
}

func TestCircuitBreaker_DefaultsOnZeroLimit(t *testing.T) {
	cb := NewCircuitBreaker(0, nil)

	// Default limit should be 3
	fp := "step1:deterministic:auth failed"
	assert.False(t, cb.Record(fp, FailureClassDeterministic)) // 1
	assert.False(t, cb.Record(fp, FailureClassDeterministic)) // 2
	assert.True(t, cb.Record(fp, FailureClassDeterministic))  // 3 = limit

	// Default tracked classes: deterministic, contract_failure, test_failure
	fpContract := "step1:contract_failure:schema mismatch"
	assert.False(t, cb.Record(fpContract, FailureClassContractFailure))
	assert.Equal(t, 1, cb.Count(fpContract))

	fpTest := "step1:test_failure:tests failed"
	assert.False(t, cb.Record(fpTest, FailureClassTestFailure))
	assert.Equal(t, 1, cb.Count(fpTest))

	// Transient should NOT be tracked by default
	fpTransient := "step1:transient:timeout"
	assert.False(t, cb.Record(fpTransient, FailureClassTransient))
	assert.Equal(t, 0, cb.Count(fpTransient))

	// Canceled should NOT be tracked by default
	fpCanceled := "step1:canceled:canceled"
	assert.False(t, cb.Record(fpCanceled, FailureClassCanceled))
	assert.Equal(t, 0, cb.Count(fpCanceled))

	// Budget exhausted should NOT be tracked by default
	fpBudget := "step1:budget_exhausted:context window full"
	assert.False(t, cb.Record(fpBudget, FailureClassBudgetExhausted))
	assert.Equal(t, 0, cb.Count(fpBudget))
}

func TestCircuitBreaker_LoadFromAttempts(t *testing.T) {
	cb := NewCircuitBreaker(3, []string{FailureClassDeterministic, FailureClassTestFailure})

	attempts := []StepAttemptReplay{
		{StepID: "build", FailureClass: FailureClassDeterministic, ErrorMessage: "auth failed"},
		{StepID: "build", FailureClass: FailureClassDeterministic, ErrorMessage: "auth failed"},
		{StepID: "test", FailureClass: FailureClassTestFailure, ErrorMessage: "tests failed"},
		{StepID: "deploy", FailureClass: FailureClassTransient, ErrorMessage: "timeout"}, // not tracked
	}

	cb.LoadFromAttempts(attempts)

	// Build fingerprint for "build" deterministic failures
	fpBuild := NormalizeFingerprint("build", FailureClassDeterministic, "auth failed")
	assert.Equal(t, 2, cb.Count(fpBuild))

	// Build fingerprint for "test" test_failure
	fpTest := NormalizeFingerprint("test", FailureClassTestFailure, "tests failed")
	assert.Equal(t, 1, cb.Count(fpTest))

	// Transient was not tracked, so no count
	fpDeploy := NormalizeFingerprint("deploy", FailureClassTransient, "timeout")
	assert.Equal(t, 0, cb.Count(fpDeploy))

	// Recording one more for build should trip the breaker (2 + 1 = 3)
	tripped := cb.Record(fpBuild, FailureClassDeterministic)
	require.True(t, tripped, "circuit breaker should trip after replayed attempts + new record reach limit")
}

func TestCircuitBreaker_ConcurrentSafety(t *testing.T) {
	cb := NewCircuitBreaker(1000, []string{FailureClassDeterministic})

	const goroutines = 50
	const recordsPerGoroutine = 20

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < recordsPerGoroutine; j++ {
				fp := fmt.Sprintf("step%d:deterministic:err%d", id%5, j%3)
				cb.Record(fp, FailureClassDeterministic)
			}
		}(i)
	}

	wg.Wait()

	// Verify no panic occurred and counts are positive.
	// With 50 goroutines each doing 20 records across 5 step variants and 3 error variants,
	// total records = 1000, spread across 15 fingerprints.
	totalCount := 0
	for i := 0; i < 5; i++ {
		for j := 0; j < 3; j++ {
			fp := fmt.Sprintf("step%d:deterministic:err%d", i, j)
			count := cb.Count(fp)
			assert.Greater(t, count, 0, "fingerprint %s should have recorded at least once", fp)
			totalCount += count
		}
	}
	assert.Equal(t, goroutines*recordsPerGoroutine, totalCount, "total records should match")
}
