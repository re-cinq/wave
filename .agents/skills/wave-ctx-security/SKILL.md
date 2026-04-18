---
name: wave-ctx-security
description: Domain context for Three-layer defense system — permission enforcement (deny-first tool access control), input sanitization (prompt injection detection, risk scoring), path validation (traversal prevention, symlink rejection, Unicode homograph detection), sandbox isolation (Docker with full hardening, bubblewrap via Nix), credential scrubbing in audit logs, and curated subprocess environments.
---

# Security Context

Three-layer defense system — permission enforcement (deny-first tool access control), input sanitization (prompt injection detection, risk scoring), path validation (traversal prevention, symlink rejection, Unicode homograph detection), sandbox isolation (Docker with full hardening, bubblewrap via Nix), credential scrubbing in audit logs, and curated subprocess environments.

## Invariants

- Deny patterns ALWAYS take precedence over allow patterns — deny rules checked first, match blocks regardless of allow rules
- Path traversal sequences always rejected — checks .., ./, ../, ..\\, URL-encoded variants (%2e%2e, %252e%252e, ..%2f, ..%5c)
- Paths must be within approved directories — resolved to absolute, checked via filepath.Rel, relative path must not start with '..'
- Symlinks rejected by default — every path component checked via os.Lstat for os.ModeSymlink when AllowSymlinks is false
- Unicode homograph attacks detected and blocked — UTF-7 sequences rejected, mixed confusable scripts (Latin+Cyrillic, Latin+Greek, Latin+Arabic, Latin+Hebrew) rejected; CJK intentionally allowed
- Path length must not exceed MaxPathLength (default 255)
- Prompt injection detection enabled by default in strict (MustPass) mode — seven regex patterns detect injection attempts
- Input length capped at MaxInputLength (default 10000 characters) — excess truncated
- Schema content size must not exceed ContentSizeLimit (default 1MB) — hard error, not truncation
- Script tags, event handlers, and javascript: URLs always stripped from schema content
- TodoWrite always injected into the deny list for every adapter invocation
- Docker containers always run with --read-only, --cap-drop=ALL, --security-opt=no-new-privileges, --network=none
- Docker containers get tmpfs mounts with nosuid,nodev for /tmp, /var/run, /home/wave
- Artifact directories mounted read-only in Docker; output and workspace directories read-write
- Docker UID/GID mapping defaults to host user — prevents root execution inside container
- All logged output scrubbed for credential patterns before writing to disk — 8 patterns: API_KEY, TOKEN, SECRET, PASSWORD, CREDENTIAL, AUTH, PRIVATE_KEY, ACCESS_KEY
- Credential scrubbing applies to ALL log methods — tool calls, file ops, step start/end, error messages
- DebugTracer writes are thread-safe — sync.Mutex protects file handle, verified with 50-goroutine concurrency test
- Adapter subprocesses receive curated environment, not full host environment — only HOME, PATH, TERM, TMPDIR=/tmp plus whitelisted vars
- Claude Code telemetry suppressed — DISABLE_TELEMETRY, DISABLE_ERROR_REPORTING, CLAUDE_CODE_DISABLE_FEEDBACK_SURVEY, DISABLE_BUG_COMMAND
- Claude Code always runs with --dangerously-skip-permissions — Wave enforces permissions via agent frontmatter instead
- All embedded personas, pipelines, and prompts scanned for unsafe CLI interpolation patterns — prevents shell injection via double-quoted variable expansion
- MaxPathLength, MaxInputLength, and ContentSizeLimit must all be positive — non-positive values produce non-retryable SecurityValidationError
- AllowSymlinks defaults to false, AllowUnknownPersonas defaults to false, ValidatePersonaReferences defaults to true
- Risk score capped at 100 — scores >=50 considered high risk
- Security errors are structured with retryability metadata — path traversal and config errors non-retryable; injection and input errors retryable

## Key Decisions

- Three-layer permission enforcement: agent frontmatter (tool availability) + runtime CLAUDE.md (behavioral guidance) + sandbox (hard OS isolation) — defense-in-depth even if LLM ignores instructions
- Deny-first permission model — safest default, deny match always blocks regardless of allow patterns
- Sandbox architecture split: bubblewrap handled by Nix dev shell (external, returns NoneSandbox passthrough), Docker is in-process implementation with full hardening
- Credential scrubbing via compiled regex — trades precision for simplicity and performance, 8 keyword patterns compiled once and reused
- Risk scoring is additive with cap at 100 — base 20 for any sanitization, +50 injection, +30 suspicious, +15 shell metacharacters, +10 length truncation, +5 per credential keyword
- Schema content cached process-lifetime — sync.Map keyed by absolute path, no TTL needed since schemas don't change during pipeline run
- Fresh memory at every step boundary — --no-session-persistence flag, no chat history inheritance
- Paths in display/logs are sanitized — >50 chars replaced with <path:N chars>, .. replaced with [..]

## Domain Vocabulary

| Term | Meaning |
|------|--------|
| SecurityConfig | Top-level config aggregating path validation, sanitization, and persona validation settings |
| PathValidator | Validates file paths against directory allowlists, symlink policies, traversal patterns, and Unicode attacks |
| InputSanitizer | Sanitizes user input for prompt injection, length limits, and suspicious content with risk scoring |
| SecurityValidationError | Structured error with type, retryability, details, and suggested fix |
| SecurityViolationEvent | Immutable event record for detected security violations with severity and source |
| InputSanitizationRecord | Tracks sanitization actions with SHA-256 hash and risk score |
| PermissionChecker | Validates tool operations against allow/deny glob patterns — deny-first evaluation |
| PermissionError | Error carrying persona name, tool, argument, and denial reason |
| Sandbox | Interface with Wrap, Validate, Cleanup methods for OS-level isolation |
| DockerSandbox | Docker container sandbox with --read-only, --cap-drop=ALL, --network=none, --security-opt=no-new-privileges |
| NoneSandbox | Passthrough no-op sandbox — used when bubblewrap is handled externally by Nix |
| TraceLogger | NDJSON audit logger with credential scrubbing for all logged output |
| DebugTracer | NDJSON debug tracer with credential scrubbing, optional stderr mirror, and thread-safe writes |
| TraceEvent | Single structured trace event with timestamp, type, step ID, duration, metadata |
| Permissions | Struct with AllowedTools and Deny lists — the manifest-level permission definition |
| PersonaSandbox | Per-persona sandbox config with AllowedDomains network whitelist |
| RuntimeSandbox | Global sandbox config: backend, image, default domains, env passthrough |
| SandboxOnlySettings | Minimal settings.json written only when sandbox is enabled |
| Severity | Security event severity: LOW, MEDIUM, HIGH, CRITICAL |
| ViolationType | Security violation category: path_traversal, prompt_injection, invalid_persona, malformed_json, input_validation |
| PersonaSpec | Decoupled subset of persona config for agent markdown compilation — model, allowed tools, deny tools |

## Neighboring Contexts

- **execution**
- **configuration**
- **validation**

## Key Files

- `internal/adapter/permissions.go`
- `internal/security/path.go`
- `internal/security/sanitize.go`
- `internal/adapter/claude.go`
- `internal/security/config.go`
- `internal/audit/logger.go`
- `internal/audit/trace.go`
- `internal/sandbox/docker.go`
- `internal/sandbox/factory.go`
- `internal/adapter/environment.go`
- `internal/security/events.go`
- `internal/security/errors.go`

