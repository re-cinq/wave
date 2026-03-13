package forge

import (
	"os/exec"
	"strings"
)

// ForgeType identifies the source code hosting platform.
type ForgeType string

const (
	ForgeGitHub    ForgeType = "github"
	ForgeGitLab    ForgeType = "gitlab"
	ForgeBitbucket ForgeType = "bitbucket"
	ForgeGitea     ForgeType = "gitea"
	ForgeUnknown   ForgeType = "unknown"
)

// ForgeInfo describes the detected forge and its associated metadata.
type ForgeInfo struct {
	Type           ForgeType `json:"type"`
	Host           string    `json:"host"`
	Owner          string    `json:"owner"`
	Repo           string    `json:"repo"`
	CLITool        string    `json:"cli_tool"`
	PipelinePrefix string    `json:"pipeline_prefix"`
	PRTerm         string    `json:"pr_term"`    // "Pull Request" or "Merge Request"
	PRCommand      string    `json:"pr_command"` // "pr" or "mr"
}

// Slug returns "owner/repo" if both are set, otherwise empty string.
func (fi ForgeInfo) Slug() string {
	if fi.Owner != "" && fi.Repo != "" {
		return fi.Owner + "/" + fi.Repo
	}
	return ""
}

// Detect classifies a remote URL into a ForgeInfo.
// Supports SSH (git@host:owner/repo.git) and HTTPS (https://host/owner/repo.git) formats.
func Detect(remoteURL string) ForgeInfo {
	host, owner, repo := parseRemoteURL(remoteURL)
	if host == "" {
		return ForgeInfo{Type: ForgeUnknown}
	}

	ft := classifyHost(host)
	cli, prefix, prTerm, prCommand := forgeMetadata(ft)

	return ForgeInfo{
		Type:           ft,
		Host:           host,
		Owner:          owner,
		Repo:           repo,
		CLITool:        cli,
		PipelinePrefix: prefix,
		PRTerm:         prTerm,
		PRCommand:      prCommand,
	}
}

// DetectFromGitRemotes shells out to `git remote -v` and classifies the first
// fetch remote. Returns ForgeUnknown info if git is unavailable or no remotes exist.
func DetectFromGitRemotes() (ForgeInfo, error) {
	out, err := exec.Command("git", "remote", "-v").Output()
	if err != nil {
		return ForgeInfo{Type: ForgeUnknown}, nil
	}

	// Parse first fetch remote
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Format: "origin\thttps://github.com/owner/repo.git (fetch)"
		if !strings.Contains(line, "(fetch)") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		return Detect(fields[1]), nil
	}

	return ForgeInfo{Type: ForgeUnknown}, nil
}

// FilterPipelinesByForge returns pipeline names that match the given forge's
// prefix convention. Pipelines without a forge prefix are always included.
func FilterPipelinesByForge(ft ForgeType, names []string) []string {
	_, prefix, _, _ := forgeMetadata(ft)
	if prefix == "" {
		return names
	}

	var result []string
	for _, name := range names {
		// Include pipelines that match the forge prefix or have no forge prefix
		if strings.HasPrefix(name, prefix+"-") || !hasForgePrefix(name) {
			result = append(result, name)
		}
	}
	return result
}

// parseRemoteURL extracts host, owner, and repo from SSH or HTTPS remote URLs.
func parseRemoteURL(url string) (host, owner, repo string) {
	url = strings.TrimSpace(url)
	if url == "" {
		return "", "", ""
	}

	// SSH format: git@host:owner/repo.git or ssh://git@host/owner/repo.git
	if strings.HasPrefix(url, "ssh://") {
		url = strings.TrimPrefix(url, "ssh://")
		// ssh://git@host/owner/repo.git
		if idx := strings.Index(url, "@"); idx >= 0 {
			url = url[idx+1:]
		}
		return parseHTTPSPath(url)
	}

	if strings.Contains(url, "@") && strings.Contains(url, ":") {
		// git@host:owner/repo.git
		atIdx := strings.Index(url, "@")
		colonIdx := strings.Index(url[atIdx:], ":")
		if colonIdx < 0 {
			return "", "", ""
		}
		colonIdx += atIdx

		host = url[atIdx+1 : colonIdx]
		path := strings.TrimSuffix(url[colonIdx+1:], ".git")
		parts := strings.SplitN(path, "/", 2)
		if len(parts) == 2 {
			return host, parts[0], parts[1]
		}
		return host, "", ""
	}

	// HTTPS format: https://host/owner/repo.git
	for _, scheme := range []string{"https://", "http://"} {
		if strings.HasPrefix(url, scheme) {
			url = strings.TrimPrefix(url, scheme)
			return parseHTTPSPath(url)
		}
	}

	return "", "", ""
}

// parseHTTPSPath extracts host, owner, repo from "host/owner/repo" format.
func parseHTTPSPath(path string) (host, owner, repo string) {
	path = strings.TrimSuffix(path, ".git")
	parts := strings.SplitN(path, "/", 3)
	if len(parts) < 3 {
		if len(parts) == 1 {
			return parts[0], "", ""
		}
		return parts[0], parts[1], ""
	}
	return parts[0], parts[1], parts[2]
}

// classifyHost maps a hostname to a ForgeType.
func classifyHost(host string) ForgeType {
	host = strings.ToLower(host)
	switch {
	case host == "github.com" || strings.HasSuffix(host, ".github.com"):
		return ForgeGitHub
	case host == "gitlab.com" || strings.HasSuffix(host, ".gitlab.com"):
		return ForgeGitLab
	case host == "bitbucket.org" || strings.HasSuffix(host, ".bitbucket.org"):
		return ForgeBitbucket
	case strings.Contains(host, "gitea"):
		return ForgeGitea
	default:
		return ForgeUnknown
	}
}

// forgeMetadata returns the CLI tool, pipeline prefix, PR term, and PR command for a forge type.
func forgeMetadata(ft ForgeType) (cli, prefix, prTerm, prCommand string) {
	switch ft {
	case ForgeGitHub:
		return "gh", "gh", "Pull Request", "pr"
	case ForgeGitLab:
		return "glab", "gl", "Merge Request", "mr"
	case ForgeBitbucket:
		return "bb", "bb", "Pull Request", "pr"
	case ForgeGitea:
		return "tea", "gt", "Pull Request", "pr"
	default:
		return "", "", "", ""
	}
}

// hasForgePrefix checks if a pipeline name starts with any known forge prefix.
func hasForgePrefix(name string) bool {
	prefixes := []string{"gh-", "gl-", "bb-", "gt-"}
	for _, p := range prefixes {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}
