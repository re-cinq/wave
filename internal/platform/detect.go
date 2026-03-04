// Package platform provides hosting platform detection from git remote URLs.
// It identifies GitHub, GitLab, Bitbucket, and Gitea platforms by inspecting
// remote URL patterns, extracting owner/repo metadata and setting appropriate
// API URLs, CLI tools, and pipeline families.
package platform

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// PlatformType identifies the hosting platform for a git repository.
type PlatformType string

const (
	PlatformGitHub    PlatformType = "github"
	PlatformGitLab    PlatformType = "gitlab"
	PlatformBitbucket PlatformType = "bitbucket"
	PlatformGitea     PlatformType = "gitea"
	PlatformUnknown   PlatformType = "unknown"
)

// PlatformProfile contains the detected hosting platform identity along with
// associated metadata for API access, CLI tooling, and pipeline routing.
type PlatformProfile struct {
	Type              PlatformType `json:"type"`
	Owner             string       `json:"owner"`
	Repo              string       `json:"repo"`
	APIURL            string       `json:"api_url,omitempty"`
	CLITool           string       `json:"cli_tool,omitempty"`
	PipelineFamily    string       `json:"pipeline_family"`
	AdditionalRemotes []RemoteInfo `json:"additional_remotes,omitempty"`
}

// RemoteInfo describes a single git remote and its detected platform.
type RemoteInfo struct {
	Name     string       `json:"name"`
	URL      string       `json:"url"`
	Platform PlatformType `json:"platform"`
}

// sshPattern matches SSH remote URLs like:
//
//	git@github.com:owner/repo.git
//	ssh://git@github.com/owner/repo.git
//	ssh://git@github.com:2222/owner/repo.git
var sshPattern = regexp.MustCompile(
	`^(?:ssh://)?(?:[a-zA-Z0-9._-]+@)?([a-zA-Z0-9._-]+(?:\.[a-zA-Z0-9._-]+)+)(?::[0-9]+)?[:/]([^/]+)/([^/]+?)(?:\.git)?$`,
)

// httpsPattern matches HTTPS remote URLs like:
//
//	https://github.com/owner/repo.git
//	https://github.com/owner/repo
//	http://gitlab.example.com:8080/owner/repo.git
var httpsPattern = regexp.MustCompile(
	`^https?://([a-zA-Z0-9._-]+(?:\.[a-zA-Z0-9._-]+)*)(?::[0-9]+)?/([^/]+)/([^/]+?)(?:\.git)?$`,
)

// gitLabSubgroupHTTPS matches GitLab subgroup URLs like:
//
//	https://gitlab.com/group/subgroup/repo.git
var gitLabSubgroupHTTPS = regexp.MustCompile(
	`^https?://([a-zA-Z0-9._-]+(?:\.[a-zA-Z0-9._-]+)*)(?::[0-9]+)?/(.+)/([^/]+?)(?:\.git)?$`,
)

// gitLabSubgroupSSH matches GitLab subgroup SSH URLs like:
//
//	git@gitlab.com:group/subgroup/repo.git
var gitLabSubgroupSSH = regexp.MustCompile(
	`^(?:ssh://)?(?:[a-zA-Z0-9._-]+@)?([a-zA-Z0-9._-]+(?:\.[a-zA-Z0-9._-]+)+)(?::[0-9]+)?[:/](.+)/([^/]+?)(?:\.git)?$`,
)

// platformHosts maps well-known hostnames to platform types.
var platformHosts = map[string]PlatformType{
	"github.com":    PlatformGitHub,
	"gitlab.com":    PlatformGitLab,
	"bitbucket.org": PlatformBitbucket,
}

// Detect analyzes a remote URL and returns a PlatformProfile describing the
// hosting platform, owner, and repository. Unrecognized URLs produce a profile
// with Type set to PlatformUnknown.
func Detect(remoteURL string) PlatformProfile {
	remoteURL = strings.TrimSpace(remoteURL)
	if remoteURL == "" {
		return PlatformProfile{Type: PlatformUnknown, PipelineFamily: "unknown"}
	}

	host, owner, repo := parseRemoteURL(remoteURL)
	if host == "" {
		return PlatformProfile{Type: PlatformUnknown, PipelineFamily: "unknown"}
	}

	platformType := identifyPlatform(host)

	profile := PlatformProfile{
		Type:  platformType,
		Owner: owner,
		Repo:  repo,
	}

	switch platformType {
	case PlatformGitHub:
		profile.APIURL = "https://api.github.com"
		profile.CLITool = "gh"
		profile.PipelineFamily = "gh"
	case PlatformGitLab:
		profile.APIURL = fmt.Sprintf("https://%s/api/v4", host)
		profile.CLITool = "glab"
		profile.PipelineFamily = "gl"
	case PlatformBitbucket:
		profile.APIURL = "https://api.bitbucket.org/2.0"
		profile.CLITool = "bb"
		profile.PipelineFamily = "bb"
	case PlatformGitea:
		profile.APIURL = fmt.Sprintf("https://%s/api/v1", host)
		profile.CLITool = "tea"
		profile.PipelineFamily = "gt"
	default:
		profile.PipelineFamily = "unknown"
	}

	return profile
}

