# Prompt Content Quality Checklist

**Feature**: Persona Prompt Optimization (Issue #96)
**Focus**: Quality of requirements governing the 17 persona prompts and base protocol content
**Date**: 2026-02-20

## Base Protocol Content Requirements

- [ ] CHK-PC001 - Are the five base protocol content areas (fresh memory, artifact I/O, workspace isolation, contract compliance, permission enforcement) defined with enough precision to produce a deterministic file, or could two implementers produce significantly different content? [Completeness]
- [ ] CHK-PC002 - Does the spec define the target tone and register for base protocol content — imperative, declarative, instructional? Is this consistent with persona prompt tone? [Clarity]
- [ ] CHK-PC003 - Does the base protocol token budget (80-120) account for markdown structure overhead (headings, bullets) or only prose content? [Clarity]
- [ ] CHK-PC004 - Does the spec define whether the base protocol should reference "you" (the agent), "the system", or use passive voice? [Clarity]
- [ ] CHK-PC005 - Is the shared contract output instruction ("When a contract schema is provided...") explicitly listed as content to move from personas to the base protocol, or is it only mentioned in the research doc? [Completeness]

## Persona Optimization Criteria

- [ ] CHK-PC006 - Does the spec provide a rubric or decision tree for determining whether a given piece of persona content is "role-differentiating" vs "generic"? [Completeness]
- [ ] CHK-PC007 - Are the three mandatory structural elements (identity, responsibilities, output contract) defined with structural markers (e.g., must be H1, must be a section heading) or only semantically? [Clarity]
- [ ] CHK-PC008 - Does the spec define acceptance criteria for the QUALITY of optimized prompts beyond token count — e.g., can a 100-token prompt that says nothing useful pass? [Coverage]
- [ ] CHK-PC009 - Are the anti-patterns to remove (FR-006) exhaustive, or should the implementer also identify and remove other generic content not listed? [Completeness]
- [ ] CHK-PC010 - Does the spec address whether persona-specific behavioral constraints (e.g., "NEVER modify source code") should use the same phrasing across personas or can they vary? [Clarity]

## Per-Persona Coverage

- [ ] CHK-PC011 - Does the spec or plan provide per-persona optimization guidance for all 17 personas, or only for the ones with identified issues in the research? [Completeness]
- [ ] CHK-PC012 - Are the per-persona token ranges in the plan justified by analysis, or are they estimates that could cause rework if a persona's genuinely differentiating content exceeds the planned range? [Coverage]
- [ ] CHK-PC013 - Does the spec address how to handle the researcher persona's "Source Evaluation Criteria" and "Handling Conflicting Information" sections — are these role-differentiating or generic? [Clarity]
- [ ] CHK-PC014 - Does the spec address whether the supervisor and provocateur personas (currently the longest) need different optimization strategies than shorter personas? [Coverage]
- [ ] CHK-PC015 - Does the spec define what "no language-specific references" means for edge cases — e.g., the word "Go" as a verb, "Type" as a concept, "Class" as a category? [Clarity]

## Content Validation

- [ ] CHK-PC016 - Does the spec define how SC-002 (zero content duplication) will be mechanically verified — substring matching, semantic analysis, or manual review? [Completeness]
- [ ] CHK-PC017 - Does the spec define the language-specific keyword list exhaustively (Contract 3 lists 12 languages) — are less common languages (Elixir, Haskell, Scala, PHP) intentionally excluded? [Completeness]
- [ ] CHK-PC018 - Does the spec define whether SC-007 (mandatory sections) is verified by heading text matching, section content analysis, or structural position? [Clarity]
