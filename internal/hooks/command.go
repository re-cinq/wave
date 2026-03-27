package hooks

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
)

// executeCommand runs a shell command and interprets its exit code.
// Exit 0 = proceed, exit 2 = block with JSON reason on stderr, other = block.
func executeCommand(ctx context.Context, hook *LifecycleHookDef, evt HookEvent) HookResult {
	timeout := hook.GetTimeout()
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", os.ExpandEnv(hook.Command))

	// Set hook event as environment variables
	cmd.Env = append(os.Environ(),
		"WAVE_HOOK_EVENT="+string(evt.Type),
		"WAVE_HOOK_PIPELINE="+evt.PipelineID,
		"WAVE_HOOK_STEP="+evt.StepID,
		"WAVE_HOOK_WORKSPACE="+evt.Workspace,
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		return HookResult{
			HookName: hook.Name,
			Decision: DecisionProceed,
		}
	}

	// Extract exit code
	exitCode := 1
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	}

	reason := stderr.String()
	if exitCode == 2 {
		// Try to parse JSON reason from stderr
		var jsonReason struct {
			Reason string `json:"reason"`
		}
		if json.Unmarshal(stderr.Bytes(), &jsonReason) == nil && jsonReason.Reason != "" {
			reason = jsonReason.Reason
		}
	}

	if reason == "" {
		reason = err.Error()
	}

	return HookResult{
		HookName: hook.Name,
		Decision: DecisionBlock,
		Reason:   reason,
		Err:      err,
	}
}
