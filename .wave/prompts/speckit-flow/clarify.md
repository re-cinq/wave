You are refining a feature specification by identifying and resolving ambiguities.

Feature context: {{ input }}

## Working Directory

You are running in an **isolated git worktree** shared with previous pipeline steps.
Your working directory IS the project root. The feature branch was created by a
previous step and is already checked out.

A status report from the previous step is available at `.wave/artifacts/spec_info`.
Read it to find the branch name, spec file, and feature directory.

## Instructions

Follow the `/speckit.clarify` workflow:

1. Read `.wave/artifacts/spec_info` to find the feature directory and spec file path
2. Run `.specify/scripts/bash/check-prerequisites.sh --json --paths-only` to confirm paths
3. Load the current spec and perform a focused ambiguity scan across:
   - Functional scope and domain model
   - Integration points and edge cases
   - Terminology consistency
4. Generate up to 5 clarification questions (prioritized)
5. For each question, select the best option based on codebase context
6. Integrate each resolution directly into the spec file
7. Save the updated spec

## CONSTRAINTS

- Do NOT spawn Task subagents — work directly in the main context
- Do NOT use WebSearch — all clarifications should be resolved from codebase
  context and the existing spec. The specify step already did the research.
- Keep the scope tight: only fix genuine ambiguities, don't redesign the spec

## Non-Interactive Mode

Since this runs in a pipeline, resolve all clarifications autonomously:
- Select the recommended option based on codebase patterns and existing architecture
- Document the rationale for each choice in the Clarifications section
- Err on the side of commonly-accepted industry standards

## Output

Write a JSON status report to .wave/output/clarify-status.json with:
```json
{
  "clarifications_resolved": 3,
  "sections_updated": ["section1", "section2"],
  "spec_file": "path to updated spec.md",
  "feature_dir": "path to feature directory",
  "summary": "brief description of clarifications made"
}
```
