package hooks

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func boolPtr(b bool) *bool { return &b }

func TestIsBlocking(t *testing.T) {
	tests := []struct {
		name     string
		hook     LifecycleHookDef
		expected bool
	}{
		// Default blocking behavior per event type
		{
			name:     "run_start defaults to blocking",
			hook:     LifecycleHookDef{Event: EventRunStart},
			expected: true,
		},
		{
			name:     "step_start defaults to blocking",
			hook:     LifecycleHookDef{Event: EventStepStart},
			expected: true,
		},
		{
			name:     "step_completed defaults to blocking",
			hook:     LifecycleHookDef{Event: EventStepCompleted},
			expected: true,
		},
		{
			name:     "run_completed defaults to non-blocking",
			hook:     LifecycleHookDef{Event: EventRunCompleted},
			expected: false,
		},
		{
			name:     "run_failed defaults to non-blocking",
			hook:     LifecycleHookDef{Event: EventRunFailed},
			expected: false,
		},
		{
			name:     "step_failed defaults to non-blocking",
			hook:     LifecycleHookDef{Event: EventStepFailed},
			expected: false,
		},
		{
			name:     "step_retrying defaults to non-blocking",
			hook:     LifecycleHookDef{Event: EventStepRetrying},
			expected: false,
		},
		{
			name:     "contract_validated defaults to non-blocking",
			hook:     LifecycleHookDef{Event: EventContractValidated},
			expected: false,
		},
		{
			name:     "artifact_created defaults to non-blocking",
			hook:     LifecycleHookDef{Event: EventArtifactCreated},
			expected: false,
		},
		{
			name:     "workspace_created defaults to non-blocking",
			hook:     LifecycleHookDef{Event: EventWorkspaceCreated},
			expected: false,
		},
		// Explicit override
		{
			name:     "explicit blocking=true on non-blocking event",
			hook:     LifecycleHookDef{Event: EventRunFailed, Blocking: boolPtr(true)},
			expected: true,
		},
		{
			name:     "explicit blocking=false on blocking event",
			hook:     LifecycleHookDef{Event: EventRunStart, Blocking: boolPtr(false)},
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.hook.IsBlocking())
		})
	}
}

func TestIsFailOpen(t *testing.T) {
	tests := []struct {
		name     string
		hook     LifecycleHookDef
		expected bool
	}{
		// Default fail_open per hook type
		{
			name:     "command defaults to fail_open=false",
			hook:     LifecycleHookDef{Type: HookTypeCommand},
			expected: false,
		},
		{
			name:     "script defaults to fail_open=false",
			hook:     LifecycleHookDef{Type: HookTypeScript},
			expected: false,
		},
		{
			name:     "http defaults to fail_open=true",
			hook:     LifecycleHookDef{Type: HookTypeHTTP},
			expected: true,
		},
		{
			name:     "llm_judge defaults to fail_open=true",
			hook:     LifecycleHookDef{Type: HookTypeLLMJudge},
			expected: true,
		},
		// Explicit override
		{
			name:     "explicit fail_open=true on command",
			hook:     LifecycleHookDef{Type: HookTypeCommand, FailOpen: boolPtr(true)},
			expected: true,
		},
		{
			name:     "explicit fail_open=false on http",
			hook:     LifecycleHookDef{Type: HookTypeHTTP, FailOpen: boolPtr(false)},
			expected: false,
		},
		{
			name:     "explicit fail_open=false on llm_judge",
			hook:     LifecycleHookDef{Type: HookTypeLLMJudge, FailOpen: boolPtr(false)},
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.hook.IsFailOpen())
		})
	}
}

func TestGetTimeout(t *testing.T) {
	tests := []struct {
		name     string
		hook     LifecycleHookDef
		expected time.Duration
	}{
		// Default timeouts per type
		{
			name:     "command defaults to 30s",
			hook:     LifecycleHookDef{Type: HookTypeCommand},
			expected: 30 * time.Second,
		},
		{
			name:     "http defaults to 10s",
			hook:     LifecycleHookDef{Type: HookTypeHTTP},
			expected: 10 * time.Second,
		},
		{
			name:     "llm_judge defaults to 60s",
			hook:     LifecycleHookDef{Type: HookTypeLLMJudge},
			expected: 60 * time.Second,
		},
		{
			name:     "script defaults to 30s",
			hook:     LifecycleHookDef{Type: HookTypeScript},
			expected: 30 * time.Second,
		},
		// Custom timeout
		{
			name:     "custom timeout 5s",
			hook:     LifecycleHookDef{Type: HookTypeCommand, Timeout: "5s"},
			expected: 5 * time.Second,
		},
		{
			name:     "custom timeout 2m",
			hook:     LifecycleHookDef{Type: HookTypeHTTP, Timeout: "2m"},
			expected: 2 * time.Minute,
		},
		{
			name:     "custom timeout 500ms",
			hook:     LifecycleHookDef{Type: HookTypeScript, Timeout: "500ms"},
			expected: 500 * time.Millisecond,
		},
		// Invalid timeout falls back to type default
		{
			name:     "invalid timeout falls back to command default",
			hook:     LifecycleHookDef{Type: HookTypeCommand, Timeout: "not-a-duration"},
			expected: 30 * time.Second,
		},
		{
			name:     "invalid timeout falls back to http default",
			hook:     LifecycleHookDef{Type: HookTypeHTTP, Timeout: "abc"},
			expected: 10 * time.Second,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.hook.GetTimeout())
		})
	}
}
