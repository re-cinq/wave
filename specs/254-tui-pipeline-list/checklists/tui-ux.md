# TUI/UX Quality Checklist: TUI Pipeline List Left Pane

**Purpose**: Validate TUI-specific and user experience quality concerns for the pipeline list component.  
**Feature**: #254 — TUI Pipeline List Left Pane  
**Created**: 2026-03-05  
**Spec**: [spec.md](../spec.md)

---

## Visual Hierarchy & Layout

- [ ] CHK101 - Are visual weight rules defined for section headers vs pipeline items (font weight, indentation, spacing) to establish clear hierarchy? [Visual Hierarchy]
- [ ] CHK102 - Is the vertical spacing between sections specified (blank line, divider, or flush)? [Visual Hierarchy]
- [ ] CHK103 - Is the truncation strategy for pipeline names specified in priority order (which metadata is truncated first when space is constrained)? [Visual Hierarchy]
- [ ] CHK104 - Are color/styling requirements expressed as semantic roles (e.g., "success", "error") rather than hard-coded values, ensuring theme adaptability? [Visual Hierarchy]

## Interaction Design

- [ ] CHK105 - Is focus transfer behavior defined (how does the user move focus from left pane to future right pane and back)? [Interaction]
- [ ] CHK106 - Are the keymap conflicts documented between filter mode (text input active) and normal mode (↑/↓ navigation)? [Interaction]
- [ ] CHK107 - Is the transition animation/rendering specified when switching between filter mode and normal mode (smooth or instant)? [Interaction]
- [ ] CHK108 - Is the scroll position preservation behavior defined when exiting and re-entering filter mode? [Interaction]
- [ ] CHK109 - Is the cursor placement rule specified after a data refresh adds/removes items (preserve selection by identity or by index)? [Interaction]

## Responsiveness & Performance

- [ ] CHK110 - Are rendering performance requirements specified for the target item count (100 items per SC-004)? [Performance]
- [ ] CHK111 - Is the minimum usable terminal size documented as a requirement (not just the left pane minimum of 25 columns)? [Performance]
- [ ] CHK112 - Is the behavior specified when the terminal height is too small to show even one item per section? [Performance]

## Data Display

- [ ] CHK113 - Is the elapsed time format specified precisely (e.g., "2m30s" vs "2:30" vs "2 minutes")? [Data Display]
- [ ] CHK114 - Is the duration format for Finished items specified precisely (same format as elapsed time?)? [Data Display]
- [ ] CHK115 - Are the status indicator symbols specified (✓/✗) and their fallback for terminals without Unicode support? [Data Display]
- [ ] CHK116 - Is the "cancelled" status display specified with the same precision as "completed" and "failed"? [Data Display]
- [ ] CHK117 - Is the empty section body display specified (blank, placeholder text like "No running pipelines", or section hidden)? [Data Display]

---

**Total items**: 17  
**Dimensions**: Visual Hierarchy (4), Interaction (5), Responsiveness (3), Data Display (5)
