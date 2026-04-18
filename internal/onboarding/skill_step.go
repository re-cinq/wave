package onboarding

import (
	"fmt"
	"os"
)

// SkillSelectionStep is a no-op wizard step left in place so the wizard's
// step count and reconfigure semantics remain stable. The earlier
// tessl/bmad/openspec/speckit ecosystem selector was removed in #1113 along
// with the source adapters that backed it. Skills are now installed post-
// onboarding via `wave skills add <path>` (or `--project` to commit them).
type SkillSelectionStep struct{}

// Name returns the display name for this wizard step.
func (s *SkillSelectionStep) Name() string { return "Skill Selection" }

// Run preserves any existing skills (reconfigure path) and otherwise returns
// an empty list. In interactive mode it prints a one-line hint pointing at
// the new install command.
func (s *SkillSelectionStep) Run(cfg *WizardConfig) (*StepResult, error) {
	existing := []string{}
	if cfg.Reconfigure && cfg.Existing != nil {
		existing = cfg.Existing.Skills
	}

	if cfg.Interactive {
		fmt.Fprintln(os.Stderr, "\n  Skills are now installed post-setup via `wave skills add <path>`")
		fmt.Fprintln(os.Stderr, "  (use `--project` to commit a skill to .agents/skills/).")
		fmt.Fprintln(os.Stderr, "  Run `wave skills list` to see what's already discoverable.")
		fmt.Fprintln(os.Stderr)
	}

	return &StepResult{
		Data: map[string]interface{}{
			"skills": existing,
		},
	}, nil
}
