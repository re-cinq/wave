package hooks

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteCommand(t *testing.T) {
	tests := []struct {
		name             string
		command          string
		expectedDecision HookDecision
		expectedReason   string
		checkReason      bool
	}{
		{name: "exit 0 returns proceed", command: "true", expectedDecision: DecisionProceed},
		{name: "exit 1 returns block", command: "exit 1", expectedDecision: DecisionBlock},
		{name: "exit 2 with JSON stderr", command: `echo '{"reason":"bad code"}' >&2; exit 2`, expectedDecision: DecisionBlock, expectedReason: "bad code", checkReason: true},
		{name: "exit 2 with plain stderr", command: `echo "plain error" >&2; exit 2`, expectedDecision: DecisionBlock, expectedReason: "plain error", checkReason: true},
		{name: "exit 1 with stderr", command: `echo "something went wrong" >&2; exit 1`, expectedDecision: DecisionBlock, expectedReason: "something went wrong", checkReason: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			hook := &LifecycleHookDef{Name: "test-hook", Type: HookTypeCommand, Command: tc.command, Timeout: "5s"}
			evt := HookEvent{Type: EventStepStart, PipelineID: "test-pipeline", StepID: "test-step"}
			result := executeCommand(context.Background(), hook, evt)
			assert.Equal(t, tc.expectedDecision, result.Decision)
			assert.Equal(t, "test-hook", result.HookName)
			if tc.checkReason {
				assert.Contains(t, result.Reason, tc.expectedReason)
			}
			if tc.expectedDecision == DecisionProceed {
				assert.NoError(t, result.Err)
			} else {
				assert.NotNil(t, result.Err)
			}
		})
	}
}

func TestExecuteCommandTimeout(t *testing.T) {
	hook := &LifecycleHookDef{Name: "timeout-hook", Type: HookTypeCommand, Command: "sleep 60", Timeout: "100ms"}
	result := executeCommand(context.Background(), hook, HookEvent{Type: EventStepStart, PipelineID: "test-pipeline"})
	assert.Equal(t, DecisionBlock, result.Decision)
	assert.NotNil(t, result.Err)
}

func TestExecuteCommandWaveHookEnvVars(t *testing.T) {
	hook := &LifecycleHookDef{Name: "env-check-hook", Type: HookTypeCommand, Command: `printenv WAVE_HOOK_EVENT | grep -q step_start && printenv WAVE_HOOK_PIPELINE | grep -q my-pipeline && printenv WAVE_HOOK_STEP_ID | grep -q my-step`, Timeout: "5s"}
	result := executeCommand(context.Background(), hook, HookEvent{Type: EventStepStart, PipelineID: "my-pipeline", StepID: "my-step", Workspace: "/tmp/test-workspace"})
	assert.Equal(t, DecisionProceed, result.Decision, "WAVE_HOOK_* env vars should be available; got reason: %s", result.Reason)
}

func TestExecuteCommandWorkspaceEnvVar(t *testing.T) {
	tmpDir := t.TempDir()
	hook := &LifecycleHookDef{Name: "workspace-env-hook", Type: HookTypeCommand, Command: `printenv WAVE_HOOK_WORKSPACE | grep -qF "` + tmpDir + `"`, Timeout: "5s"}
	result := executeCommand(context.Background(), hook, HookEvent{Type: EventStepStart, PipelineID: "test-pipeline", StepID: "test-step", Workspace: tmpDir})
	require.Equal(t, DecisionProceed, result.Decision, "WAVE_HOOK_WORKSPACE should be set; err: %v, reason: %s", result.Err, result.Reason)
}

func TestExecuteCommandCuratedEnv(t *testing.T) {
	t.Setenv("WAVE_SECRET_TOKEN", "should-not-leak")
	hook := &LifecycleHookDef{Name: "curated-env-hook", Type: HookTypeCommand, Command: `printenv WAVE_SECRET_TOKEN 2>/dev/null && exit 1 || exit 0`, Timeout: "5s"}
	result := executeCommand(context.Background(), hook, HookEvent{Type: EventStepStart, PipelineID: "test-pipeline"})
	assert.Equal(t, DecisionProceed, result.Decision, "arbitrary host env vars should NOT leak into hook subprocess")
}

func TestExecuteCommandBaseEnvVars(t *testing.T) {
	hook := &LifecycleHookDef{Name: "base-env-hook", Type: HookTypeCommand, Command: `test -n "$HOME" && test -n "$PATH"`, Timeout: "5s"}
	result := executeCommand(context.Background(), hook, HookEvent{Type: EventStepStart, PipelineID: "test-pipeline"})
	assert.Equal(t, DecisionProceed, result.Decision, "HOME and PATH should be set; reason: %s", result.Reason)
}
