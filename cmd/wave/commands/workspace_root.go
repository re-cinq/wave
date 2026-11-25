package commands

import (
	"fmt"
	"os"
	"path/filepath"

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

// manifestProjectDir returns the project directory for manifestPath: the
// directory holding the manifest.
func manifestProjectDir(manifestPath string) string {
	if manifestPath == "" {
		manifestPath = "wave.yaml"
	}
	dir, err := filepath.Abs(filepath.Dir(manifestPath))
	if err != nil {
		return "."
	}
	return dir
}

// unsafeRootError is the CLI error for a workspace root that equals or
// contains the project.
func unsafeRootError(err error) error {
	return NewCLIError(CodeSecurityViolation, fmt.Sprintf("refusing to touch workspaces: %s", err),
		"Point runtime.workspace_root at a dedicated directory outside the project, or remove it to use the default")
}
