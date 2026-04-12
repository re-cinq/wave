# Implementation Plan: Wave Task Classifier

**Branch**: `772-task-classifier` | **Date**: 2026-04-12 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/772-task-classifier/spec.md`

## Summary

Create `internal/classify` package with three files: `profile.go` (types), `analyzer.go` (keyword-based input classification), `selector.go` (profile-to-pipeline routing). The package reuses `suggest.ClassifyInput` for URL/ref detection, adds keyword analysis for domain/complexity, and maps profiles to pipeline names per AGENTS.md routing table. No new external dependencies.

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: `github.com/recinq/wave/internal/suggest` (InputType reuse)
**Storage**: N/A (pure in-memory classification)
**Testing**: `go test` with table-driven tests, same-package test files
**Target Platform**: Linux (Wave CLI binary)
**Project Type**: Single Go binary — new internal package
**Performance Goals**: <1ms per classification (SC-006)
**Constraints**: No external dependencies (SC-005), no disk I/O, no network calls
**Scale/Scope**: 3 source files + 3 test files, ~400-500 lines total

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

| Principle | Status | Notes |
|-----------|--------|-------|
| P1: Single Binary | PASS | No new external deps, pure Go |
| P2: Manifest as SSOT | N/A | No manifest changes needed |
| P3: Persona-Scoped Execution | N/A | Library code, not persona |
| P4: Fresh Memory | N/A | No step boundary changes |
| P5: Navigator-First | N/A | Library code |
| P6: Contracts at Handover | N/A | No pipeline step changes |
| P7: Relay via Summarizer | N/A | No relay changes |
| P8: Ephemeral Workspaces | N/A | No workspace changes |
| P9: Credentials Never Touch Disk | N/A | No credential handling |
| P10: Observable Progress | N/A | No event changes |
| P11: Bounded Recursion | N/A | No recursion changes |
| P12: Minimal Step State Machine | N/A | No state changes |
| P13: Test Ownership | PASS | All 3 source files get corresponding _test.go files with 10+ table-driven cases each |

**Result**: No violations. All applicable principles satisfied.

## Project Structure

### Documentation (this feature)

```
specs/772-task-classifier/
├── plan.md              # This file
├── spec.md              # Feature specification
├── research.md          # Phase 0 research output
├── data-model.md        # Phase 1 data model
├── contracts/           # Phase 1 API contracts
│   └── classify.go      # API surface documentation
├── checklists/          # Requirement checklists
│   └── requirements.md
└── tasks.md             # Phase 2 output (NOT created by plan)
```

### Source Code (repository root)

```
internal/classify/
├── profile.go           # TaskProfile, PipelineConfig, enums
├── profile_test.go      # Profile validation tests
├── analyzer.go          # Classify() function
├── analyzer_test.go     # Analyzer tests (10+ table-driven cases)
├── selector.go          # SelectPipeline() function
└── selector_test.go     # Selector tests (10+ table-driven cases)
```

**Structure Decision**: Single new package under `internal/` following established patterns (`internal/suggest/`, `internal/pipeline/`). No sub-packages needed — three files with clear single responsibilities.

## Implementation Components

### Component 1: profile.go — Type Definitions

**FR coverage**: FR-001, FR-008

Define all exported types:
- `Complexity` string enum with 4 constants
- `Domain` string enum with 7 constants
- `VerificationDepth` string enum with 3 constants
- `TaskProfile` struct with 5 fields (BlastRadius, Complexity, Domain, VerificationDepth, InputType)
- `PipelineConfig` struct with 2 fields (Name, Reason)

**Dependencies**: `internal/suggest` (for `suggest.InputType`)

### Component 2: analyzer.go — Input Classification

**FR coverage**: FR-002, FR-003, FR-004, FR-009, FR-010, FR-011, FR-012

Implement `Classify(input string, issueBody string) TaskProfile`:

1. Call `suggest.ClassifyInput(input)` to get InputType
2. Combine `input` + `issueBody` into analysis text (lowercased)
3. Match domain keywords with priority ordering:
   - security: "vulnerability", "cve", "injection", "xss", "csrf", "auth bypass", "security"
   - performance: "slow", "latency", "performance", "optimize", "benchmark", "memory leak"
   - bug: "bug", "fix", "broken", "crash", "error", "null pointer", "panic", "doesn't work"
   - refactor: "refactor", "restructure", "reorganize", "clean up", "technical debt"
   - research: "research", "investigate", "analyze", "compare", "evaluate", "explore"
   - docs: "documentation", "readme", "typo", "comment", "docs", "docstring"
   - feature (default): "add", "implement", "create", "new", "feature", "support"
4. Match complexity keywords:
   - architectural: "architecture", "redesign", "across.*packages", "system-wide", "entire"
   - complex: "multiple", "several", "complex", "integration", "across"
   - simple: "typo", "rename", "single", "minor", "small", "trivial"
   - medium (default)
5. Derive blast_radius: base(complexity) + modifier(domain), clamped [0.0, 1.0]
6. Derive verification_depth from complexity

**Edge case handling**:
- Empty/whitespace → default profile (simple, feature, 0.1, structural_only)
- No keyword matches → medium, feature, 0.3, behavioral
- Mixed domains → highest priority wins per FR-011

### Component 3: selector.go — Pipeline Selection

**FR coverage**: FR-005, FR-006

Implement `SelectPipeline(profile TaskProfile) PipelineConfig`:

Sequential priority evaluation:
1. `InputType == pr_url` → `ops-pr-review`
2. `Domain == security` → `audit-security`
3. `Domain == research` → `impl-research`
4. `Domain == docs` → `doc-fix`
5. `Domain == refactor` AND `Complexity ∈ {complex, architectural}` → `impl-speckit`
6. `Complexity ∈ {simple, medium}` → `impl-issue`
7. `Complexity ∈ {complex, architectural}` → `impl-speckit`

Each case includes a descriptive `Reason` string.

### Component 4: Tests

**FR coverage**: FR-007, SC-001 through SC-004

**profile_test.go**: Validate enum constants are correct strings, TaskProfile zero-value behavior, PipelineConfig field access. 10+ cases.

**analyzer_test.go**: Table-driven tests covering:
- All 7 domain classifications (security, performance, bug, refactor, feature, docs, research)
- All 4 complexity levels
- Empty/whitespace input defaults
- Mixed domain signal priority
- URL + issue body combination
- PR URL input type detection
- No-keyword fallback
- 10+ cases minimum

**selector_test.go**: Table-driven tests covering:
- All 7 routing rules from FR-006
- PR URL short-circuit
- Security override (any complexity)
- Domain-specific routes (research, docs)
- Architectural refactor → impl-speckit
- Simple/medium → impl-issue
- Complex/architectural → impl-speckit
- Reason field non-empty
- 10+ cases minimum

## Implementation Order

1. `profile.go` — types first (no logic, no dependencies except suggest)
2. `profile_test.go` — validate types compile and constants are correct
3. `selector.go` — pipeline routing (depends only on types)
4. `selector_test.go` — verify routing table
5. `analyzer.go` — input classification (depends on types + suggest)
6. `analyzer_test.go` — verify classification accuracy

Rationale: Types → selector → analyzer. Selector is simpler (pure mapping) so implement before analyzer (keyword matching). Tests interleaved with implementation for immediate validation.

## Complexity Tracking

No constitution violations. No complexity justifications needed.
