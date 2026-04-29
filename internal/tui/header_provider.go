package tui

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/recinq/wave/internal/forge"
	wgit "github.com/recinq/wave/internal/git"
	"github.com/recinq/wave/internal/manifest"
)

// DefaultMetadataProvider fetches project metadata from external sources
// (git CLI, wave.yaml manifest, gh CLI, and an optional health-check function).
type DefaultMetadataProvider struct {
	// ManifestPath is the path to wave.yaml. Defaults to "wave.yaml" if empty.
	ManifestPath string

	// HealthCheckFunc is an optional callback injected by the application layer
	// to aggregate pipeline run health from the state database. This avoids a
	// direct dependency on the state package. When nil, FetchPipelineHealth
	// returns HealthOK.
	HealthCheckFunc func() (HealthStatus, error)
}

// FetchGitState shells out to git to determine the current branch, abbreviated
// commit hash, dirty/clean working tree status, and the first configured remote.
// If git is unavailable or the directory is not a repository, placeholder values
// are returned with a nil error so the header can still render.
func (p *DefaultMetadataProvider) FetchGitState() (GitState, error) {
	var state GitState

	// Get current branch name via the centralized git helper.
	branch, err := wgit.Branch()
	if err != nil {
		// Not a git repo or git not installed — return safe placeholders.
		return GitState{Branch: "[no git]", CommitHash: "[no git]"}, nil
	}
	state.Branch = branch

	if hash, err := wgit.ShortHash(); err == nil {
		state.CommitHash = hash
	}

	if dirty, err := wgit.IsDirty(); err == nil {
		state.IsDirty = dirty
	}

	if remote, err := wgit.FirstRemote(); err == nil {
		state.RemoteName = remote
	}

	return state, nil
}

// FetchManifestInfo loads wave.yaml and extracts the project name and repo slug.
// If the manifest is missing or malformed, placeholder values are returned with
// a nil error so the header can still render.
func (p *DefaultMetadataProvider) FetchManifestInfo() (ManifestInfo, error) {
	path := p.ManifestPath
	if path == "" {
		path = "wave.yaml"
	}

	m, err := manifest.Load(path)
	if err != nil {
		return ManifestInfo{ProjectName: "[no project]"}, nil
	}

	info := ManifestInfo{
		ProjectName: m.Metadata.Name,
		RepoName:    m.Metadata.Repo,
	}
	if info.ProjectName == "" {
		info.ProjectName = "[no project]"
	}

	// Auto-detect repo slug from git remote if not set in manifest
	if info.RepoName == "" {
		info.RepoName = detectRepoFromGitRemote()
	}

	return info, nil
}

// FetchGitHubInfo checks gh CLI authentication and, when connected, fetches the
// open issues count for the given repo (owner/repo format). Three auth states
// are distinguished:
//   - GitHubNotConfigured: gh CLI is not installed or not authenticated
//   - GitHubOffline: authenticated but the API call failed (network, rate-limit)
//   - GitHubConnected: authenticated and the API returned data
//
// An empty repo string results in GitHubNotConfigured.
func (p *DefaultMetadataProvider) FetchGitHubInfo(repo string) (GitHubInfo, error) {
	if repo == "" {
		return GitHubInfo{AuthState: GitHubNotConfigured}, nil
	}

	// Verify that gh CLI is authenticated.
	if err := exec.Command("gh", "auth", "status").Run(); err != nil {
		return GitHubInfo{AuthState: GitHubNotConfigured}, nil
	}

	// Fetch the repository's open_issues_count from the GitHub API.
	out, err := exec.Command(
		"gh", "api",
		fmt.Sprintf("repos/%s", repo),
		"--jq", ".open_issues_count",
	).Output()
	if err != nil {
		return GitHubInfo{AuthState: GitHubOffline}, nil
	}

	var count int
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(out))), &count); err != nil {
		return GitHubInfo{AuthState: GitHubOffline}, nil
	}

	return GitHubInfo{
		AuthState:   GitHubConnected,
		IssuesCount: count,
	}, nil
}

// FetchPipelineHealth delegates to the injected HealthCheckFunc when set.
// Returns HealthOK if no health-check function has been provided.
func (p *DefaultMetadataProvider) FetchPipelineHealth() (HealthStatus, error) {
	if p.HealthCheckFunc != nil {
		return p.HealthCheckFunc()
	}
	return HealthOK, nil
}

// detectRepoFromGitRemote extracts owner/repo from the origin remote URL.
// Delegates to the forge package for URL parsing.
func detectRepoFromGitRemote() string {
	fi, err := forge.DetectFromGitRemotes()
	if err != nil {
		return ""
	}
	return fi.Slug()
}
