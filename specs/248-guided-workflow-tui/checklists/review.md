# Requirements Quality Review Checklist

**Feature**: Guided Workflow Orchestrator TUI (#248)
**Spec**: `specs/248-guided-workflow-tui/spec.md`

---

## Completeness

- [ ] CHK001 - Are all view states in the state machine (Health, Proposals, Fleet, Attached) fully specified with entry conditions, exit conditions, and allowed transitions? [Completeness]
- [ ] CHK002 - Are error handling requirements defined for each view state (e.g., what happens if health checks fail, if proposal launch fails, if fleet view loses connection to a running pipeline)? [Completeness]
- [ ] CHK003 - Is the behavior of the `n` key (manual launch from empty proposals) fully specified — what screen appears, what inputs are required, what pipeline list is shown? [Completeness]
- [ ] CHK004 - Are requirements defined for what happens when the user returns to Proposals after launching pipelines — are launched proposals marked, removed, or re-shown? [Completeness]
- [ ] CHK005 - Is the modify overlay (`m` key) fully specified — field dimensions, validation rules, maximum input length, multi-line support? [Completeness]
- [ ] CHK006 - Are requirements defined for the health summary header data format, refresh behavior, and staleness handling in the proposals view? [Completeness]
- [ ] CHK007 - Is the archive divider behavior specified for edge cases — what happens with zero running pipelines, zero completed pipelines, or all pipelines in one category? [Completeness]
- [ ] CHK008 - Are keyboard shortcut conflicts documented — does `s` (skip) conflict with any existing binding in the suggest view? Does `m` (modify) conflict? [Completeness]
- [ ] CHK009 - Is the Ctrl+C behavior fully specified across all guided mode states (Health, Proposals, Fleet, Attached)? [Completeness]
- [ ] CHK010 - Are requirements defined for what happens when new proposals arrive while the user is in Fleet view — is there a notification, badge count, or auto-refresh? [Completeness]

## Clarity

- [ ] CHK011 - Is the term "proposal" unambiguously distinguished from "suggestion" and "pipeline" — are these used consistently throughout the spec? [Clarity]
- [ ] CHK012 - Is the meaning of "auto-transition" precisely defined — does it mean immediate switch, animated transition, or timed delay with user override? [Clarity]
- [ ] CHK013 - Is the DAG preview format explicitly defined — is it ASCII art, box-drawing characters, or styled text? Are example renderings provided for single, sequence, and parallel types? [Clarity]
- [ ] CHK014 - Is the "archive divider" visual specification clear — line style, color, label text, positioning relative to items? [Clarity]
- [ ] CHK015 - Is the "dimmed" rendering for skipped proposals precisely defined — opacity level, color, strikethrough, or other visual treatment? [Clarity]
- [ ] CHK016 - Are the sequence grouping indicators (pending, running, completed) precisely defined — which Unicode glyphs, colors, and indentation levels? [Clarity]
- [ ] CHK017 - Is "within 500ms" (SC-002) measured from the last check result arriving or from the last check rendering — could render latency cause this to fail? [Clarity]

## Consistency

- [ ] CHK018 - Is the Tab key behavior consistent — spec says Tab toggles Proposals/Fleet, but US4 acceptance #6 also mentions `p` for the same toggle. Are both required or is one preferred? [Consistency]
- [ ] CHK019 - Is the "combined execution plan" (US3 acceptance #3) consistent with the "DAG preview" (US2 acceptance #3) — are these the same rendering or different formats? [Consistency]
- [ ] CHK020 - Are the FR-017 and FR-018 execution mechanisms consistent with the existing `PipelineLauncher` API — does `LaunchSequence` support both sequence and parallel as specified? [Consistency]
- [ ] CHK021 - Is the health summary in FR-003 ("12 open issues, 3 PRs awaiting review, all deps OK") consistent with the data sources identified in clarification C2 (infrastructure vs codebase health)? [Consistency]
- [ ] CHK022 - Are success criteria timing requirements (SC-001: 200ms first frame, SC-002: 500ms transition) consistent with the actual data loading costs of `HealthDataProvider` and `SuggestDataProvider`? [Consistency]
- [ ] CHK023 - Is the quit behavior (US5 acceptance #7: `q` exits, Ctrl+C for cancel-first) consistent across all states including when the input overlay is active? [Consistency]

## Coverage

- [ ] CHK024 - Are accessibility requirements addressed — keyboard-only navigation, screen reader compatibility, high-contrast terminal support? [Coverage]
- [ ] CHK025 - Are multi-monitor or SSH session scenarios covered — does the TUI degrade gracefully in non-interactive or limited terminal environments? [Coverage]
- [ ] CHK026 - Are concurrent user action requirements covered — what happens if the user presses Tab while a launch is in progress, or presses Enter twice quickly? [Coverage]
- [ ] CHK027 - Are data refresh requirements covered — how often does the fleet view poll for pipeline status updates, and is there a manual refresh mechanism? [Coverage]
- [ ] CHK028 - Are requirements defined for persistence across TUI restarts — if the user exits and re-launches `wave`, do they see previously launched pipelines in the fleet view? [Coverage]
- [ ] CHK029 - Is the interaction between guided mode and the existing secondary view switcher (mentioned in C1: `[`/`]` or `v` key) fully specified? [Coverage]
- [ ] CHK030 - Are requirements defined for how the TUI handles pipelines launched outside the TUI (via `wave run`) appearing in the fleet view? [Coverage]
