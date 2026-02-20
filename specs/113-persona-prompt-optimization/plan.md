# Implementation Plan: Persona Prompt Optimization

**Branch**: `113-persona-prompt-optimization` | **Date**: 2026-02-20 | **Spec**: `specs/113-persona-prompt-optimization/spec.md`
**Input**: Feature specification from `specs/113-persona-prompt-optimization/spec.md`

## Summary

Optimize all 17 Wave persona prompts for high-signal context engineering. The approach has three pillars: (1) create a shared base protocol (`base-protocol.md`) containing Wave-universal operational context injected at runtime before every persona prompt; (2) compact each persona file to 100–400 tokens of role-differentiating content only, removing generic process descriptions, duplicated constraints, and anti-patterns; (3) add ~10 lines to `prepareWorkspace` in the Claude adapter to read and prepend the base protocol. Parity between `internal/defaults/personas/` and `.wave/personas/` is enforced by a new Go test.

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: `gopkg.in/yaml.v3`, `github.com/spf13/cobra` (existing — no new dependencies)
**Storage**: Filesystem (persona `.md` files embedded via `//go:embed`)
**Testing**: `go test ./...` with `-race` flag
**Target Platform**: Linux/macOS (single static binary)
**Project Type**: Single Go binary
**Performance Goals**: N/A (prompt content changes, no runtime performance impact)
**Constraints**: All existing tests must pass; no new runtime dependencies; single binary deployment
**Scale/Scope**: 17 persona files + 1 base protocol file + ~10 LOC adapter change

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

| Principle | Status | Notes |
|-----------|--------|-------|
| P1: Single Binary | PASS | No new dependencies. `base-protocol.md` is embedded via existing `//go:embed` directive. |
| P2: Manifest as SSOT | PASS | No manifest schema changes. Base protocol is not a persona — no manifest entry needed. |
| P3: Persona-Scoped Execution | PASS | Each invocation still scoped by exactly one persona. Base protocol is operational context, not a persona. Permission enforcement unchanged. |
| P4: Fresh Memory | PASS | Base protocol explicitly reinforces fresh memory constraint. No chat history inheritance. |
| P5: Navigator-First | N/A | No change to pipeline step ordering or navigator behavior. |
| P6: Contracts at Handover | PASS | No change to contract validation. Base protocol reminds agents of contract compliance. |
| P7: Relay via Summarizer | N/A | No change to relay/compaction mechanism. |
| P8: Ephemeral Workspaces | PASS | No change to workspace management. Base protocol reinforces workspace isolation. |
| P9: Credentials Never Touch Disk | PASS | No credential handling changes. |
| P10: Observable Progress | N/A | No change to progress events. |
| P11: Bounded Recursion | N/A | No change to recursion/resource limits. |
| P12: Minimal Step State Machine | N/A | No change to step states. |
| P13: Test Ownership | PASS | All existing tests must pass. New tests added for base protocol injection. |

**Result**: All applicable principles PASS. No violations to track.

## Project Structure

### Documentation (this feature)

```
specs/113-persona-prompt-optimization/
├── plan.md              # This file
├── research.md          # Phase 0 output — unknowns and decisions
├── data-model.md        # Phase 1 output — entity definitions
├── contracts/           # Phase 1 output — validation contracts
│   └── persona-validation.md
└── tasks.md             # Phase 2 output (NOT created by /speckit.plan)
```

### Source Code (repository root)

```
internal/
├── adapter/
│   ├── claude.go           # MODIFY: add base protocol prepend in prepareWorkspace
│   └── claude_test.go      # MODIFY: add tests for base protocol injection
└── defaults/
    ├── embed.go            # UNCHANGED: //go:embed personas/*.md already captures all .md
    └── personas/
        ├── base-protocol.md    # NEW: shared Wave operational context
        ├── navigator.md        # MODIFY: optimize to 100-150 tokens
        ├── implementer.md      # MODIFY: optimize to 100-180 tokens
        ├── reviewer.md         # MODIFY: optimize to 100-180 tokens
        ├── planner.md          # MODIFY: optimize to 100-180 tokens
        ├── researcher.md       # MODIFY: optimize to 150-300 tokens
        ├── debugger.md         # MODIFY: optimize to 100-200 tokens
        ├── auditor.md          # MODIFY: optimize to 100-180 tokens
        ├── craftsman.md        # MODIFY: optimize to 100-180 tokens
        ├── summarizer.md       # MODIFY: optimize to 100-180 tokens
        ├── github-analyst.md   # MODIFY: optimize to 100-250 tokens
        ├── github-commenter.md # MODIFY: optimize to 100-250 tokens
        ├── github-enhancer.md  # MODIFY: optimize to 100-200 tokens
        ├── philosopher.md      # MODIFY: optimize to 100-150 tokens
        ├── provocateur.md      # MODIFY: optimize to 200-400 tokens
        ├── validator.md        # MODIFY: optimize to 100-250 tokens
        ├── synthesizer.md      # MODIFY: optimize to 100-200 tokens
        └── supervisor.md       # MODIFY: optimize to 200-400 tokens

.wave/
└── personas/               # MODIFY: mirror all changes from internal/defaults/personas/
    ├── base-protocol.md    # NEW: byte-identical copy
    └── *.md                # MODIFY: byte-identical copies

tests/                      # Potential location for parity test
```

**Structure Decision**: Existing Go project structure. Changes are confined to `internal/adapter/` (runtime injection), `internal/defaults/personas/` (prompt content), and `.wave/personas/` (parity copies). No new packages, no new directories beyond what exists.

## Design Decisions

### D1: Base Protocol Injection Point

**Location**: `internal/adapter/claude.go`, `prepareWorkspace` method, lines 260-274.

**Current flow**:
1. Read `cfg.SystemPrompt` OR `.wave/personas/<persona>.md`
2. Append restriction section
3. Write CLAUDE.md

**New flow**:
1. **Read `.wave/personas/base-protocol.md`** (NEW)
2. **Write base protocol content** (NEW)
3. **Write `---` separator** (NEW)
4. Read `cfg.SystemPrompt` OR `.wave/personas/<persona>.md` (existing)
5. Append restriction section (existing)
6. Write CLAUDE.md (existing)

**Error handling**: If `base-protocol.md` cannot be read, `prepareWorkspace` returns an error. The pipeline step fails-secure rather than running without operational context.

**Edge case**: When `cfg.SystemPrompt` is set directly (e.g., pipeline steps with inline prompts), the base protocol is still prepended. This ensures all pipeline steps get Wave operational context regardless of how the prompt is configured.

### D2: Persona Optimization Strategy

For each persona, apply this checklist:
1. **Keep**: Identity statement (H1), unique responsibilities, output contract, role-specific behavioral constraints
2. **Remove**: Generic process descriptions (read-analyze-report workflows), "Communication Style" sections, "Domain Expertise" restating responsibilities, shared contract output boilerplate ("When a contract schema is provided...")
3. **Compact**: Merge overlapping bullet points, eliminate filler phrases
4. **Verify**: No language-specific references, no base protocol duplication, within 100-400 token range

### D3: Parity Enforcement

A Go test in `internal/defaults/` (or `tests/`) will:
1. Read all files from `internal/defaults/personas/` via the embed FS
2. Read all files from `.wave/personas/` via `os.ReadFile`
3. Assert byte-identical content for every file
4. Fail with a clear message identifying which files diverge

## Complexity Tracking

_No constitution violations to track._

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|-----------|--------------------------------------|
| (none) | — | — |
