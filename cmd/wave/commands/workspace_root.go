package commands

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/recinq/wave/internal/workspace"
	"gopkg.in/yaml.v3"
)

// manifestWorkspaceRoot resolves the workspace root for the project whose
// manifest is at manifestPath: runtime.workspace_root when set, otherwise the
// per-project default outside the checkout. The manifest is read leniently
// so housekeeping commands keep working when it is missing or invalid.
func manifestWorkspaceRoot(manifestPath string) string {
	if manifestPath == "" {
		manifestPath = "wave.yaml"
	}
	var m struct {
		Runtime struct {
			WorkspaceRoot string `yaml:"workspace_root"`
		} `yaml:"runtime"`
	}
	if data, err := os.ReadFile(manifestPath); err == nil {
		_ = yaml.Unmarshal(data, &m)
	}
	return workspace.ResolveRoot(m.Runtime.WorkspaceRoot)
}

// workspaceRootsToSweep returns the resolved workspace root plus the legacy
// in-checkout root (.agents/workspaces) when that is a different directory,
// so cleanup also reaches workspaces created by older Wave versions.
func workspaceRootsToSweep(manifestPath string) []string {
	root := manifestWorkspaceRoot(manifestPath)
	roots := []string{root}
	if !samePath(root, workspace.LegacyRoot) {
		roots = append(roots, workspace.LegacyRoot)
	}
	return roots
}

// samePath reports whether a and b name the same directory after making
// both absolute.
func samePath(a, b string) bool {
	absA, errA := filepath.Abs(a)
	absB, errB := filepath.Abs(b)
	if errA != nil || errB != nil {
		return a == b
	}
	return absA == absB
}

// containsCwd reports whether dir is the working directory or one of its
// ancestors; removing such a directory would delete the project itself.
func containsCwd(dir string) bool {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return true
	}
	wd, err := os.Getwd()
	if err != nil {
		return true
	}
	rel, err := filepath.Rel(absDir, wd)
	return err == nil && (rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))))
}
