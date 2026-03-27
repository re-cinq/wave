package hooks

import (
	"context"
	"os"
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
		checkReason      bool // if true, assert reason contains expectedReason
	}{
		{
			name:             "exit 0 returns proceed",
			command:          "true",
			expectedDecision: DecisionProceed,
		},
		{
			name:             "exit 1 returns block",
			command:          "exit 1",
			expectedDecision: DecisionBlock,
		},
		{
			name:             "exit 2 with JSON stderr returns block with parsed reason",
			command:          `echo '{"reason":"bad code"}' >&2; exit 2`,
			expectedDecision: DecisionBlock,
			expectedReason:   "bad code",
			checkReason:      true,
		},
		{
			name:             "exit 2 with plain stderr returns block with stderr text",
			command:          `echo "plain error" >&2; exit 2`,
			expectedDecision: DecisionBlock,
			expectedReason:   "plain error",
			checkReason:      true,
		},
		{
			name:             "exit 1 with stderr returns block with stderr",
			command:          `echo "something went wrong" >&2; exit 1`,
			expectedDecision: DecisionBlock,
			expectedReason:   "something went wrong",
			checkReason:      true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			hook := &LifecycleHookDef{
				Name:    "test-hook",
				Type:    HookTypeCommand,
				Command: tc.command,
				Timeout: "5s",
			}
			evt := HookEvent{
				Type:       EventStepStart,
				PipelineID: "test-pipeline",
				StepID:     "test-step",
			}

			result := executeCommand(context.Background(), hook, evt)

			assert.Equal(t, tc.expectedDecision, result.Decision)
			assert.Equal(t, "test-hook", result.HookName)
			if tc.checkReason {
				assert.Contains(t, result.Reason, tc.expectedReason)
			}
			if tc.expectedDecision == DecisionProceed {
				assert.NoError(t, result.Err)
			} else {
				// Block decisions from failed commands should have an error
				assert.NotNil(t, result.Err)
			}
		})
	}
}

func TestExecuteCommandTimeout(t *testing.T) {
	hook := &LifecycleHookDef{
		Name:    "timeout-hook",
		Type:    HookTypeCommand,
		Command: "sleep 60",
		Timeout: "100ms",
	}
	evt := HookEvent{
		Type:       EventStepStart,
		PipelineID: "test-pipeline",
	}

	result := executeCommand(context.Background(), hook, evt)

	assert.Equal(t, DecisionBlock, result.Decision)
	assert.NotNil(t, result.Err)
	assert.NotEmpty(t, result.Reason)
}

func TestExecuteCommandEnvExpansion(t *testing.T) {
	// Set an environment variable and verify the command can expand it
	t.Setenv("WAVE_TEST_VAR", "hello_world")

	hook := &LifecycleHookDef{
		Name:    "env-hook",
		Type:    HookTypeCommand,
		Command: `test "$WAVE_TEST_VAR" = "hello_world"`,
		Timeout: "5s",
	}
	evt := HookEvent{
		Type:       EventStepStart,
		PipelineID: "test-pipeline",
	}

	result := executeCommand(context.Background(), hook, evt)

	assert.Equal(t, DecisionProceed, result.Decision)
}

func TestExecuteCommandWaveHookEnvVars(t *testing.T) {
	// WAVE_HOOK_* vars are set in cmd.Env (child process), but os.ExpandEnv runs
	// in the parent process first. We must avoid $VAR syntax in the command string
	// because os.ExpandEnv would expand them to empty. Instead we use printenv which
	// reads from the child process environment.
	hook := &LifecycleHookDef{
		Name:    "env-check-hook",
		Type:    HookTypeCommand,
		Command: `printenv WAVE_HOOK_EVENT | grep -q step_start && printenv WAVE_HOOK_PIPELINE | grep -q my-pipeline && printenv WAVE_HOOK_STEP | grep -q my-step`,
		Timeout: "5s",
	}
	evt := HookEvent{
		Type:       EventStepStart,
		PipelineID: "my-pipeline",
		StepID:     "my-step",
		Workspace:  "/tmp/test-workspace",
	}

	result := executeCommand(context.Background(), hook, evt)

	assert.Equal(t, DecisionProceed, result.Decision,
		"WAVE_HOOK_* env vars should be available; got reason: %s", result.Reason)
}

func TestExecuteCommandWorkspaceEnvVar(t *testing.T) {
	// WAVE_HOOK_WORKSPACE is set in cmd.Env. We use printenv to read it from
	// the child process environment, avoiding os.ExpandEnv in the parent.
	tmpDir := t.TempDir()
	hook := &LifecycleHookDef{
		Name:    "workspace-env-hook",
		Type:    HookTypeCommand,
		Command: `printenv WAVE_HOOK_WORKSPACE | grep -qF "` + tmpDir + `"`,
		Timeout: "5s",
	}
	evt := HookEvent{
		Type:       EventStepStart,
		PipelineID: "test-pipeline",
		StepID:     "test-step",
		Workspace:  tmpDir,
	}

	result := executeCommand(context.Background(), hook, evt)

	require.Equal(t, DecisionProceed, result.Decision,
		"WAVE_HOOK_WORKSPACE should be set; err: %v, reason: %s", result.Err, result.Reason)
}

func TestExecuteCommandOsExpandEnv(t *testing.T) {
	// Verify os.ExpandEnv expands variables present in the current process env
	key := "WAVE_EXPAND_TEST_VAR"
	os.Setenv(key, "expanded_value")
	defer os.Unsetenv(key)

	hook := &LifecycleHookDef{
		Name: "expand-hook",
		Type: HookTypeCommand,
		// os.ExpandEnv will replace $WAVE_EXPAND_TEST_VAR before execution
		Command: `echo $WAVE_EXPAND_TEST_VAR | grep expanded_value`,
		Timeout: "5s",
	}
	evt := HookEvent{
		Type:       EventStepStart,
		PipelineID: "test-pipeline",
	}

	result := executeCommand(context.Background(), hook, evt)

	assert.Equal(t, DecisionProceed, result.Decision)
}
