# Triage Review Criteria

Evaluate the triaged-findings output for sane bucketing.

## Criteria

1. **Bucket conservatism** — Findings the model cannot fix in a single bounded commit are NOT placed in `actionable`. Architectural / cross-cutting feedback is `deferred`.
2. **Rejection rationale** — Every entry in `rejected` carries a non-empty `reason` that names a concrete cause (false positive, duplicate, out-of-scope) — not generic dismissals.
3. **No invented findings** — Every triaged entry traces back to one entry in the input findings array. No hallucinated additions.
4. **No silent drops** — Total of actionable + deferred + rejected equals the input findings count (post-deduplication).
5. **Field preservation** — Bucketing carries Finding fields through verbatim. Severity is not inflated or deflated during triage.
6. **Actionable bucket is bounded** — The `actionable` array does not exceed ~10 items even on noisy reviews. Excess findings belong in `deferred` for human selection.

## Output Format

**CRITICAL: Your output must be ONLY the review JSON object. No preamble, no markdown fences, no wrapper text, no other output before or after the JSON.**

## Verdict

- **pass**: All criteria met. Triage is sane and the resolve step has a tractable workload.
- **warn**: Minor over-allocation to `actionable` or weak rejection reasons, but no invented or dropped findings.
- **fail**: Findings invented, dropped silently, or rejected without rationale.
