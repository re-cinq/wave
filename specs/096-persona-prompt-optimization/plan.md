# Implementation Plan: Persona Prompt Optimization

## Objective

Optimize all 30 persona .md files (+base-protocol.md) to maximize behavioral signal per token by removing low-signal content (generic process, duplicated Wave protocol, communication style), strengthening role-differentiating content, and maintaining strict parity between the two persona directories.

## Approach

### Strategy: Audit-then-optimize, one persona at a time

Each persona file is edited individually (per CLAUDE.md rule: "personas are functional code, edit individually"). The optimization follows a consistent pattern:

1. **Audit** the current persona for low-signal content
2. **Remove** generic sections already covered by base-protocol.md (artifact I/O, workspace isolation, contract compliance)
3. **Remove** "Communication Style", "Domain Expertise", and generic process sections if present
4. **Sharpen** the identity statement and responsibilities to be maximally differentiating
5. **Preserve** Anti-Patterns, Quality Checklist, and Constraints sections where they add genuine value
6. **Verify** token count is within 100-400 range (words * 100/75 heuristic)
7. **Mirror** the change to the paired directory

### base-protocol.md review

Review and potentially enhance base-protocol.md to ensure it covers all shared concerns that are being removed from individual personas. The current content already covers: fresh context, artifact I/O, workspace isolation, contract compliance, permission enforcement, real execution only, no TodoWrite. This is comprehensive and likely needs no changes.

## File Mapping

### Files to modify (30 persona .md files x 2 directories = 60 edits)

**Source of truth**: `internal/defaults/personas/*.md`
**Mirror**: `.wave/personas/*.md`

Each file below is modified in BOTH locations (parity test enforces byte-identical content):

| Persona | Current State | Optimization Needed |
|---------|--------------|-------------------|
| base-protocol.md | Already lean (~240 tokens) | Review only -- likely no changes needed |
| navigator.md | Good structure, has Anti-Patterns + Quality Checklist | Minor: check for low-signal content |
| implementer.md | Very lean (~120 tokens) | Already optimized, verify token minimum |
| reviewer.md | Has Ontology-vs-Code section (pipeline-specific) | Keep -- genuinely differentiating |
| planner.md | Has Scope Boundary section | Good -- differentiates from philosopher |
| researcher.md | Has Composition Pipeline section | Keep -- genuinely needed for pipelines |
| debugger.md | Good structure | Minor optimization possible |
| auditor.md | Lean, overlaps with reviewer | Ensure differentiation from reviewer |
| craftsman.md | Has Guidelines + Anti-Patterns + Quality Checklist | May be slightly verbose -- tighten |
| summarizer.md | Has Anti-Patterns + Quality Checklist | Good, verify within range |
| github-analyst.md | Lean, step-by-step format | Already optimized |
| github-commenter.md | Has Core Capabilities with examples | Keep -- CLI syntax is high-signal |
| github-enhancer.md | Lean | Already optimized |
| github-scoper.md | Has Decomposition Guidelines + Template | Keep -- procedural knowledge is differentiating |
| philosopher.md | Has Ontology Extraction section | Keep |
| provocateur.md | Has Thinking Style + Evidence Gathering + Ontology | Well-structured, verify range |
| validator.md | Lean, focused | Already well-optimized |
| synthesizer.md | Has JSON output emphasis + Ontology Evolution | Keep |
| supervisor.md | Has Evidence Gathering + Evaluation Criteria | Verify within range |
| gitea-analyst.md | Forge variant of github-analyst | Same pattern, different CLI |
| gitea-commenter.md | Forge variant of github-commenter | Same pattern, different CLI |
| gitea-enhancer.md | Forge variant of github-enhancer | Same pattern, different CLI |
| gitea-scoper.md | Forge variant of github-scoper | Same pattern, different CLI |
| gitlab-analyst.md | Forge variant of github-analyst | Same pattern, different CLI |
| gitlab-commenter.md | Forge variant of github-commenter | Same pattern, different CLI |
| gitlab-enhancer.md | Forge variant of github-enhancer | Same pattern, different CLI |
| gitlab-scoper.md | Forge variant of github-scoper | Same pattern, different CLI |
| bitbucket-analyst.md | Forge variant, uses REST API | Same pattern, API-specific |
| bitbucket-commenter.md | Forge variant, uses REST API | Same pattern, API-specific |
| bitbucket-enhancer.md | Forge variant, uses REST API | Same pattern, API-specific |
| bitbucket-scoper.md | Forge variant, uses REST API | Same pattern, API-specific |

