# Audit Doc Review Criteria

Evaluate the synthesised audit document against the audit-issue quality bar. The agent_review handover for the synthesize step uses these criteria to decide pass / warn / fail.

## Criteria

1. **Novel signal first** — The opening paragraph leads with what the audit discovered that was NOT already stated in the issue body. Recap of the issue body is bounded to ≤25% of the doc length.
2. **Severity rubric, no count target** — Every gap is classified `critical | high | medium | low | info`. The doc does NOT pad to hit a count target ("10 each", "exactly 5"). If the audit found 3 high-severity gaps and nothing else, the doc reports 3 — not 10 with hairsplit additions.
3. **File:line edit anchor on every code-tier recommendation** — For each gap whose `axis` is `code` (or `cross-axis` involving code), the recommendation names a concrete `path/to/file.go:line` anchor a craftsman could open and edit. "Refactor the auth layer" without a file:line pointer fails this criterion.
4. **Screenshots inlined for UX-tier gaps** — For every gap with `ux_tier: true` (or `axis: webui`), the doc inlines the captured screenshot from `.agents/output/screenshots/` via a relative markdown image link. A UX-tier gap without an inlined screenshot fails this criterion.
5. **Follow-up spec emitted** — A `followup-specs.json` file (or equivalent typed artifact) sits alongside the audit doc, with one entry per high-or-above gap. Drafting the follow-up in the body alone — without the typed artifact — fails this criterion (the file-each-followup step relies on the artifact).
6. **No speckit boilerplate** — The deliverable is the audit doc + screenshots + follow-up specs + (optionally) PR. Speckit's `specs/<branch>/{spec,plan,tasks}.md` shape does NOT belong in a doc-only audit deliverable.
7. **Cross-axis correlations called out** — If two or more evidence axes surface the same root cause, the doc explicitly links them under one gap rather than reporting them as duplicate findings.

## Verdict

- **pass**: All seven criteria satisfied. Audit doc is ready to ship as-is.
- **warn**: 1-2 criteria partially met (e.g. one UX-tier gap missing its screenshot). Pipeline proceeds; the deficit is logged in the run summary.
- **fail**: 3+ criteria missed, or any of #2 (count padding) / #4 (missing screenshots for UX-tier) / #5 (no typed follow-up spec) is violated outright. Pipeline routes to the revise loop (`max_iterations: 2`).
