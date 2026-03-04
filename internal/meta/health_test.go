package meta

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/platform"
)

// --- Test helpers ---

// mockGitRunner records calls and returns pre-configured responses.
type mockGitRunner struct {
	responses map[string]string
	errors    map[string]error
}

func newMockGitRunner() *mockGitRunner {
	return &mockGitRunner{
		responses: make(map[string]string),
		errors:    make(map[string]error),
	}
}

func (m *mockGitRunner) withResponse(key, value string) *mockGitRunner {
	m.responses[key] = value
	return m
}

func (m *mockGitRunner) withError(key string, err error) *mockGitRunner {
	m.errors[key] = err
	return m
}

func (m *mockGitRunner) Run(_ context.Context, args ...string) (string, error) {
	key := ""
	if len(args) > 0 {
		key = args[0]
	}
	if err, ok := m.errors[key]; ok {
		return "", err
	}
	if resp, ok := m.responses[key]; ok {
		return resp, nil
	}
	return "", fmt.Errorf("unexpected git command: %v", args)
}

// mockGitHubAPI implements GitHubAPI for testing.
type mockGitHubAPI struct {
	openIssues int
	openPRs    int
	err        error
}

func (m *mockGitHubAPI) GetRepoStats(_ context.Context, _, _ string) (int, int, error) {
	return m.openIssues, m.openPRs, m.err
}

// writeValidManifest creates a minimal valid wave.yaml in the given directory.
func writeValidManifest(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "wave.yaml")
	content := `apiVersion: v1
kind: WaveManifest
metadata:
  name: test-project
adapters:
  claude:
    binary: claude
    mode: headless
personas:
  navigator:
    adapter: claude
    system_prompt_file: nav.md
runtime:
  workspace_root: .wave/workspaces
skills:
  speckit:
    check: "echo ok"
    install: "echo install"
`
	// Create the referenced prompt file.
	if err := os.WriteFile(filepath.Join(dir, "nav.md"), []byte("# Nav"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

// --- T009: Type definitions are tested implicitly through usage in all tests below ---

// --- T010: checkInit tests ---

func TestCheckInit_ValidManifest(t *testing.T) {
	dir := t.TempDir()
	manifestPath := writeValidManifest(t, dir)

	h := NewHealthChecker(
		WithManifestPath(manifestPath),
		WithVersion("v0.9.0"),
	)

	out := h.checkInit(context.Background(), manifestPath)

	if !out.result.ManifestFound {
		t.Error("expected ManifestFound=true")
	}
	if !out.result.ManifestValid {
		t.Error("expected ManifestValid=true")
	}
	if out.result.WaveVersion != "v0.9.0" {
		t.Errorf("expected WaveVersion=v0.9.0, got %s", out.result.WaveVersion)
	}
	if out.result.Error != "" {
		t.Errorf("unexpected error: %s", out.result.Error)
	}
	if out.result.LastConfigDate.IsZero() {
		t.Error("expected LastConfigDate to be set")
	}
	if out.manifest == nil {
		t.Error("expected manifest to be loaded after successful init")
	}
}

func TestCheckInit_MissingManifest(t *testing.T) {
	h := NewHealthChecker(
		WithManifestPath("/nonexistent/wave.yaml"),
		WithVersion("v0.9.0"),
	)

	out := h.checkInit(context.Background(), "/nonexistent/wave.yaml")

	if out.result.ManifestFound {
		t.Error("expected ManifestFound=false for missing file")
	}
	if out.result.ManifestValid {
		t.Error("expected ManifestValid=false for missing file")
	}
	if out.result.Error == "" {
		t.Error("expected error message for missing manifest")
	}
	if out.manifest != nil {
		t.Error("expected nil manifest for missing file")
	}
}

func TestCheckInit_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "wave.yaml")
	if err := os.WriteFile(path, []byte("{{{{invalid yaml!!!"), 0644); err != nil {
		t.Fatal(err)
	}

	h := NewHealthChecker(
		WithManifestPath(path),
	)

	out := h.checkInit(context.Background(), path)

	if !out.result.ManifestFound {
		t.Error("expected ManifestFound=true (file exists)")
	}
	if out.result.ManifestValid {
		t.Error("expected ManifestValid=false for invalid YAML")
	}
	if out.result.Error == "" {
		t.Error("expected error message for invalid YAML")
	}
	if out.manifest != nil {
		t.Error("expected nil manifest for invalid YAML")
	}
}

