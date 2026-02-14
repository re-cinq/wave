// Package contracts defines the Provisioner contract for skill command
// file provisioning into step workspaces.
//
// NOTE: This file is a design artifact, not production code.

package contracts

// --- Provisioner API Contract ---

// ProvisionerContract defines the public API of the skill Provisioner.
//
// Implementation location: internal/skill/skill.go
//
// No changes to existing API — the interface is already correct.
// The only fix needed is in the CALLER (executor.go:512) which must
// pass the correct repoRoot value instead of "".
type ProvisionerContract interface {
	// Provision copies skill command files into the workspace's staging directory.
	// Files are staged at workspacePath/.claude/commands/ (the provisioner target).
	// The adapter layer then copies these to its own settings directory.
	//
	// Parameters:
	//   workspacePath: base path for the staging directory
	//   skillNames: names of skills whose commands to provision
	//
	// Behavior:
	//   - Skips undeclared skills silently
	//   - Uses commands_glob from SkillConfig, falls back to .claude/commands/<name>.*.md
	//   - Resolves glob relative to repoRoot (constructor parameter)
	Provision(workspacePath string, skillNames []string) error

	// DiscoverCommands returns command files available for named skills.
	// Returns map[skillName][]relativePaths.
	DiscoverCommands(skillNames []string) (map[string][]string, error)

	// ProvisionAll provisions commands for all declared skills.
	ProvisionAll(workspacePath string) error
}
