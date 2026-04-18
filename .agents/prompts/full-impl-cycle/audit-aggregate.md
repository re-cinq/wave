## Objective

Merge findings from all audit pipelines (security, correctness, architecture, tests,
coverage) into a single aggregated findings document. Your job is to flatten all findings
into one array, tag each with its source audit, deduplicate overlapping findings, and
produce a summary of audit results.

## Context

You have access to findings artifacts from up to five audit pipelines. Each artifact
follows the shared-findings schema with a `findings` array. Some audits may have failed
or been skipped — include their status in the `source_audits` tracking array.

Read all available findings artifacts in the injected workspace.

## Requirements

1. **Collect all findings**: Read each injected audit findings artifact. For each finding,
   add a `source_audit` field identifying which audit pipeline produced it.

2. **Deduplicate**: If multiple audits flag the same file+line with overlapping concerns
   (e.g., correctness and coverage both flag a missing feature), keep the higher-severity
   finding and note both sources.

3. **Track source audits**: Produce a `source_audits` array listing each audit pipeline
   name, the number of findings it contributed, and its completion status (completed,
   failed, skipped).

4. **Compute totals**: Set `total_findings` to the count of deduplicated findings.

5. **Generate summary**: Write a one-line summary: "N findings from M audits
   (X critical, Y high, Z medium)".

## Severity Mapping

The input findings use severity levels: critical, high, medium, low, info.
Preserve these in the output — do NOT map to the major/minor/suggestion scale.
The rework gate step handles that mapping.

## Constraints

- Do NOT modify finding descriptions or evidence. Pass them through as-is.
- Do NOT add new findings. Only merge what the audit pipelines produced.
- Do NOT skip failed or empty audits — record them in `source_audits` with
  `finding_count: 0` and `status: failed` or `status: skipped`.

## Output Format

Produce JSON output matching the aggregated-findings schema at the output artifact path.
