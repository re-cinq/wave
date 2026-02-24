# Implementation Plan: Persona Architecture Evaluation

## Objective

Evaluate Wave's persona system against Claude Code's team-based architecture, produce a comprehensive comparison document, and deliver at least 3 actionable proposals for improving persona definitions, consolidation opportunities, and coordination patterns.

## Approach

This is primarily a **research and documentation task** with targeted code changes to persona definitions and manifest configuration. The deliverables are:

1. A comparison document analyzing both systems
2. Concrete persona consolidation/improvement proposals
3. Updated persona `.md` files and `wave.yaml` entries where changes are warranted
4. Updated pipelines that reference consolidated personas

## File Mapping

### Files to Create

| Path | Purpose |
|------|---------|
| `docs/persona-architecture-evaluation.md` | Comprehensive comparison document |

### Files to Modify

| Path | Purpose |
|------|---------|
| `.wave/personas/reviewer.md` | Consolidate auditor+reviewer into a unified reviewer |
| `.wave/personas/auditor.md` | Remove (merge into reviewer) or narrow scope |
| `.wave/personas/implementer.md` | Consolidate with craftsman or clarify differentiation |
| `.wave/personas/craftsman.md` | Absorb implementer capabilities or refine |
| `.wave/personas/planner.md` | Consolidate with philosopher or clarify roles |
| `.wave/personas/philosopher.md` | Absorb planner capabilities or refine |
| `.wave/personas/navigator.md` | Potentially enhance with research capabilities |
| `.wave/personas/base-protocol.md` | Add coordination protocol enhancements |
| `wave.yaml` | Update persona definitions for consolidated personas |
| `.wave/pipelines/*.yaml` | Update persona references in affected pipelines |

### Files Potentially to Delete

| Path | Reason |
|------|--------|
| `.wave/personas/auditor.md` | If consolidated into reviewer |
| `.wave/personas/implementer.md` | If consolidated into craftsman |
| `.wave/personas/planner.md` | If consolidated into philosopher |

## Architecture Decisions

### AD-1: Consolidation Strategy

**Decision**: Consolidate personas where role overlap exceeds 70% and pipeline usage patterns show interchangeability.

**Rationale**: Having 18 personas creates cognitive overhead for pipeline authors. Claude Code Teams uses fewer, more capable roles. Consolidation reduces the persona catalog while preserving specialization where it matters.

**Candidates**:
- `auditor` + `reviewer` → unified `reviewer` with security audit capabilities
- `craftsman` + `implementer` → unified `craftsman` (implementer is a subset)
- `planner` + `philosopher` → unified `architect` or keep separate with clearer boundaries

### AD-2: Preserve Constitutional Properties

**Decision**: All changes must preserve:
- Fresh memory at step boundaries
- Contract validation at handovers
- Permission enforcement via allow/deny patterns
- Ephemeral workspace isolation

**Rationale**: These are Wave's core architectural invariants. Claude Code Teams' dynamic coordination (spawn, join, broadcast) operates differently and cannot be directly adopted without violating fresh-memory guarantees.

### AD-3: Adopt Patterns Selectively

**Decision**: Adopt Claude Code patterns that fit Wave's pipeline model:
- **Watchdog pattern** → Can be implemented as a parallel pipeline step with read-only permissions
- **Council pattern** → Can be implemented as parallel steps with a synthesizer merge step
- Claude Code's leader-worker and swarm patterns are NOT compatible with Wave's fresh-memory architecture

### AD-4: Persona Prompt Enhancement

**Decision**: Enhance persona `.md` files with:
- Explicit anti-patterns (what NOT to do)
- Cross-persona awareness (what artifacts to expect/produce)
- Output quality checklist

**Rationale**: Claude Code Teams uses structured role definitions with clear capability boundaries. Wave personas are currently terse and could benefit from richer role descriptions.

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Pipeline breakage from persona renames | Medium | High | Update all pipeline YAML references; run test suite |
| Consolidated personas lose specialization | Low | Medium | Keep granular permission models even with consolidated prompts |
| Over-consolidation reduces pipeline flexibility | Medium | Medium | Start with clear consolidation candidates; keep option to split later |
| Tests reference specific persona names | Medium | High | Search all tests for persona name strings; update accordingly |

## Testing Strategy

1. **Grep for persona references**: Find all code and YAML files referencing persona names
2. **Pipeline validation**: Ensure all pipeline YAML files parse correctly after changes
3. **Permission tests**: Verify permission models are preserved for consolidated personas
4. **Integration tests**: Run `go test ./...` to catch regressions
5. **Manual pipeline smoke test**: Run a simple pipeline to verify end-to-end flow
