You are generating quality checklists to validate requirement completeness before
implementation.

Feature context: {{ input }}

## Working Directory

You are running in an **isolated git worktree** shared with previous pipeline steps.
Your working directory IS the project root. The feature branch was created by a
previous step and is already checked out.

## Instructions

Follow the `/speckit.checklist` workflow:

1. Find the feature directory and spec file path from the spec info artifact
2. Run `.specify/scripts/bash/check-prerequisites.sh --json` to get FEATURE_DIR
3. Load feature context: spec.md, plan.md, tasks.md
4. Generate focused checklists as "unit tests for requirements":
   - Each item tests the QUALITY of requirements, not the implementation
   - Use format: `- [ ] CHK### - Question about requirement quality [Dimension]`
   - Group by quality dimensions: Completeness, Clarity, Consistency, Coverage

5. Create the following checklist files in `FEATURE_DIR/checklists/`:
   - `review.md` — overall requirements quality validation
   - Additional domain-specific checklists as warranted by the feature

## CONSTRAINTS

- Do NOT spawn Task subagents — work directly in the main context
- Do NOT use WebSearch — all information is in the spec artifacts

## Checklist Anti-Patterns (AVOID)

- WRONG: "Verify the button clicks correctly" (tests implementation)
- RIGHT: "Are interaction requirements defined for all clickable elements?" (tests requirements)

## Output

Produce a JSON status report matching the injected output schema.
