# Display & UX Requirements Checklist

**Feature**: Pipeline Output UX — Surface Key Outcomes
**Spec**: `specs/120-pipeline-output-ux/spec.md`
**Date**: 2026-02-20

This checklist validates the quality of display-related and UX-related requirements.

---

## Terminal Rendering

- [ ] CHK101 - Are the formatting primitives to be used for each outcome element specified (Bold, Success, Muted, Primary, Warning, Error)? [Clarity]
- [ ] CHK102 - Is the indentation/nesting structure of the outcomes section defined (e.g., header at column 0, items indented 2 spaces)? [Clarity]
- [ ] CHK103 - Are separator/divider requirements between the outcomes section and subsequent output defined? [Completeness]
- [ ] CHK104 - Is the "Next Steps" arrow format ("  → Label") specified with enough precision, including behavior when the label contains a long URL? [Clarity]
- [ ] CHK105 - Are icon requirements for each outcome type specified for both Unicode-capable and ASCII-fallback terminals? [Completeness]
- [ ] CHK106 - Is the behavior defined for when Formatter methods (Bold, Success, etc.) are unavailable or return empty in degraded terminals? [Completeness]

## Output Mode Interactions

- [ ] CHK107 - Is the interaction between `--verbose` and `--output json` defined (does verbose affect JSON output fields)? [Completeness]
- [ ] CHK108 - Is the rendering target (stdout vs stderr) for the outcomes section explicitly specified for each output mode? [Clarity]
- [ ] CHK109 - Is the behavior defined for `--output text` (non-TUI) versus `--output auto` on a TTY — are outcomes rendered identically? [Clarity]
- [ ] CHK110 - Is there a requirement for how the outcomes section interacts with the BubbleTea TUI teardown in auto mode? [Completeness]

## Narrow Terminal / Responsive Layout

- [ ] CHK111 - Is the minimum supported terminal width for outcomes rendering defined (the spec mentions <60 columns but doesn't define a hard minimum)? [Completeness]
- [ ] CHK112 - Are truncation rules for long URLs specified (middle truncation, end truncation, or full display with wrapping)? [Clarity]
- [ ] CHK113 - Is the behavior for outcome lines that exceed terminal width defined (wrap, truncate, or ellipsis)? [Clarity]

## Accessibility & Readability

- [ ] CHK114 - Are color choices specified with consideration for color-blind users (e.g., not relying solely on red/green distinction)? [Completeness]
- [ ] CHK115 - Is the "first 5 lines" measurability criterion in SC-001 defined relative to a specific terminal width or assumed standard (80 columns)? [Clarity]
- [ ] CHK116 - Are the conditions for "actionable follow-ups are available" in FR-011 exhaustively enumerated, or is this an open-ended heuristic? [Clarity]
