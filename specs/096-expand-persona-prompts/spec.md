# Expand Persona Definitions with Detailed System Prompts and Role Specifications

**Issue**: [#96](https://github.com/re-cinq/wave/issues/96)
**Feature Branch**: `096-expand-persona-prompts`
**Labels**: enhancement, personas
**Status**: Draft

## Summary

Expand Wave's persona definitions from brief bullet-point descriptions into comprehensive, detailed system prompts that fully specify each persona's role, expertise, communication style, constraints, and behavioral guidelines.

## Background

Currently, Wave personas (defined in `.wave/personas/`) consist of relatively brief descriptions (10-25 lines each). To improve the quality and consistency of AI agent outputs, each persona should have a rich system prompt that provides:

- **Role definition**: Clear statement of the persona's purpose within the Wave pipeline
- **Domain expertise**: Specific technical domains and knowledge areas
- **Communication style**: Tone, verbosity, formatting preferences
- **Behavioral constraints**: What the persona should and should not do
- **Output expectations**: Expected format and quality standards for deliverables

Since Wave uses fresh memory at every step boundary, persona prompts are the **primary mechanism** for agent behavior — they need to be self-contained.

## Personas to Elaborate

The following 13 personas in `.wave/personas/` need expanded definitions:

1. `navigator.md` — Pipeline orchestration and planning (currently 19 lines)
2. `implementer.md` — Code implementation and feature development (currently 23 lines)
3. `reviewer.md` — Code review and quality assurance (currently 30 lines)
4. `planner.md` — Task planning and decomposition (currently 28 lines)
5. `researcher.md` — Codebase research and analysis (currently 51 lines)
6. `debugger.md` — Bug investigation and resolution (currently 36 lines)
7. `auditor.md` — Security and compliance auditing (currently 25 lines)
8. `craftsman.md` — Code craftsmanship and refactoring (currently 23 lines)
9. `summarizer.md` — Context compaction and summarization (currently 23 lines)
10. `github-analyst.md` — GitHub issue analysis (currently 33 lines)
11. `github-commenter.md` — GitHub issue commenting (currently 48 lines)
12. `github-enhancer.md` — GitHub issue enhancement (currently 29 lines)
13. `philosopher.md` — Design philosophy and architecture (currently 20 lines)

## Acceptance Criteria

- [ ] Each persona definition is at least 30 lines covering role, expertise, constraints, and communication style
- [ ] Persona definitions are specific to Wave's domain (software engineering, pipeline orchestration, AI agent coordination)
- [ ] Each persona includes a clear "You are..." identity statement
- [ ] Each persona specifies what tools and permissions it expects
- [ ] Each persona defines its output format expectations
- [ ] Elaborated personas integrate correctly with the existing persona loading mechanism in `internal/manifest/`
- [ ] Existing tests continue to pass after persona updates
- [ ] No behavioral regressions in pipeline execution with updated personas

## Implementation Notes

- Persona files are Markdown in `.wave/personas/` and loaded by the manifest system via `system_prompt_file` field in `wave.yaml`
- The manifest parser (`internal/manifest/parser.go`) validates that the referenced `system_prompt_file` exists; the content itself is read at runtime by the adapter
- Permission enforcement uses deny/allow patterns projected into `settings.json` and `CLAUDE.md` restriction sections — this is separate from persona content
- Each persona's permissions are defined in `wave.yaml`, not in the persona markdown file — the markdown file documents expected tools for self-awareness but does not enforce them
- Reference existing implementations: issue #24 (decoupled schema details) and PR #92 (persona-related changes)

## Out of Scope

- Changes to the persona loading mechanism itself
- Adding new personas (this issue is about improving existing ones)
- Changes to permission enforcement or pipeline execution logic
- Changes to `wave.yaml` persona configuration (only `.md` file content changes)

## Structural Template for Expanded Personas

Each persona should follow this consistent structure:

```markdown
# [Persona Name]

You are [identity statement — who you are and your primary function within Wave].

## Domain Expertise
- [Area 1]: [specific knowledge]
- [Area 2]: [specific knowledge]
- [Area 3]: [specific knowledge]

## Responsibilities
- [Responsibility 1]
- [Responsibility 2]
- [Responsibility 3]

## Communication Style
- [Tone and voice]
- [Formatting preferences]
- [Verbosity level]

## Process
1. [Step 1]
2. [Step 2]
3. [Step 3]

## Tools and Permissions
- Expected tools: [list]
- Read-only / read-write scope
- Restrictions awareness

## Output Format
[Expected output format and quality standards]

## Constraints
- [Hard constraint 1]
- [Hard constraint 2]
- [Behavioral boundary]
```

## Success Criteria

- All 13 persona files updated with expanded definitions
- Each file is at least 30 lines
- All existing tests pass (`go test ./...`)
- No changes to Go source code or `wave.yaml`