### Files NOT modified

- `internal/defaults/personas/*.yaml` -- permission configs are out of scope
- `internal/adapter/claude.go` -- CLAUDE.md assembly mechanism unchanged
- `internal/pipeline/executor.go` -- pipeline execution unchanged
- `internal/defaults/embed.go` -- embed mechanism unchanged
- Test files -- tests define the constraints we optimize WITHIN, not what we change

## Architecture Decisions

### AD-1: base-protocol.md is the shared base (already exists)

The issue proposes extracting shared content into a composable base prompt. This is ALREADY implemented:
- `base-protocol.md` is loaded and prepended at `internal/adapter/claude.go:258-265`
- All personas receive it as a preamble via `---` separator
- No code changes needed -- just ensure persona .md files don't duplicate base-protocol content

### AD-2: Edit both directories simultaneously

The parity test (`TestPersonaFilesParity`) enforces byte-identical content. Every edit must be applied to BOTH `internal/defaults/personas/` AND `.wave/personas/`. The approach is: edit the source of truth first, then copy to .wave.

### AD-3: Preserve pipeline-specific sections

Sections like "Ontology-vs-Code Validation" (reviewer), "Composition Pipeline Integration" (researcher), "Ontology Extraction Patterns" (philosopher), "Ontology Challenge Patterns" (provocateur), "Ontology Evolution Output" (synthesizer) are genuinely differentiating and should be preserved. They encode pipeline-specific knowledge that no other persona has.

### AD-4: Forge variants share structure with platform-specific CLI syntax

The forge-specific personas (gitea-*, gitlab-*, bitbucket-*) mirror the github-* structure but with different CLI commands. These are already lean and well-structured. Optimization should focus on ensuring consistency across forge families rather than cutting content -- the CLI syntax examples are high-signal.

### AD-5: Token budget is 100-400 per persona

The test `TestPersonaFilesTokenRange` uses `words * 100 / 75` as the token heuristic. Current personas are already within range. The optimization goal is signal density, not token reduction.

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Over-optimization removes differentiating content | Medium | High | Check each removal against "does any other persona have this?" |
| Token count drops below 100 minimum | Low | Medium | Run token check after each edit |
| Parity test fails from partial edits | Medium | High | Always edit both dirs as atomic operation |
| Behavioral regression in pipeline execution | Low | Medium | Run `go test ./internal/defaults/...` after all edits |
| Base-protocol.md changes break assembly | Very Low | High | Don't change base-protocol.md unless absolutely necessary |

## Testing Strategy

### Automated (existing tests -- run after all changes)

```bash
go test ./internal/defaults/... -v
```

This runs:
- `TestPersonaFilesTokenRange` -- 100-400 token range
- `TestPersonaFilesNoLanguageReferences` -- no language-specific keywords
- `TestPersonaFilesMandatorySections` -- H1, Responsibilities/Step-by-Step, Output section
- `TestAllPersonasCovered` -- exactly 30 personas (excluding base-protocol)
- `TestPersonaFilesParity` -- byte-identical between internal/defaults and .wave
- `TestGetPersonaConfigs_MatchesPersonaFiles` -- every .md has a .yaml config

### Manual verification

- Spot-check 3-4 personas for signal density after optimization
- Verify no content from base-protocol.md is duplicated in any persona
- Verify forge variants are consistent within their family
