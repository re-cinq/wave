package scope

import (
	"testing"
)

// T007: Parse() tests
func TestParse(t *testing.T) {
	t.Run("valid scopes", func(t *testing.T) {
		tests := []struct {
			name      string
			input     string
			wantScope TokenScope
			wantWarns int
		}{
			{
				name:  "issues:read",
				input: "issues:read",
				wantScope: TokenScope{
					Resource:   "issues",
					Permission: "read",
					EnvVar:     "",
				},
				wantWarns: 0,
			},
			{
				name:  "pulls:write",
				input: "pulls:write",
				wantScope: TokenScope{
					Resource:   "pulls",
					Permission: "write",
					EnvVar:     "",
				},
				wantWarns: 0,
			},
			{
				name:  "repos:admin",
				input: "repos:admin",
				wantScope: TokenScope{
					Resource:   "repos",
					Permission: "admin",
					EnvVar:     "",
				},
				wantWarns: 0,
			},
			{
				name:  "packages:read",
				input: "packages:read",
				wantScope: TokenScope{
					Resource:   "packages",
					Permission: "read",
					EnvVar:     "",
				},
				wantWarns: 0,
			},
			{
				name:  "actions:write",
				input: "actions:write",
				wantScope: TokenScope{
					Resource:   "actions",
					Permission: "write",
					EnvVar:     "",
				},
				wantWarns: 0,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, warns, err := Parse(tt.input)
				if err != nil {
					t.Fatalf("Parse(%q) unexpected error: %v", tt.input, err)
				}
				if len(warns) != tt.wantWarns {
					t.Errorf("Parse(%q) got %d warnings, want %d", tt.input, len(warns), tt.wantWarns)
				}
				if got.Resource != tt.wantScope.Resource {
					t.Errorf("Resource = %q, want %q", got.Resource, tt.wantScope.Resource)
				}
				if got.Permission != tt.wantScope.Permission {
					t.Errorf("Permission = %q, want %q", got.Permission, tt.wantScope.Permission)
				}
				if got.EnvVar != tt.wantScope.EnvVar {
					t.Errorf("EnvVar = %q, want %q", got.EnvVar, tt.wantScope.EnvVar)
				}
			})
		}
	})

	t.Run("valid scopes with @ENV_VAR", func(t *testing.T) {
		tests := []struct {
			name      string
			input     string
			wantScope TokenScope
		}{
			{
				name:  "issues:read@GH_TOKEN",
				input: "issues:read@GH_TOKEN",
				wantScope: TokenScope{
					Resource:   "issues",
					Permission: "read",
					EnvVar:     "GH_TOKEN",
				},
			},
			{
				name:  "pulls:write@GITLAB_TOKEN",
				input: "pulls:write@GITLAB_TOKEN",
				wantScope: TokenScope{
					Resource:   "pulls",
					Permission: "write",
					EnvVar:     "GITLAB_TOKEN",
				},
			},
			{
				name:  "repos:admin@MY_CUSTOM_TOKEN",
				input: "repos:admin@MY_CUSTOM_TOKEN",
				wantScope: TokenScope{
					Resource:   "repos",
					Permission: "admin",
					EnvVar:     "MY_CUSTOM_TOKEN",
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, warns, err := Parse(tt.input)
				if err != nil {
					t.Fatalf("Parse(%q) unexpected error: %v", tt.input, err)
				}
				if len(warns) != 0 {
					t.Errorf("Parse(%q) got %d warnings, want 0", tt.input, len(warns))
				}
				if got.Resource != tt.wantScope.Resource {
					t.Errorf("Resource = %q, want %q", got.Resource, tt.wantScope.Resource)
				}
				if got.Permission != tt.wantScope.Permission {
					t.Errorf("Permission = %q, want %q", got.Permission, tt.wantScope.Permission)
				}
				if got.EnvVar != tt.wantScope.EnvVar {
					t.Errorf("EnvVar = %q, want %q", got.EnvVar, tt.wantScope.EnvVar)
				}
			})
		}
	})

	t.Run("invalid scopes return errors", func(t *testing.T) {
		tests := []struct {
			name  string
			input string
		}{
			{
				name:  "empty string",
				input: "",
			},
			{
				name:  "whitespace only",
				input: "   ",
			},
			{
				name:  "no colon separator",
				input: "invalid",
			},
			{
				name:  "missing permission",
				input: "issues:",
			},
			{
				name:  "missing resource",
				input: ":read",
			},
			{
				name:  "unknown permission",
				input: "issues:unknown",
			},
			{
				name:  "empty env var after @",
				input: "issues:read@",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, _, err := Parse(tt.input)
				if err == nil {
					t.Errorf("Parse(%q) expected error, got nil", tt.input)
				}
			})
		}
	})

	t.Run("unknown resources produce warnings but no error", func(t *testing.T) {
		tests := []struct {
			name  string
			input string
		}{
			{
				name:  "custom:read",
				input: "custom:read",
			},
			{
				name:  "deployments:write",
				input: "deployments:write",
			},
			{
				name:  "webhooks:admin",
				input: "webhooks:admin",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, warns, err := Parse(tt.input)
				if err != nil {
					t.Errorf("Parse(%q) unexpected error: %v", tt.input, err)
				}
				if len(warns) == 0 {
					t.Errorf("Parse(%q) expected at least one warning for unknown resource, got none", tt.input)
				}
			})
		}
	})

	t.Run("case insensitive parsing", func(t *testing.T) {
		tests := []struct {
			name           string
			input          string
			wantResource   string
			wantPermission string
		}{
			{
				name:           "Issues:READ",
				input:          "Issues:READ",
				wantResource:   "issues",
				wantPermission: "read",
			},
			{
				name:           "PULLS:WRITE",
				input:          "PULLS:WRITE",
				wantResource:   "pulls",
				wantPermission: "write",
			},
			{
				name:           "Repos:Admin",
				input:          "Repos:Admin",
				wantResource:   "repos",
				wantPermission: "admin",
			},
			{
				name:           "PACKAGES:READ",
				input:          "PACKAGES:READ",
				wantResource:   "packages",
				wantPermission: "read",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, _, err := Parse(tt.input)
				if err != nil {
					t.Fatalf("Parse(%q) unexpected error: %v", tt.input, err)
				}
				if got.Resource != tt.wantResource {
					t.Errorf("Resource = %q, want %q", got.Resource, tt.wantResource)
				}
				if got.Permission != tt.wantPermission {
					t.Errorf("Permission = %q, want %q", got.Permission, tt.wantPermission)
				}
			})
		}
	})

	t.Run("String() round-trips correctly", func(t *testing.T) {
		tests := []struct {
			name  string
			input string
			want  string
		}{
			{
				name:  "simple scope",
				input: "issues:read",
				want:  "issues:read",
			},
			{
				name:  "scope with env var",
				input: "pulls:write@GH_TOKEN",
				want:  "pulls:write@GH_TOKEN",
			},
			{
				name:  "mixed case normalises",
				input: "Issues:READ",
				want:  "issues:read",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, _, err := Parse(tt.input)
				if err != nil {
					t.Fatalf("Parse(%q) unexpected error: %v", tt.input, err)
				}
				if got.String() != tt.want {
					t.Errorf("String() = %q, want %q", got.String(), tt.want)
				}
			})
		}
	})
}

