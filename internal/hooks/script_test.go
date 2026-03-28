package hooks

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteScript(t *testing.T) {
	tests := []struct {
		name             string
		script           string
		expectedDecision HookDecision
		expectedReason   string
		checkReason      bool
	}{
		{
			name:             "exit 0 returns proceed",
			script:           "#!/bin/sh\nexit 0",
			expectedDecision: DecisionProceed,
		},
		{
			name:             "exit 1 returns block",
			script:           "#!/bin/sh\nexit 1",
			expectedDecision: DecisionBlock,
		},
		{
			name:             "exit 2 with JSON reason on stderr returns block with parsed reason",
			script:           "#!/bin/sh\necho '{\"reason\":\"validation failed\"}' >&2\nexit 2",
			expectedDecision: DecisionBlock,
			expectedReason:   "validation failed",
			checkReason:      true,
		},
		{
			name:             "exit 2 with plain stderr returns raw stderr",
			script:           "#!/bin/sh\necho 'plain error text' >&2\nexit 2",
			expectedDecision: DecisionBlock,
			expectedReason:   "plain error text",
			checkReason:      true,
		},
		{
			name:             "exit 1 with stderr returns stderr text",
			script:           "#!/bin/sh\necho 'error message' >&2\nexit 1",
			expectedDecision: DecisionBlock,
			expectedReason:   "error message",
			checkReason:      true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			hook := &LifecycleHookDef{
				Name:    "test-script-hook",
				Type:    HookTypeScript,
				Script:  tc.script,
				Timeout: "5s",
			}
			evt := HookEvent{
				Type:       EventStepStart,
				PipelineID: "test-pipeline",
				StepID:     "test-step",
			}

			result := executeScript(context.Background(), hook, evt)

			assert.Equal(t, tc.expectedDecision, result.Decision)
			assert.Equal(t, "test-script-hook", result.HookName)
			if tc.checkReason {
				assert.Contains(t, result.Reason, tc.expectedReason)
			}
		})
	}
}

func TestExecuteScriptMultiLine(t *testing.T) {
	script := `#!/bin/sh
RESULT="hello"
if [ "$RESULT" = "hello" ]; then
    echo "check passed" >&2
    exit 0
fi
exit 1
`
	hook := &LifecycleHookDef{
		Name:    "multiline-hook",
		Type:    HookTypeScript,
		Script:  script,
		Timeout: "5s",
	}
	evt := HookEvent{
		Type:       EventStepStart,
		PipelineID: "test-pipeline",
	}

	result := executeScript(context.Background(), hook, evt)

	assert.Equal(t, DecisionProceed, result.Decision)
}

func TestExecuteScriptWaveHookEnvVars(t *testing.T) {
	script := `#!/bin/sh
test "$WAVE_HOOK_EVENT" = "step_start" || exit 1
test "$WAVE_HOOK_PIPELINE" = "my-pipeline" || exit 1
test "$WAVE_HOOK_STEP_ID" = "my-step" || exit 1
exit 0
`
	hook := &LifecycleHookDef{
		Name:    "env-script-hook",
		Type:    HookTypeScript,
		Script:  script,
		Timeout: "5s",
	}
	evt := HookEvent{
		Type:       EventStepStart,
		PipelineID: "my-pipeline",
		StepID:     "my-step",
	}

	result := executeScript(context.Background(), hook, evt)

	assert.Equal(t, DecisionProceed, result.Decision,
		"WAVE_HOOK_* env vars should be available; reason: %s", result.Reason)
}

func TestExecuteScriptTempFileCleanup(t *testing.T) {
	hook := &LifecycleHookDef{
		Name:    "cleanup-hook",
		Type:    HookTypeScript,
		Script:  "#!/bin/sh\nexit 0",
		Timeout: "5s",
	}
	evt := HookEvent{
		Type:       EventStepStart,
		PipelineID: "test-pipeline",
	}

	// Get the temp dir contents before
	tmpDir := os.TempDir()
	beforeEntries, err := filepath.Glob(filepath.Join(tmpDir, "wave-hook-*.sh"))
	require.NoError(t, err)

	_ = executeScript(context.Background(), hook, evt)

	// Get the temp dir contents after
	afterEntries, err := filepath.Glob(filepath.Join(tmpDir, "wave-hook-*.sh"))
	require.NoError(t, err)

	// Find any new temp files (there should be none since cleanup runs)
	newFiles := diffSlices(beforeEntries, afterEntries)
	assert.Empty(t, newFiles, "temp files should be cleaned up after script execution")
}

func TestExecuteScriptTimeout(t *testing.T) {
	// Use a trap-based approach so the process can be killed cleanly by context timeout.
	hook := &LifecycleHookDef{
		Name:    "timeout-script-hook",
		Type:    HookTypeScript,
		Script:  "#!/bin/sh\nwhile true; do sleep 0.01; done",
		Timeout: "200ms",
	}
	evt := HookEvent{
		Type:       EventStepStart,
		PipelineID: "test-pipeline",
	}

	start := time.Now()
	result := executeScript(context.Background(), hook, evt)
	elapsed := time.Since(start)

	assert.Equal(t, DecisionBlock, result.Decision)
	assert.NotNil(t, result.Err)
	assert.Less(t, elapsed, 5*time.Second, "timeout should kill the script quickly")
}

// diffSlices returns elements in b that are not in a.
func diffSlices(a, b []string) []string {
	set := make(map[string]bool, len(a))
	for _, s := range a {
		set[s] = true
	}
	var diff []string
	for _, s := range b {
		if !set[s] {
			diff = append(diff, s)
		}
	}
	return diff
}

func TestExecuteScriptFailedScriptProducesErrorResult(t *testing.T) {
	// Script that does actual work but fails
	script := `#!/bin/sh
echo "doing work..."
echo "work failed" >&2
exit 1
`
	hook := &LifecycleHookDef{
		Name:    "work-script",
		Type:    HookTypeScript,
		Script:  script,
		Timeout: "5s",
	}
	evt := HookEvent{
		Type:       EventStepStart,
		PipelineID: "test-pipeline",
	}

	result := executeScript(context.Background(), hook, evt)

	assert.Equal(t, DecisionBlock, result.Decision)
	assert.NotNil(t, result.Err)
	assert.True(t, strings.Contains(result.Reason, "work failed"),
		"reason should contain stderr output, got: %s", result.Reason)
}
