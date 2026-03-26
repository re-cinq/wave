---
name: wave-ctx-security
description: Domain context for Wave's security bounded context
---

# Security Context

Permission enforcement, input sanitization, path validation, sandbox isolation, and audit logging.

## Invariants

- All user input is sanitized for prompt injection before reaching any adapter; detection patterns are compiled once at init and reused
- Deny rules are projected into BOTH `settings.json` (adapter-level permission enforcement) AND the runtime CLAUDE.md (behavioral restriction section) -- neither alone is sufficient
- No credentials are written to disk; the audit logger scrubs all values matching credential patterns (`API_KEY`, `TOKEN`, `SECRET`, `PASSWORD`, `CREDENTIAL`, `AUTH`, `PRIVATE_KEY`, `ACCESS_KEY`) before writing trace entries
- Path validation rejects traversal attempts (`../`), enforces approved directories, normalizes Unicode to NFC, and detects homograph attacks before any file operation
- Symlink following is disabled by default (`AllowSymlinks: false`)
- Only environment variables listed in `runtime.sandbox.env_passthrough` reach adapter subprocesses
- Security violations are logged with severity levels and a blocked/allowed flag for audit trail

## Key Decisions

- Three-layer defense: outer sandbox (bubblewrap/Docker with read-only FS), adapter sandbox (`settings.json` allow/deny), and prompt restrictions (CLAUDE.md restriction section)
- `deny: ["Bash(*)"]` removes Bash, Write, AND Edit from the model's tool list entirely -- this is a Claude Code CLI behavior, not a Wave design choice, but it must be accounted for when authoring personas
- Input sanitization runs in strict mode by default (`MustPass: true`) -- detected prompt injection rejects the input outright rather than stripping patterns
- `SecurityConfig` uses a builder pattern with `DefaultSecurityConfig()` providing sensible defaults (255 char max path, 10KB max input, 1MB content size limit)
- Persona validation checks that all persona references in pipelines and meta-pipelines resolve to defined personas in the manifest; unknown personas are rejected unless `AllowUnknownPersonas` is set for testing
- Path lengths over 50 characters are truncated in log output to prevent log injection

## Domain Vocabulary

| Term | Meaning |
|------|---------|
| InputSanitizer | Validates and cleans user input; detects prompt injection via compiled regex patterns |
| PathValidator | Validates file paths against approved directories, traversal attacks, Unicode homographs, and length limits |
| SecurityLogger | Structured security event logger with severity levels (critical, high, medium, low) |
| SecurityConfig | Aggregated configuration for all security subsystems (path validation, sanitization, persona validation) |
| Violation | A detected security event classified by type (path_traversal, prompt_injection, input_validation) and source |
| Severity | Event severity level: `critical` (blocked), `high` (blocked), `medium` (warned), `low` (logged) |
| Sandbox | Execution isolation layer; backends are `bubblewrap` (Linux namespaces), `docker` (container), or `none` |
| Credential scrubbing | Regex-based removal of secret values from audit log entries before they hit disk |
| Approved directory | A path prefix that file operations are restricted to (`.wave/contracts/`, `.wave/schemas/`, `contracts/`, `schemas/`) |
| Env passthrough | The explicit list of host environment variables that reach adapter subprocesses through the sandbox |

## Neighboring Contexts

- **Execution** (`internal/pipeline/`) -- the executor instantiates security infrastructure (`PathValidator`, `InputSanitizer`, `SecurityLogger`) and passes it through the step lifecycle
- **Configuration** (`internal/manifest/`) -- `RuntimeSandbox` config, persona `Permissions`, and adapter `DefaultPermissions` feed into security enforcement
- **Audit** (`internal/audit/`) -- `TraceLogger` handles structured logging with credential scrubbing; it consumes security events but is a separate package

## Key Files

- `internal/security/sanitize.go` -- `InputSanitizer`, prompt injection detection, content size enforcement
- `internal/security/path.go` -- `PathValidator`, traversal detection, approved directory enforcement, Unicode normalization
- `internal/security/config.go` -- `SecurityConfig`, `DefaultSecurityConfig()`, validation method
- `internal/security/logging.go` -- `SecurityLogger`, violation logging, path sanitization for log output
- `internal/security/errors.go` -- `SecurityValidationError`, `PathTraversalError`, `InputValidationError` types
- `internal/security/events.go` -- `SecurityViolationEvent`, violation and source type constants
- `internal/sandbox/sandbox.go` -- `Sandbox` interface with `Wrap()` and `Cleanup()` methods
- `internal/sandbox/docker.go` -- Docker sandbox backend implementation
- `internal/sandbox/factory.go` -- `NewSandbox()` factory dispatching on `SandboxBackendType`
- `internal/sandbox/types.go` -- `Config` struct, `SandboxBackendType` enum (none, docker, bubblewrap)
- `internal/audit/logger.go` -- `AuditLogger` interface, `TraceLogger` with credential scrubbing regex
- `internal/audit/trace.go` -- `DebugTracer` for structured NDJSON trace file output
- `internal/adapter/permissions.go` -- Permission normalization and settings.json projection
