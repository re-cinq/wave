# Research: Add Missing Personas

**Feature**: 021-add-missing-personas
**Date**: 2026-02-04

## Research Summary

This is a configuration-focused feature requiring no external research. All patterns are derived from existing Wave personas.

## Decision 1: Implementer Persona Permissions

**Decision**: Model implementer after `craftsman` persona with full execution permissions

**Rationale**:
- Implementer is used in pipelines for executing code changes (gh-poor-issues, umami)
- Requires ability to read files, write changes, edit existing code, and run bash commands
- Craftsman has proven safe permission set for code modification tasks

**Alternatives Considered**:
- Narrower permissions like navigator (rejected: implementer needs to write)
- Even broader permissions (rejected: unnecessary for typical tasks)

## Decision 2: Reviewer Persona Permissions

**Decision**: Model reviewer after `auditor` persona with read-heavy permissions plus artifact write

**Rationale**:
- Reviewer is used for quality assessment and validation (docs-to-impl, gh-poor-issues)
- Needs to read code, search patterns, run tests, but NOT modify source
- Must write artifact.json for pipeline handoff (auditor cannot write at all)

**Alternatives Considered**:
- Full auditor permissions with no write (rejected: cannot produce artifacts)
- Implementer-level permissions (rejected: too broad for review tasks)

## Decision 3: Persona System Prompt Structure

**Decision**: Follow existing Wave persona markdown structure

**Structure**:
```
# Persona Name

Brief role description (1-2 sentences)

## Responsibilities
- Bullet list of duties

## Output Format
Structured output guidance (JSON for contract compatibility)

## Constraints
- NEVER statements for hard limits
- Focus and scope boundaries
```

**Rationale**: Consistency with existing personas (navigator, craftsman, auditor)

## Decision 4: Embedded Defaults Handling

**Decision**: Add files to `internal/defaults/personas/` directory

**Rationale**:
- embed.go uses `//go:embed personas/*` directive
- New .md files in the directory are automatically embedded
- No code changes needed to embed.go

**Verification Needed**: Confirm glob pattern includes new files (see agent research)

## Open Questions (Resolved)

| Question | Resolution |
|----------|------------|
| What permissions does implementer need? | Read, Write, Edit, Bash (like craftsman) |
| What permissions does reviewer need? | Read, Glob, Grep, Write(artifact.json), limited Bash for tests |
| Are code changes needed for wave init? | No - embed directive auto-includes new files |
| What output format for contract compatibility? | JSON with schema injection at runtime |

## References

- Existing personas: `.wave/personas/craftsman.md`, `.wave/personas/auditor.md`
- Permission patterns: `wave.yaml` lines 25-110
- Pipeline usage: `.wave/pipelines/gh-poor-issues.yaml`, `.wave/pipelines/umami.yaml`
