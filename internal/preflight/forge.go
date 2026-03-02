package preflight

import (
	"fmt"
	"os/exec"
	"strings"
)

// ForgeInfo represents a detected forge type and its expected CLI binary.
type ForgeInfo struct {
	Type string // "github", "gitlab", "gitea", "bitbucket", or "unknown"
	CLI  string // Expected CLI binary name: "gh", "glab", "tea", "bb", or ""
}

// DetectForge maps a git remote URL to a forge type and its expected CLI binary.
// It supports SSH (git@host:...) and HTTPS (https://host/...) URL formats.
// Returns ForgeInfo with Type="unknown" and empty CLI if the forge cannot be determined.
func DetectForge(remoteURL string) ForgeInfo {
	host := extractHost(remoteURL)
	if host == "" {
		return ForgeInfo{Type: "unknown"}
	}

	host = strings.ToLower(host)

	switch {
	case host == "github.com" || strings.HasSuffix(host, ".github.com"):
		return ForgeInfo{Type: "github", CLI: "gh"}
	case host == "gitlab.com" || strings.HasSuffix(host, ".gitlab.com") || strings.Contains(host, "gitlab"):
		return ForgeInfo{Type: "gitlab", CLI: "glab"}
	case strings.Contains(host, "gitea") || strings.Contains(host, "forgejo") || strings.Contains(host, "codeberg"):
		return ForgeInfo{Type: "gitea", CLI: "tea"}
	case host == "bitbucket.org" || strings.HasSuffix(host, ".bitbucket.org") || strings.Contains(host, "bitbucket"):
		return ForgeInfo{Type: "bitbucket", CLI: "bb"}
	default:
		return ForgeInfo{Type: "unknown"}
	}
}

// extractHost extracts the hostname from a git remote URL.
// Supports formats:
//   - https://github.com/org/repo.git
//   - git@github.com:org/repo.git
//   - ssh://git@github.com/org/repo.git
func extractHost(remoteURL string) string {
	url := strings.TrimSpace(remoteURL)
	if url == "" {
		return ""
	}

	// SSH format: git@host:org/repo.git
	if strings.Contains(url, "@") && strings.Contains(url, ":") && !strings.Contains(url, "://") {
		parts := strings.SplitN(url, "@", 2)
		if len(parts) == 2 {
			hostPart := strings.SplitN(parts[1], ":", 2)
			if len(hostPart) == 2 {
				return hostPart[0]
			}
		}
		return ""
	}

	// URL format: https://host/... or ssh://git@host/...
	// Strip scheme
	idx := strings.Index(url, "://")
	if idx >= 0 {
		url = url[idx+3:]
	}

	// Strip user@
	if atIdx := strings.Index(url, "@"); atIdx >= 0 {
		url = url[atIdx+1:]
	}

	// Take host (before first / or :)
	for i, c := range url {
		if c == '/' || c == ':' {
			return url[:i]
		}
	}

	return url
}

// CheckForgeCLI detects the forge type from the given git remote URL and checks
// that the corresponding CLI tool is available on PATH.
func (c *Checker) CheckForgeCLI(remoteURL string) ([]Result, error) {
	if remoteURL == "" {
		return []Result{{
			Name:    "forge-cli",
			Kind:    "forge",
			OK:      true,
			Message: "no git remote configured — forge CLI check skipped",
		}}, nil
	}

	forge := DetectForge(remoteURL)
	if forge.Type == "unknown" || forge.CLI == "" {
		return []Result{{
			Name:    "forge-cli",
			Kind:    "forge",
			OK:      true,
			Message: fmt.Sprintf("unrecognized forge for remote %q — forge CLI check skipped", remoteURL),
		}}, nil
	}

	lookPath := c.lookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}

	_, err := lookPath(forge.CLI)
	if err != nil {
		remediation := forgeCLIRemediation(forge)
		return []Result{{
			Name:        forge.CLI,
			Kind:        "forge",
			OK:          false,
			Message:     fmt.Sprintf("%s CLI %q not found on PATH (detected forge: %s)", forge.Type, forge.CLI, forge.Type),
			Remediation: remediation,
		}}, &ToolError{
			MissingTools: []string{forge.CLI},
		}
	}

	return []Result{{
		Name:    forge.CLI,
		Kind:    "forge",
		OK:      true,
		Message: fmt.Sprintf("%s CLI %q found (detected forge: %s)", forge.Type, forge.CLI, forge.Type),
	}}, nil
}

// forgeCLIRemediation returns install guidance for the given forge CLI.
func forgeCLIRemediation(forge ForgeInfo) string {
	switch forge.Type {
	case "github":
		return "Install the GitHub CLI: https://cli.github.com/"
	case "gitlab":
		return "Install the GitLab CLI: https://gitlab.com/gitlab-org/cli"
	case "gitea":
		return "Install the Gitea CLI (tea): https://gitea.com/gitea/tea"
	case "bitbucket":
		return "Install the Bitbucket CLI (bb): https://bitbucket.org/atlassian/bitbucket-cli"
	default:
		return fmt.Sprintf("Install the %s CLI %q and ensure it is on your PATH", forge.Type, forge.CLI)
	}
}
