# feat(pipelines): scope audits to PR diff and improve ops-pr-respond signal-to-noise

- **Issue**: [re-cinq/wave#1411](https://github.com/re-cinq/wave/issues/1411)
- **Author**: nextlevelshit
- **Labels**: enhancement
- **State**: OPEN
- **Branch**: `1411-scope-audits-pr`

## Context

`ops-pr-respond` (introduced in #1407) ran on PR #1407 itself and produced this triage shape:

- **127 raw findings** across six audit axes
- **19 actionable** (per-finding fixes applied via `impl-finding`)
- **6 deferred** (security findings requiring design decisions)
- **~102 rejected** (out-of-PR-scope)

The PR touches 10 files (two new pipelines, one new schema, AGENTS.md, three spec docs, plus `internal/defaults` mirrors). Yet the architecture, test-coverage, duplicate-code, and dead-code audits all returned findings on unrelated `internal/*` and `cmd/*` files — every one of those was rejected as out-of-PR-scope by the triage step.

The two axes that produced relevant signal:

- `audit-security` — 13 findings on the new YAML/schema (7 actionable, 6 deferred for design)
- `audit-doc-scan` — 12 findings on the new pipelines + spec docs (12 actionable)

## Problem

Audit pipelines run with no awareness of PR scope when invoked through `ops-pr-respond`'s `parallel-review` step. They scan the whole repo, produce findings everywhere, and rely on the downstream `triage` step to filter out-of-scope noise. Token-wise this means:

1. Each audit re-walks the whole repository.
2. Triage spends most of its budget rejecting findings, not classifying them.
3. The signal-to-noise ratio is ~15% (19 useful / 127 raw).

## Proposed improvements

### 1. PR-scoped audit prompts

Add a `scope_files` / `scope_diff` input to each `audit-*` pipeline. When invoked via `ops-pr-respond`, the parent passes:

- `pr_context.changed_files` as `scope_files`
- `.agents/output/pr.diff` as `scope_diff`

The audit prompt then constrains itself: "Only flag issues in `<scope_files>`. The diff at `<scope_diff>` is the authoritative scope."

### 2. Triage skip for empty axes

If an audit returns zero in-scope findings, skip the triage step's normalisation of that axis entirely. Currently triage parses heterogeneous markdown blobs from every audit even when nothing applies.

### 3. Pre-triage out-of-scope filter

A small, deterministic filter step between `merge-findings` and `triage`:

- Drop findings whose `file` is not in `pr_context.changed_files`.
- Drop findings whose cited symbol does not appear in the diff.

This shrinks the planner's input from 127 to ~25 before the LLM-based classification runs.

### 4. Defer-by-design output bucket

The 6 security findings deferred for design decisions (prompt-injection threat model, parallel push race, build-time deduplication) need a structured handover. Either:

- `triaged-findings.json` gains a `design_questions` array with one entry per deferred-by-design item, and `comment-back` posts those as a separate "Design questions" section.
- Or `comment-back` opens a follow-up issue per deferred design item.

### 5. Prompt hardening for the new pipelines themselves

Independent of the scoping work: tighten `ops-pr-respond` and `impl-finding` prompts so their fan-out items don't redundantly fetch PR metadata, re-clone the repo, or re-parse the diff. The first run had each `impl-finding` child shell-out to `gh` even though `pr_context` was already injected.

## Acceptance Criteria

- [ ] `ops-pr-respond` on a 10-file PR produces ≤ 30 raw findings (down from 127) before triage.
- [ ] Triage signal-to-noise rises to ≥ 60% actionable.
- [ ] Deferred-by-design items surface as either a structured PR section or follow-up issues, not silent dropouts.
- [ ] No audit step writes findings against files outside `pr_context.changed_files`.

## Source

Empirical baseline: run `ops-pr-respond-20260426-203623-b73e` on PR #1407.
