# Data Model: Persona Prompt Optimization

**Feature Branch**: `113-persona-prompt-optimization`
**Date**: 2026-02-20

## Entities

### 1. Shared Base Protocol

**File**: `base-protocol.md`
**Locations**: `internal/defaults/personas/base-protocol.md`, `.wave/personas/base-protocol.md`
**Format**: Markdown
**Token Budget**: 80–120 tokens

The base protocol is a new file containing Wave-universal operational context. It is NOT a persona — it has no identity, no responsibilities, and no output contract. It is a preamble injected before every persona prompt.

**Content Structure**:
```markdown
# Wave Agent Protocol

You are operating within a Wave pipeline step.

## Operational Context

- **Fresh context**: You have no memory of prior steps. Each step starts clean.
- **Artifact I/O**: Read inputs from injected artifacts. Write outputs to artifact files.
- **Workspace isolation**: You are in an ephemeral worktree. Changes here do not affect the source repository directly.
- **Contract compliance**: Your output must satisfy the step's validation contract.
- **Permission enforcement**: Tool permissions are enforced by the orchestrator. Do not attempt to bypass restrictions listed below.
```

**Constraints**:
- MUST NOT contain role-specific guidance
- MUST NOT list specific tool allow/deny permissions
- MUST NOT contain programming language references
- MUST be byte-identical across both file locations
- MUST be under 120 tokens

---

### 2. Persona Prompt (optimized)

**Files**: `<persona-name>.md` (17 files)
**Locations**: `internal/defaults/personas/`, `.wave/personas/`
**Format**: Markdown
**Token Budget**: 100–400 tokens per file

Each persona prompt contains ONLY role-differentiating content.

**Required Sections** (per FR-003, FR-004, FR-005):

1. **Identity Statement** (H1 heading): Single sentence establishing who this persona is and what differentiates it.
2. **Responsibilities**: Bulleted list of unique responsibilities NOT shared with other personas.
3. **Output Contract**: Expected output format for this role (JSON schema reference, markdown structure, etc.).

**Optional Sections** (as needed):
4. **Behavioral Constraints**: Role-specific behavioral reminders as defense-in-depth (e.g., "NEVER modify source code" for read-only personas). NOT tool lists.

**Prohibited Content** (per FR-006, FR-007):
- "Communication Style" sections
- "Domain Expertise" sections restating responsibilities
- Process sections describing generic read-process-output workflows
- Programming language references
- Content duplicated from the base protocol

---

### 3. CLAUDE.md (generated at runtime — not a stored entity)

**Location**: `<workspace>/CLAUDE.md` (ephemeral, generated per step)
**Format**: Markdown
**Assembled from**:

```
[base-protocol.md content]
---
[persona-specific .md content]
---
[restriction section from manifest permissions]
```

**Assembly Logic** (in `prepareWorkspace`):
1. Read `base-protocol.md` from `.wave/personas/base-protocol.md`
2. Read persona prompt from `.wave/personas/<persona>.md` OR use `cfg.SystemPrompt`
3. Concatenate: base protocol + `\n\n---\n\n` + persona prompt
4. Append restriction section (existing logic, unchanged)

**Error Handling**:
- Missing `base-protocol.md` → return error (fail-secure)
- Missing persona `.md` → fall back to generic header (existing behavior, unchanged)

---

## Entity Relationships

```
base-protocol.md ──────┐
                        ├──→ CLAUDE.md (runtime assembly)
persona-name.md ────────┤
                        │
restriction section ────┘
(from manifest)
```

- **1:many**: One base protocol serves all 17 personas
- **1:1**: One persona prompt per persona name
- **1:1**: One CLAUDE.md per pipeline step execution

---

## File Inventory (post-implementation)

### `internal/defaults/personas/` (18 files — 17 personas + 1 base protocol)

| File | Type | Token Range |
|------|------|-------------|
| `base-protocol.md` | Base Protocol | 80–120 |
| `navigator.md` | Persona | 100–150 |
| `implementer.md` | Persona | 100–180 |
| `reviewer.md` | Persona | 100–180 |
| `planner.md` | Persona | 100–180 |
| `researcher.md` | Persona | 150–300 |
| `debugger.md` | Persona | 100–200 |
| `auditor.md` | Persona | 100–180 |
| `craftsman.md` | Persona | 100–180 |
| `summarizer.md` | Persona | 100–180 |
| `github-analyst.md` | Persona | 100–250 |
| `github-commenter.md` | Persona | 100–250 |
| `github-enhancer.md` | Persona | 100–200 |
| `philosopher.md` | Persona | 100–150 |
| `provocateur.md` | Persona | 200–400 |
| `validator.md` | Persona | 100–250 |
| `synthesizer.md` | Persona | 100–200 |
| `supervisor.md` | Persona | 200–400 |

### `.wave/personas/` (18 files — byte-identical copies)

Same as above.

---

## Code Changes Summary

### Modified Files

| File | Change |
|------|--------|
| `internal/adapter/claude.go` | Add base protocol read+prepend in `prepareWorkspace` (~10 LOC) |
| `internal/adapter/claude_test.go` | Add test for base protocol injection, missing file error |
| `internal/defaults/personas/*.md` | Optimize all 17 persona files + add `base-protocol.md` |
| `.wave/personas/*.md` | Mirror all changes from `internal/defaults/personas/` |

### New Files

| File | Purpose |
|------|---------|
| `internal/defaults/personas/base-protocol.md` | Shared Wave operational context (embedded) |
| `.wave/personas/base-protocol.md` | Workspace copy of base protocol |

### Unchanged Files

| File | Why |
|------|-----|
| `internal/defaults/embed.go` | `//go:embed personas/*.md` already captures all .md files |
| `cmd/wave/commands/init.go` | `GetPersonas()` includes base-protocol.md naturally; manifest template is hardcoded |
| `wave.yaml` manifest template | No new persona entry needed — base protocol is not a persona |
