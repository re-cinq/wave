# Research: Critical Security Vulnerability Fixes

**Phase 0 Research** | **Date**: 2026-02-02
**Feature**: Critical security fixes for Wave pipeline system

## Path Traversal Prevention

**Decision**: Implement allowlist-based path validation with `filepath.Clean()` and base directory restrictions

**Rationale**: Path traversal attacks can expose sensitive files. Go's `filepath.Clean()` normalizes paths, and allowlist validation ensures only approved directories are accessible for schema files.

**Alternatives considered**:
- Blacklist approach (rejected: too easy to bypass)
- Regular expression validation (rejected: complex and error-prone)
- Chroot jail (rejected: overkill for this use case)

## Input Sanitization Strategy

**Decision**: Multi-layer sanitization with length limits, content filtering, and prompt injection detection

**Rationale**: Prompt injection requires detecting and neutralizing attempts to override AI instructions while preserving legitimate user input functionality.

**Alternatives considered**:
- Complete input blocking (rejected: breaks functionality)
- Regex-only filtering (rejected: insufficient for sophisticated attacks)
- Full natural language processing (rejected: too complex/slow)

## JSON Comment Handling

**Decision**: Pre-processing step to strip comments before JSON parsing with error recovery

**Rationale**: Standard JSON parsers reject comments. Pre-processing allows graceful handling while maintaining strict JSON compliance for contract validation.

**Alternatives considered**:
- Custom JSON parser with comment support (rejected: security risk)
- Replacing JSON with YAML (rejected: breaking change)
- Error-only approach (rejected: breaks existing functionality)

## Persona Validation for Meta-Pipelines

**Decision**: Inject available persona list into philosopher prompt and validate generated YAML against manifest

**Rationale**: Two-layer validation: AI is constrained by available personas, and generated pipeline is validated before execution.

**Alternatives considered**:
- Runtime persona creation (rejected: security risk)
- Soft validation with warnings (rejected: pipelines still fail)
- Manual persona mapping (rejected: not scalable)

## Security Event Logging

**Decision**: Structured logging with sanitized details using existing Wave logging infrastructure

**Rationale**: Security events need auditing without exposing sensitive data. Leverage existing logging to avoid architectural changes.

**Alternatives considered**:
- Separate security log files (rejected: adds complexity)
- No logging (rejected: security compliance requirement)
- Full request/response logging (rejected: privacy concerns)

## Performance Impact Mitigation

**Decision**: Optimize sanitization with compiled regexes and caching, benchmark to ensure <5ms overhead

**Rationale**: Security cannot significantly impact pipeline performance. Compiled patterns and caching reduce repeated computation overhead.

**Alternatives considered**:
- No performance constraints (rejected: user experience impact)
- Async processing (rejected: adds complexity without benefit)
- Optional security (rejected: defeats the purpose)

## Backward Compatibility Strategy

**Decision**: Default-secure configuration with graceful degradation for existing pipelines

**Rationale**: Security fixes must not break existing functionality. New validation is strict but provides clear error messages for migration.

**Alternatives considered**:
- Breaking changes with migration tool (rejected: too disruptive)
- Opt-in security (rejected: leaves vulnerabilities exposed)
- Version-dependent behavior (rejected: confusing for users)