// T008: ValidateScopes() tests
func TestValidateScopes(t *testing.T) {
	t.Run("all valid scopes returns nil", func(t *testing.T) {
		scopes := []string{
			"issues:read",
			"pulls:write",
			"repos:admin",
			"packages:read",
			"actions:write",
		}
		errs := ValidateScopes(scopes)
		if len(errs) != 0 {
			t.Errorf("ValidateScopes(%v) got %d errors, want 0: %v", scopes, len(errs), errs)
		}
	})

	t.Run("mix of valid and invalid returns only invalid as errors", func(t *testing.T) {
		scopes := []string{
			"issues:read",   // valid
			"invalid",       // invalid — no colon
			"pulls:write",   // valid
			"repos:",        // invalid — missing permission
			"packages:read", // valid
			":admin",        // invalid — missing resource
		}
		errs := ValidateScopes(scopes)
		wantErrCount := 3
		if len(errs) != wantErrCount {
			t.Errorf("ValidateScopes(%v) got %d errors, want %d: %v", scopes, len(errs), wantErrCount, errs)
		}
	})

	t.Run("all invalid scopes returns all as errors", func(t *testing.T) {
		scopes := []string{
			"",
			"invalid",
			"issues:",
			":read",
			"issues:unknown",
			"issues:read@",
		}
		errs := ValidateScopes(scopes)
		if len(errs) != len(scopes) {
			t.Errorf("ValidateScopes(%v) got %d errors, want %d: %v", scopes, len(errs), len(scopes), errs)
		}
	})

	t.Run("empty slice returns nil", func(t *testing.T) {
		errs := ValidateScopes([]string{})
		if len(errs) != 0 {
			t.Errorf("ValidateScopes([]) got %d errors, want 0: %v", len(errs), errs)
		}
	})

	t.Run("nil slice returns nil", func(t *testing.T) {
		errs := ValidateScopes(nil)
		if len(errs) != 0 {
			t.Errorf("ValidateScopes(nil) got %d errors, want 0: %v", len(errs), errs)
		}
	})

	t.Run("unknown resource is not an error", func(t *testing.T) {
		// Unknown resources produce warnings during Parse but ValidateScopes only returns errors.
		scopes := []string{
			"custom:read",
			"deployments:write",
		}
		errs := ValidateScopes(scopes)
		if len(errs) != 0 {
			t.Errorf("ValidateScopes(%v) got %d errors for unknown resources, want 0: %v", scopes, len(errs), errs)
		}
	})
}

