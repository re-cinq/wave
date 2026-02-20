package adapter

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
			name:          "rate limit in content returns rate_limit",
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

// =============================================================================
// T029: SIGKILL Exit Code 137 Reports Termination Error
// Tests that ClassifyFailure correctly handles SIGKILL scenarios (exit code 137).
// Exit code 137 = 128 + 9 (SIGKILL signal number).
// =============================================================================

// TestSIGKILL_ExitCode137_Classification tests that exit code 137 scenarios
// are correctly classified. Exit code 137 indicates the process was killed
// by SIGKILL (signal 9), typically due to timeout or OOM.
func TestSIGKILL_ExitCode137_Classification(t *testing.T) {
	// Exit code 137 messages that should be classified as general_error
	// (since we don't have a specific "terminated" reason yet)
	testCases := []struct {
		name          string
		subtype       string
		resultContent string
		ctxErr        error
		want          string
	}{
		{
			name:          "exit code 137 mentioned in content",
			resultContent: "Process terminated with exit code 137",
			want:          FailureReasonGeneralError,
		},
		{
			name:          "killed by signal 9 (SIGKILL)",
			resultContent: "Process killed by signal 9",
			want:          FailureReasonGeneralError,
		},
		{
			name:          "OOM killer terminated process",
			resultContent: "Out of memory: Killed process 12345",
			want:          FailureReasonGeneralError,
		},
		{
			name:   "timeout context with 137 is still timeout",
			ctxErr: context.DeadlineExceeded,
			// Even if content mentions 137, timeout takes precedence
			resultContent: "Process terminated with exit code 137",
			want:          FailureReasonTimeout,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyFailure(tt.subtype, tt.resultContent, tt.ctxErr)
			if got != tt.want {
				t.Errorf("ClassifyFailure(%q, %q, %v) = %q, want %q",
					tt.subtype, tt.resultContent, tt.ctxErr, got, tt.want)
			}
		})
	}
}

// TestSIGKILL_ExitCode137_ErrorIntegration tests that SIGKILL scenarios
// can be properly wrapped in StepError and unwrapped.
func TestSIGKILL_ExitCode137_ErrorIntegration(t *testing.T) {
	// Create a StepError for a SIGKILL scenario
	cause := fmt.Errorf("process terminated: exit code 137 (SIGKILL)")
	se := NewStepError(FailureReasonGeneralError, cause, 10000, "error_during_execution")

	// Verify error message includes cause
	errMsg := se.Error()
	if !strings.Contains(errMsg, "general_error") {
		t.Errorf("expected error message to contain 'general_error', got: %s", errMsg)
	}

	// Verify cause can be retrieved
	if !strings.Contains(se.Cause.Error(), "137") {
		t.Errorf("expected cause to contain '137', got: %s", se.Cause.Error())
	}

	// Verify errors.As works for wrapped SIGKILL errors
	wrappedErr := fmt.Errorf("step failed: %w", se)
	var target *StepError
	if !errors.As(wrappedErr, &target) {
		t.Fatal("errors.As should find StepError in wrapped SIGKILL error")
	}
	if target.FailureReason != FailureReasonGeneralError {
		t.Errorf("FailureReason = %q, want %q", target.FailureReason, FailureReasonGeneralError)
	}
}

// TestSIGKILL_ExitCode137_CommonScenarios tests common scenarios where
// exit code 137 appears in production.
func TestSIGKILL_ExitCode137_CommonScenarios(t *testing.T) {
	scenarios := []struct {
		name        string
		description string
		exitCode    int
		expectClass string
	}{
		{
			name:        "SIGKILL from timeout",
			description: "Process killed due to step timeout",
			exitCode:    137, // 128 + 9 (SIGKILL)
			expectClass: FailureReasonGeneralError,
		},
		{
			name:        "SIGTERM graceful shutdown",
			description: "Process terminated gracefully",
			exitCode:    143, // 128 + 15 (SIGTERM)
			expectClass: FailureReasonGeneralError,
		},
		{
			name:        "SIGINT interrupt",
			description: "Process interrupted",
			exitCode:    130, // 128 + 2 (SIGINT)
			expectClass: FailureReasonGeneralError,
		},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			// Create error with exit code information
			resultContent := fmt.Sprintf("%s (exit code %d)", sc.description, sc.exitCode)
			class := ClassifyFailure("", resultContent, nil)

			if class != sc.expectClass {
				t.Errorf("ClassifyFailure for %s = %q, want %q", sc.name, class, sc.expectClass)
			}
		})
	}
}

// TestExitCodeFromSignal verifies exit code to signal number mapping.
// This is informational/documentation test showing how signal exit codes work.
func TestExitCodeFromSignal(t *testing.T) {
	// Standard UNIX convention: exit code = 128 + signal number
	signalExitCodes := map[string]int{
		"SIGKILL (9)":  137,
		"SIGTERM (15)": 143,
		"SIGINT (2)":   130,
		"SIGQUIT (3)":  131,
		"SIGABRT (6)":  134,
	}

	for name, expectedCode := range signalExitCodes {
		signalNum := expectedCode - 128
		calculatedCode := 128 + signalNum

		if calculatedCode != expectedCode {
			t.Errorf("%s: calculated %d != expected %d", name, calculatedCode, expectedCode)
		}
	}
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
