# Persona Validation Contracts

These contracts define the machine-verifiable checks for persona prompt optimization.

## Contract 1: Token Range (SC-001)

**Type**: Structural
**Applies to**: All 17 persona `.md` files
**Check**: Word count × (100/75) ≥ 100 AND word count × (100/75) ≤ 400
**Tool**: `wc -w` on each persona file, excluding `base-protocol.md`

## Contract 2: Zero Duplication (SC-002)

**Type**: Content
**Applies to**: Each persona file vs. `base-protocol.md`
**Check**: No paragraph (3+ consecutive words) from `base-protocol.md` appears verbatim in any persona file
**Tool**: Text diff analysis or substring matching

## Contract 3: No Language References (SC-003)

**Type**: Content
**Applies to**: All persona files + `base-protocol.md`
**Check**: `grep -i` for: `\bGo\b`, `\bGolang\b`, `\bPython\b`, `\bTypeScript\b`, `\bJavaScript\b`, `\bJava\b`, `\bRust\b`, `\bRuby\b`, `\bSwift\b`, `\bKotlin\b`, `\bC\+\+\b`, `\bC#\b` returns zero matches
**Exclusions**: "Go" as a verb (context-dependent) — use `\bGolang\b` and `\bGo\s+(code|program|module|package|binary|runtime|compiler)\b` patterns
**Tool**: `grep -P` or Go test with regex

## Contract 4: All Personas Covered (SC-004)

**Type**: Completeness
**Check**: Exactly 17 persona files exist (excluding `base-protocol.md`) with these names: navigator, implementer, reviewer, planner, researcher, debugger, auditor, craftsman, summarizer, github-analyst, github-commenter, github-enhancer, philosopher, provocateur, validator, synthesizer, supervisor
**Tool**: File listing + name matching

## Contract 5: Test Suite Green (SC-005)

**Type**: Behavioral
**Check**: `go test ./...` exits 0
**Tool**: Go test runner

## Contract 6: Parity (SC-006)

**Type**: Structural
**Check**: `diff -r internal/defaults/personas/ .wave/personas/` returns zero differences
**Tool**: `diff` or Go test with byte comparison

## Contract 7: Mandatory Sections (SC-007)

**Type**: Structural
**Applies to**: All 17 persona files
**Check**: Each file contains:
- An H1 heading (identity statement)
- A section with "Responsibilities" or role-specific responsibilities content
- A section with "Output" in the heading (output contract)
**Tool**: Regex or markdown parser

## Contract 8: Base Protocol Injection (SC-008)

**Type**: Behavioral
**Check**: After `prepareWorkspace` runs, the generated CLAUDE.md contains:
1. Base protocol content (at least the "Wave Agent Protocol" heading)
2. A `---` separator
3. Persona-specific content
**Tool**: Go integration test
