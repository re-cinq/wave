# Implementation Plan: Critical Security Vulnerability Fixes

**Branch**: `017-fix-critical-security` | **Date**: 2026-02-02 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/017-fix-critical-security/spec.md`

## Summary

This plan addresses critical security vulnerabilities and system bugs in the Wave pipeline execution system: path traversal in schema injection, prompt injection via malicious schemas, unvalidated user input enabling AI manipulation, invalid persona references in meta-pipeline generation, and malformed JSON output breaking contract validation. The technical approach focuses on input validation, sanitization, and robust error handling while maintaining backward compatibility.

## Technical Context

**Language/Version**: Go 1.21+ (existing Wave codebase)
**Primary Dependencies**: jsonschema/v6, existing Wave internal packages, YAML parsing libraries
**Storage**: File system (schema files, logs), SQLite state database (existing)
**Testing**: Go testing framework, table-driven tests, security-focused test cases
**Target Platform**: Linux/macOS/Windows servers and development machines
**Project Type**: Security fixes to existing CLI tool and multi-agent pipeline system
**Performance Goals**: No degradation of existing pipeline performance, sanitization overhead <5ms per operation
**Constraints**: Must maintain backward compatibility, zero breaking changes to public APIs
**Scale/Scope**: Affects all Wave installations, 5 critical vulnerabilities, 12 functional requirements

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

✅ **Principle 1 (Single Binary)**: No new dependencies, only internal security fixes
✅ **Principle 2 (Manifest Truth)**: Persona validation uses existing manifest structure
✅ **Principle 3 (Persona-Scoped)**: Security fixes enhance persona permission enforcement
✅ **Principle 4 (Fresh Memory)**: No changes to step boundary memory model
✅ **Principle 5 (Navigator-First)**: No changes to navigation architecture
✅ **Principle 6 (Contracts)**: Fixes improve contract validation robustness
✅ **Principle 7 (Relay/Summarizer)**: No changes to relay mechanism
✅ **Principle 8 (Ephemeral Workspaces)**: Security improvements complement workspace isolation
✅ **Principle 9 (No Credentials on Disk)**: Security fixes help prevent credential exposure
✅ **Principle 10 (Observable Progress)**: Security events will be logged appropriately
✅ **Principle 11 (Bounded Recursion)**: Fixes improve meta-pipeline resource limits
✅ **Principle 12 (Step State Machine)**: No changes to state transitions

**No constitutional violations identified**

## Project Structure

### Documentation (this feature)

```
specs/017-fix-critical-security/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```
internal/
├── pipeline/
│   ├── executor.go       # Schema injection fixes, input sanitization
│   ├── meta.go           # Persona validation, JSON comment prevention
│   └── security.go       # New: Security validation utilities
├── contract/
│   ├── jsonschema.go     # JSON cleaning, must_pass handling fixes
│   └── validator.go     # Enhanced validation with sanitization
├── manifest/
│   ├── types.go          # Persona validation structures
│   └── loader.go         # Manifest persona enumeration
└── security/             # New package for security utilities
    ├── path.go           # Path traversal prevention
    ├── sanitize.go       # Input/content sanitization
    └── logging.go        # Security event logging

tests/
├── security/             # New: Security-specific test suite
│   ├── path_test.go      # Path traversal attack tests
│   ├── injection_test.go # Prompt injection tests
│   └── persona_test.go   # Meta-pipeline persona validation tests
├── pipeline/
│   ├── executor_test.go  # Enhanced with security test cases
│   └── meta_test.go      # JSON comment and persona tests
└── contract/
    └── jsonschema_test.go # Enhanced validation tests
```

**Structure Decision**: Single project structure with new `internal/security` package for reusable security utilities. Security fixes integrated into existing pipeline and contract packages to minimize architectural changes while centralizing security logic for easier maintenance.

## Complexity Tracking

_No constitutional violations requiring justification_