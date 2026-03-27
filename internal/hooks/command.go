package hooks

import (
	"bytes"
	"context"
	"os/exec"
)

// executeCommand runs a shell command and interprets its exit code.
// Exit 0 = proceed, exit 2 = block with JSON reason on stderr, other = block.
func executeCommand(ctx context.Context, hook *LifecycleHookDef, evt HookEvent) HookResult {
	timeout := hook.GetTimeout()
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// The shell itself handles variable expansion — no need for os.ExpandEnv
	// which would incorrectly expand WAVE_HOOK_* vars in the parent process.
	cmd := exec.CommandContext(ctx, "sh", "-c", hook.Command)

	// Curated environment: only base system vars + WAVE_HOOK_* variables.
	// The full host environment is NOT inherited to prevent sandbox bypass.
	cmd.Env = buildHookEnv(evt)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return interpretExitCode(hook.Name, err, stderr.Bytes())
}
