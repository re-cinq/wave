package forge

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/recinq/wave/internal/github"
)

// ResolveToken returns an authentication token for the given forge type.
func ResolveToken(ft ForgeType) string {
	switch ft {
	case ForgeGitHub:
		return resolveGitHubToken()
	case ForgeGitLab:
		return resolveGitLabToken()
	case ForgeBitbucket:
		return resolveBitbucketToken()
	case ForgeGitea, ForgeForgejo:
		return resolveGiteaToken()
	case ForgeCodeberg:
		return resolveCodebergToken()
	default:
		return ""
	}
}

func resolveGitHubToken() string {
	if t := os.Getenv("GH_TOKEN"); t != "" {
		return t
	}
	if t := os.Getenv("GITHUB_TOKEN"); t != "" {
		return t
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "gh", "auth", "token").Output()
	if err == nil {
		if t := strings.TrimSpace(string(out)); t != "" {
			return t
		}
	}
	return ""
}

func resolveGitLabToken() string {
	if t := os.Getenv("GITLAB_TOKEN"); t != "" {
		return t
	}
	if t := os.Getenv("GL_TOKEN"); t != "" {
		return t
	}
	return ""
}

func resolveBitbucketToken() string {
	if t := os.Getenv("BITBUCKET_TOKEN"); t != "" {
		return t
	}
	if t := os.Getenv("BB_TOKEN"); t != "" {
		return t
	}
	return ""
}

func resolveGiteaToken() string {
	if t := os.Getenv("GITEA_TOKEN"); t != "" {
		return t
	}
	return ""
}

func resolveCodebergToken() string {
	if t := os.Getenv("CODEBERG_TOKEN"); t != "" {
		return t
	}
	// Fall back to GITEA_TOKEN since Codeberg is Forgejo
	if t := os.Getenv("GITEA_TOKEN"); t != "" {
		return t
	}
	return ""
}

// NewClient creates a forge.Client for the given ForgeInfo, resolving
// authentication tokens automatically. Returns (nil, nil) if no token is
// found or the forge type is not supported, so callers' nil-guard checks
// show "not configured" rather than a cryptic error. Returns a non-nil
// error only when client construction itself fails.
func NewClient(info ForgeInfo) (Client, error) {
	token := ResolveToken(info.Type)
	if token == "" {
		return nil, nil
	}

	switch info.Type {
	case ForgeGitHub:
		ghClient := github.NewClient(github.ClientConfig{Token: token})
		return NewGitHubClient(ghClient)
	default:
		// Non-GitHub forges not yet supported — return (nil, nil) so callers'
		// nil-guard checks show "not configured" rather than cryptic errors.
		return nil, nil
	}
}
