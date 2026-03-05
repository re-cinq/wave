# TUI Architecture Checklist: TUI Bubble Tea Scaffold (#252)

**Feature**: 252-tui-bubbletea-scaffold (part 1 of 10, parent: #251)
**Generated**: 2026-03-05
**Purpose**: Validate TUI-specific requirement quality for layout, interaction, and extensibility

---

## Layout Requirements

- [ ] CHK031 - Are the fixed heights for header (3 lines) and status bar (1 line) specified as requirements with rationale, or are they implementation details that leaked into the spec? [Clarity]
- [ ] CHK032 - Does the spec define behavior when content area height calculates to zero or negative (e.g., terminal exactly 4 rows: 3 header + 1 status = 0 content)? [Completeness]
- [ ] CHK033 - Is the minimum size threshold (80×24) justified? Does FR-008 explain why these specific dimensions were chosen and whether they should be configurable? [Clarity]
- [ ] CHK034 - Does the layout spec account for terminals wider than 200 columns? Are there maximum width constraints or does content stretch infinitely? [Completeness]
- [ ] CHK035 - Is the transition between "degradation message" and "normal layout" when resizing across the 80×24 threshold defined as instant or animated? [Clarity]

## Interaction Requirements

- [ ] CHK036 - Are all keyboard inputs that the TUI must respond to exhaustively listed? Beyond `q` and Ctrl-C, does the spec address or explicitly exclude other common keys (Escape, Ctrl-D, Ctrl-Z)? [Completeness]
- [ ] CHK037 - Does the spec define whether mouse input is enabled or disabled for the scaffold? Bubble Tea supports mouse events — should they be explicitly disabled? [Completeness]
- [ ] CHK038 - Is the `q` key exit behavior scoped correctly for future extensibility? When text input is added in later issues, pressing `q` should type the character, not exit. Does the spec acknowledge this? [Completeness]
- [ ] CHK039 - Does the spec define focus behavior? In the scaffold, is there a focused component, or are all components passive? [Completeness]

## Extensibility for Remaining 9 Issues

- [ ] CHK040 - Does the spec define the interface contract between `AppModel` and its child components clearly enough that future issues can add new views without modifying the spec? [Completeness]
- [ ] CHK041 - Does the spec acknowledge that `ContentModel` is a placeholder that will be replaced by real views? Is the replacement mechanism (swap, wrap, extend) defined? [Clarity]
- [ ] CHK042 - Are the component boundaries (header owns its state, content owns its state, etc.) specified clearly enough to prevent coupling that would complicate future issues? [Clarity]
- [ ] CHK043 - Does the spec define whether the status bar keybinding hints are hardcoded or context-sensitive? FR-006 says "context-sensitive" but the data model shows a fixed string. [Consistency]

## CLI Integration

- [ ] CHK044 - Does the spec define the precedence order for all TTY detection signals? (`--no-tui` > `WAVE_FORCE_TTY` > `TERM=dumb` > actual TTY check)? [Completeness]
- [ ] CHK045 - Is the behavior of `wave --no-tui` without any subcommand defined? Does it print help text, show version, or do something else? [Clarity]
- [ ] CHK046 - Does the spec address the interaction between `--debug` flag and TUI mode? Should debug logs appear in the TUI, go to stderr, or be suppressed? [Completeness]
- [ ] CHK047 - Is the `WAVE_FORCE_TTY` env var documented as a user-facing feature or an internal testing mechanism? This affects whether it needs help text and documentation. [Clarity]
