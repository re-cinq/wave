package adapter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SentinelFile is written into every Wave-provisioned skill directory so future
// runs can distinguish Wave-managed dirs from user-committed ones. Only dirs
// containing this file are eligible for removal during re-provisioning.
const SentinelFile = ".wave-managed"

// ProvisionSkills installs the given skills into <workspacePath>/<targetSubdir>/
// using the adapter's native skill discovery layout. The flow is:
//
//  1. Workspace-scope assertion: panic if targetSubdir resolves outside
//     workspacePath. This is a defensive guard against accidental host damage.
//  2. Clear stale Wave-managed skills: walk the existing target dir and
//     RemoveAll any subdir containing the sentinel file. User-committed skills
//     (no sentinel) are left untouched.
//  3. Copy each skill source dir into <target>/<name>/ and drop the sentinel.
//
// targetSubdir must be a relative path (e.g. ".claude/skills" or
// ".agents/skills"). workspacePath must be absolute or otherwise unambiguous;
// the safety assertion uses string-prefix matching after Clean().
func ProvisionSkills(workspacePath, targetSubdir string, skills []SkillRef) error {
	if workspacePath == "" {
		return fmt.Errorf("workspacePath is required for skill provisioning")
	}
	if filepath.IsAbs(targetSubdir) {
		return fmt.Errorf("targetSubdir must be relative, got %q", targetSubdir)
	}

	workspaceClean := filepath.Clean(workspacePath)
	skillsDir := filepath.Clean(filepath.Join(workspaceClean, targetSubdir))

	// Safety: refuse to act on a path outside the workspace. This protects
	// against config tampering or path-traversal in targetSubdir.
	prefix := workspaceClean + string(os.PathSeparator)
	if skillsDir != workspaceClean && !strings.HasPrefix(skillsDir, prefix) {
		panic(fmt.Sprintf("refusing skill provisioning outside workspace: workspace=%s target=%s", workspaceClean, skillsDir))
	}

	if err := clearWaveManagedSkills(skillsDir); err != nil {
		return fmt.Errorf("clear stale skills in %s: %w", skillsDir, err)
	}

	if len(skills) == 0 {
		return nil
	}

	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		return fmt.Errorf("create skills dir %s: %w", skillsDir, err)
	}

	for _, ref := range skills {
		if ref.SourcePath == "" {
			continue
		}
		dst := filepath.Join(skillsDir, ref.Name)
		if err := copySkillDir(ref.SourcePath, dst); err != nil {
			return fmt.Errorf("provision skill %q to %s: %w", ref.Name, dst, err)
		}
		sentinel := filepath.Join(dst, SentinelFile)
		if err := os.WriteFile(sentinel, []byte("wave-provisioned\n"), 0o644); err != nil {
			return fmt.Errorf("write sentinel for skill %q: %w", ref.Name, err)
		}
	}
	return nil
}

// clearWaveManagedSkills removes only subdirectories that contain SentinelFile.
// Subdirs without the sentinel are user-committed and preserved.
func clearWaveManagedSkills(skillsDir string) error {
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(skillsDir, entry.Name())
		if _, statErr := os.Stat(filepath.Join(path, SentinelFile)); statErr != nil {
			continue
		}
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("remove %s: %w", path, err)
		}
	}
	return nil
}
