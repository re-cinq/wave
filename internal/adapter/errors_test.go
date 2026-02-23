package adapter

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

func TestClassifyFailure(t *testing.T) {
	tests := []struct {
		name          string
		subtype       string
		resultContent string
		ctxErr        error
		want          string
	}{
		{
			name:   "deadline exceeded returns timeout",
			ctxErr: context.DeadlineExceeded,
			want:   FailureReasonTimeout,
		},
		{
			name:   "deadline exceeded takes priority over subtype",
			subtype: "error_max_turns",
			ctxErr: context.DeadlineExceeded,
			want:   FailureReasonTimeout,
		},
		{
			name:    "error_max_turns returns context_exhaustion",
			subtype: "error_max_turns",
			want:    FailureReasonContextExhaustion,
		},
		{
			name:          "prompt is too long in content returns context_exhaustion",
			resultContent: "Error: prompt is too long for the model",
			want:          FailureReasonContextExhaustion,
		},
		{
			name:          "case insensitive prompt is too long detection",
			resultContent: "PROMPT IS TOO LONG",
			want:          FailureReasonContextExhaustion,
		},
		{
			name:    "success subtype returns general_error",
			subtype: "success",
			want:    FailureReasonGeneralError,
		},
		{
			name:    "error_during_execution returns general_error",
			subtype: "error_during_execution",
			want:    FailureReasonGeneralError,
		},
		{
			name: "no subtype and no context error returns general_error",
			want: FailureReasonGeneralError,
		},
		{
			name:          "you've hit your limit returns rate_limit",
			resultContent: "You've hit your limit for the day. Please wait and try again.",
			want:          FailureReasonRateLimit,
		},
		{
			name:          "rate limit exceeded in content returns rate_limit",
			resultContent: "Error: rate limit exceeded, please retry later",
			want:          FailureReasonRateLimit,
		},
		{
			name:          "too many requests returns rate_limit",
			resultContent: "HTTP 429: Too Many Requests",
			want:          FailureReasonRateLimit,
		},
		{
			name:          "case insensitive rate limit detection",
			resultContent: "RATE LIMIT reached",
			want:          FailureReasonRateLimit,
		},
		{
			name:          "unrelated content returns general_error",
			resultContent: "some other error message",
			want:          FailureReasonGeneralError,
		},
		{
			name:          "security review mentioning rate limiting is not rate_limit",
			resultContent: "No rate limiting on chat endpoints (cost amplification risk)",
			want:          FailureReasonGeneralError,
		},
		{
			name:   "context canceled is not treated as timeout",
			ctxErr: context.Canceled,
			want:   FailureReasonGeneralError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyFailure(tt.subtype, tt.resultContent, tt.ctxErr)
			if got != tt.want {
				t.Errorf("ClassifyFailure(%q, %q, %v) = %q, want %q",
					tt.subtype, tt.resultContent, tt.ctxErr, got, tt.want)
			}
		})
	}
}

func TestStepErrorInterface(t *testing.T) {
	cause := fmt.Errorf("underlying error")
	se := NewStepError(FailureReasonTimeout, cause, 50000, "error_during_execution")

	t.Run("Error includes reason and cause", func(t *testing.T) {
		got := se.Error()
		want := "timeout: underlying error"
		if got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("Error without cause", func(t *testing.T) {
		se2 := NewStepError(FailureReasonGeneralError, nil, 0, "")
		got := se2.Error()
		if got != FailureReasonGeneralError {
			t.Errorf("Error() = %q, want %q", got, FailureReasonGeneralError)
		}
	})

	t.Run("Unwrap returns cause", func(t *testing.T) {
		got := se.Unwrap()
		if got != cause {
			t.Errorf("Unwrap() = %v, want %v", got, cause)
		}
	})

	t.Run("errors.As works", func(t *testing.T) {
		wrappedErr := fmt.Errorf("adapter failed: %w", se)
		var target *StepError
		if !errors.As(wrappedErr, &target) {
			t.Fatal("errors.As should find StepError")
		}
		if target.FailureReason != FailureReasonTimeout {
			t.Errorf("FailureReason = %q, want %q", target.FailureReason, FailureReasonTimeout)
		}
		if target.TokensUsed != 50000 {
			t.Errorf("TokensUsed = %d, want 50000", target.TokensUsed)
		}
		if target.Subtype != "error_during_execution" {
			t.Errorf("Subtype = %q, want %q", target.Subtype, "error_during_execution")
		}
	})

	t.Run("errors.Is works through StepError", func(t *testing.T) {
		se3 := NewStepError(FailureReasonTimeout, context.DeadlineExceeded, 0, "")
		if !errors.Is(se3, context.DeadlineExceeded) {
			t.Error("errors.Is should find DeadlineExceeded through StepError.Unwrap()")
		}
	})
}

func TestStepErrorRemediation(t *testing.T) {
	tests := []struct {
		reason string
		want   string
	}{
		{
			reason: FailureReasonTimeout,
			want:   "Consider increasing the step timeout with --timeout or breaking the task into smaller steps.",
		},
		{
			reason: FailureReasonContextExhaustion,
			want:   "The context window was exhausted. Consider breaking the task into smaller steps or adjusting relay compaction thresholds (relay.token_threshold_percent).",
		},
		{
			reason: FailureReasonRateLimit,
			want:   "API rate limit reached. Wait for the limit to reset and retry.",
		},
		{
			reason: FailureReasonGeneralError,
			want:   "Check the adapter output and logs for details.",
		},
		{
			reason: "unknown_reason",
			want:   "Check the adapter output and logs for details.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.reason, func(t *testing.T) {
			se := NewStepError(tt.reason, nil, 0, "")
			if se.Remediation != tt.want {
				t.Errorf("Remediation = %q, want %q", se.Remediation, tt.want)
			}
		})
	}
}
