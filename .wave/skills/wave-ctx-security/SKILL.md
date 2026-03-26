---
name: wave-ctx-security
description: Domain context for Wave's permission enforcement and input sanitization bounded context
---

# Security Context

Permission enforcement, input sanitization, path validation, sandbox configuration, audit logging, and credential scrubbing.

## Invariants

- All user input is sanitized for prompt injection before being used in prompts, file paths, or shell commands
- Deny rules are projected into BOTH adapter settings.json AND the runtime CLAUDE.md restriction section -- dual enforcement prevents bypass
- No credentials are written to disk; audit logs scrub sensitive values using pattern-based and entropy-based detection
- Workspace isolation prevents data leakage between steps and between pipeline runs
- Path validation rejects traversal attempts (`../`), symlink escapes, and paths outside the approved directory set
- Persona permissions use a glob-pattern allow/deny model; `deny` rules are security boundaries (hard block), `allow` rules are convenience (auto-approve prompts)
- Sandbox configuration supports two backends: bubblewrap (`bwrap`) for Linux and Docker for cross-platform; both enforce read-only filesystem, hidden $HOME, and curated environment variables
- Only variables listed in `runtime.sandbox.env_passthrough` reach adapter subprocesses; all others are stripped
- Network access is restricted to `runtime.sandbox.default_allowed_domains` plus persona-specific `sandbox.allowed_domains`; the effective allowlist is the union

## Key Decisions

- `SecurityConfig` is a single struct that bundles `PathValidator`, `InputSanitizer`, and `SecurityLogger` -- instantiated once per pipeline run and threaded through the executor
- The sanitizer operates in multiple passes: first HTML entity decoding, then pattern matching for known injection vectors, then structural validation
- Audit events are emitted as structured JSON with tool name, arguments (scrubbed), timestamp, and step context -- consumed by the trace writer
- Credential detection uses both pattern matching (API key prefixes, token formats) and Shannon entropy thresholds for high-entropy strings
- Bubblewrap sandbox uses `--ro-bind` for source mounts and `--bind` only for the workspace directory; `/proc`, `/dev/null`, `/dev/urandom` are selectively mounted
- Docker sandbox generates ephemeral containers with volume mounts matching the bubblewrap bind topology

## Domain Vocabulary

| Term | Meaning |
|------|---------|
| PathValidator | Validates file paths against an approved directory set; rejects traversal and symlink escapes |
| InputSanitizer | Scrubs user-provided strings for prompt injection patterns before use in prompts or commands |
| SecurityLogger | Audit logger that scrubs credentials from log output using pattern and entropy detection |
| Sandbox | Process isolation layer (bubblewrap or Docker) that constrains adapter subprocess filesystem and network access |
| Deny rule | A glob pattern in persona permissions that hard-blocks a tool; projected into both settings.json and CLAUDE.md |
| Allow rule | A glob pattern that auto-approves tool usage without interactive prompts; does NOT grant access to denied tools |
| Credential scrubbing | Multi-pass detection of secrets in log output: known prefixes (sk-, ghp_, etc.), regex patterns, and entropy scoring |
| Env passthrough | Explicit allowlist of environment variables forwarded to adapter subprocesses; all others are stripped |
| Domain allowlist | Network domains the adapter subprocess may contact; union of runtime defaults and persona overrides |

## Neighboring Contexts

- **Execution** (`internal/pipeline/`) -- the executor instantiates `SecurityConfig` and passes it through step execution; permission enforcement happens at adapter launch time
- **Configuration** (`internal/manifest/`) -- persona permissions, sandbox config, and allowed domains originate from `wave.yaml` manifest definitions
- **Validation** (`internal/contract/`) -- schema file paths are validated by `PathValidator` before loading

## Key Files

- `internal/security/sanitizer.go` -- `InputSanitizer`, multi-pass sanitization, injection pattern detection
- `internal/security/path_validator.go` -- `PathValidator`, directory allowlist, traversal detection, symlink resolution
- `internal/security/logger.go` -- `SecurityLogger`, credential scrubbing, entropy-based secret detection
- `internal/security/config.go` -- `SecurityConfig` struct bundling validator, sanitizer, and logger
- `internal/sandbox/bubblewrap.go` -- bubblewrap sandbox backend, bind mount configuration, process group isolation
- `internal/sandbox/docker.go` -- Docker sandbox backend, container lifecycle, volume mount generation
- `internal/sandbox/factory.go` -- sandbox backend selection based on platform and configuration
- `internal/audit/audit.go` -- audit event types, trace file writer, credential pattern registry
- `internal/adapter/claude.go` -- settings.json generation with deny rule projection and domain allowlisting
