## Objective

Analyze the aggregated findings from all audit pipelines and produce a pass/fail verdict
determining whether the implementation meets the quality bar for PR creation, or whether
it needs rework.

## Context

The injected `aggregated_findings` artifact contains merged findings from up to five
audit dimensions (security, correctness, architecture, tests, coverage). Each finding
has a severity level and source audit. Your job is to synthesize these into a binary
decision: pass (proceed to PR creation) or fail (trigger rework loop).

## Severity Mapping

Input findings use the shared-findings severity scale. Map to the verdict scale:
- `critical` -> `critical` (always blocks)
- `high` -> `major` (blocks by default)
- `medium` -> `minor` (does not block)
- `low` -> `suggestion` (does not block)
- `info` -> `suggestion` (does not block)

## Gate Logic

Apply these rules strictly:

1. **Any critical finding** -> decision: `fail`
2. **Any major (high) finding** -> decision: `fail`
3. **Only minor/suggestion findings** -> decision: `pass`
4. **No findings** -> decision: `pass`

The severity threshold can be overridden by pipeline configuration. If the configured
threshold is "minor", then minor findings also block. The default threshold is "major".

## Requirements

1. **Count findings by mapped severity**: Produce the `findings_summary` object with
   counts for critical, major, minor, and suggestion.

2. **Determine decision**: Apply the gate logic above.

3. **Write reason**: Explain the verdict in one sentence referencing the finding counts.

4. **Set iteration**: Use the current rework iteration number (1 for first pass).

5. **On fail — produce aggregated_feedback**: Concatenate all blocking findings
   (critical + major) into a structured feedback string that the rework step can use.
   Format: one finding per line with `[source_audit] severity: file:line — description`.

6. **On fail — list blocking_findings**: Include the array of findings that caused the
   gate to fail, with source_audit, severity, file, and description.

## Constraints

- Do NOT change finding severities or reinterpret them. Map mechanically.
- Do NOT add subjective assessment beyond the gate logic. The gate is algorithmic.
- Do NOT pass when blocking findings exist, regardless of context or justification.
- Do NOT fail when only non-blocking findings exist, regardless of their count.

## Output Format

Produce JSON output matching the rework-gate-verdict schema at the output artifact path.
