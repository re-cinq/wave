# Implementation Plan: Expand Persona Definitions

## Objective

Expand all 13 existing Wave persona markdown files in `.wave/personas/` from brief bullet-point descriptions into comprehensive system prompts (30+ lines each) that fully specify role, expertise, communication style, constraints, and behavioral guidelines — without changing any Go source code or `wave.yaml`.

## Approach

This is a content-only change affecting 13 markdown files. The approach is:

1. **Define a consistent template** — All personas follow the same structural sections (identity, expertise, responsibilities, communication style, process, tools/permissions, output format, constraints) while content is unique to each role.
2. **Preserve existing content** — Current content should be retained and expanded, not replaced. Existing responsibilities, constraints, and output format sections already present should be kept and enriched.
3. **Ground in Wave architecture** — Each persona should reference Wave-specific concepts (pipelines, contracts, artifacts, fresh memory boundaries, workspace isolation) where relevant.
4. **Mirror `wave.yaml` permissions** — The tools/permissions section in each persona should document (not enforce) the same tools listed in `wave.yaml` for that persona, so the agent is self-aware of its capabilities.
5. **Batch by functional group** — Personas can be grouped and worked in parallel since they are independent files.

## File Mapping

All changes are modifications to existing files. No files are created or deleted.

| File | Action | Description |
|------|--------|-------------|
| `.wave/personas/navigator.md` | modify | Expand from 19 to 30+ lines |
| `.wave/personas/implementer.md` | modify | Expand from 23 to 30+ lines |
| `.wave/personas/reviewer.md` | modify | Expand from 30 to 40+ lines |
| `.wave/personas/planner.md` | modify | Expand from 28 to 30+ lines |
| `.wave/personas/researcher.md` | modify | Expand from 51 to 60+ lines (already most detailed) |
| `.wave/personas/debugger.md` | modify | Expand from 36 to 45+ lines |
| `.wave/personas/auditor.md` | modify | Expand from 25 to 30+ lines |
| `.wave/personas/craftsman.md` | modify | Expand from 23 to 30+ lines |
| `.wave/personas/summarizer.md` | modify | Expand from 23 to 30+ lines |
| `.wave/personas/github-analyst.md` | modify | Expand from 33 to 40+ lines |
| `.wave/personas/github-commenter.md` | modify | Expand from 48 to 55+ lines (already detailed) |
| `.wave/personas/github-enhancer.md` | modify | Expand from 29 to 35+ lines |
| `.wave/personas/philosopher.md` | modify | Expand from 20 to 30+ lines |

## Architecture Decisions

### AD-1: Content-only changes

No Go source code modifications. The manifest parser reads the file path from `wave.yaml` and passes the content to the adapter — so persona file content can be changed freely without affecting code.

### AD-2: Consistent structure, unique content

All personas follow the same section template:
- `# Name` + identity statement
- `## Domain Expertise`
- `## Responsibilities`
- `## Communication Style`
- `## Process` (workflow steps)
- `## Tools and Permissions`
- `## Output Format`
- `## Constraints`

But section depth and detail varies by persona complexity. Simple personas (summarizer) may have shorter sections than complex ones (implementer, researcher).

### AD-3: Document permissions, don't enforce them

Persona files describe expected tools for the agent's self-awareness. Actual enforcement is via `wave.yaml` deny/allow patterns projected into `settings.json` and `CLAUDE.md` restriction sections. The persona markdown should say "You have access to Read, Glob, Grep" to help the agent understand its capabilities, but this is documentation, not enforcement.

### AD-4: Self-contained prompts

Since Wave uses fresh memory at every step boundary, each persona prompt must be fully self-contained. It should not assume the agent has seen previous conversation context. All necessary behavioral instructions must be in the prompt itself.

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Expanded prompts cause unexpected behavioral changes | Medium | Medium | Preserve all existing constraints; add detail, don't change intent |
| Prompt length increases token usage per step | Low | Low | Keep prompts concise (30-60 lines, not 200+); focus on density over length |
| Inconsistency between persona markdown and `wave.yaml` permissions | Medium | Low | Cross-reference `wave.yaml` when writing each persona's Tools section |
| Tests fail due to content changes | Very Low | Low | Tests reference file existence, not content; run full test suite to verify |

## Testing Strategy

### Unit Tests
- No new unit tests needed — this is a content-only change to markdown files
- Existing tests validate persona file existence (via `system_prompt_file` in `wave.yaml`), which is unchanged

### Integration Tests
- Run `go test ./...` to ensure all existing tests pass
- Run `go test -race ./...` for race condition checking
- Key test packages to verify:
  - `internal/manifest/` — validates persona file existence
  - `internal/pipeline/` — uses personas in execution
  - `internal/adapter/` — reads persona content
  - `cmd/wave/commands/` — validates manifest

### Manual Validation
- Verify each expanded persona file is at least 30 lines
- Verify each has a "You are..." identity statement
- Verify tools/permissions sections match `wave.yaml`
- Verify consistent structure across all 13 files

## Grouping Strategy

Personas can be grouped by functional similarity for efficient batch writing:

**Group A: Core Pipeline Roles** (4 personas)
- navigator, implementer, reviewer, craftsman

**Group B: Planning & Analysis** (3 personas)
- planner, philosopher, researcher

**Group C: Specialized Operations** (3 personas)
- debugger, auditor, summarizer

**Group D: GitHub Integration** (3 personas)
- github-analyst, github-commenter, github-enhancer

All groups are independent and can be worked in parallel.