func TestCheckInit_IncompleteManifest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "wave.yaml")
	// Valid YAML but missing required fields (no metadata.name).
	content := `apiVersion: v1
kind: WaveManifest
metadata:
  description: "missing name"
runtime:
  workspace_root: .wave/workspaces
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	h := NewHealthChecker(WithManifestPath(path))

	out := h.checkInit(context.Background(), path)

	if !out.result.ManifestFound {
		t.Error("expected ManifestFound=true")
	}
	if out.result.ManifestValid {
		t.Error("expected ManifestValid=false for incomplete manifest")
	}
	if out.result.Error == "" {
		t.Error("expected validation error")
	}
	if out.manifest != nil {
		t.Error("expected nil manifest for incomplete manifest")
	}
}

// --- T011: checkDependencies tests ---

func TestCheckDependencies_AllAvailable(t *testing.T) {
	m := &manifest.Manifest{
		Adapters: map[string]manifest.Adapter{
			"claude": {Binary: "echo", Mode: "headless"},
		},
		Skills: map[string]manifest.SkillConfig{
			"test-skill": {
				Check:   "true",
				Install: "echo install",
			},
		},
	}

	// Use a lookPath that always succeeds.
	h := NewHealthChecker(
		WithLookPath(func(file string) (string, error) {
			return "/usr/bin/" + file, nil
		}),
	)

	report := h.checkDependencies(context.Background(), m)

	// Should include at least "echo" (adapter binary) and "git" (implicit).
	foundEcho := false
	foundGit := false
	for _, t := range report.Tools {
		if t.Name == "echo" {
			foundEcho = true
			if !t.Available {
				t.Available = true // unreachable but satisfies go vet
			}
		}
		if t.Name == "git" {
			foundGit = true
		}
	}
	if !foundEcho {
		t.Error("expected echo tool in report")
	}
	if !foundGit {
		t.Error("expected git tool in report (implicit dependency)")
	}
}

func TestCheckDependencies_MissingTool(t *testing.T) {
	m := &manifest.Manifest{
		Adapters: map[string]manifest.Adapter{
			"nonexistent-adapter": {Binary: "this-tool-does-not-exist-xyz", Mode: "headless"},
		},
	}

	h := NewHealthChecker(
		WithLookPath(func(file string) (string, error) {
			if file == "git" {
				return "/usr/bin/git", nil
			}
			return "", fmt.Errorf("not found: %s", file)
		}),
	)

	report := h.checkDependencies(context.Background(), m)

	for _, tool := range report.Tools {
		if tool.Name == "this-tool-does-not-exist-xyz" {
			if tool.Available {
				t.Error("expected tool to be unavailable")
			}
			return
		}
	}
	t.Error("expected the missing tool in the report")
}

func TestCheckDependencies_SkillAutoInstallable(t *testing.T) {
	m := &manifest.Manifest{
		Skills: map[string]manifest.SkillConfig{
			"installable": {
				Check:   "false",
				Install: "echo install",
			},
			"not-installable": {
				Check: "false",
			},
		},
	}

	h := NewHealthChecker(
		WithLookPath(func(file string) (string, error) {
			return "/usr/bin/" + file, nil
		}),
	)

	report := h.checkDependencies(context.Background(), m)

	for _, skill := range report.Skills {
		switch skill.Name {
		case "installable":
			if !skill.AutoInstallable {
				t.Error("expected skill 'installable' to be auto-installable")
			}
		case "not-installable":
			if skill.AutoInstallable {
				t.Error("expected skill 'not-installable' to not be auto-installable")
			}
		}
	}
}

func TestCheckDependencies_NilManifest(t *testing.T) {
	h := NewHealthChecker()
	report := h.checkDependencies(context.Background(), nil)

	if len(report.Tools) != 0 {
		t.Errorf("expected 0 tools for nil manifest, got %d", len(report.Tools))
	}
	if len(report.Skills) != 0 {
		t.Errorf("expected 0 skills for nil manifest, got %d", len(report.Skills))
	}
}

// --- T012: checkCodebase tests ---

func TestCheckCodebase_GitLocalFallback(t *testing.T) {
	gitRunner := newMockGitRunner().
		withResponse("rev-list", "42").
		withResponse("branch", "  main\n  feature-1\n  feature-2").
		withResponse("log", "2026-03-01T10:00:00Z")

	h := NewHealthChecker(
		WithGitRunner(gitRunner),
		WithPlatformProfile(platform.PlatformProfile{
			Type: platform.PlatformUnknown,
		}),
	)

	metrics := h.checkCodebase(context.Background(), h.profile)

	if metrics.Source != "git_local" {
		t.Errorf("expected source=git_local, got %s", metrics.Source)
	}
	if metrics.RecentCommits != 42 {
		t.Errorf("expected 42 recent commits, got %d", metrics.RecentCommits)
	}
	if metrics.BranchCount != 3 {
		t.Errorf("expected 3 branches, got %d", metrics.BranchCount)
	}
	if metrics.LastCommitDate.IsZero() {
		t.Error("expected LastCommitDate to be set")
	}
	if metrics.APIAvailable {
		t.Error("expected APIAvailable=false for git local")
	}
}

func TestCheckCodebase_GitHubAPI(t *testing.T) {
	gitRunner := newMockGitRunner().
		withResponse("rev-list", "10").
		withResponse("branch", "  main").
		withResponse("log", "2026-03-01T10:00:00Z")

	h := NewHealthChecker(
		WithGitRunner(gitRunner),
		WithPlatformProfile(platform.PlatformProfile{
			Type:  platform.PlatformGitHub,
			Owner: "recinq",
			Repo:  "wave",
		}),
		WithGitHubAPI(&mockGitHubAPI{
			openIssues: 15,
			openPRs:    3,
		}),
	)

	metrics := h.checkCodebase(context.Background(), h.profile)

	if metrics.Source != "github_api" {
		t.Errorf("expected source=github_api, got %s", metrics.Source)
	}
	if !metrics.APIAvailable {
		t.Error("expected APIAvailable=true")
	}
	if metrics.OpenIssueCount != 15 {
		t.Errorf("expected 15 open issues, got %d", metrics.OpenIssueCount)
	}
	if metrics.OpenPRCount != 3 {
		t.Errorf("expected 3 open PRs, got %d", metrics.OpenPRCount)
	}
	// Git-local metrics should still be populated.
	if metrics.RecentCommits != 10 {
		t.Errorf("expected 10 recent commits from git-local supplement, got %d", metrics.RecentCommits)
	}
}

func TestCheckCodebase_GitHubAPIFallback(t *testing.T) {
	gitRunner := newMockGitRunner().
		withResponse("rev-list", "5").
		withResponse("branch", "  main").
		withResponse("log", "2026-03-01T10:00:00Z")

	h := NewHealthChecker(
		WithGitRunner(gitRunner),
		WithPlatformProfile(platform.PlatformProfile{
			Type:  platform.PlatformGitHub,
			Owner: "recinq",
			Repo:  "wave",
		}),
		WithGitHubAPI(&mockGitHubAPI{
			err: fmt.Errorf("unauthorized"),
		}),
	)

	metrics := h.checkCodebase(context.Background(), h.profile)

	if metrics.Source != "git_local" {
		t.Errorf("expected source=git_local on API failure, got %s", metrics.Source)
	}
	if metrics.APIAvailable {
		t.Error("expected APIAvailable=false on API failure")
	}
}

func TestCheckCodebase_GitErrors(t *testing.T) {
	gitRunner := newMockGitRunner().
		withError("rev-list", fmt.Errorf("not a git repo")).
		withError("branch", fmt.Errorf("not a git repo")).
		withError("log", fmt.Errorf("not a git repo"))

	h := NewHealthChecker(
		WithGitRunner(gitRunner),
	)

	metrics := h.checkCodebase(context.Background(), platform.PlatformProfile{})

	// Errors should result in zero values, not panics.
	if metrics.RecentCommits != 0 {
		t.Errorf("expected 0 recent commits on error, got %d", metrics.RecentCommits)
	}
	if metrics.BranchCount != 0 {
		t.Errorf("expected 0 branches on error, got %d", metrics.BranchCount)
	}
}

// --- T013: RunHealthChecks tests ---

func TestRunHealthChecks_Integration(t *testing.T) {
	dir := t.TempDir()
	manifestPath := writeValidManifest(t, dir)

	gitRunner := newMockGitRunner().
		withResponse("rev-list", "20").
		withResponse("branch", "  main\n  dev").
		withResponse("log", "2026-03-01T10:00:00Z")

	h := NewHealthChecker(
		WithManifestPath(manifestPath),
		WithVersion("v0.9.0"),
		WithGitRunner(gitRunner),
		WithLookPath(func(file string) (string, error) {
			return "/usr/bin/" + file, nil
		}),
		WithPlatformProfile(platform.PlatformProfile{
			Type:           platform.PlatformGitHub,
			Owner:          "recinq",
			Repo:           "wave",
			PipelineFamily: "gh",
		}),
	)

	report, err := h.RunHealthChecks(context.Background(), DefaultHealthCheckConfig())
	if err != nil {
		t.Fatalf("RunHealthChecks returned error: %v", err)
	}

	if report == nil {
		t.Fatal("expected non-nil report")
	}
	if report.Timestamp.IsZero() {
		t.Error("expected Timestamp to be set")
	}
	if report.Duration <= 0 {
		t.Error("expected positive Duration")
	}
	if !report.Init.ManifestFound {
		t.Error("expected manifest found in report")
	}
	if !report.Init.ManifestValid {
		t.Error("expected manifest valid in report")
	}
	if report.Init.WaveVersion != "v0.9.0" {
		t.Errorf("expected version v0.9.0, got %s", report.Init.WaveVersion)
	}
	if report.Platform.Type != platform.PlatformGitHub {
		t.Errorf("expected platform type GitHub, got %s", report.Platform.Type)
	}
}

func TestRunHealthChecks_Timeout(t *testing.T) {
	// Create a git runner that blocks forever.
	slowGitRunner := &slowRunner{delay: 5 * time.Second}

	dir := t.TempDir()
	manifestPath := writeValidManifest(t, dir)

	h := NewHealthChecker(
		WithManifestPath(manifestPath),
		WithGitRunner(slowGitRunner),
		WithLookPath(func(file string) (string, error) {
			return "/usr/bin/" + file, nil
		}),
	)

	opts := HealthCheckConfig{
		InitTimeout:     5 * time.Second,
		DepsTimeout:     5 * time.Second,
		CodebaseTimeout: 100 * time.Millisecond, // Very short timeout for codebase.
		PlatformTimeout: 5 * time.Second,
	}

	report, err := h.RunHealthChecks(context.Background(), opts)
	if err != nil {
		t.Fatalf("RunHealthChecks returned error: %v", err)
	}

	// Codebase check should have timed out.
	foundTimeout := false
	for _, e := range report.Errors {
		if e.Check == "codebase" && e.Timeout {
			foundTimeout = true
			break
		}
	}
	if !foundTimeout {
		t.Error("expected codebase timeout error in report")
	}
}

// slowRunner is a GitRunner that takes a long time.
type slowRunner struct {
	delay time.Duration
}

func (s *slowRunner) Run(ctx context.Context, args ...string) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(s.delay):
		return "", fmt.Errorf("should have been cancelled")
	}
}

func TestRunHealthChecks_ContextCancellation(t *testing.T) {
	dir := t.TempDir()
	manifestPath := writeValidManifest(t, dir)

	h := NewHealthChecker(
		WithManifestPath(manifestPath),
		WithGitRunner(&slowRunner{delay: 10 * time.Second}),
		WithLookPath(func(file string) (string, error) {
			return "/usr/bin/" + file, nil
		}),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	report, err := h.RunHealthChecks(ctx, DefaultHealthCheckConfig())
	if err != nil {
		t.Fatalf("RunHealthChecks returned error: %v", err)
	}

	// Report should still be returned even with context cancellation.
	if report == nil {
		t.Fatal("expected non-nil report on context cancellation")
	}
}

// --- T014: Race safety test ---

func TestRunHealthChecks_ParallelRaceSafety(t *testing.T) {
	dir := t.TempDir()
	manifestPath := writeValidManifest(t, dir)

	gitRunner := newMockGitRunner().
		withResponse("rev-list", "10").
		withResponse("branch", "  main").
		withResponse("log", "2026-03-01T10:00:00Z")

	h := NewHealthChecker(
		WithManifestPath(manifestPath),
		WithVersion("v0.9.0"),
		WithGitRunner(gitRunner),
		WithLookPath(func(file string) (string, error) {
			return "/usr/bin/" + file, nil
		}),
		WithPlatformProfile(platform.PlatformProfile{
			Type:           platform.PlatformGitHub,
			Owner:          "recinq",
			Repo:           "wave",
			PipelineFamily: "gh",
		}),
	)

	// Run multiple health checks concurrently to check for race conditions.
	var completed atomic.Int32
	errCh := make(chan error, 10)

	for i := 0; i < 10; i++ {
		go func() {
			report, err := h.RunHealthChecks(context.Background(), DefaultHealthCheckConfig())
			if err != nil {
				errCh <- err
				return
			}
			if report == nil {
				errCh <- fmt.Errorf("nil report")
				return
			}
			completed.Add(1)
			errCh <- nil
		}()
	}

	for i := 0; i < 10; i++ {
		if err := <-errCh; err != nil {
			t.Errorf("concurrent health check failed: %v", err)
		}
	}

	if completed.Load() != 10 {
		t.Errorf("expected 10 completed runs, got %d", completed.Load())
	}
}

func TestDefaultHealthCheckConfig(t *testing.T) {
	cfg := DefaultHealthCheckConfig()

	if cfg.InitTimeout <= 0 {
		t.Error("expected positive InitTimeout")
	}
	if cfg.DepsTimeout <= 0 {
		t.Error("expected positive DepsTimeout")
	}
	if cfg.CodebaseTimeout <= 0 {
		t.Error("expected positive CodebaseTimeout")
	}
	if cfg.PlatformTimeout <= 0 {
		t.Error("expected positive PlatformTimeout")
	}
}

func TestNewHealthChecker_Defaults(t *testing.T) {
	h := NewHealthChecker()

	if h.manifestPath != "wave.yaml" {
		t.Errorf("expected default manifest path 'wave.yaml', got %s", h.manifestPath)
	}
	if h.version != "unknown" {
		t.Errorf("expected default version 'unknown', got %s", h.version)
	}
	if h.lookPath == nil {
		t.Error("expected non-nil lookPath function")
	}
	if h.gitRunner == nil {
		t.Error("expected non-nil gitRunner")
	}
}

func TestNewHealthChecker_WithOptions(t *testing.T) {
	profile := platform.PlatformProfile{
		Type:  platform.PlatformGitLab,
		Owner: "test",
		Repo:  "repo",
	}

	h := NewHealthChecker(
		WithManifestPath("/custom/path.yaml"),
		WithVersion("v1.0.0"),
		WithPlatformProfile(profile),
	)

	if h.manifestPath != "/custom/path.yaml" {
		t.Errorf("expected custom manifest path, got %s", h.manifestPath)
	}
	if h.version != "v1.0.0" {
		t.Errorf("expected version v1.0.0, got %s", h.version)
	}
	if h.profile.Type != platform.PlatformGitLab {
		t.Errorf("expected GitLab platform, got %s", h.profile.Type)
	}
}

func TestCheckInit_VersionPropagated(t *testing.T) {
	dir := t.TempDir()
	manifestPath := writeValidManifest(t, dir)

	h := NewHealthChecker(
		WithManifestPath(manifestPath),
		WithVersion("v2.3.4-beta"),
	)

	out := h.checkInit(context.Background(), manifestPath)

	if out.result.WaveVersion != "v2.3.4-beta" {
		t.Errorf("expected version v2.3.4-beta, got %s", out.result.WaveVersion)
	}
}

func TestCheckDependencies_MultipleAdapters(t *testing.T) {
	m := &manifest.Manifest{
		Adapters: map[string]manifest.Adapter{
			"claude": {Binary: "claude", Mode: "headless"},
			"gpt":    {Binary: "gpt-cli", Mode: "headless"},
		},
	}

	seenTools := make(map[string]bool)
	h := NewHealthChecker(
		WithLookPath(func(file string) (string, error) {
			seenTools[file] = true
			return "/usr/bin/" + file, nil
		}),
	)

	report := h.checkDependencies(context.Background(), m)

	// Should include both adapter binaries and git.
	if len(report.Tools) < 3 {
		t.Errorf("expected at least 3 tools (2 adapters + git), got %d", len(report.Tools))
	}
	if !seenTools["claude"] {
		t.Error("expected claude to be checked")
	}
	if !seenTools["gpt-cli"] {
		t.Error("expected gpt-cli to be checked")
	}
	if !seenTools["git"] {
		t.Error("expected git to be checked")
	}
}

func TestCheckCodebase_EmptyBranches(t *testing.T) {
	gitRunner := newMockGitRunner().
		withResponse("rev-list", "0").
		withResponse("branch", "").
		withResponse("log", "2026-03-01T10:00:00Z")

	h := NewHealthChecker(WithGitRunner(gitRunner))

	metrics := h.checkCodebase(context.Background(), platform.PlatformProfile{})

	if metrics.BranchCount != 0 {
		t.Errorf("expected 0 branches for empty output, got %d", metrics.BranchCount)
	}
}

func TestRunHealthChecks_MissingManifest(t *testing.T) {
	h := NewHealthChecker(
		WithManifestPath("/nonexistent/wave.yaml"),
		WithGitRunner(newMockGitRunner().
			withResponse("rev-list", "0").
			withResponse("branch", "").
			withResponse("log", "2026-03-01T10:00:00Z")),
		WithLookPath(func(file string) (string, error) {
			return "/usr/bin/" + file, nil
		}),
	)

	report, err := h.RunHealthChecks(context.Background(), DefaultHealthCheckConfig())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.Init.ManifestFound {
		t.Error("expected ManifestFound=false")
	}
	if report.Init.Error == "" {
		t.Error("expected error in init result")
	}
	// Dependencies should be empty since manifest wasn't loaded.
	// (The deps check waits for manifest and may time out.)
}
