package hooks

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
)

// executeScript writes an inline script to a temp file, executes it, and cleans up.
func executeScript(ctx context.Context, hook *LifecycleHookDef, evt HookEvent) HookResult {
	timeout := hook.GetTimeout()
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Write script to temp file
	tmpFile, err := os.CreateTemp("", "wave-hook-*.sh")
	if err != nil {
		return HookResult{
			HookName: hook.Name,
			Decision: DecisionBlock,
			Reason:   "failed to create temp script file: " + err.Error(),
			Err:      err,
		}
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(hook.Script); err != nil {
		tmpFile.Close()
		return HookResult{
			HookName: hook.Name,
			Decision: DecisionBlock,
			Reason:   "failed to write script: " + err.Error(),
			Err:      err,
		}
	}
	tmpFile.Close()

	if err := os.Chmod(tmpFile.Name(), 0700); err != nil {
		return HookResult{
			HookName: hook.Name,
			Decision: DecisionBlock,
			Reason:   "failed to make script executable: " + err.Error(),
			Err:      err,
		}
	}

	cmd := exec.CommandContext(ctx, "sh", tmpFile.Name())
	cmd.Env = append(os.Environ(),
		"WAVE_HOOK_EVENT="+string(evt.Type),
		"WAVE_HOOK_PIPELINE="+evt.PipelineID,
		"WAVE_HOOK_STEP="+evt.StepID,
		"WAVE_HOOK_WORKSPACE="+evt.Workspace,
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	if runErr == nil {
		return HookResult{
			HookName: hook.Name,
			Decision: DecisionProceed,
		}
	}

	exitCode := 1
	if exitErr, ok := runErr.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	}

	reason := stderr.String()
	if exitCode == 2 {
		var jsonReason struct {
			Reason string `json:"reason"`
		}
		if json.Unmarshal(stderr.Bytes(), &jsonReason) == nil && jsonReason.Reason != "" {
			reason = jsonReason.Reason
		}
	}

	if reason == "" {
		reason = runErr.Error()
	}

	return HookResult{
		HookName: hook.Name,
		Decision: DecisionBlock,
		Reason:   reason,
		Err:      runErr,
	}
}
