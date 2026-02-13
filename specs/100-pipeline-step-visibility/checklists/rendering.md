# Rendering Requirements Quality Checklist

**Feature**: Pipeline Step Visibility in Default Run Mode
**Spec**: `specs/100-pipeline-step-visibility/spec.md`
**Date**: 2026-02-14
**Focus**: Visual rendering, TUI layout, and display behavior requirements

## Visual Design Completeness

- [ ] CHK101 - Are all six indicator characters defined with their exact Unicode code points (not just visual descriptions)? [Completeness]
- [ ] CHK102 - Is the spinner character set for the running state defined precisely (braille set `⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏`), or does the spec rely on implicit "existing behavior"? [Completeness]
- [ ] CHK103 - Is the step list indentation or alignment specified? Should all step names start at the same column, or are indicators variable-width? [Completeness]
- [ ] CHK104 - Are ASCII fallback indicators defined for all six states for terminals that don't support Unicode? The spec mentions `UnicodeCharSet`/`AsciiOnly` fallback but doesn't define ASCII equivalents for pending (`○`), skipped (`—`), or cancelled (`⊛`). [Completeness]
- [ ] CHK105 - Is the vertical spacing between step lines defined (single line, double-spaced, blank line separators)? [Completeness]
- [ ] CHK106 - Does the spec define whether the running step's additional context lines (tool activity, current action) are indented below the step line or rendered inline? [Completeness]

## Layout and Composition Clarity

- [ ] CHK107 - Is the relationship between the progress bar component and the step list component clearly defined? Does the step list appear above, below, or replace the progress bar? [Clarity]
- [ ] CHK108 - Is the rendering order within a single step line unambiguous: indicator, then step name, then persona in parens, then timing in parens? Could "persona (timing)" be confused with "step-name (persona)"? [Clarity]
- [ ] CHK109 - For the deliverable tree attached to completed steps, is the visual nesting relationship to the step line clear enough to implement? [Clarity]
- [ ] CHK110 - Is the "pulsating effect" for the running step in the Dashboard path defined with enough specificity to replicate, or does the spec depend on existing behavior? [Clarity]

## Multi-Path Consistency

- [ ] CHK111 - Are the rendering requirements for the BubbleTea path and the Dashboard path specified to produce visually equivalent output, or are differences acceptable? [Consistency]
- [ ] CHK112 - Is the `getStatusIcon()` update in the Dashboard path required to use the same Unicode characters as the BubbleTea path, or can the two paths use different icon sets? [Consistency]
- [ ] CHK113 - Does the plan's instruction to "remove the blank line separator" (Change 4) align with any spec requirement, or is it an implementation-level decision that should be elevated to the spec? [Consistency]
- [ ] CHK114 - Is the color palette for each state consistent between the BubbleTea path (lipgloss colors) and the Dashboard path (ANSI colors)? [Consistency]

## Edge Case Coverage

- [ ] CHK115 - Is the "rapid step completion" edge case (step completes in <1s) testable with a specific rendering assertion, or only described qualitatively? [Coverage]
- [ ] CHK116 - Is the behavior for extremely long step names or persona names defined (truncation, wrapping, or overflow)? [Coverage]
- [ ] CHK117 - Is the atomicity requirement (no frame shows two spinners) testable at the rendering layer, or does it depend on the event layer guarantees? [Coverage]
- [ ] CHK118 - Is the step list rendering behavior defined for a pipeline with zero steps (degenerate case)? [Coverage]
