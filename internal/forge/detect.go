package forge

import (
	"crypto/tls"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// ForgeType identifies the source code hosting platform.
type ForgeType string

const (
	ForgeGitHub    ForgeType = "github"
	ForgeGitLab    ForgeType = "gitlab"
	ForgeBitbucket ForgeType = "bitbucket"
	ForgeGitea     ForgeType = "gitea"
	ForgeForgejo   ForgeType = "forgejo"
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
	return DetectWithOverride(remoteURL, "")
}

// DetectWithOverride classifies a remote URL into a ForgeInfo, using the
// manifest forge override if non-empty. The override value should be a valid
// ForgeType string (e.g. "github", "gitlab", "gitea", "forgejo", "bitbucket").
func DetectWithOverride(remoteURL, forgeOverride string) ForgeInfo {
	host, owner, repo := parseRemoteURL(remoteURL)
	if host == "" {
		return ForgeInfo{Type: ForgeUnknown}
	}

	var ft ForgeType
	if forgeOverride != "" {
		ft = ForgeType(strings.ToLower(forgeOverride))
	} else {
		ft = classifyHost(host)
	}

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
	return DetectFromGitRemotesWithOverride(""), nil
}

// DetectFromGitRemotesWithOverride is like DetectFromGitRemotes but accepts a
// manifest forge override string. When non-empty, the override bypasses hostname
// matching and endpoint probing.
func DetectFromGitRemotesWithOverride(forgeOverride string) ForgeInfo {
	out, err := exec.Command("git", "remote", "-v").Output()
	if err != nil {
		return ForgeInfo{Type: ForgeUnknown}
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
		return DetectWithOverride(fields[1], forgeOverride)
	}

	return ForgeInfo{Type: ForgeUnknown}
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
// When the hostname doesn't match known patterns, it falls back to
// tea CLI detection and HTTP endpoint probing.
func classifyHost(host string) ForgeType {
	h := strings.ToLower(host)
	switch {
	case h == "github.com" || strings.HasSuffix(h, ".github.com"):
		return ForgeGitHub
	case h == "gitlab.com" || strings.HasSuffix(h, ".gitlab.com"):
		return ForgeGitLab
	case h == "bitbucket.org" || strings.HasSuffix(h, ".bitbucket.org"):
		return ForgeBitbucket
	case strings.Contains(h, "gitea"):
		return ForgeGitea
	case strings.Contains(h, "forgejo"):
		return ForgeForgejo
	}

	// Check if tea CLI knows about this host (registered Gitea/Forgejo instance).
	if ft := checkTeaCLI(host); ft != ForgeUnknown {
		return ft
	}

	// Probe well-known forge API endpoints as a last resort.
	return probeForgeType(host)
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
	case ForgeGitea, ForgeForgejo:
		return "tea", "gt", "Pull Request", "pr"
	default:
		return "", "", "", ""
	}
}

// probeHTTPClient is the HTTP client used by probeForgeType. Package-level
// variable so tests can replace it with a client pointing at httptest servers.
var probeHTTPClient = &http.Client{
	Timeout: 3 * time.Second,
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // self-hosted instances often use self-signed certs
	},
	CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse // don't follow redirects
	},
}

// probeForgeType probes well-known API endpoints on the given host to identify
// the forge software. Each probe uses a 3s HTTP timeout. Returns the first
// matching forge type, or ForgeUnknown if all probes fail.
func probeForgeType(host string) ForgeType {
	// Probe order: Forgejo before Gitea because Forgejo also serves the Gitea
	// endpoint, so checking the Forgejo-specific one first avoids misclassification.
	probes := []struct {
		path     string
		forgeType ForgeType
	}{
		{"/api/forgejo/v1/version", ForgeForgejo},
		{"/api/v1/version", ForgeGitea},
		{"/api/v4/version", ForgeGitLab},
		{"/rest/api/1.0/application-properties", ForgeBitbucket},
	}

	for _, p := range probes {
		url := "https://" + host + p.path
		resp, err := probeHTTPClient.Get(url) //nolint:noctx // fire-and-forget probe; context not needed
		if err != nil {
			continue
		}
		resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			return p.forgeType
		}
	}
	return ForgeUnknown
}

// checkTeaCLIFunc allows tests to replace the tea CLI lookup.
var checkTeaCLIFunc func(host string) ForgeType

// checkTeaCLI runs `tea login list` and checks if the given host appears as
// a registered Gitea/Forgejo instance. Returns ForgeGitea if found,
// ForgeUnknown otherwise.
func checkTeaCLI(host string) ForgeType {
	if checkTeaCLIFunc != nil {
		return checkTeaCLIFunc(host)
	}

	// Only attempt if tea binary is available.
	if _, err := exec.LookPath("tea"); err != nil {
		return ForgeUnknown
	}

	out, err := exec.Command("tea", "login", "list").Output()
	if err != nil {
		return ForgeUnknown
	}

	hostLower := strings.ToLower(host)
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(strings.ToLower(line), hostLower) {
			return ForgeGitea
		}
	}
	return ForgeUnknown
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
