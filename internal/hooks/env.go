package hooks

import "os"

// buildHookEnv constructs a curated environment for hook subprocesses.
// Only WAVE_HOOK_* variables and base system variables (HOME, PATH, TERM, TMPDIR)
// are included. The full host environment is NOT inherited — this prevents
// sandbox bypass via environment leakage.
func buildHookEnv(evt HookEvent) []string {
	return []string{
		"HOME=" + os.Getenv("HOME"),
		"PATH=" + os.Getenv("PATH"),
		"TERM=" + getenvDefault("TERM", "xterm-256color"),
		"TMPDIR=/tmp",
		"WAVE_HOOK_EVENT=" + string(evt.Type),
		"WAVE_HOOK_PIPELINE=" + evt.PipelineID,
		"WAVE_HOOK_STEP=" + evt.StepID,
		"WAVE_HOOK_WORKSPACE=" + evt.Workspace,
	}
}

func getenvDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
