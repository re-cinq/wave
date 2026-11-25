package workspace

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// LegacyRoot is where workspaces lived before they moved out of the project
// checkout. It is still honoured when a manifest sets it explicitly as
// runtime.workspace_root, and cleanup commands also sweep it so workspaces
// left behind by older versions can be removed.
const LegacyRoot = ".agents/workspaces"

var unsafeNameChars = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

// ResolveRoot returns the workspace root to use for a project. A non-empty
// configured value (runtime.workspace_root) wins; otherwise the per-project
// default outside the checkout is returned (see DefaultRoot).
func ResolveRoot(configured string) string {
	if strings.TrimSpace(configured) != "" {
		return configured
	}
	return DefaultRoot()
}

// DefaultRoot returns the default workspace root for the project in the
// current working directory: <user cache dir>/wave/workspaces/<name>-<hash>,
// where <name> is the project directory's base name and <hash> is derived
// from its absolute path. On Linux the cache dir is $XDG_CACHE_HOME or
// ~/.cache.
//
// Workspaces live outside the checkout on purpose: when a step's git
// worktree sits inside the project directory, agents regularly walk up and
// edit the parent checkout instead of the worktree they were given.
func DefaultRoot() string {
	return DefaultRootFor(projectDir())
}

// DefaultRootFor is DefaultRoot for an explicit project directory.
func DefaultRootFor(project string) string {
	if abs, err := filepath.Abs(project); err == nil {
		project = abs
	}
	if resolved, err := filepath.EvalSymlinks(project); err == nil {
		project = resolved
	}
	sum := sha256.Sum256([]byte(project))
	name := unsafeNameChars.ReplaceAllString(filepath.Base(project), "_")
	if name == "" || name == "." || name == string(filepath.Separator) {
		name = "project"
	}
	return filepath.Join(cacheBase(), "wave", "workspaces", name+"-"+hex.EncodeToString(sum[:])[:12])
}

func projectDir() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

func cacheBase() string {
	if dir, err := os.UserCacheDir(); err == nil && dir != "" {
		return dir
	}
	return os.TempDir()
}
