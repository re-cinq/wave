package workspace

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
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

// RunMarker is the file Wave writes into every run directory it creates
// under a workspace root. Cleanup commands only delete directories that
// carry it (or that live in the legacy in-checkout root), so a
// misconfigured workspace_root can never make them delete foreign data.
const RunMarker = ".wave-run"

// MarkRunDir records dir as a Wave-created run directory. dir must exist.
// The marker holds the run ID (the directory name), so a marker copied or
// planted into another directory does not make that directory Wave's.
func MarkRunDir(dir string) error {
	return os.WriteFile(filepath.Join(dir, RunMarker), []byte(filepath.Base(dir)+"\n"), 0644)
}

// hasRunMarker reports whether dir holds a regular RunMarker file whose
// content is dir's own name.
func hasRunMarker(dir string) bool {
	marker := filepath.Join(dir, RunMarker)
	info, err := os.Lstat(marker)
	if err != nil || !info.Mode().IsRegular() || info.Size() > 4096 {
		return false
	}
	data, err := os.ReadFile(marker)
	return err == nil && strings.TrimSpace(string(data)) == filepath.Base(dir)
}

// canonical returns the absolute, symlink-resolved form of path. Missing
// trailing components are kept as-is on top of the resolved existing
// prefix, so paths that do not exist yet still compare correctly.
func canonical(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	rest := ""
	cur := abs
	for {
		if resolved, err := filepath.EvalSymlinks(cur); err == nil {
			return filepath.Join(resolved, rest), nil
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return abs, nil
		}
		rest = filepath.Join(filepath.Base(cur), rest)
		cur = parent
	}
}

// within reports whether path is root or lies below it. Both must be
// canonical.
func within(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

// CheckRoot returns an error when root cannot safely hold workspaces for
// the project in projectDir: when it equals or contains the project
// directory (or the git repository it belongs to). Deleting anything
// under such a root could delete the project itself.
func CheckRoot(root, projectDir string) error {
	r, err := canonical(root)
	if err != nil {
		return fmt.Errorf("resolve workspace root %q: %w", root, err)
	}
	projects := []string{projectDir}
	if top, err := gitTopLevel(projectDir); err == nil {
		projects = append(projects, top)
	}
	for _, p := range projects {
		cp, err := canonical(p)
		if err != nil {
			return fmt.Errorf("resolve project dir %q: %w", p, err)
		}
		if within(r, cp) {
			return fmt.Errorf("workspace root %s equals or contains the project directory %s", r, cp)
		}
	}
	return nil
}

func gitTopLevel(dir string) (string, error) {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// isLegacyRoot reports whether root is the pre-marker in-checkout root
// <projectDir>/.agents/workspaces as a real directory, whose entries all
// belong to Wave. If .agents or .agents/workspaces is a symlink (a cloned
// repo can ship one) it points somewhere Wave does not own, so it is not
// treated as the legacy root.
func isLegacyRoot(root, projectDir string) bool {
	if LegacyRootIsSymlink(projectDir) {
		return false
	}
	r, err1 := canonical(root)
	p, err2 := canonical(projectDir)
	return err1 == nil && err2 == nil && r == filepath.Join(p, LegacyRoot)
}

// LegacyRootIsSymlink reports whether <projectDir>/.agents or
// <projectDir>/.agents/workspaces is a symbolic link. Cleanup must then
// remove at most the link itself and never descend into its target.
func LegacyRootIsSymlink(projectDir string) bool {
	for _, rel := range []string{".agents", LegacyRoot} {
		if info, err := os.Lstat(filepath.Join(projectDir, rel)); err == nil && info.Mode()&os.ModeSymlink != 0 {
			return true
		}
	}
	return false
}

// OwnedRunDir reports whether dir is a run directory Wave created directly
// below root: a single path component below the canonical root, a real
// directory (not a symlink), carrying RunMarker unless root is the legacy
// in-checkout root, and neither equal to nor containing the project.
func OwnedRunDir(root, dir, projectDir string) bool {
	r, err := canonical(root)
	if err != nil {
		return false
	}
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() {
		return false
	}
	d, err := canonical(dir)
	if err != nil || filepath.Dir(d) != r {
		return false
	}
	if p, err := canonical(projectDir); err != nil || within(d, p) {
		return false
	}
	if isLegacyRoot(root, projectDir) {
		return true
	}
	return hasRunMarker(d)
}

// OwnedRunDirs lists the run directories under root that OwnedRunDir
// accepts. A missing root yields no entries.
func OwnedRunDirs(root, projectDir string) []string {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var dirs []string
	for _, e := range entries {
		p := filepath.Join(root, e.Name())
		if OwnedRunDir(root, p, projectDir) {
			dirs = append(dirs, p)
		}
	}
	return dirs
}

// OwnedWorktree reports whether wt is a step worktree Wave created under
// root: <root>/<owned run dir>/__wt_<name>, and not the project itself.
func OwnedWorktree(root, wt, projectDir string) bool {
	w, err := canonical(wt)
	if err != nil || !strings.HasPrefix(filepath.Base(w), "__wt_") {
		return false
	}
	if p, err := canonical(projectDir); err != nil || within(w, p) {
		return false
	}
	return OwnedRunDir(root, filepath.Dir(w), projectDir)
}
