# Optimize persona prompts for high-signal context engineering

**Issue**: [re-cinq/wave#96](https://github.com/re-cinq/wave/issues/96)
**Labels**: enhancement, personas
**Author**: nextlevelshit

## Summary

Optimize Wave's persona definitions to maximize behavioral signal per token, guided by the context engineering principle: "Find the smallest set of high-signal tokens that maximize the likelihood of some desired outcome."

The goal is **not** to make personas longer or shorter -- it's to ensure every token in a persona prompt earns its place by differentiating that persona's behavior from others.

## Background

Wave personas are the primary behavioral lever for agent output quality. Because of fresh memory at every step boundary, the persona prompt is the only persistent instruction the agent receives. Each persona prompt is injected at every pipeline step. With 30 personas and multi-step pipelines, bloated prompts create compounding overhead.

### Key techniques (Rajasekaran et al. 2025)

1. **Compaction** -- Persona prompts should be the compacted version of role knowledge
2. **Progressive disclosure** -- Persona prompts set identity and constraints, not step-by-step procedures
3. **Sub-agent architecture** -- Wave already does this. Each persona IS a sub-agent with a focused task

## Scope

### All 30 persona .md files in `internal/defaults/personas/` + `base-protocol.md`:

**Core 17 (from issue):**
- navigator, implementer, reviewer, planner, researcher, debugger, auditor, craftsman, summarizer, github-analyst, github-commenter, github-enhancer, philosopher, provocateur, validator, synthesizer, supervisor

**Forge-specific 13 (discovered in codebase):**
- gitea-analyst, gitea-commenter, gitea-enhancer, gitea-scoper
- gitlab-analyst, gitlab-commenter, gitlab-enhancer, gitlab-scoper
- bitbucket-analyst, bitbucket-commenter, bitbucket-enhancer, bitbucket-scoper
- github-scoper

### What makes a high-signal persona prompt

**Must have:**
- Identity statement differentiating this persona from all others
- Role-specific responsibilities unique to this persona
- Output contract (what the step produces)
- Constraints (behavioral guardrails)
- Anti-Patterns (where applicable, to prevent known failure modes)
- Quality Checklist (where applicable, for self-verification)

**Should NOT have:**
- "Domain Expertise" lists that restate Responsibilities
- "Communication Style" sections identical across personas
- Process sections describing generic read->process->output flow
- Length padding ("at least 30 lines")
- Wave architecture details already covered by base-protocol.md

### Shared base prompt

`base-protocol.md` already exists and is injected as a preamble at runtime by `internal/adapter/claude.go:258-265`. Individual persona files should contain ONLY what's unique to that role. Content that duplicates base-protocol.md must be removed from individual personas.

## Acceptance Criteria

- [ ] Each persona prompt contains only role-differentiating content + output contract + constraints
- [ ] Shared Wave protocol in base-protocol.md is sufficient (injected at runtime, not duplicated per persona)
- [ ] No generic "Communication Style" or "Domain Expertise" sections that duplicate Responsibilities
- [ ] Persona prompts target 100-400 tokens each (existing test: `TestPersonaFilesTokenRange`)
- [ ] Permission enforcement (allow/deny) preserved in .yaml configs -- non-negotiable for security
- [ ] Language-agnostic -- no Go/Python/JS/etc references (existing test: `TestPersonaFilesNoLanguageReferences`)
- [ ] All 30 personas covered (existing test: `TestAllPersonasCovered`)
- [ ] Parity between `internal/defaults/personas/` and `.wave/personas/` maintained (existing test: `TestPersonaFilesParity`)
- [ ] Mandatory sections present: H1 heading, Responsibilities/Step-by-Step, Output section (existing test: `TestPersonaFilesMandatorySections`)
- [ ] Existing tests pass: `go test ./internal/defaults/...`

## Author Comment

> 1. should not be too specific and general for all languages
> 2. parity between .wave and internal/defaults

## Out of Scope

- Adding new personas beyond the current 30
- Changes to persona loading mechanism or permission enforcement
- Changes to pipeline execution logic
- Empirical A/B testing of prompt effectiveness
- Changes to .yaml persona configs (permissions, temperature, model)
