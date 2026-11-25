package workspace

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveRoot_ConfiguredWins(t *testing.T) {
	if got := ResolveRoot("/srv/ws"); got != "/srv/ws" {
		t.Fatalf("ResolveRoot(/srv/ws) = %q", got)
	}
	if got := ResolveRoot(LegacyRoot); got != LegacyRoot {
		t.Fatalf("ResolveRoot(%q) = %q", LegacyRoot, got)
	}
}

func TestDefaultRoot_OutsideProjectUnderCache(t *testing.T) {
	cache := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cache)
	project := t.TempDir()
	t.Chdir(project)

	root := ResolveRoot("  ")
	if root != DefaultRoot() {
		t.Fatalf("empty workspace_root should resolve to DefaultRoot, got %q", root)
	}
	if !strings.HasPrefix(root, filepath.Join(cache, "wave", "workspaces")+string(filepath.Separator)) {
		t.Fatalf("default root %q not under %s", root, cache)
	}
	resolvedProject, _ := filepath.EvalSymlinks(project)
	if rel, err := filepath.Rel(resolvedProject, root); err == nil && !strings.HasPrefix(rel, "..") {
		t.Fatalf("default root %q is inside the project %q", root, resolvedProject)
	}
	if !strings.HasPrefix(filepath.Base(root), filepath.Base(project)+"-") {
		t.Fatalf("default root %q should start with the project name", root)
	}
}

func TestDefaultRootFor_StablePerProject(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	a, b := t.TempDir(), t.TempDir()
	first, second := DefaultRootFor(a), DefaultRootFor(a)
	if first != second {
		t.Fatal("default root must be stable for the same project")
	}
	if DefaultRootFor(a) == DefaultRootFor(b) {
		t.Fatal("different projects must get different roots")
	}
	if DefaultRootFor(a) != DefaultRootFor(a+string(filepath.Separator)+".") {
		t.Fatal("equivalent paths must map to the same root")
	}
}
