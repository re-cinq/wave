package scope

import (
	"fmt"
	"strings"

	"github.com/recinq/wave/internal/forge"
)

// ScopeViolation represents a single scope mismatch for a persona.
type ScopeViolation struct {
	PersonaName  string   // Which persona has the violation
	MissingScope string   // The abstract scope that's missing (e.g., "issues:write")
	EnvVar       string   // Which token env var was checked
	Required     []string // Platform-specific scopes needed
	Available    []string // Platform-specific scopes the token actually has
	Hint         string   // Human-readable remediation guidance
}

// ValidationResult holds the aggregate result of scope validation.
type ValidationResult struct {
	Violations []ScopeViolation
	Warnings   []string // Non-blocking issues (e.g., unknown forge, introspection failure)
}

// HasViolations returns true if there are any scope violations.
func (r *ValidationResult) HasViolations() bool {
	return len(r.Violations) > 0
}

// Error returns a formatted error string listing all violations.
func (r *ValidationResult) Error() string {
	if !r.HasViolations() {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("token scope validation failed:\n")
	for _, v := range r.Violations {
		fmt.Fprintf(&sb, "  persona %q: missing scope %q (env: %s)\n", v.PersonaName, v.MissingScope, v.EnvVar)
		if len(v.Required) > 0 {
			fmt.Fprintf(&sb, "    required platform scopes: %s\n", strings.Join(v.Required, ", "))
		}
		if v.Hint != "" {
			fmt.Fprintf(&sb, "    hint: %s\n", v.Hint)
		}
	}
	return sb.String()
}

// Validator checks that forge tokens satisfy persona scope requirements.
type Validator struct {
	resolver       *ScopeResolver
	introspector   TokenIntrospector
	forgeInfo      forge.ForgeInfo
	envPassthrough []string
}

// NewValidator creates a Validator with the given components.
func NewValidator(resolver *ScopeResolver, introspector TokenIntrospector, forgeInfo forge.ForgeInfo, envPassthrough []string) *Validator {
	return &Validator{
		resolver:       resolver,
		introspector:   introspector,
		forgeInfo:      forgeInfo,
		envPassthrough: envPassthrough,
	}
}

// defaultTokenEnvVar returns the default token environment variable for the forge type.
func defaultTokenEnvVar(ft forge.ForgeType) string {
	switch ft {
	case forge.ForgeGitHub:
		return "GH_TOKEN"
	case forge.ForgeGitLab:
		return "GITLAB_TOKEN"
	case forge.ForgeGitea, forge.ForgeForgejo, forge.ForgeCodeberg:
		return "GITEA_TOKEN"
	default:
		return ""
	}
}

// ValidatePersonas checks all personas' scope requirements against active tokens.
// The personas argument maps persona name to its token_scopes slice.
// Returns all violations aggregated (FR-006).
func (v *Validator) ValidatePersonas(personas map[string][]string) (*ValidationResult, error) {
	result := &ValidationResult{}

	// If no introspector available (unknown forge), warn and skip
	if v.introspector == nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("no token introspector available for forge type %q; skipping scope validation", v.forgeInfo.Type))
		return result, nil
	}

	// Cache introspection results per env var
	tokenCache := make(map[string]*TokenInfo)

	for name, tokenScopes := range personas {
		if len(tokenScopes) == 0 {
			continue // FR-010: opt-in enforcement
		}

		for _, scopeStr := range tokenScopes {
			ts, _, err := Parse(scopeStr)
			if err != nil {
				// Parse errors should have been caught during manifest validation
				result.Warnings = append(result.Warnings, fmt.Sprintf("persona %q: failed to parse scope %q: %v", name, scopeStr, err))
				continue
			}

			// Determine which env var to check
			envVar := ts.EnvVar
			if envVar == "" {
				envVar = defaultTokenEnvVar(v.forgeInfo.Type)
			}

			if envVar == "" {
				result.Warnings = append(result.Warnings, fmt.Sprintf("persona %q: cannot determine token env var for forge %q", name, v.forgeInfo.Type))
				continue
			}

			// Check env_passthrough includes the required token var
			if !v.isEnvPassthrough(envVar) {
				result.Violations = append(result.Violations, ScopeViolation{
					PersonaName:  name,
					MissingScope: scopeStr,
					EnvVar:       envVar,
					Hint:         fmt.Sprintf("add %q to runtime.sandbox.env_passthrough in wave.yaml", envVar),
				})
				continue
			}

			// Resolve abstract scope to platform-specific scopes
			required, err := v.resolver.Resolve(ts)
			if err != nil {
				result.Violations = append(result.Violations, ScopeViolation{
					PersonaName:  name,
					MissingScope: scopeStr,
					EnvVar:       envVar,
					Hint:         fmt.Sprintf("scope validation not supported for forge %q; %v", v.forgeInfo.Type, err),
				})
				continue
			}

			// Introspect token (cached per env var)
			tokenInfo, ok := tokenCache[envVar]
			if !ok {
				tokenInfo, err = v.introspector.Introspect(envVar)
				if err != nil {
					result.Warnings = append(result.Warnings, fmt.Sprintf("token introspection failed for %s: %v", envVar, err))
					continue
				}
				tokenCache[envVar] = tokenInfo
			}

			// If introspection had an error, emit a ScopeViolation instead of warning
			if tokenInfo.Error != nil {
				hint := fmt.Sprintf("token introspection failed: %v", tokenInfo.Error)
				if tokenInfo.TokenType == "fine-grained" {
					hint = "fine-grained PATs cannot be introspected; recreate as classic PAT or use --skip-scope-check"
				}
				result.Violations = append(result.Violations, ScopeViolation{
					PersonaName:  name,
					MissingScope: scopeStr,
					EnvVar:       envVar,
					Hint:         hint,
				})
				continue
			}

			// Check if token has the required scopes
			if !hasRequiredScopes(tokenInfo.Scopes, required) {
				result.Violations = append(result.Violations, ScopeViolation{
					PersonaName:  name,
					MissingScope: scopeStr,
					EnvVar:       envVar,
					Required:     required,
					Available:    tokenInfo.Scopes,
					Hint:         fmt.Sprintf("create a token with scopes: %s", strings.Join(required, ", ")),
				})
			}
		}
	}

	return result, nil
}

// isEnvPassthrough checks if the given env var is in the passthrough list.
// If the passthrough list is empty, all env vars are considered allowed.
func (v *Validator) isEnvPassthrough(envVar string) bool {
	if len(v.envPassthrough) == 0 {
		return true // No restrictions
	}
	for _, e := range v.envPassthrough {
		if e == envVar {
			return true
		}
	}
	return false
}

// hasRequiredScopes checks if the available scopes contain all required scopes.
func hasRequiredScopes(available, required []string) bool {
	availSet := make(map[string]bool, len(available))
	for _, s := range available {
		availSet[s] = true
	}
	for _, r := range required {
		if !availSet[r] {
			return false
		}
	}
	return true
}
