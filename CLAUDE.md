# Wave Development Guidelines

**Wave** is a multi-agent pipeline orchestrator written in Go that wraps Claude Code and other LLM CLIs via subprocess execution. It composes personas, pipelines, contracts, and relay/compaction into a continuous development system.

## Critical Constraints

1. **Single static binary** — no runtime dependencies except adapter binaries
2. **Test ownership** — every failing test is YOUR concern. Fix or delete (with justification), never ignore. Changes to personas, pipelines, contracts, or meta-pipelines require `go test ./...`
3. **Security first** — all inputs validated, paths sanitized, permissions enforced
4. **Constitutional compliance** — navigator-first architecture, fresh memory at step boundaries, contract validation at handovers, ephemeral workspace isolation, observable progress events
5. **Observable execution** — structured progress events for monitoring
6. **No backward compatibility constraint** during prototype phase — move fast, let tests catch regressions
7. **No `t.Skip()`** without a linked issue. Delete tests only with clear justification

## How Wave Works at Runtime

Each pipeline is a **topologically-sorted DAG** of steps. For every step:

1. **Workspace creation** — an ephemeral worktree is created under `.wave/workspaces/<pipeline>/<step>/`. Steps can share workspaces via `workspace.ref`. Mounts support readonly/readwrite modes
2. **Artifact injection** — outputs from prior steps are injected into `.wave/artifacts/` before execution begins. The system validates existence, enforces optional/required semantics, and checks schemas if `ref.SchemaPath` is specified
3. **Runtime CLAUDE.md assembly** — a per-step CLAUDE.md is generated from four layers:
   - Base protocol preamble (`.wave/personas/base-protocol.md`)
   - Persona system prompt (role, responsibilities, constraints)
   - Contract compliance section (auto-generated from step contract schema)
   - Restriction section (denied/allowed tools, network domains from manifest permissions)
4. **Adapter execution** — the persona runs in isolated context with fresh memory (no chat history inheritance)
5. **Contract validation** — step output is validated against its contract (json_schema, test_suite, typescript, quality_gate) **before** marking the step successful. Hard failures block; soft failures log warnings

Key source files: `internal/pipeline/executor.go`, `internal/adapter/claude.go`, `internal/contract/`, `internal/workspace/`

## Architecture

### Active Technologies
- Go 1.25+ with `gopkg.in/yaml.v3`, `github.com/spf13/cobra`
- SQLite for pipeline state, filesystem for workspaces and artifacts

### Core Components
- **Manifests** (`wave.yaml`) — single source of truth for configuration
- **Personas** — AI agents with specific roles, permissions, and system prompts
- **Pipelines** — multi-step workflows with dependency resolution
- **Contracts** — output validation (JSON schema, TypeScript, test suites)
- **Workspaces** — ephemeral isolated execution environments
- **State Management** — SQLite-backed persistence and resumption

### Security Model
- Fresh memory at every step boundary — no chat history inheritance
- Permission enforcement with deny/allow patterns — strictly enforced
- Ephemeral workspaces — isolated filesystem execution
- Contract validation — all outputs validated before step completion
- Audit logging — credential scrubbing and tool call tracking

## File Structure
```
internal/
├── adapter/      # Subprocess execution and adapter management
├── audit/        # Audit logging and credential scrubbing
├── contract/     # Output validation (JSON, TypeScript, test suites)
├── deliverable/  # Pipeline deliverable tracking and output
├── display/      # Terminal progress display and formatting
├── event/        # Progress event emission and monitoring
├── github/       # GitHub API integration for issue enhancement
├── manifest/     # Configuration loading and validation
├── pipeline/     # Pipeline execution and step management
├── preflight/    # Pipeline dependency validation and auto-install
├── relay/        # Context compaction and summarization
├── security/     # Security validation and sanitization
├── skill/        # Skill discovery, provisioning, and command management
├── state/        # SQLite persistence and state management
├── worktree/     # Git worktree lifecycle for isolated workspaces
└── workspace/    # Ephemeral workspace management

cmd/wave/         # CLI command structure
tests/            # Test coverage
.wave/            # Default personas, pipelines, contracts
```

## Security

- All user input sanitized for prompt injection; file paths validated against approved directories
- Persona permissions strictly enforced at runtime; deny rules projected into `settings.json` AND runtime `CLAUDE.md`
- **Outer sandbox**: Nix dev shell with bubblewrap (read-only FS, hidden `$HOME`, curated env)
- **Adapter sandbox**: `settings.json` sandbox settings with network domain allowlisting
- **Prompt restrictions**: runtime `CLAUDE.md` restriction section generated from manifest
- **Environment hygiene**: only `runtime.sandbox.env_passthrough` vars reach subprocesses
- No credentials on disk; sanitized logging; workspace isolation prevents data leakage

## Development

### Code Standards
- Follow effective Go practices (`gofmt`, `go vet`), single responsibility per package
- Use interfaces for testability and dependency injection
- Comprehensive error types with structured details
- Table-driven tests with edge case coverage
- **Find & replace**: prefer `perl -pi -e` over `sed`/`awk` for in-place substitutions — `sed` and `awk` are unreliable with escaping, multiline, and cross-platform differences (macOS vs Linux)

### Testing
```bash
go test ./...            # Run all tests
go test -race ./...      # Run with race detector (required for PR)
```

See `docs/migrations.md` for database migration documentation.

## Git Commits

- **No Co-Authored-By** — never include Co-Authored-By lines in commit messages
- **No AI attribution** — do not add "Generated with Claude Code" or similar
- Use conventional commit prefixes: `feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:`

## Versioning

Automated semantic versioning from conventional commits. Every merge to `main` produces a release.

| Commit prefix | Bump | Example |
|---------------|------|---------|
| `fix:`, `docs:`, `refactor:`, `test:`, `chore:` | **patch** (0.0.X) | v0.1.0 → v0.1.1 |
| `feat:` | **minor** (0.X.0) | v0.1.1 → v0.2.0 |
| `BREAKING CHANGE:` or `!:` (e.g. `feat!:`) | **major** (X.0.0) | v0.2.0 → v1.0.0 |

## Debugging
- Use `--debug` flag for detailed execution logging
- Check `.wave/traces/` for audit logs
- Workspace contents preserved for post-mortem analysis

## Recent Changes
- 086-pipeline-recovery-hints: Added Go 1.25+ + `github.com/spf13/cobra` (CLI), `gopkg.in/yaml.v3` (config) — no new dependencies
- 085-web-operations-dashboard: Added Go 1.25+ (existing project) + `net/http` stdlib (Go 1.22+ enhanced `ServeMux`), `html/template`, `go:embed`, `modernc.org/sqlite` (existing)
- 029-release-gated-embedding: Added Go 1.25+ + `gopkg.in/yaml.v3`, `github.com/spf13/cobra`
- 021-add-missing-personas: Added implementer and reviewer personas, updated persona prompts to decouple schema details per issue #24

<!-- MANUAL ADDITIONS START -->

1. NEVER write contract or artifact schemas in prompts. Wave has to parse, validate and inject them properly into the proper pipeline step. **Exception**: `gh pr create --body-file .wave/artifacts/<name>` and similar CLI commands that require a literal file path are acceptable — the persona needs the path to pass to external tools.
2. NEVER pass validations silently. If a validation fails, it must be reported as an error and the step should not complete successfully.
3. NEVER make bulk-edits to the codebase except it is not functional code; BEWARE personas, pipelines etc. are all functional code and should be edited with the same care as any other code: Individually!

<!-- MANUAL ADDITIONS END -->
