# Quality Review Checklist: 096-expand-persona-prompts

> Unit tests for requirement quality — each item validates the specification,
> not the implementation.

## Completeness

- [ ] CHK001 - Are acceptance criteria defined for every persona that needs FR-008 remediation (craftsman, reviewer, auditor, debugger)? [Completeness]
- [ ] CHK002 - Does the spec define what "language-agnostic" means precisely enough to distinguish acceptable from unacceptable content? [Completeness]
- [ ] CHK003 - Are the exact replacement strings for all FR-008 violations documented for each of the 4 affected files? [Completeness]
- [ ] CHK004 - Does the spec enumerate all 13 persona files with their current state (line counts, FR-008 status) to prevent ambiguity about scope? [Completeness]
- [ ] CHK005 - Is the structural template defined with enough specificity to validate conformance objectively (all 7 required concepts listed)? [Completeness]
- [ ] CHK006 - Are validation steps for SC-005 (parity) specified as executable commands, not just prose descriptions? [Completeness]
- [ ] CHK007 - Does the spec cover both file locations (.wave/personas/ and internal/defaults/personas/) in every relevant requirement? [Completeness]

## Clarity

- [ ] CHK008 - Is the distinction between "Go source code" and "Markdown files in internal/defaults/" explicitly disambiguated to prevent FR-011 misinterpretation? [Clarity]
- [ ] CHK009 - Is the relationship between the structural template section names and the 7 required concepts clearly mapped, given that section heading flexibility is allowed? [Clarity]
- [ ] CHK010 - Does the spec clearly distinguish between the already-completed initial expansion (commit 6fdb3e9) and the remaining refinement scope? [Clarity]
- [ ] CHK011 - Is the precedence rule between contract schemas and persona output format sections clearly stated? [Clarity]
- [ ] CHK012 - Are the terms "language-agnostic", "language-specific toolchain reference", and "hardcoded reference" defined with enough precision to apply consistently across all 13 files? [Clarity]
- [ ] CHK013 - Is the "Tools and Permissions" section requirement clear about when to use Wave permission syntax vs. generic descriptions? [Clarity]

## Consistency

- [ ] CHK014 - Are all functional requirements (FR-001 through FR-013) traceable to at least one user story? [Consistency]
- [ ] CHK015 - Are all success criteria (SC-001 through SC-007) traceable to at least one functional requirement? [Consistency]
- [ ] CHK016 - Does the "Out of Scope" section align with FR-011 (no Go source code changes) without contradicting FR-010 (parity sync requires updating files under internal/)? [Consistency]
- [ ] CHK017 - Is the line count range (30-200) consistently applied in FR-009, FR-013, and SC-002? [Consistency]
- [ ] CHK018 - Do the edge cases in the spec align with the constraints documented in the structural template (contract precedence, orchestrator enforcement, tools section)? [Consistency]
- [ ] CHK019 - Does the plan's task list cover all functional requirements, or are any FRs missing corresponding tasks? [Consistency]

## Coverage

- [ ] CHK020 - Are all GitHub issue comments (language-agnostic, parity) explicitly traced to functional requirements? [Coverage]
- [ ] CHK021 - Does the spec address what happens when a persona that currently passes FR-008 is accidentally modified to introduce language-specific content? [Coverage]
- [ ] CHK022 - Is there a requirement for the 9 personas that already pass FR-008 to remain compliant, not just the 4 that need fixes? [Coverage]
- [ ] CHK023 - Does the spec address whether the "Tools and Permissions" section's language-agnostic wording must also apply to the internal/defaults/ copies (trivially yes via FR-010, but explicitly stated)? [Coverage]
- [ ] CHK024 - Is regression testing coverage defined (go test ./...) and is the expected outcome (zero failures) stated as a hard requirement? [Coverage]
- [ ] CHK025 - Does the spec define the verification approach for structural template conformance (manual review vs. automated check)? [Coverage]
- [ ] CHK026 - Are the clarifications (C-001 through C-005) incorporated back into the main requirements text, or do they exist only in the clarifications section? [Coverage]
