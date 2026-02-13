# Wave Development Guidelines

You are working on **Wave** - a multi-agent pipeline orchestrator written in Go that wraps Claude Code and other LLM CLIs via subprocess execution.

## Project Overview

Wave composes personas, pipelines, contracts, and relay/compaction into a continuous development system. It executes multi-step workflows where each step is performed by a specialized AI persona with specific permissions and tools.

## Architecture Principles

### Active Technologies
- Go 1.25+ + gopkg.in/yaml.v3, github.com/spf13/cobra (existing Wave dependencies)
- SQLite for pipeline state, filesystem for workspaces and artifacts

### Core Components
- **Manifests** (`wave.yaml`) - Single source of truth for configuration
- **Personas** - AI agents with specific roles, permissions, and system prompts
- **Pipelines** - Multi-step workflows with dependency resolution
- **Contracts** - Output validation (JSON schema, TypeScript, test suites)
- **Workspaces** - Ephemeral isolated execution environments
- **State Management** - SQLite-backed persistence and resumption

### Security Model
- **Fresh memory** at every step boundary - no chat history inheritance
- **Permission enforcement** with deny/allow patterns - strictly enforced
- **Ephemeral workspaces** - isolated filesystem execution
- **Contract validation** - all outputs validated before step completion
- **Audit logging** - credential scrubbing and tool call tracking

## Development Guidelines

### Code Standards
- **Go conventions** - Follow effective Go practices and formatting
- **Single responsibility** - Each package has a clear, focused purpose
- **Interface design** - Use interfaces for testability and flexibility
- **Error handling** - Comprehensive error types with structured details
- **Testing** - Table-driven tests with comprehensive edge case coverage

### Critical Constraints
1. **Single static binary** - No runtime dependencies except adapter binaries
2. **Constitutional compliance** - All changes must align with Wave constitution
3. **Test ownership** - Every failing test is the concern of the worker who caused it. Fix or delete (with justification), never ignore. Changes to personas, pipelines, contracts, or meta-pipelines require running the full test suite.
4. **Security first** - All inputs validated, paths sanitized, permissions enforced
5. **Observable execution** - Structured progress events for monitoring

**Note**: Backward compatibility is NOT a constraint during prototype phase. We move fast and let tests catch regressions.

### File Structure
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
tests/            # Comprehensive test coverage
.wave/            # Default personas, pipelines, contracts
```

### Key Implementation Patterns

#### Pipeline Execution
- Each step runs in isolated workspace with persona-specific permissions
- Fresh context at every boundary (no memory inheritance)
- Artifact injection for inter-step communication
- Contract validation before step completion

#### Security Validation
- Path traversal prevention with allowlisted directories
- Input sanitization for prompt injection prevention
- Schema content validation before AI processing
- Security event logging for audit trails

#### Error Handling
- Structured error types with detailed context
- Retry mechanisms based on error type and configuration
- Graceful degradation when possible
- Clear, actionable error messages

### Testing Requirements
- **Unit tests** for all public interfaces
- **Integration tests** for pipeline execution flows
- **Security tests** for validation and sanitization
- **Race condition testing** with `-race` flag
- **Performance tests** for critical paths

### Test Ownership (Prototype Discipline)

**Hypothesis**: Non-deterministic systems can act predictably with the right guardrails.

Tests ARE those guardrails. When you change core primitives (personas, pipelines, contracts, meta-pipelines):

1. Run `go test ./...` before committing
2. If tests fail, YOU own fixing them — not "someone later"
3. Delete tests only with clear justification (outdated, wrong assumption)
4. No `t.Skip()` without a linked issue

This lets us move fast while maintaining confidence that Wave actually works.

### Constitutional Compliance
All development must comply with the Wave Constitution:
- Navigator-first architecture
- Fresh memory at step boundaries
- Contract validation at handovers
- Ephemeral workspace isolation
- Single binary deployment
- Observable progress events

## Security Considerations

### Input Validation
- All user input sanitized for prompt injection
- File paths validated against approved directories
- Schema content cleaned before AI processing
- Length limits enforced on all inputs

### Permission Enforcement
- Persona permissions strictly enforced at runtime
- Deny rules projected into `settings.json` AND `CLAUDE.md` restriction section
- No escalation or bypass mechanisms
- Audit trail for all permission decisions
- Fail-secure on permission violations

### Sandbox Isolation
- **Outer sandbox**: Nix dev shell with bubblewrap (read-only FS, hidden `$HOME`, curated env)
- **Adapter sandbox**: `settings.json` sandbox settings with network domain allowlisting
- **Prompt restrictions**: `CLAUDE.md` restriction section generated from manifest
- **Environment hygiene**: Only `runtime.sandbox.env_passthrough` vars reach subprocesses

### Data Protection
- No credentials stored on disk
- Curated environment passthrough (not full `os.Environ()`)
- Sanitized logging (no sensitive data)
- Workspace isolation prevents data leakage

## Common Tasks

### Adding New Commands
1. Create command in `cmd/wave/commands/`
2. Register in main command structure
3. Add comprehensive help text and examples
4. Implement with proper error handling
5. Add unit tests for all code paths

### Adding New Contract Types
1. Implement validator interface in `internal/contract/`
2. Add to validator registry
3. Update configuration types
4. Add comprehensive test coverage
5. Document in user guides

### Adding Security Features
1. Implement in `internal/security/` package
2. Integrate with existing validation flows
3. Add security event logging
4. Comprehensive attack vector testing
5. Update security documentation

## Performance Considerations
- Pipeline execution should complete steps in reasonable time
- State queries must be fast (< 100ms for status checks)
- Memory usage should remain bounded during execution
- Concurrent pipeline support without resource contention

## Database Migrations

Wave uses a comprehensive migration system for schema management:

### Adding New Migrations
1. Add migration definition in `internal/state/migration_definitions.go`
2. Include both `Up` (forward) and `Down` (rollback) SQL
3. Write comprehensive tests in `*_test.go` files
4. Test rollback functionality thoroughly
5. Update documentation for user-facing changes

### Environment Configuration
- `WAVE_MIGRATION_ENABLED=true` - Enable migration system (default: true)
- `WAVE_AUTO_MIGRATE=true` - Auto-apply on startup (default: true)
- `WAVE_MAX_MIGRATION_VERSION=N` - Limit migrations for gradual rollout
- `WAVE_SKIP_MIGRATION_VALIDATION=true` - Skip checksums (dev only)

### CLI Commands
```bash
# Check migration status
wave migrate status