// T008: PermissionSatisfies() tests
func TestPermissionSatisfies(t *testing.T) {
	t.Run("admin satisfies admin, write, and read", func(t *testing.T) {
		tests := []struct {
			need string
			want bool
		}{
			{need: "admin", want: true},
			{need: "write", want: true},
			{need: "read", want: true},
		}
		for _, tt := range tests {
			t.Run("admin satisfies "+tt.need, func(t *testing.T) {
				got := PermissionSatisfies("admin", tt.need)
				if got != tt.want {
					t.Errorf("PermissionSatisfies(\"admin\", %q) = %v, want %v", tt.need, got, tt.want)
				}
			})
		}
	})

	t.Run("write satisfies write and read but not admin", func(t *testing.T) {
		tests := []struct {
			need string
			want bool
		}{
			{need: "admin", want: false},
			{need: "write", want: true},
			{need: "read", want: true},
		}
		for _, tt := range tests {
			t.Run("write satisfies "+tt.need+" = "+boolStr(tt.want), func(t *testing.T) {
				got := PermissionSatisfies("write", tt.need)
				if got != tt.want {
					t.Errorf("PermissionSatisfies(\"write\", %q) = %v, want %v", tt.need, got, tt.want)
				}
			})
		}
	})

	t.Run("read satisfies only read", func(t *testing.T) {
		tests := []struct {
			need string
			want bool
		}{
			{need: "admin", want: false},
			{need: "write", want: false},
			{need: "read", want: true},
		}
		for _, tt := range tests {
			t.Run("read satisfies "+tt.need+" = "+boolStr(tt.want), func(t *testing.T) {
				got := PermissionSatisfies("read", tt.need)
				if got != tt.want {
					t.Errorf("PermissionSatisfies(\"read\", %q) = %v, want %v", tt.need, got, tt.want)
				}
			})
		}
	})

	t.Run("unknown permission returns false", func(t *testing.T) {
		tests := []struct {
			have string
			need string
		}{
			{have: "unknown", need: "read"},
			{have: "read", need: "unknown"},
			{have: "superadmin", need: "admin"},
			{have: "admin", need: "superadmin"},
			{have: "", need: "read"},
			{have: "read", need: ""},
			{have: "", need: ""},
		}
		for _, tt := range tests {
			t.Run("PermissionSatisfies("+tt.have+","+tt.need+")", func(t *testing.T) {
				got := PermissionSatisfies(tt.have, tt.need)
				if got {
					t.Errorf("PermissionSatisfies(%q, %q) = true, want false", tt.have, tt.need)
				}
			})
		}
	})

	t.Run("case insensitive comparisons", func(t *testing.T) {
		tests := []struct {
			have string
			need string
			want bool
		}{
			{have: "ADMIN", need: "read", want: true},
			{have: "Write", need: "READ", want: true},
			{have: "READ", need: "WRITE", want: false},
			{have: "Admin", need: "Admin", want: true},
		}
		for _, tt := range tests {
			t.Run("PermissionSatisfies("+tt.have+","+tt.need+")", func(t *testing.T) {
				got := PermissionSatisfies(tt.have, tt.need)
				if got != tt.want {
					t.Errorf("PermissionSatisfies(%q, %q) = %v, want %v", tt.have, tt.need, got, tt.want)
				}
			})
		}
	})
}

// boolStr returns "true" or "false" as a string for use in subtest names.
func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