// parseRemoteURL extracts the hostname, owner, and repository name from
// a git remote URL. It handles both SSH and HTTPS formats, including
// GitLab subgroup paths. Returns empty strings if parsing fails.
func parseRemoteURL(url string) (host, owner, repo string) {
	// Try standard SSH pattern first (2-segment path: owner/repo)
	if matches := sshPattern.FindStringSubmatch(url); matches != nil {
		return matches[1], matches[2], cleanRepoName(matches[3])
	}

	// Try standard HTTPS pattern (2-segment path: owner/repo)
	if matches := httpsPattern.FindStringSubmatch(url); matches != nil {
		return matches[1], matches[2], cleanRepoName(matches[3])
	}

	// Try GitLab subgroup patterns (3+ segment paths: group/subgroup/.../repo)
	// Only apply these if the host is recognized as GitLab
	if matches := gitLabSubgroupHTTPS.FindStringSubmatch(url); matches != nil {
		h := matches[1]
		if identifyPlatform(h) == PlatformGitLab {
			return h, matches[2], cleanRepoName(matches[3])
		}
	}

	if matches := gitLabSubgroupSSH.FindStringSubmatch(url); matches != nil {
		h := matches[1]
		if identifyPlatform(h) == PlatformGitLab {
			return h, matches[2], cleanRepoName(matches[3])
		}
	}

	return "", "", ""
}

// identifyPlatform determines the platform type from a hostname.
// It checks well-known hosts first, then applies heuristic pattern matching
// for self-hosted instances (e.g., "gitea" in hostname).
func identifyPlatform(host string) PlatformType {
	host = strings.ToLower(host)

	// Check well-known hosts
	if pt, ok := platformHosts[host]; ok {
		return pt
	}

	// Heuristic: hostnames containing "gitlab" suggest a GitLab instance
	if strings.Contains(host, "gitlab") {
		return PlatformGitLab
	}

	// Heuristic: hostnames containing "gitea" suggest a Gitea instance
	if strings.Contains(host, "gitea") {
		return PlatformGitea
	}

	// Heuristic: hostnames containing "bitbucket" suggest a Bitbucket instance
	if strings.Contains(host, "bitbucket") {
		return PlatformBitbucket
	}

	// Heuristic: hostnames containing "github" suggest a GitHub instance
	if strings.Contains(host, "github") {
		return PlatformGitHub
	}

	return PlatformUnknown
}

// cleanRepoName removes the .git suffix and trims whitespace from a
// repository name.
func cleanRepoName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.TrimSuffix(name, ".git")
	return name
}

// gitCommandRunner abstracts git command execution for testing.
type gitCommandRunner func(args ...string) ([]byte, error)

// defaultGitRunner executes git commands via os/exec.
func defaultGitRunner(args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	return cmd.Output()
}

// DetectFromGit inspects git remotes in the current repository and returns
// a PlatformProfile based on the origin remote. Additional remotes are
// collected into AdditionalRemotes. Returns an error if git is not
// available or the directory is not a git repository.
func DetectFromGit() (PlatformProfile, error) {
	return detectFromGitWith(defaultGitRunner)
}

// detectFromGitWith is the testable implementation of DetectFromGit that
// accepts an injectable git command runner.
func detectFromGitWith(run gitCommandRunner) (PlatformProfile, error) {
	output, err := run("remote", "-v")
	if err != nil {
		return PlatformProfile{Type: PlatformUnknown, PipelineFamily: "unknown"},
			fmt.Errorf("failed to list git remotes: %w", err)
	}

	remotes := parseGitRemoteOutput(output)
	if len(remotes) == 0 {
		return PlatformProfile{Type: PlatformUnknown, PipelineFamily: "unknown"},
			fmt.Errorf("no git remotes configured")
	}

	// Find origin remote — use it as the primary
	var primary *RemoteInfo
	var additional []RemoteInfo

	for i := range remotes {
		if remotes[i].Name == "origin" {
			primary = &remotes[i]
		} else {
			additional = append(additional, remotes[i])
		}
	}

	// If no origin, use the first remote as primary
	if primary == nil {
		primary = &remotes[0]
		additional = remotes[1:]
	}

	profile := Detect(primary.URL)
	profile.AdditionalRemotes = additional

	return profile, nil
}

// parseGitRemoteOutput parses the output of `git remote -v` into a
// deduplicated slice of RemoteInfo. Each remote appears twice in the output
// (fetch and push); we keep only unique name entries.
func parseGitRemoteOutput(output []byte) []RemoteInfo {
	var remotes []RemoteInfo
	seen := make(map[string]bool)

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Format: <name>\t<url> (fetch|push)
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		name := parts[0]
		url := parts[1]

		// Deduplicate: git remote -v shows each remote twice (fetch/push)
		if seen[name] {
			continue
		}
		seen[name] = true

		platform := Detect(url).Type

		remotes = append(remotes, RemoteInfo{
			Name:     name,
			URL:      url,
			Platform: platform,
		})
	}

	return remotes
}
