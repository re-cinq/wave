package hooks

import (
	"bytes"
	"context"
	"os"
	"os/exec"
)

// executeScript writes an inline script to a temp file, executes it, and cleans up.
func executeScript(ctx context.Context, hook *LifecycleHookDef, evt HookEvent) HookResult {
	timeout := hook.GetTimeout()
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	tmpFile, err := os.CreateTemp("", "wave-hook-*.sh")
	if err != nil {
		return HookResult{HookName: hook.Name, Decision: DecisionBlock, Reason: "failed to create temp script file: " + err.Error(), Err: err}
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	if _, err := tmpFile.WriteString(hook.Script); err != nil {
		tmpFile.Close()
		return HookResult{HookName: hook.Name, Decision: DecisionBlock, Reason: "failed to write script: " + err.Error(), Err: err}
	}
	tmpFile.Close()

	if err := os.Chmod(tmpFile.Name(), 0700); err != nil {
		return HookResult{HookName: hook.Name, Decision: DecisionBlock, Reason: "failed to make script executable: " + err.Error(), Err: err}
	}

	cmd := exec.CommandContext(ctx, "sh", tmpFile.Name())
	cmd.Env = buildHookEnv(evt)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	return interpretExitCode(hook.Name, runErr, stderr.Bytes())
}
