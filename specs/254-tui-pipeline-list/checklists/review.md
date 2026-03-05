# Quality Review Checklist: TUI Pipeline List Left Pane

**Purpose**: Validate requirements quality across completeness, clarity, consistency, and coverage dimensions before implementation begins.  
**Feature**: #254 — TUI Pipeline List Left Pane  
**Created**: 2026-03-05  
**Spec**: [spec.md](../spec.md) | **Plan**: [plan.md](../plan.md) | **Tasks**: [tasks.md](../tasks.md)

---

## Completeness

- [ ] CHK001 - Are error handling requirements defined for all three data sources (SQLite for Running/Finished, manifest for Available)? [Completeness]
- [ ] CHK002 - Is the behavior specified when the PipelineDataProvider returns errors mid-session (after successful initial load)? [Completeness]
- [ ] CHK003 - Are loading/initialization states specified for the left pane before the first data fetch completes? [Completeness]
- [ ] CHK004 - Is the right pane placeholder content and behavior fully specified (dimensions, styling, alignment)? [Completeness]
- [ ] CHK005 - Are keyboard shortcut discoverability requirements defined (how does the user learn about `/` for filter, Enter for collapse)? [Completeness]
- [ ] CHK006 - Is the elapsed time update frequency for Running items specified (does it update live or only on 5s polling refresh)? [Completeness]
- [ ] CHK007 - Are the sort criteria for the Finished section specified (most recent first? by completion time or start time)? [Completeness]
- [ ] CHK008 - Is the maximum number of finished pipelines to display defined as a hard requirement (FR-014 says "default limit: 20" — is this configurable or fixed)? [Completeness]
- [ ] CHK009 - Are accessibility requirements defined for the pipeline list (screen reader support, high contrast mode)? [Completeness]
- [ ] CHK010 - Is the section ordering specified as fixed (Running → Finished → Available) or is it left to implementation? [Completeness]

## Clarity

- [ ] CHK011 - Is the exact visual layout of a Running item row unambiguous (name position, elapsed time position, separator/padding)? [Clarity]
- [ ] CHK012 - Is the exact visual layout of a Finished item row unambiguous (name, status icon, status text, duration — positioning and truncation priority)? [Clarity]
- [ ] CHK013 - Is "case-insensitive substring match" for the filter clearly defined for non-ASCII pipeline names (Unicode handling)? [Clarity]
- [ ] CHK014 - Are the collapsed/expanded indicators (▸/▾) clearly specified in terms of position (before or after section label)? [Clarity]
- [ ] CHK015 - Is the filter input activation behavior unambiguous (does `/` insert a `/` character or only activate the input)? [Clarity]
- [ ] CHK016 - Is "visual selection indicator" sufficiently precise — are the exact styling properties (color, bold, background) defined or left to the theme? [Clarity]
- [ ] CHK017 - Is the minimum terminal size for correct rendering specified (FR-001 says min 25 columns for left pane, but what about overall minimum)? [Clarity]

## Consistency

- [ ] CHK018 - Does the PipelineSelectedMsg contract match the existing header_messages.go definition exactly (field types, empty value semantics)? [Consistency]
- [ ] CHK019 - Does the 5-second polling interval align with the header's separate polling cycle, or could they race/conflict? [Consistency]
- [ ] CHK020 - Does the PipelineDataProvider interface follow the MetadataProvider pattern precisely (method naming, error return, constructor pattern)? [Consistency]
- [ ] CHK021 - Is the left pane width calculation (30%, min 25, max 50) consistent with how the header bar handles terminal width constraints? [Consistency]
- [ ] CHK022 - Does the "focused by default" requirement (FR-012) align with how other TUI components handle focus (are there competing focus claims)? [Consistency]
- [ ] CHK023 - Is the NO_COLOR requirement (FR-018) consistent with how existing TUI components (header, status bar) handle NO_COLOR? [Consistency]
- [ ] CHK024 - Does the section collapse toggle key (Enter) conflict with any existing key bindings in the TUI? [Consistency]

## Coverage

- [ ] CHK025 - Is the behavior specified when a pipeline run finishes while the user has it selected in the Running section (cursor stability)? [Coverage]
- [ ] CHK026 - Is the behavior specified when the user is in filter mode and a data refresh changes the filtered results? [Coverage]
- [ ] CHK027 - Is the behavior specified when terminal resize occurs during active filter mode? [Coverage]
- [ ] CHK028 - Is the behavior specified when multiple pipelines share the same name (duplicate names across sections)? [Coverage]
- [ ] CHK029 - Is the behavior specified for very rapid ↑/↓ key repeats (debouncing or queue handling)? [Coverage]
- [ ] CHK030 - Is the behavior specified when the database becomes unavailable after initial load (stale data display vs error state)? [Coverage]
- [ ] CHK031 - Is the behavior specified when a collapsed section receives new items via polling refresh (does it stay collapsed)? [Coverage]

---

**Total items**: 31  
**Dimensions**: Completeness (10), Clarity (7), Consistency (7), Coverage (7)
