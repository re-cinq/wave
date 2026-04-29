package scope

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/recinq/wave/internal/config"
	"github.com/recinq/wave/internal/forge"
)

// TokenInfo holds the introspection result for a single token.
type TokenInfo struct {
	EnvVar    string   // Which env var was checked
	Scopes    []string // Actual scopes/permissions the token has
	TokenType string   // "classic", "fine-grained", "project", "unknown"
	Error     error    // Non-nil if introspection failed (warn, don't block)
}

// TokenIntrospector queries forge tokens for their actual permissions.
type TokenIntrospector interface {
	Introspect(envVar string) (*TokenInfo, error)
}

// CommandRunner abstracts command execution for testing.
type CommandRunner func(name string, args ...string) ([]byte, error)

// defaultRunner runs commands via exec.Command.
func defaultRunner(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).CombinedOutput()
}

// GitHubIntrospector uses `gh api` to discover token scopes.
type GitHubIntrospector struct {
	runCmd CommandRunner
}

// GitLabIntrospector uses `glab api` to discover token scopes.
type GitLabIntrospector struct {
	runCmd CommandRunner
}

// GiteaIntrospector uses Gitea API to discover token scopes.
type GiteaIntrospector struct {
	runCmd CommandRunner
}

// NewIntrospector creates a TokenIntrospector for the given forge type.
func NewIntrospector(forgeType forge.ForgeType) TokenIntrospector {
	switch forgeType {
	case forge.ForgeGitHub:
		return &GitHubIntrospector{runCmd: defaultRunner}
	case forge.ForgeGitLab:
		return &GitLabIntrospector{runCmd: defaultRunner}
	case forge.ForgeGitea, forge.ForgeForgejo, forge.ForgeCodeberg:
		return &GiteaIntrospector{runCmd: defaultRunner}
	default:
		return nil
	}
}

// Introspect queries GitHub for the token's OAuth scopes via `gh api user --include`.
func (g *GitHubIntrospector) Introspect(envVar string) (*TokenInfo, error) {
	info := &TokenInfo{EnvVar: envVar, TokenType: "unknown"}

	// Check that the env var is set
	if !config.EnvPresent(envVar) {
		info.Error = fmt.Errorf("environment variable %s is not set", envVar)
		return info, nil
	}

	out, err := g.runCmd("gh", "api", "user", "--include")
	if err != nil {
		info.Error = fmt.Errorf("gh api user failed: %w", err)
		return info, nil
	}

	output := string(out)

	// Parse X-OAuth-Scopes header from response headers
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(line), "x-oauth-scopes:") {
			info.TokenType = "classic"
			scopeStr := strings.TrimSpace(line[len("x-oauth-scopes:"):])
			if scopeStr == "" {
				// Classic PAT with no scopes
				info.Scopes = []string{}
				return info, nil
			}
			for _, s := range strings.Split(scopeStr, ",") {
				s = strings.TrimSpace(s)
				if s != "" {
					info.Scopes = append(info.Scopes, s)
				}
			}
			return info, nil
		}
	}

	// No X-OAuth-Scopes header — likely a fine-grained PAT
	info.TokenType = "fine-grained"
	info.Error = fmt.Errorf("fine-grained PAT detected; scope introspection not available via headers")
	return info, nil
}

// Introspect queries GitLab for the token's scopes via `glab api /personal_access_tokens/self`.
func (g *GitLabIntrospector) Introspect(envVar string) (*TokenInfo, error) {
	info := &TokenInfo{EnvVar: envVar, TokenType: "unknown"}

	if !config.EnvPresent(envVar) {
		info.Error = fmt.Errorf("environment variable %s is not set", envVar)
		return info, nil
	}

	out, err := g.runCmd("glab", "api", "/personal_access_tokens/self")
	if err != nil {
		info.Error = fmt.Errorf("glab api failed: %w", err)
		return info, nil
	}

	var result struct {
		Scopes []string `json:"scopes"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		info.Error = fmt.Errorf("failed to parse GitLab token response: %w", err)
		return info, nil
	}

	info.TokenType = "project"
	info.Scopes = result.Scopes
	return info, nil
}

// Introspect queries Gitea for token permissions via API.
func (g *GiteaIntrospector) Introspect(envVar string) (*TokenInfo, error) {
	info := &TokenInfo{EnvVar: envVar, TokenType: "unknown"}

	if !config.EnvPresent(envVar) {
		info.Error = fmt.Errorf("environment variable %s is not set", envVar)
		return info, nil
	}

	// Try using tea CLI first
	out, err := g.runCmd("tea", "whoami")
	if err != nil {
		info.Error = fmt.Errorf("gitea token introspection failed: %w", err)
		return info, nil
	}

	// Basic check — if we can authenticate, we assume the token is valid
	// Gitea doesn't expose token scopes easily via CLI
	if len(out) > 0 {
		info.TokenType = "unknown"
		info.Error = fmt.Errorf("gitea does not expose token scopes via CLI; skipping detailed scope validation")
	}

	return info, nil
}
