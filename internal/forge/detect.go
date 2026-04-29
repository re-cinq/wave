package forge

import (
	"context"
	"crypto/tls"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/recinq/wave/internal/httpx"
)

// ForgeType identifies the source code hosting platform.
type ForgeType string

const (
	ForgeGitHub    ForgeType = "github"
	ForgeGitLab    ForgeType = "gitlab"
	ForgeBitbucket ForgeType = "bitbucket"
	ForgeGitea     ForgeType = "gitea"
	ForgeForgejo   ForgeType = "forgejo"
	ForgeCodeberg  ForgeType = "codeberg"
	ForgeLocal     ForgeType = "local"
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
// ForgeType string (e.g. "github", "gitlab", "gitea", "forgejo", "codeberg",
// "bitbucket", "local").
// When the override is "local", a ForgeLocal info is returned regardless of
// the remote URL — this supports fully forgeless operation.
func DetectWithOverride(remoteURL, forgeOverride string) ForgeInfo {
	// "local" or "none" override short-circuits all detection — no forge needed.
	if strings.EqualFold(forgeOverride, string(ForgeLocal)) || strings.EqualFold(forgeOverride, "none") {
		cli, prefix, prTerm, prCommand := forgeMetadata(ForgeLocal)
		return ForgeInfo{
			Type:           ForgeLocal,
			CLITool:        cli,
			PipelinePrefix: prefix,
			PRTerm:         prTerm,
			PRCommand:      prCommand,
		}
	}

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
// fetch remote. Returns ForgeLocal info if git is unavailable or no remotes
// exist, since a repo without remotes is inherently local-only.
func DetectFromGitRemotes() (ForgeInfo, error) {
	return DetectFromGitRemotesWithOverride(""), nil
}

// DetectFromGitRemotesWithOverride is like DetectFromGitRemotes but accepts a
// manifest forge override string. When non-empty, the override bypasses hostname
// matching and endpoint probing. When the override is "local", ForgeLocal is
// returned immediately without checking git remotes. When no remotes are found
// and no override is set, ForgeLocal is returned instead of ForgeUnknown —
// a repo without remotes is inherently local-only.
func DetectFromGitRemotesWithOverride(forgeOverride string) ForgeInfo {
	// "local" or "none" override returns immediately — no git remote inspection needed.
	if strings.EqualFold(forgeOverride, string(ForgeLocal)) || strings.EqualFold(forgeOverride, "none") {
		return DetectWithOverride("", forgeOverride)
	}

	out, err := exec.Command("git", "remote", "-v").Output()
	if err != nil {
		// git unavailable or not a repo — treat as local.
		return ForgeInfo{Type: ForgeLocal, PipelinePrefix: "local"}
	}

	remoteURL := pickFetchRemoteURL(string(out))
	if remoteURL != "" {
		return DetectWithOverride(remoteURL, forgeOverride)
	}

	// No fetch remotes found — repo is local-only.
	return ForgeInfo{Type: ForgeLocal, PipelinePrefix: "local"}
}

// pickFetchRemoteURL parses `git remote -v` output and returns the best fetch
// remote URL. It prefers "origin" over any other remote. If "origin" has no
// fetch URL, it falls back to the first available fetch remote. Returns empty
// string when no fetch remotes are found.
func pickFetchRemoteURL(gitRemoteOutput string) string {
	var originURL string
	var fallbackURL string
	for _, line := range strings.Split(gitRemoteOutput, "\n") {
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
		if fields[0] == "origin" {
			originURL = fields[1]
			break // origin found — no need to check further
		}
		if fallbackURL == "" {
			fallbackURL = fields[1]
		}
	}

	if originURL != "" {
		return originURL
	}
	return fallbackURL
}

// FilterPipelinesByForge returns pipeline names that match the given forge's
// prefix convention. Pipelines without a forge prefix are always included.
// For ForgeLocal, only pipelines with no forge prefix or the "local-" prefix
// are included — forge-specific pipelines (gh-, gl-, bb-, gt-, cb-) are excluded.
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

	if strings.Contains(url, "@") && !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
		// git@host:owner/repo.git — only match when NOT an HTTPS URL with credentials
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
// Strips any userinfo@ prefix (e.g. "gho_TOKEN@github.com/...") before parsing.
func parseHTTPSPath(path string) (host, owner, repo string) {
	// Strip userinfo (token or username) from HTTPS URLs: "user@host/..." → "host/..."
	if slashIdx := strings.Index(path, "/"); slashIdx > 0 {
		hostPart := path[:slashIdx]
		if atIdx := strings.LastIndex(hostPart, "@"); atIdx >= 0 {
			path = path[atIdx+1:]
		}
	}
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
	case h == "codeberg.org" || strings.HasSuffix(h, ".codeberg.org"):
		return ForgeCodeberg
	// Re-cinq-hosted Gitea instance — ship it explicitly so the boot path
	// classifier doesn't pay the 3s HTTP probe on every wave init / wave
	// run inside a librete.ch repo.
	case h == "git.librete.ch":
		return ForgeGitea
	}

	// Check if tea CLI knows about this host (registered Gitea/Forgejo instance).
	if ft := checkTeaCLI(host); ft != ForgeUnknown {
		return ft
	}

	// Probe well-known forge API endpoints as a last resort.
	return probeForgeType(host)
}

// forgeMetadata returns the CLI tool, pipeline prefix, PR term, and PR command for a forge type.
// ForgeLocal returns empty strings for CLI tool, PR term, and PR command since
// local-only operation has no forge CLI or pull request concepts. The pipeline
// prefix "local" is used for pipeline filtering.
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
	case ForgeCodeberg:
		return "tea", "gt", "Pull Request", "pr"
	case ForgeLocal:
		return "", "local", "", ""
	default:
		return "", "", "", ""
	}
}

// probeHTTPClient is the HTTP client used by probeForgeType. Package-level
// variable so tests can replace it with a client pointing at httptest
// servers. Single-shot (MaxRetries: 0) — probes are best-effort and a
// failed probe just means "this isn't that forge", retrying would only
// slow detection. Self-signed TLS is tolerated because self-hosted Gitea
// or GitLab instances frequently ship with non-public CAs.
var probeHTTPClient = httpx.New(httpx.Config{
	Timeout:    3 * time.Second,
	MaxRetries: 0,
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // self-hosted instances often use self-signed certs
	},
	CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse // don't follow redirects
	},
})

// probeForgeType probes well-known API endpoints on the given host to identify
// the forge software. Each probe uses a 3s HTTP timeout. Returns the first
// matching forge type, or ForgeUnknown if all probes fail.
func probeForgeType(host string) ForgeType {
	// Probe order: Forgejo before Gitea because Forgejo also serves the Gitea
	// endpoint, so checking the Forgejo-specific one first avoids misclassification.
	probes := []struct {
		path      string
		forgeType ForgeType
	}{
		{"/api/forgejo/v1/version", ForgeForgejo},
		{"/api/v1/version", ForgeGitea},
		{"/api/v4/version", ForgeGitLab},
		{"/rest/api/1.0/application-properties", ForgeBitbucket},
	}

	for _, p := range probes {
		url := "https://" + host + p.path
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		resp, err := probeHTTPClient.Get(ctx, url)
		cancel()
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
	prefixes := []string{"gh-", "gl-", "bb-", "gt-", "local-"}
	for _, p := range prefixes {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}
