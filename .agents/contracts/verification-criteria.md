# Verification Criteria

Evaluate whether the verification step produced a sound go/no-go recommendation.

## Criteria

1. **Evidence-based** — Verdict cites specific test results, code inspection findings, or behavioral checks
2. **Completeness** — All changed code paths were verified, not just the happy path
3. **Risk assessment** — Edge cases and failure modes were considered
4. **Regression check** — Existing functionality was verified to still work
5. **Clear verdict** — The recommendation is unambiguous (PASS/FAIL with justification)

## Output Format

**CRITICAL: Your output must be ONLY the review JSON object. No preamble, no markdown fences, no wrapper text, no other output before or after the JSON.**

## Verdict

- **pass**: Verdict is well-justified with evidence covering all changed paths
- **warn**: Verdict is reasonable but missing edge case coverage
- **fail**: Verdict lacks evidence or misses obvious verification gaps
