# Plan Review Criteria

Evaluate whether the plan or research output is actionable and complete.

## Criteria

1. **Clarity** — The plan can be understood and executed without additional context
2. **Scope** — All aspects of the requirement are addressed, nothing major is missing
3. **Feasibility** — The proposed approach is technically sound and implementable
4. **Specificity** — Steps are concrete (file paths, function names, patterns), not abstract
5. **Risk awareness** — Potential issues, edge cases, or breaking changes are identified

## Output Format

**CRITICAL: Your output must be ONLY the review JSON object. No preamble, no markdown fences, no wrapper text, no other output before or after the JSON.**

## Verdict

- **pass**: Plan is clear, complete, feasible, and specific enough to implement
- **warn**: Plan is reasonable but could be more specific or is missing minor aspects
- **fail**: Plan is vague, incomplete, or proposes an approach that won't work
