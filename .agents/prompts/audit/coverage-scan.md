## Objective

Perform a requirements coverage audit by parsing the issue's acceptance criteria and
validating that the implementation addresses each criterion. Your job is to create a
traceability map from requirements to implementation, identifying unaddressed or partially
addressed requirements. This is the reconnaissance phase — breadth over depth.

## Context

This is the first step of a two-step coverage audit pipeline. The issue assessment artifact
from the implementation step describes the acceptance criteria and requirements. You have
read-only access to the entire project. Your output feeds into an aggregation step that
merges findings across all audit dimensions.

## Requirements

1. **Extract acceptance criteria**: Parse the issue assessment artifact to identify all
   acceptance criteria, functional requirements, and success metrics. Create an explicit
   checklist of what the implementation must deliver.

2. **Trace each criterion to implementation**: For each acceptance criterion:
   - Find the code that addresses it (file path, function, line range)
   - Determine if the criterion is fully addressed, partially addressed, or unaddressed
   - Note the evidence (what code exists and what it does)
   - Rate the coverage: complete, partial, or missing

3. **Identify gaps**: For each unaddressed or partially addressed criterion:
   - Describe what is missing
   - Assess the severity: is this a core requirement (critical) or a nice-to-have (low)?
   - Suggest what needs to be added

4. **Check for scope creep**: Identify any implementation that goes beyond the stated
   requirements. This is informational, not necessarily a problem — but it should be
   visible for review.

5. **Validate success metrics**: If the issue specifies measurable success criteria
   (e.g., "response time under 100ms", "100% of existing tests pass"), verify whether
   the implementation includes validation for those metrics.

## Constraints and Anti-patterns

- Do NOT assess code quality, architecture, or security. This is a coverage audit only.
- Do NOT flag implementation approaches as wrong — only flag whether requirements are met.
- Do NOT modify any files. This is a read-only scan.
- Do NOT invent requirements that are not in the issue assessment. Only trace what was
  explicitly requested.

## Output Format

Write your findings to the output artifact path matching the contract schema. Each finding
must include: type="coverage", severity (critical/high/medium/low/info), the affected file
path (or "N/A" for missing implementations), a description of the coverage gap, evidence
of what exists or is missing, and a recommendation (fix/investigate).

## Quality Bar

A passing coverage scan creates a complete traceability map from every acceptance criterion
to its implementation (or lack thereof). Missing an unaddressed core requirement constitutes
a failure. Flagging a fully-addressed requirement as missing also constitutes a failure.
