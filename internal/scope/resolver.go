package scope

import (
	"fmt"

	"github.com/recinq/wave/internal/forge"
)

// ScopeResolver maps abstract scopes to forge-native identifiers.
type ScopeResolver struct {
	forgeType forge.ForgeType
}

// NewResolver creates a ScopeResolver for the given forge platform.
func NewResolver(forgeType forge.ForgeType) *ScopeResolver {
	return &ScopeResolver{forgeType: forgeType}
}

// Resolve translates a TokenScope to the forge-native scope string(s) required.
func (r *ScopeResolver) Resolve(scope TokenScope) ([]string, error) {
	switch r.forgeType {
	case forge.ForgeGitHub:
		return r.resolveGitHub(scope)
	case forge.ForgeGitLab:
		return r.resolveGitLab(scope)
	case forge.ForgeGitea, forge.ForgeForgejo, forge.ForgeCodeberg:
		return r.resolveGitea(scope)
	case forge.ForgeBitbucket:
		return nil, fmt.Errorf("bitbucket token scope validation is not yet supported; skipping enforcement")
	default:
		return nil, fmt.Errorf("unknown forge type %q; skipping scope enforcement", r.forgeType)
	}
}

// GitHub classic PAT scope mappings.
// GitHub's OAuth scopes are coarse-grained: "repo" covers issues, pulls, repos, actions.
func (r *ScopeResolver) resolveGitHub(scope TokenScope) ([]string, error) {
	switch scope.Resource {
	case "issues":
		return []string{"repo"}, nil
	case "pulls":
		return []string{"repo"}, nil
	case "repos":
		return []string{"repo"}, nil
	case "actions":
		return []string{"repo"}, nil
	case "packages":
		switch scope.Permission {
		case "read":
			return []string{"read:packages"}, nil
		case "write", "admin":
			return []string{"write:packages"}, nil
		}
	}
	// Unknown resource — pass through as-is
	return []string{scope.Resource + ":" + scope.Permission}, nil
}

// GitLab scope mappings.
func (r *ScopeResolver) resolveGitLab(scope TokenScope) ([]string, error) {
	switch scope.Resource {
	case "issues", "pulls", "actions":
		switch scope.Permission {
		case "read":
			return []string{"read_api"}, nil
		default:
			return []string{"api"}, nil
		}
	case "repos":
		switch scope.Permission {
		case "read":
			return []string{"read_repository"}, nil
		default:
			return []string{"write_repository"}, nil
		}
	case "packages":
		switch scope.Permission {
		case "read":
			return []string{"read_api"}, nil
		default:
			return []string{"api"}, nil
		}
	}
	return []string{scope.Resource + ":" + scope.Permission}, nil
}

// Gitea scope mappings.
func (r *ScopeResolver) resolveGitea(scope TokenScope) ([]string, error) {
	switch scope.Resource {
	case "issues":
		switch scope.Permission {
		case "read":
			return []string{"read:issue"}, nil
		default:
			return []string{"write:issue"}, nil
		}
	case "pulls":
		switch scope.Permission {
		case "read":
			return []string{"read:issue"}, nil
		default:
			return []string{"write:issue"}, nil
		}
	case "repos":
		switch scope.Permission {
		case "read":
			return []string{"read:repository"}, nil
		default:
			return []string{"write:repository"}, nil
		}
	case "actions":
		// Gitea doesn't have dedicated action scopes
		return []string{"read:repository"}, nil
	case "packages":
		switch scope.Permission {
		case "read":
			return []string{"read:package"}, nil
		default:
			return []string{"write:package"}, nil
		}
	}
	return []string{scope.Resource + ":" + scope.Permission}, nil
}
