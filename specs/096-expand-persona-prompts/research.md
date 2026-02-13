# Research: Expand Persona Definitions with Detailed System Prompts

**Feature**: 096-expand-persona-prompts
**Date**: 2026-02-13

## Current State Analysis

### Expansion Status

Commit `6fdb3e9` already expanded all 13 persona files in `.wave/personas/`. The expanded versions range from 57-93 lines, all meeting the FR-009 minimum of 30 lines and the FR-013 maximum of 200 lines.

However, `internal/defaults/personas/` still contains the **original brief versions** (18-50 lines). All 13 files differ between the two directories — SC-005 (byte-identical parity) is completely unmet.

| File | `.wave/` Lines | `internal/defaults/` Lines | Parity |
|------|---------------|--------------------------|--------|
| navigator.md | 57 | 18 | FAIL |
| philosopher.md | 63 | 19 | FAIL |
| planner.md | 65 | 27 | FAIL |
| craftsman.md | 60 | 22 | FAIL |
| implementer.md | 58 | 23 | FAIL |
| reviewer.md | 70 | 29 | FAIL |
| auditor.md | 60 | 24 | FAIL |
| debugger.md | 72 | 35 | FAIL |
| researcher.md | 93 | 50 | FAIL |
| summarizer.md | 60 | 23 | FAIL |
| github-analyst.md | 75 | 32 | FAIL |
| github-commenter.md | 91 | 47 | FAIL |
| github-enhancer.md | 76 | 28 | FAIL |

### FR-008 Violations (Language-Specific References)

Four persona files in `.wave/personas/` contain language-specific references that must be generalized:

#### 1. craftsman.md — Line 12
- **Violation**: `Go conventions including effective Go practices, formatting, and idiomatic patterns`
- **Fix**: → `Language conventions and idiomatic patterns for the target codebase`
- **Violation**: Line 46 — `go test, go build, go vet, etc.` in Tools section
- **Fix**: → `build, test, and static analysis commands for the project's toolchain`

#### 2. reviewer.md — Line 35
- **Violation**: `Run available tests (\`go test\`, \`npm test\`) to verify passing state`
- **Fix**: → `Run the project's test suite to verify passing state`
- **Violation**: Lines 46-47 — `Bash(go test*)`, `Bash(npm test*)` in Tools section
- **Fix**: → `Bash(...)`: Run the project's test suite to validate implementation behavior

#### 3. auditor.md — Lines 2-3
- **Violation**: `specializing in Go systems and multi-agent pipeline architectures`
- **Fix**: → `specializing in software systems and multi-agent pipeline architectures`
- **Violation**: Line 16 — `Go-specific security concerns: unsafe pointer usage, race conditions, path traversal`
- **Fix**: → `Language-specific security concerns: memory safety, race conditions, path traversal, type confusion`
- **Violation**: Line 33 — `Run static analysis tools (\`go vet\`)`
- **Fix**: → `Run static analysis tools available in the project's toolchain`
- **Violation**: Line 43 — `Bash(go vet*)`, Line 44 — `Bash(npm audit*)`
- **Fix**: → Generalize to `Bash(...)` with language-agnostic description

#### 4. debugger.md — Lines 2-3
- **Violation**: `specializing in Go systems and multi-agent pipelines`
- **Fix**: → `specializing in software systems and multi-agent pipelines`
- **Violation**: Line 9 — `Root cause analysis and fault isolation in concurrent Go programs`
- **Fix**: → `Root cause analysis and fault isolation in concurrent programs`
- **Violation**: Line 13 — `Go-specific debugging: goroutine leaks, race conditions, deadlocks, channel misuse`
- **Fix**: → `Concurrency debugging: race conditions, deadlocks, resource leaks, and synchronization issues`
- **Violation**: Line 51 — `Bash(go test*)`
- **Fix**: → `Bash(...)`: Run the project's test suite to reproduce failures and validate hypotheses

### Structural Template Conformance

All 13 expanded personas were checked against the 7 required concepts:

| Concept | nav | phil | plan | craft | impl | rev | aud | dbg | res | sum | gh-a | gh-c | gh-e |
|---------|-----|------|------|-------|------|-----|-----|-----|-----|-----|------|------|------|
| 1. Identity | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| 2. Expertise | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| 3. Responsibilities | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| 4. Process | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| 5. Tools | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| 6. Output | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| 7. Constraints | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |

All 13 personas already have all 7 required structural concepts. Template conformance is satisfied.

## Unknowns and Decisions

### Decision 1: Fix Strategy for FR-008 Violations

**Decision**: In-place edit of the 4 failing persona files in `.wave/personas/`, then copy all 13 to `internal/defaults/personas/` for parity.

**Rationale**: The spec is explicit about what to fix (C-001 provides exact replacements). Editing in `.wave/personas/` first preserves them as the source of truth, then syncing to `internal/defaults/` satisfies FR-010.

**Alternatives rejected**:
- Edit `internal/defaults/` separately (risk: divergence, double work)
- Use a script to auto-sync (over-engineering for a one-time content update)

### Decision 2: Parity Sync Approach

**Decision**: After all edits are complete on `.wave/personas/`, copy each file byte-for-byte to `internal/defaults/personas/`. Validate with `diff -r`.

**Rationale**: FR-010 and SC-005 require byte-identical content. The simplest approach is to use `.wave/personas/` as the canonical source and copy.

**Alternatives rejected**:
- Symlinks (Go `//go:embed` doesn't follow symlinks)
- Generate from template (over-engineering; content is hand-authored)

### Decision 3: No Go Source Code Changes Required

**Decision**: This is a content-only change. No `.go` files, `wave.yaml`, or JSON schemas need modification.

**Rationale**: FR-011 explicitly prohibits Go source changes. The persona loading mechanism (`system_prompt_file` in `wave.yaml` → file read) is unchanged. The `//go:embed` directive in Go source already embeds `internal/defaults/personas/*.md` — updating the `.md` content updates what gets embedded at compile time without touching the `.go` file.

### Decision 4: Test Validation Strategy

**Decision**: Run `go test ./...` after all persona files are updated to confirm zero regressions (SC-006).

**Rationale**: Since no Go code changes, tests should pass. But constitution Principle 13 requires validation. Any test that hardcodes persona content would need updating — but tests should validate behavior (file loads successfully), not content.

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Test hardcodes persona content | Low | Medium | Check test fixtures; update if needed |
| Persona exceeds LLM context with step prompt | Low | Low | All personas are 57-93 lines, well under 200-line limit |
| FR-008 fix introduces incorrect tool references | Low | Medium | Review each fix against wave.yaml persona config |
| Parity sync misses a file | Low | High | Use `diff -r` validation step |
