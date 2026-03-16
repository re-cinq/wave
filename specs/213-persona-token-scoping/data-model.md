# Data Model: Persona Token Scoping

**Branch**: `213-persona-token-scoping` | **Date**: 2026-03-16

## Entities

### TokenScope (Value Object)

Represents a single declared permission requirement parsed from the `token_scopes` field.

```go
// Package: internal/scope

// TokenScope represents a parsed scope declaration from a persona's token_scopes field.
type TokenScope struct {
    Resource   string // Canonical resource name: issues, pulls, repos, actions, packages
    Permission string // Permission level: read, write, admin
    EnvVar     string // Optional token env var override (from @ENV_VAR suffix); empty = use default
}
```

**Parsing rules**:
- Input format: `<resource>:<permission>` or `<resource>:<permission>@<ENV_VAR>`
- `Resource` must be from canonical set or generates lint warning
- `Permission` must be one of: `read`, `write`, `admin`
- `Permission` hierarchy: `admin` ⊇ `write` ⊇ `read`
- Empty/malformed strings produce a validation error

**Example values**:
- `issues:read` → `{Resource: "issues", Permission: "read", EnvVar: ""}`
- `pulls:write@GH_TOKEN` → `{Resource: "pulls", Permission: "write", EnvVar: "GH_TOKEN"}`

### ScopeResolver (Service)

Translates abstract `TokenScope` declarations into platform-specific scope identifiers.

```go
// Package: internal/scope

// ScopeResolver maps abstract scopes to forge-native identifiers.
type ScopeResolver struct {
    forgeType forge.ForgeType
}

// Resolve translates a TokenScope to the forge-native scope string(s) required.
// Returns the expected scope names that the token must have.
func (r *ScopeResolver) Resolve(scope TokenScope) ([]string, error)
```

**Behavior**:
- GitHub classic PAT: maps to OAuth scope names (e.g., `repo`, `read:packages`)
- GitLab: maps to token scope names (e.g., `api`, `read_repository`)
- Gitea: maps to Gitea permission names (e.g., `read:issue`, `write:repository`)
- Unknown/Bitbucket: returns error (caller decides whether to warn or fail)

### TokenIntrospector (Service)

Queries the actual scopes of a forge API token at runtime.

```go
// Package: internal/scope

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

// GitHubIntrospector uses `gh api` to discover token scopes.
type GitHubIntrospector struct{}

// GitLabIntrospector uses `glab api` to discover token scopes.
type GitLabIntrospector struct{}

// GiteaIntrospector uses Gitea API to discover token scopes.
type GiteaIntrospector struct{}
```

**Behavior**:
- Each implementation shells out to the forge CLI or uses curl
- Results are cached per `envVar` for the duration of the pipeline run
- Introspection failure is non-fatal: returns `TokenInfo` with `Error` set
- Caller checks `Error` and decides whether to warn or skip validation

### ScopeValidator (Service)

Orchestrates the full validation flow: parse → resolve → introspect → compare.

```go
// Package: internal/scope

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

// Validator checks that forge tokens satisfy persona scope requirements.
type Validator struct {
    resolver     *ScopeResolver
    introspector TokenIntrospector
    forgeInfo    forge.ForgeInfo
    envPassthrough []string // From runtime.sandbox.env_passthrough
}

// ValidatePersonas checks all personas' scope requirements against active tokens.
// Returns all violations aggregated (FR-006).
func (v *Validator) ValidatePersonas(personas map[string]manifest.Persona) (*ValidationResult, error)
```

**Behavior**:
- Iterates all personas, skips those without `token_scopes` (FR-010)
- Parses each scope string into `TokenScope`
- Resolves to platform-specific scopes via `ScopeResolver`
- Introspects actual token scopes via `TokenIntrospector`
- Compares and collects all violations before returning (FR-006)
- Checks `env_passthrough` includes required token vars
- Unknown forge → warning, skip enforcement (FR-007)

## Manifest Schema Extension

### Persona.TokenScopes field

```yaml
# wave.yaml
personas:
  navigator:
    adapter: claude
    system_prompt_file: .wave/personas/navigator.md
    token_scopes:        # NEW — optional field
      - issues:read
      - pulls:read
    permissions:
      allowed_tools: [Read, Grep, Glob]
      deny: [Write(*), Edit(*)]

  implementer:
    adapter: claude
    system_prompt_file: .wave/personas/implementer.md
    token_scopes:        # NEW — optional field
      - issues:read
      - pulls:write
      - repos:write
    permissions:
      allowed_tools: [Read, Write, Edit, Bash]
      deny: [Bash(rm -rf /*)]
```

### Go type change

```go
// In internal/manifest/types.go
type Persona struct {
    Adapter          string          `yaml:"adapter"`
    Description      string          `yaml:"description,omitempty"`
    SystemPromptFile string          `yaml:"system_prompt_file"`
    Temperature      float64         `yaml:"temperature,omitempty"`
    Model            string          `yaml:"model,omitempty"`
    Permissions      Permissions     `yaml:"permissions,omitempty"`
    Hooks            HookConfig      `yaml:"hooks,omitempty"`
    Sandbox          *PersonaSandbox `yaml:"sandbox,omitempty"`
    Skills           []string        `yaml:"skills,omitempty"`
    TokenScopes      []string        `yaml:"token_scopes,omitempty"` // NEW
}
```

## Package Dependencies

```
internal/scope (NEW)
├── imports: internal/forge (ForgeType, ForgeInfo)
├── imports: internal/manifest (Persona — for TokenScopes field)
└── imported by: internal/pipeline (executor.go — preflight phase)

internal/manifest (MODIFIED)
└── Persona struct gains TokenScopes field

internal/pipeline (MODIFIED)
└── executor.go gains scope validation call after preflight

internal/onboarding (MODIFIED)
└── Manifest generation includes token_scopes comments
```

## Validation Flow Sequence

```
wave run <pipeline> <input>
  │
  ├─ Load manifest (manifest.Load)
  │   └─ Parse token_scopes per persona (YAML unmarshal — automatic)
  │   └─ Validate scope syntax (scope.ParseScope — called from manifest validator)
  │
  ├─ Preflight: tools & skills (preflight.Checker.Run)
  │
  ├─ Token scope validation (scope.Validator.ValidatePersonas)  ← NEW
  │   ├─ For each persona with token_scopes:
  │   │   ├─ Parse scope strings → []TokenScope
  │   │   ├─ Check env_passthrough includes required token vars
  │   │   ├─ Resolve abstract → platform scopes (ScopeResolver)
  │   │   ├─ Introspect actual token scopes (TokenIntrospector)
  │   │   └─ Compare required vs actual → ScopeViolation if mismatch
  │   ├─ Aggregate all violations
  │   └─ Return ValidationResult
  │       ├─ Violations → fail pipeline with error listing all missing scopes
  │       └─ Warnings → emit events, continue execution
  │
  └─ Execute pipeline steps
```
