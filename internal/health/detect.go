package health

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// DetectForge runs "git remote -v" in the given repoPath, parses the output
// to find the origin remote (falling back to the first remote), and returns
// the forge type and "owner/repo" identifier extracted from the remote URL.
func DetectForge(repoPath string) (ForgeType, string, error) {
	cmd := exec.Command("git", "remote", "-v")
	cmd.Dir = repoPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return Unknown, "", fmt.Errorf("git remote -v failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	// Parse remote lines. Each line looks like:
	//   origin	https://github.com/owner/repo.git (fetch)
	//   origin	git@github.com:owner/repo.git (push)
	type remote struct {
		name string
		url  string
	}

	var remotes []remote
	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Split into fields: name, url, (fetch|push)
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		// Only consider fetch entries to avoid duplicates.
		if len(fields) >= 3 && fields[2] != "(fetch)" {
			continue
		}

		remotes = append(remotes, remote{name: fields[0], url: fields[1]})
	}

	if len(remotes) == 0 {
		return Unknown, "", fmt.Errorf("no remotes found in repository at %s", repoPath)
	}

	// Prefer "origin", fall back to first remote.
	chosen := remotes[0]
	for _, r := range remotes {
		if r.name == "origin" {
			chosen = r
			break
		}
	}

	forgeType, repoID := parseRemoteURL(chosen.url)
	return forgeType, repoID, nil
}

// parseRemoteURL extracts the forge type and "owner/repo" identifier from a
// git remote URL. It handles both HTTPS and SSH formats:
//
//	HTTPS: https://github.com/owner/repo.git
//	SSH:   git@github.com:owner/repo.git
//
// The .git suffix is stripped from the repo identifier.
func parseRemoteURL(rawURL string) (ForgeType, string) {
	var host, path string

	if strings.HasPrefix(rawURL, "https://") || strings.HasPrefix(rawURL, "http://") {
		// HTTPS format: https://github.com/owner/repo.git
		// Strip the scheme.
		stripped := rawURL
		if idx := strings.Index(stripped, "://"); idx >= 0 {
			stripped = stripped[idx+3:]
		}
		// Split into host and path at the first slash.
		slashIdx := strings.Index(stripped, "/")
		if slashIdx < 0 {
			return Unknown, ""
		}
		host = stripped[:slashIdx]
		path = stripped[slashIdx+1:]
	} else if strings.Contains(rawURL, "@") && strings.Contains(rawURL, ":") {
		// SSH format: git@github.com:owner/repo.git
		atIdx := strings.Index(rawURL, "@")
		colonIdx := strings.Index(rawURL[atIdx:], ":")
		if colonIdx < 0 {
			return Unknown, ""
		}
		host = rawURL[atIdx+1 : atIdx+colonIdx]
		path = rawURL[atIdx+colonIdx+1:]
	} else {
		return Unknown, ""
	}

	// Strip port from host if present (e.g., github.com:8080).
	// For SSH URLs the colon separates host from path, so this only applies
	// to HTTPS URLs where the host might include a port.
	if colonIdx := strings.Index(host, ":"); colonIdx >= 0 {
		host = host[:colonIdx]
	}

	// Strip .git suffix from path.
	path = strings.TrimSuffix(path, ".git")

	// Trim any trailing slashes.
	path = strings.TrimRight(path, "/")

	// Determine forge type from host.
	forgeType := matchHost(host)

	return forgeType, path
}

// matchHost returns the ForgeType for a given hostname.
func matchHost(host string) ForgeType {
	host = strings.ToLower(host)

	switch host {
	case "github.com":
		return GitHub
	case "gitlab.com":
		return GitLab
	case "bitbucket.org":
		return Bitbucket
	case "codeberg.org":
		return Gitea
	default:
		if strings.Contains(host, "gitea") {
			return Gitea
		}
		return Unknown
	}
}
