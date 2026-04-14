package skill

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGhCliSkillSplit validates that the gh-cli content split is intact:
// SKILL.md must be lean (core patterns only) and the full reference must
// be discoverable as a resource so ProvisionFromStore copies it to the workspace.
func TestGhCliSkillSplit(t *testing.T) {
	store := NewDirectoryStore(SkillSource{Root: "../../skills", Precedence: 1})

	s, err := store.Read("gh-cli")
	require.NoError(t, err, "gh-cli skill must be readable from skills/")

	// Body must be present but lean — full reference lives in references/
	require.NotEmpty(t, s.Body, "gh-cli SKILL.md body must not be empty")
	lines := strings.Count(s.Body, "\n")
	assert.Less(t, lines, 200,
		"gh-cli SKILL.md body is %d lines — split reference material into references/", lines)

	// Agent must be able to find the full reference from the body pointer
	assert.Contains(t, s.Body, "references/full-reference.md",
		"SKILL.md body must reference full-reference.md so agents can locate it")

	// ProvisionFromStore copies ResourcePaths to the workspace —
	// if full-reference.md is not discovered here it will not be available at runtime
	assert.Contains(t, s.ResourcePaths, "references/full-reference.md",
		"references/full-reference.md must be in ResourcePaths for provisioning")
}

// TestGhCliNativeSkillSplit validates the same constraints for the .claude/skills/
// copy, which is what Claude Code injects natively into interactive sessions and
// into pipeline worktrees (since worktrees inherit .claude/ from the project root).
func TestGhCliNativeSkillSplit(t *testing.T) {
	store := NewDirectoryStore(SkillSource{Root: "../../.claude/skills", Precedence: 1})

	s, err := store.Read("gh-cli")
	require.NoError(t, err, "gh-cli skill must be readable from .claude/skills/")

	lines := strings.Count(s.Body, "\n")
	assert.Less(t, lines, 200,
		".claude/skills/gh-cli SKILL.md is %d lines — must match trimmed skills/gh-cli/SKILL.md", lines)

	assert.Contains(t, s.Body, "references/full-reference.md",
		".claude/skills/gh-cli/SKILL.md must reference full-reference.md")

	assert.Contains(t, s.ResourcePaths, "references/full-reference.md",
		"references/full-reference.md must be in ResourcePaths")
}
