# Research: Persona Token Scoping

**Branch**: `213-persona-token-scoping` | **Date**: 2026-03-16

## Unknowns Resolved

### U1: Where does token scope declaration fit in the manifest schema?

**Decision**: Add optional `token_scopes` field to `manifest.Persona` struct.

**Rationale**: The `Persona` struct in `internal/manifest/types.go` already holds `Permissions` (allowed_tools, deny). Token scopes are a complementary permission dimension — they control what the persona can do via forge APIs, while `Permissions` controls what local tools it can invoke. Adding `token_scopes []string` keeps all persona-level access control co-located.

**Alternatives rejected**:
- Global `runtime.token_scopes` — would require mapping scopes to personas separately, losing the per-persona principle of least privilege.
- Separate `security.yaml` — violates Constitution Principle 2 (Manifest as Single Source of Truth).

### U2: Where should scope validation live architecturally?

**Decision**: New `internal/scope` package, called from executor alongside existing preflight checks.

**Rationale**: The spec (C5) explicitly recommends a separate package. The existing `preflight.Checker` validates local tool/skill availability via `exec.LookPath`. Token scope validation is fundamentally different — it requires network API calls to forge endpoints and per-persona scoping. Mixing these concerns would violate single responsibility.

The executor in `internal/pipeline/executor.go` already has a clear preflight section (lines ~303-326) where `preflight.NewChecker` runs. Token scope validation slots in immediately after, using the same pattern of emitting `"preflight"` events and returning errors that block execution.

**Alternatives rejected**:
- Extending `preflight.Checker` — different dependencies (forge APIs vs. exec.LookPath), different scoping (per-persona vs. per-pipeline).
- Middleware in adapter — too late, we want fail-fast before any step executes.

### U3: How to introspect GitHub fine-grained PAT scopes?

**Decision**: Use `gh api user --include` to extract `X-OAuth-Scopes` header for classic PATs. For fine-grained PATs (no `X-OAuth-Scopes` header), use targeted API probes.

**Rationale**: The `gh` CLI is already a required tool in Wave's preflight for GitHub forges. GitHub's API returns `X-OAuth-Scopes` in response headers for classic PATs. Fine-grained PATs don't expose scopes in headers — instead, the token simply gets 403/404 on resources it can't access. The pragmatic approach is:
1. Try `gh api user --include` — parse `X-OAuth-Scopes` header
2. If no `X-OAuth-Scopes` header, assume fine-grained PAT and probe specific endpoints
3. Cache introspection results per token variable for the pipeline run

**Alternatives rejected**:
- Direct HTTP client — adds dependency, bypasses `gh auth` credential chain.
- Skip fine-grained PAT support — increasingly common, would limit adoption.

### U4: How to introspect GitLab token scopes?

**Decision**: Use `glab api /personal_access_tokens/self` which returns the token's scopes directly.

**Rationale**: GitLab's API has a dedicated introspection endpoint that returns `{"scopes": ["api", "read_repository", ...]}`. The `glab` CLI handles auth automatically. This is simpler than GitHub's approach.

### U5: How to introspect Gitea token scopes?

**Decision**: Use Gitea API `/api/v1/user/tokens` (requires `sudo` scope) or fallback to probing `/api/v1/user` response.

**Rationale**: Gitea's token API is less mature. The `tea` CLI may not be universally available, so direct API calls via `curl` with the token header are more reliable. If introspection fails, we warn and skip (FR-007).

### U6: How to map abstract scopes to platform-specific scopes?

**Decision**: Static mapping table per forge type in the `scope` package.

**Rationale**: The canonical vocabulary is small and fixed (issues, pulls, repos, actions, packages × read/write/admin). Each forge has well-documented scope/permission names. A static map is simple, testable, and requires no external data.

| Abstract Scope | GitHub (classic PAT) | GitHub (fine-grained) | GitLab | Gitea |
|---|---|---|---|---|
| `issues:read` | `repo` (or `public_repo`) | Issues: Read | `read_api` | `read:issue` |
| `issues:write` | `repo` | Issues: Read and write | `api` | `write:issue` |
| `pulls:read` | `repo` | Pull requests: Read | `read_api` | `read:issue` |
| `pulls:write` | `repo` | Pull requests: Read and write | `api` | `write:issue` |
| `repos:read` | `repo` (or `public_repo`) | Contents: Read | `read_repository` | `read:repository` |
| `repos:write` | `repo` | Contents: Read and write | `write_repository` | `write:repository` |
| `actions:read` | `repo` | Actions: Read | `read_api` | N/A |
| `actions:write` | `repo` | Actions: Read and write | `api` | N/A |
| `packages:read` | `read:packages` | Packages: Read | `read_api` | `read:package` |
| `packages:write` | `write:packages` | Packages: Read and write | `api` | `write:package` |

### U7: How to handle the `@ENV_VAR` token binding syntax?

**Decision**: Parse in `internal/scope` during manifest loading. The `TokenScope` type stores resource, permission, and optional env var override. Default env var comes from `forge.ForgeInfo` detection.

**Rationale**: The `@` suffix is unambiguous — no forge scope name contains `@`. Parsing happens once during manifest load. The default token variable is determined by the detected forge type (e.g., `GH_TOKEN` for GitHub, `GITLAB_TOKEN` for GitLab).

### U8: How to detect missing `env_passthrough` configuration?

**Decision**: During scope validation, check if each required token env var is listed in `runtime.sandbox.env_passthrough`. If not, report a configuration error with remediation hint.

**Rationale**: Even if a token exists in the host environment, it won't reach adapter subprocesses unless listed in `env_passthrough`. This is a common misconfiguration that should be caught early.
