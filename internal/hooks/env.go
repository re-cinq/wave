package hooks

import "github.com/recinq/wave/internal/config"

// buildHookEnv constructs a curated environment for hook subprocesses.
// Only WAVE_HOOK_* variables and base system variables (HOME, PATH, TERM, TMPDIR)
// are included. The full host environment is NOT inherited — this prevents
// sandbox bypass via environment leakage.
func buildHookEnv(evt HookEvent) []string {
	env := config.FromEnv()
	return []string{
		"HOME=" + env.Home,
		"PATH=" + env.Path,
		"TERM=" + env.TermOr("xterm-256color"),
		"TMPDIR=/tmp",
		"WAVE_HOOK_EVENT=" + string(evt.Type),
		"WAVE_HOOK_PIPELINE=" + evt.PipelineID,
		"WAVE_HOOK_STEP_ID=" + evt.StepID,
		"WAVE_HOOK_WORKSPACE=" + evt.Workspace,
	}
}
