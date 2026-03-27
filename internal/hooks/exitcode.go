package hooks

import (
	"encoding/json"
	"os/exec"
)

// interpretExitCode builds a HookResult from a command execution error.
// Exit 0 = proceed, exit 2 = block with optional JSON reason on stderr, other = block.
func interpretExitCode(hookName string, runErr error, stderr []byte) HookResult {
	if runErr == nil {
		return HookResult{
			HookName: hookName,
			Decision: DecisionProceed,
		}
	}
	exitCode := 1
	if exitErr, ok := runErr.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	}
	reason := string(stderr)
	if exitCode == 2 {
		var jsonReason struct {
			Reason string `json:"reason"`
		}
		if json.Unmarshal(stderr, &jsonReason) == nil && jsonReason.Reason != "" {
			reason = jsonReason.Reason
		}
	}
	if reason == "" {
		reason = runErr.Error()
	}
	return HookResult{
		HookName: hookName,
		Decision: DecisionBlock,
		Reason:   reason,
		Err:      runErr,
	}
}
