# Quality Review Criteria

Evaluate the analysis output for completeness and actionability.

## Criteria

1. **Specificity** — Findings reference specific files, functions, or line numbers, not vague generalities
2. **Actionability** — Each finding includes a concrete remediation suggestion
3. **Completeness** — The analysis covers the full scope of changes, not just surface-level observations
4. **Severity accuracy** — Severity ratings are proportional to actual impact
5. **No hallucination** — Findings reference code that actually exists in the diff

## Verdict

- **pass**: All criteria met, findings are substantive and actionable
- **warn**: Minor gaps in specificity or completeness, but core findings are sound
- **fail**: Findings are vague, generic, or reference non-existent code
