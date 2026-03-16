package scope

import (
	"fmt"
	"strings"
)

// Canonical resource names for token scopes.
var canonicalResources = map[string]bool{
	"issues":   true,
	"pulls":    true,
	"repos":    true,
	"actions":  true,
	"packages": true,
}

// Valid permission levels in hierarchical order.
var validPermissions = map[string]bool{
	"read":  true,
	"write": true,
	"admin": true,
}

// permissionLevel maps permission strings to their hierarchy level.
var permissionLevel = map[string]int{
	"read":  1,
	"write": 2,
	"admin": 3,
}

// TokenScope represents a parsed scope declaration from a persona's token_scopes field.
type TokenScope struct {
	Resource   string // Canonical resource name: issues, pulls, repos, actions, packages
	Permission string // Permission level: read, write, admin
	EnvVar     string // Optional token env var override (from @ENV_VAR suffix); empty = use default
}

// String returns the canonical string representation of the scope.
func (s TokenScope) String() string {
	base := s.Resource + ":" + s.Permission
	if s.EnvVar != "" {
		return base + "@" + s.EnvVar
	}
	return base
}

// ParseWarning represents a non-fatal issue found during scope parsing.
type ParseWarning struct {
	Scope   string
	Message string
}

// Parse parses a scope string in the format "<resource>:<permission>" or
// "<resource>:<permission>@<ENV_VAR>" into a TokenScope.
// Returns an error for invalid syntax and a warning for unknown (but syntactically valid) resources.
func Parse(scopeStr string) (TokenScope, []ParseWarning, error) {
	if strings.TrimSpace(scopeStr) == "" {
		return TokenScope{}, nil, fmt.Errorf("empty scope string")
	}

	scopeStr = strings.TrimSpace(scopeStr)

	// Split off optional @ENV_VAR suffix
	var envVar string
	if idx := strings.Index(scopeStr, "@"); idx >= 0 {
		envVar = scopeStr[idx+1:]
		scopeStr = scopeStr[:idx]
		if envVar == "" {
			return TokenScope{}, nil, fmt.Errorf("scope %q has empty env var after @", scopeStr)
		}
	}

	// Split resource:permission
	parts := strings.SplitN(scopeStr, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return TokenScope{}, nil, fmt.Errorf("invalid scope format %q: expected <resource>:<permission>", scopeStr)
	}

	resource := strings.ToLower(parts[0])
	permission := strings.ToLower(parts[1])

	if !validPermissions[permission] {
		return TokenScope{}, nil, fmt.Errorf("invalid permission %q in scope %q: must be read, write, or admin", permission, scopeStr)
	}

	var warnings []ParseWarning
	if !canonicalResources[resource] {
		warnings = append(warnings, ParseWarning{
			Scope:   scopeStr,
			Message: fmt.Sprintf("unknown resource %q in scope %q; known resources: issues, pulls, repos, actions, packages", resource, scopeStr),
		})
	}

	return TokenScope{
		Resource:   resource,
		Permission: permission,
		EnvVar:     envVar,
	}, warnings, nil
}

// ValidateScopes parses all scope strings and returns aggregated errors.
// Returns nil if all scopes are valid.
func ValidateScopes(scopes []string) []error {
	var errs []error
	for _, s := range scopes {
		if _, _, err := Parse(s); err != nil {
			errs = append(errs, fmt.Errorf("invalid token scope %q: %w", s, err))
		}
	}
	return errs
}

// PermissionSatisfies returns true if the "have" permission level satisfies
// the "need" permission level in the hierarchy: admin ⊇ write ⊇ read.
func PermissionSatisfies(have, need string) bool {
	haveLevel, haveOK := permissionLevel[strings.ToLower(have)]
	needLevel, needOK := permissionLevel[strings.ToLower(need)]
	if !haveOK || !needOK {
		return false
	}
	return haveLevel >= needLevel
}
