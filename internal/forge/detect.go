package forge

import (
	"net/url"
	"os/exec"
	"strings"
)

// defaultDomains maps well-known hostnames to forge types.
var defaultDomains = map[string]ForgeType{
	"github.com":    GitHub,
	"gitlab.com":    GitLab,
	"bitbucket.org": Bitbucket,
	"codeberg.org":  Gitea,
}

// giteaIndicators are substrings that suggest a Gitea/Forgejo instance.
var giteaIndicators = []string{"gitea", "forgejo"}

// GitRemoteFunc is the function signature for obtaining git remote output.
// This enables testing without actual git repositories.
type GitRemoteFunc func() (string, error)

// DefaultGitRemoteFunc runs "git remote -v" and returns its output.
func DefaultGitRemoteFunc() (string, error) {
	out, err := exec.Command("git", "remote", "-v").CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// Detect runs forge detection using the provided configuration and git remote function.
// If gitRemoteFn is nil, DefaultGitRemoteFunc is used.
func Detect(cfg *ForgeConfig, gitRemoteFn GitRemoteFunc) ([]ForgeDetection, error) {
	if gitRemoteFn == nil {
		gitRemoteFn = DefaultGitRemoteFunc
	}

	output, err := gitRemoteFn()
	if err != nil {
		return nil, err
	}

	return parseRemotes(output, cfg), nil
}

// parseRemotes parses "git remote -v" output and classifies each remote.
func parseRemotes(output string, cfg *ForgeConfig) []ForgeDetection {
	seen := make(map[string]bool)
	var detections []ForgeDetection

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		// Only process fetch lines to avoid duplicates
		if len(fields) >= 3 && fields[2] != "(fetch)" {
			continue
		}

		remoteURL := fields[1]
		hostname := extractHost(remoteURL)
		if hostname == "" {
			continue
		}

		if seen[hostname] {
			continue
		}
		seen[hostname] = true

		forgeType := classifyHost(hostname, cfg)
		detections = append(detections, ForgeDetection{
			Type:     forgeType,
			Remote:   remoteURL,
			Hostname: hostname,
			CLITool:  forgeType.CLITool(),
		})
	}

	return detections
}

// extractHost extracts the hostname from an SSH or HTTPS git remote URL.
func extractHost(remoteURL string) string {
	// SSH format: git@github.com:org/repo.git
	if strings.Contains(remoteURL, "@") && strings.Contains(remoteURL, ":") && !strings.Contains(remoteURL, "://") {
		atIdx := strings.Index(remoteURL, "@")
		colonIdx := strings.Index(remoteURL[atIdx:], ":")
		if colonIdx > 0 {
			host := remoteURL[atIdx+1 : atIdx+colonIdx]
			host = strings.TrimPrefix(host, "[")
			if bracketIdx := strings.Index(host, "]"); bracketIdx >= 0 {
				host = host[:bracketIdx]
			}
			return strings.ToLower(host)
		}
	}

	// SSH URL format: ssh://git@github.com/org/repo.git
	if strings.HasPrefix(remoteURL, "ssh://") {
		parsed, err := url.Parse(remoteURL)
		if err != nil {
			return ""
		}
		return strings.ToLower(parsed.Hostname())
	}

	// HTTPS format: https://github.com/org/repo.git
	if strings.Contains(remoteURL, "://") {
		parsed, err := url.Parse(remoteURL)
		if err != nil {
			return ""
		}
		return strings.ToLower(parsed.Hostname())
	}

	return ""
}

// classifyHost determines the forge type from a hostname.
func classifyHost(hostname string, cfg *ForgeConfig) ForgeType {
	// Check user-configured domains first (takes priority)
	if cfg != nil && cfg.Domains != nil {
		for domain, forgeStr := range cfg.Domains {
			if strings.EqualFold(hostname, domain) {
				return ParseForgeType(forgeStr)
			}
		}
	}

	// Check exact matches against built-in defaults
	if ft, ok := defaultDomains[hostname]; ok {
		return ft
	}

	// Check subdomain matches (e.g., github.example.com matches github.com pattern)
	for domain, ft := range defaultDomains {
		if strings.HasSuffix(hostname, "."+domain) {
			return ft
		}
	}

	// Check Gitea indicators in hostname
	for _, indicator := range giteaIndicators {
		if strings.Contains(hostname, indicator) {
			return Gitea
		}
	}

	return Unknown
}

// DetectPrimary is a convenience function that returns the first detected forge.
// If multiple forges are detected, it returns all detections so the caller can disambiguate.
// If no forges are detected, it returns a single Unknown detection.
func DetectPrimary(cfg *ForgeConfig, gitRemoteFn GitRemoteFunc) (ForgeDetection, []ForgeDetection, error) {
	detections, err := Detect(cfg, gitRemoteFn)
	if err != nil {
		return ForgeDetection{Type: Unknown}, nil, err
	}

	if len(detections) == 0 {
		return ForgeDetection{Type: Unknown}, nil, nil
	}

	// Filter out Unknown detections for primary selection
	var known []ForgeDetection
	for _, d := range detections {
		if d.Type != Unknown {
			known = append(known, d)
		}
	}

	if len(known) == 0 {
		return detections[0], detections, nil
	}

	// Check if all known detections are the same forge type
	allSame := true
	for _, d := range known[1:] {
		if d.Type != known[0].Type {
			allSame = false
			break
		}
	}

	if allSame {
		return known[0], detections, nil
	}

	// Multiple forge types detected — return first known but include all for disambiguation
	return known[0], detections, nil
}

// IsAmbiguous returns true if the detections contain multiple different known forge types.
func IsAmbiguous(detections []ForgeDetection) bool {
	var firstKnown ForgeType
	for _, d := range detections {
		if d.Type == Unknown {
			continue
		}
		if firstKnown == "" {
			firstKnown = d.Type
			continue
		}
		if d.Type != firstKnown {
			return true
		}
	}
	return false
}
