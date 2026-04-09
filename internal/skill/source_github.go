package skill

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// GitHubAdapter installs skills from GitHub repositories.
type GitHubAdapter struct {
	dep      CLIDependency
	lookPath lookPathFunc
}

// NewGitHubAdapter creates a GitHubAdapter with default exec.LookPath.
func NewGitHubAdapter() *GitHubAdapter {
	return &GitHubAdapter{
		dep: CLIDependency{
			Binary:       "git",
			Instructions: "install git from https://git-scm.com",
		},
		lookPath: exec.LookPath,
	}
}

// Prefix returns "github".
func (a *GitHubAdapter) Prefix() string { return "github" }

// GitHub naming patterns per GitHub's actual constraints.
var (
	// Owner: alphanumeric, hyphens allowed in the middle, no consecutive hyphens.
	githubOwnerPattern = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$`)
	// Repo: alphanumeric, hyphens, underscores, periods.
	githubRepoPattern = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
)

// parseGitHubRef splits a reference into owner, repo, and optional subpath.
// Format: owner/repo[/path/to/skill]
func parseGitHubRef(ref string) (owner, repo, subpath string, err error) {
	parts := strings.SplitN(ref, "/", 3)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", "", "", fmt.Errorf("invalid GitHub reference %q: expected owner/repo[/path]", ref)
	}
	owner = parts[0]
	repo = parts[1]
	if !githubOwnerPattern.MatchString(owner) {
		return "", "", "", fmt.Errorf("invalid GitHub owner %q: must match [a-zA-Z0-9-]", owner)
	}
	if !githubRepoPattern.MatchString(repo) {
		return "", "", "", fmt.Errorf("invalid GitHub repo %q: must match [a-zA-Z0-9._-]", repo)
	}
	if len(parts) == 3 {
		subpath = parts[2]
	}
	return owner, repo, subpath, nil
}

// Install clones a GitHub repository and discovers SKILL.md files.
func (a *GitHubAdapter) Install(ctx context.Context, ref string, store Store) (*InstallResult, error) {
	if err := checkDependency(a.dep, a.lookPath); err != nil {
		return nil, err
	}

	owner, repo, subpath, err := parseGitHubRef(ref)
	if err != nil {
		return nil, err
	}

	tmpDir, err := os.MkdirTemp("", "wave-skill-github-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	ctx, cancel := context.WithTimeout(ctx, CLITimeout)
	defer cancel()

	repoURL := fmt.Sprintf("https://github.com/%s/%s.git", owner, repo)
	cloneDir := filepath.Join(tmpDir, "repo")

	cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", repoURL, cloneDir)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git clone %s failed: %v\nstderr: %s", repoURL, err, stderr.String())
	}

	// If a subpath is specified, look for SKILL.md there
	if subpath != "" {
		// Validate subpath doesn't escape the clone directory
		skillDir := filepath.Join(cloneDir, subpath)
		cleanDir := filepath.Clean(skillDir)
		if !strings.HasPrefix(cleanDir, filepath.Clean(cloneDir)+string(filepath.Separator)) {
			return nil, fmt.Errorf("subpath %q escapes repository directory", subpath)
		}
		skillFile := filepath.Join(cleanDir, "SKILL.md")
		if _, err := os.Stat(skillFile); err != nil {
			return nil, fmt.Errorf("SKILL.md not found at %s in repository %s/%s", subpath, owner, repo)
		}
		return parseAndWriteSkills(ctx, []string{skillFile}, store)
	}

	// Check root for SKILL.md
	rootSkill := filepath.Join(cloneDir, "SKILL.md")
	if _, err := os.Stat(rootSkill); err == nil {
		return parseAndWriteSkills(ctx, []string{rootSkill}, store)
	}

	// No root SKILL.md — discover all SKILL.md files in the repo
	paths, err := discoverSkillFiles(cloneDir)
	if err != nil {
		return nil, err
	}

	return parseAndWriteSkills(ctx, paths, store)
}
