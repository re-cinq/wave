package pipeline

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/contract"
)

// Failure class constants for step failure classification.
const (
	FailureClassTransient       = "transient"
	FailureClassDeterministic   = "deterministic"
	FailureClassBudgetExhausted = "budget_exhausted"
	FailureClassContractFailure = "contract_failure"
	FailureClassTestFailure     = "test_failure"
	FailureClassCanceled        = "canceled"
)

// IsRetryable returns true if the given failure class is eligible for retry.
func IsRetryable(class string) bool {
	switch class {
	case FailureClassTransient, FailureClassContractFailure, FailureClassTestFailure:
		return true
	default:
		return false
	}
}

// ClassifyStepFailure determines the failure class for a step based on the
// execution error, contract validation error, and context error. It inspects
// typed errors first (StepExecutionError, ValidationError), then falls back to
// pattern matching on the error message.
func ClassifyStepFailure(err error, contractErr error, ctxErr error) string {
	// Context errors take highest priority.
	if ctxErr != nil {
		if errors.Is(ctxErr, context.Canceled) {
			return FailureClassCanceled
		}
		if errors.Is(ctxErr, context.DeadlineExceeded) {
			return FailureClassTransient
		}
	}

	// Check for adapter.StepError with structured failure reason.
	var stepErr *adapter.StepError
	if errors.As(err, &stepErr) {
		switch stepErr.FailureReason {
		case adapter.FailureReasonTimeout:
			return FailureClassTransient
		case adapter.FailureReasonRateLimit:
			return FailureClassTransient
		case adapter.FailureReasonContextExhaustion:
			return FailureClassBudgetExhausted
		}
	}

	// Check for contract.ValidationError.
	var valErr *contract.ValidationError
	if errors.As(err, &valErr) {
		return FailureClassContractFailure
	}

	// Explicit contract error parameter.
	if contractErr != nil {
		return FailureClassContractFailure
	}

	// Pattern-match on the error message for heuristic classification.
	if err != nil {
		return classifyByMessage(err.Error())
	}

	// All inputs are nil — no failure to classify.
	return ""
}

// classifyByMessage performs case-insensitive pattern matching on the error
// message to determine the failure class.
func classifyByMessage(msg string) string {
	lower := strings.ToLower(msg)

	// Auth / config errors → deterministic (no point retrying).
	authPatterns := []string{
		"invalid api key",
		"authentication failed",
		"missing binary",
		"not found in path",
		"permission denied",
		"access denied",
		"unauthorized",
		"forbidden",
	}
	for _, p := range authPatterns {
		if strings.Contains(lower, p) {
			return FailureClassDeterministic
		}
	}

	// Test failure patterns. Use specific prefixes to avoid false positives
	// (e.g. "fail:" matching unrelated log lines, "go test" matching docs).
	testPatterns := []string{
		"test failed",
		"tests failed",
		"--- fail:",
		"go test ./",
		"npm test ",
		"npm test\n",
		"pytest ",
		"pytest\n",
	}
	for _, p := range testPatterns {
		if strings.Contains(lower, p) {
			return FailureClassTestFailure
		}
	}

	// Rate limit patterns → transient. Use "status 429" or "http 429"
	// to avoid matching arbitrary numbers containing "429".
	rateLimitPatterns := []string{
		"rate limit",
		"too many requests",
		"status 429",
		"http 429",
		"error 429",
	}
	for _, p := range rateLimitPatterns {
		if strings.Contains(lower, p) {
			return FailureClassTransient
		}
	}

	// Budget exhaustion patterns.
	budgetPatterns := []string{
		"context window",
		"token limit",
		"prompt is too long",
	}
	for _, p := range budgetPatterns {
		if strings.Contains(lower, p) {
			return FailureClassBudgetExhausted
		}
	}

	// Default: transient (safe fallback, allows retry).
	return FailureClassTransient
}

// Regexp patterns for fingerprint normalization.
var (
	reTimestamp = regexp.MustCompile(`\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}[.\d]*[Z]?`)
	reLineNum   = regexp.MustCompile(`:\d+:`)
	reHexAddr   = regexp.MustCompile(`0x[0-9a-fA-F]+`)
	reUUID      = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)
	reTempPaths = regexp.MustCompile(`/tmp/[^\s]+`)
)

// NormalizeFingerprint produces a stable fingerprint for a step failure by
// stripping volatile substrings (timestamps, line numbers, hex addresses,
// UUIDs, temp paths) from the error message.
func NormalizeFingerprint(stepID, failureClass, errorMsg string) string {
	normalized := errorMsg
	normalized = reTimestamp.ReplaceAllString(normalized, "")
	normalized = reLineNum.ReplaceAllString(normalized, ":")
	normalized = reHexAddr.ReplaceAllString(normalized, "")
	normalized = reUUID.ReplaceAllString(normalized, "")
	normalized = reTempPaths.ReplaceAllString(normalized, "")
	normalized = strings.ToLower(normalized)

	if len(normalized) > 200 {
		normalized = normalized[:200]
	}

	return fmt.Sprintf("%s:%s:%s", stepID, failureClass, normalized)
}

// StepAttemptReplay holds the minimal data needed to replay failure history
// for circuit breaker initialization during pipeline resume. Defined here
// instead of importing state to avoid circular dependencies.
type StepAttemptReplay struct {
	StepID       string
	FailureClass string
	ErrorMessage string
}

// CircuitBreaker tracks repeated failures by fingerprint and trips when a
// configurable limit is reached. It is safe for concurrent use.
type CircuitBreaker struct {
	counts         map[string]int
	limit          int
	trackedClasses map[string]bool
	mu             sync.Mutex
}

// NewCircuitBreaker creates a CircuitBreaker. If limit is 0, it defaults to 3.
// If trackedClasses is empty, it defaults to deterministic, contract_failure,
// and test_failure.
func NewCircuitBreaker(limit int, trackedClasses []string) *CircuitBreaker {
	if limit <= 0 {
		limit = 3
	}

	tracked := make(map[string]bool, len(trackedClasses))
	if len(trackedClasses) == 0 {
		tracked[FailureClassDeterministic] = true
		tracked[FailureClassContractFailure] = true
		tracked[FailureClassTestFailure] = true
	} else {
		for _, c := range trackedClasses {
			tracked[c] = true
		}
	}

	return &CircuitBreaker{
		counts:         make(map[string]int),
		limit:          limit,
		trackedClasses: tracked,
	}
}

// Record increments the failure count for the given fingerprint if the
// failure class is tracked. It returns true if the circuit has tripped
// (count >= limit).
func (cb *CircuitBreaker) Record(fingerprint, failureClass string) (tripped bool) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if !cb.trackedClasses[failureClass] {
		return false
	}

	cb.counts[fingerprint]++
	return cb.counts[fingerprint] >= cb.limit
}

// LoadFromAttempts replays prior failure attempts to restore circuit breaker
// state during pipeline resume.
func (cb *CircuitBreaker) LoadFromAttempts(attempts []StepAttemptReplay) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	for _, a := range attempts {
		if !cb.trackedClasses[a.FailureClass] {
			continue
		}
		fp := NormalizeFingerprint(a.StepID, a.FailureClass, a.ErrorMessage)
		cb.counts[fp]++
	}
}

// Limit returns the configured trip threshold.
func (cb *CircuitBreaker) Limit() int {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	return cb.limit
}

// Count returns the current failure count for a fingerprint.
func (cb *CircuitBreaker) Count(fingerprint string) int {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	return cb.counts[fingerprint]
}