# Apply pending migrations
wave migrate up

# Rollback to specific version (with confirmation)
wave migrate down 3

# Validate migration integrity
wave migrate validate
```

See `docs/migrations.md` for complete migration documentation.

## Testing

```bash
# Run all tests
go test ./...

# Run with race detector (required for PR)
go test -race ./...

# Run specific package
go test ./internal/pipeline/...

# Test migration system specifically
go test ./internal/state -v -run Migration

# Run with verbose output
go test -v ./...

# Run with coverage
go test -cover ./...
```

## Code Style

Follow standard Go conventions:
- Use `gofmt` for formatting
- Run `go vet` for static analysis
- Keep functions focused and testable
- Use interfaces for dependency injection

## Git Commits

- **No Co-Authored-By** - Never include Co-Authored-By lines in commit messages
- **No AI attribution** - Do not add "Generated with Claude Code" or similar attribution
- Keep commit messages concise and focused on the change
- Use conventional commit prefixes: `feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:`

## Versioning

Wave uses **automated semantic versioning** derived from conventional commit messages.

### How It Works

On every push to `main`, the CI analyzes commit messages since the last tag and determines the version bump:

| Commit prefix | Bump | Example |
|---------------|------|---------|
| `fix:`, `docs:`, `refactor:`, `test:`, `chore:` | **patch** (0.0.X) | v0.1.0 → v0.1.1 |
| `feat:` | **minor** (0.X.0) | v0.1.1 → v0.2.0 |
| `BREAKING CHANGE:` or `!:` (e.g. `feat!:`) | **major** (X.0.0) | v0.2.0 → v1.0.0 |

The highest bump type wins when multiple commits are present. The CI then creates and pushes the tag, which triggers GoReleaser to build binaries and create a GitHub Release.

### Rules

- Every merge to `main` produces a new release automatically
- Commit prefixes determine bump level — choose them intentionally
- Use `feat!:` or include `BREAKING CHANGE:` in the commit body for major bumps
- Version starts at `v0.1.0` (first tag created by CI)

## Debugging
- Use `--debug` flag for detailed execution logging
- Check `.wave/traces/` for audit logs
- Workspace contents preserved for post-mortem analysis
- Structured events for programmatic monitoring

## Recent Changes
- 085-web-operations-dashboard: Added Go 1.25+ (existing project) + `net/http` stdlib (Go 1.22+ enhanced `ServeMux`), `html/template`, `go:embed`, `modernc.org/sqlite` (existing)
- 029-release-gated-embedding: Added Go 1.25+ + `gopkg.in/yaml.v3`, `github.com/spf13/cobra`
- 021-add-missing-personas: Added implementer and reviewer personas, updated persona prompts to decouple schema details per issue #24

<!-- MANUAL ADDITIONS START -->
<!-- MANUAL ADDITIONS END -->
