# Wave Development Guidelines

You are working on **Wave** - a multi-agent pipeline orchestrator written in Go that wraps Claude Code and other LLM CLIs via subprocess execution.

## Project Overview

Wave composes personas, pipelines, contracts, and relay/compaction into a continuous development system. It executes multi-step workflows where each step is performed by a specialized AI persona with specific permissions and tools.

## Architecture Principles

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
3. **Backward compatibility** - Never break existing manifests or APIs
4. **Security first** - All inputs validated, paths sanitized, permissions enforced
5. **Observable execution** - Structured progress events for monitoring

### File Structure
```
internal/
├── adapter/      # Subprocess execution and adapter management
├── audit/        # Audit logging and credential scrubbing
├── contract/     # Output validation (JSON, TypeScript, test suites)
├── event/        # Progress event emission and monitoring
├── manifest/     # Configuration loading and validation
├── pipeline/     # Pipeline execution and step management
├── relay/        # Context compaction and summarization
├── security/     # Security validation and sanitization
├── state/        # SQLite persistence and state management
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
- No escalation or bypass mechanisms
- Audit trail for all permission decisions
- Fail-secure on permission violations

### Data Protection
- No credentials stored on disk
- Environment-only secret passing
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

## Debugging
- Use `--debug` flag for detailed execution logging
- Check `.wave/traces/` for audit logs
- Workspace contents preserved for post-mortem analysis
- Structured events for programmatic monitoring