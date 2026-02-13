# Persona Content Quality Checklist: 096-expand-persona-prompts

> Validates that the persona-level requirements are well-specified enough to produce
> correct, complete, and language-agnostic persona definitions. Tests the requirements,
> not the personas themselves.

## Identity & Structure Requirements

- [ ] CHK101 - Is the "You are..." identity statement requirement (FR-001) specified with placement precision (within first 3 lines)? [Completeness]
- [ ] CHK102 - Does the structural template specify whether sections must appear in a fixed order or can be reordered freely? [Clarity]
- [ ] CHK103 - Are the criteria for "additional sections beyond the required seven" defined — can personas add arbitrary extra sections? [Clarity]
- [ ] CHK104 - Is the minimum content depth per section defined, or could a single bullet point per section satisfy a requirement like FR-002 (Domain Expertise)? [Completeness]

## Language-Agnostic Requirements (FR-008)

- [ ] CHK105 - Does the spec provide a definitive list of language-specific patterns to search for, or only examples? [Completeness]
- [ ] CHK106 - Is the boundary between "language-specific toolchain reference" (prohibited) and "software concept that originated in a specific language" (e.g., "goroutine" in a general concurrency context) clearly drawn? [Clarity]
- [ ] CHK107 - Does the requirement address illustrative examples — are language-specific examples permitted if clearly marked as non-prescriptive? [Clarity]
- [ ] CHK108 - Are the FR-008 remediation strings in the clarifications section (C-001) identical to those in the plan's task definitions? [Consistency]
- [ ] CHK109 - Does the requirement cover future-proofing: if new personas are added later, must they also comply with FR-008? [Coverage]
- [ ] CHK110 - Is the definition of "hardcoded reference" precise enough to distinguish `Bash(go test*)` (prohibited) from abstract mentions of testing concepts? [Clarity]

## Tools & Permissions Requirements (FR-005)

- [ ] CHK111 - Is it clear which tools should use Wave permission syntax (e.g., `Bash(git log*)`) vs. generic descriptions? [Clarity]
- [ ] CHK112 - Does the requirement specify that the "actual permissions enforced by orchestrator" disclaimer must appear in every persona's Tools section? [Completeness]
- [ ] CHK113 - Is the relationship between persona-declared tools and wave.yaml-enforced permissions documented as a non-binding expectation? [Clarity]

## Output Format Requirements (FR-006)

- [ ] CHK114 - Does the requirement specify that the contract schema precedence note must appear in every persona's Output Format section? [Completeness]
- [ ] CHK115 - Is the default output format requirement flexible enough to accommodate different persona roles (JSON output vs. prose vs. structured markdown)? [Clarity]

## Constraints Requirements (FR-007)

- [ ] CHK116 - Are there minimum constraint categories that each persona must address (e.g., scope limitations, destructive action prohibitions)? [Completeness]
- [ ] CHK117 - Does the requirement distinguish between universal constraints (all personas) and role-specific constraints? [Clarity]
