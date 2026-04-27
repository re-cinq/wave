package pipeline

import (
	"path/filepath"
	"testing"
)

// TestImplFindingDeclaresWorktreeWorkspace asserts that impl-finding's apply-fix
// step uses a per-child worktree workspace. Issue #1413: the prior mount-based
// workspace had no `origin` remote, so `git push` silently no-op'd and ~19
// commits were dropped on a real run. Switching to `type: worktree` is the
// architectural fix; this test guards against regression.
func TestImplFindingDeclaresWorktreeWorkspace(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	path := filepath.Join(repoRoot, "internal", "defaults", "pipelines", "impl-finding.yaml")

	loader := &YAMLPipelineLoader{}
	pipe, err := loader.Load(path)
	if err != nil {
		t.Fatalf("load impl-finding.yaml: %v", err)
	}

	var apply *Step
	for i := range pipe.Steps {
		if pipe.Steps[i].ID == "apply-fix" {
			apply = &pipe.Steps[i]
			break
		}
	}
	if apply == nil {
		t.Fatal("impl-finding.yaml has no apply-fix step")
	}

	if apply.Workspace.Type != "worktree" {
		t.Errorf("apply-fix workspace.type = %q, want %q", apply.Workspace.Type, "worktree")
	}
	if len(apply.Workspace.Mount) != 0 {
		t.Errorf("apply-fix workspace.mount must be empty for worktree workspaces, got %d entries", len(apply.Workspace.Mount))
	}
	if apply.Workspace.Branch == "" {
		t.Error("apply-fix workspace.branch must be set for worktree workspaces (e.g. \"{{ pipeline_id }}\")")
	}
}

// TestOpsPRRespondResolveEachInjectsPRContext asserts the resolve-each step
// inside ops-pr-respond's resolve-and-verify loop:
//   - injects pr-context into each impl-finding child via config.inject (so the
//     child can read the PR head branch from .agents/artifacts/pr-context),
//   - runs in parallel mode (per-child worktrees fully isolate working trees,
//     so the prior serial pin from #1413 is no longer needed).
func TestOpsPRRespondResolveEachInjectsPRContext(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	path := filepath.Join(repoRoot, "internal", "defaults", "pipelines", "ops-pr-respond.yaml")

	loader := &YAMLPipelineLoader{}
	pipe, err := loader.Load(path)
	if err != nil {
		t.Fatalf("load ops-pr-respond.yaml: %v", err)
	}

	var resolveAndVerify *Step
	for i := range pipe.Steps {
		if pipe.Steps[i].ID == "resolve-and-verify" {
			resolveAndVerify = &pipe.Steps[i]
			break
		}
	}
	if resolveAndVerify == nil {
		t.Fatal("ops-pr-respond.yaml has no resolve-and-verify step")
	}
	if resolveAndVerify.Loop == nil {
		t.Fatal("resolve-and-verify must declare a loop")
	}

	var resolveEach *Step
	for i := range resolveAndVerify.Loop.Steps {
		if resolveAndVerify.Loop.Steps[i].ID == "resolve-each" {
			resolveEach = &resolveAndVerify.Loop.Steps[i]
			break
		}
	}
	if resolveEach == nil {
		t.Fatal("resolve-and-verify.loop.steps has no resolve-each entry")
	}

	if resolveEach.Config == nil {
		t.Fatal("resolve-each.config must be set so pr-context can be injected into impl-finding children")
	}
	found := false
	for _, name := range resolveEach.Config.Inject {
		if name == "pr-context" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("resolve-each.config.inject must contain \"pr-context\"; got %v", resolveEach.Config.Inject)
	}

	if resolveEach.Iterate == nil {
		t.Fatal("resolve-each must declare iterate")
	}
	if resolveEach.Iterate.Mode != "parallel" {
		t.Errorf("resolve-each.iterate.mode = %q, want %q (per-child worktrees isolate working trees)", resolveEach.Iterate.Mode, "parallel")
	}
}
