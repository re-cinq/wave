package ontology

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// hookMarker tags the git post-merge snippet written by InstallStalenessHook
// so we can detect and skip re-installation without duplicating the body.
const hookMarker = "# wave-ontology-staleness"

// InstallStalenessHook writes (or appends to) a git post-merge hook that
// touches the staleness sentinel after every merge. Idempotent: existing
// hooks that already contain the marker are left untouched.
func (s *realService) InstallStalenessHook() error {
	return InstallStalenessHookAt(".git/hooks")
}

// IsStaleInDir reports whether the staleness sentinel exists under waveDir.
// This lets non-Service callers (e.g. doctor, tui) check the staleness flag
// without owning the sentinel path literal.
func IsStaleInDir(waveDir string) bool {
	_, err := os.Stat(filepath.Join(waveDir, ".ontology-stale"))
	return err == nil
}

// IsStale is a convenience wrapper for the default .agents location.
func IsStale() bool {
	_, err := os.Stat(sentinelPath)
	return err == nil
}

// InstallStalenessHookAt is the testable form of InstallStalenessHook. The
// hookDir is typically ".git/hooks" for a plain repo.
func InstallStalenessHookAt(hookDir string) error {
	if _, err := os.Stat(hookDir); err != nil {
		return fmt.Errorf("not a git repository")
	}

	hookPath := filepath.Join(hookDir, "post-merge")
	snippet := "\n" + hookMarker + "\ntouch " + sentinelPath + " 2>/dev/null\n"

	if data, err := os.ReadFile(hookPath); err == nil {
		content := string(data)
		if strings.Contains(content, hookMarker) {
			return nil
		}
		return os.WriteFile(hookPath, []byte(content+snippet), 0o755)
	}

	hook := "#!/bin/sh" + snippet
	return os.WriteFile(hookPath, []byte(hook), 0o755)
}
