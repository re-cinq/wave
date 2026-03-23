# Requirements Quality Review Checklist

**Feature**: #558 — Migrate Adapter to Agent-Based Execution
**Date**: 2026-03-23

---

## Completeness

- [ ] CHK001 - Are all agent file frontmatter fields fully enumerated with their types, defaults, and optionality? [Completeness]
- [ ] CHK002 - Is the behavior defined for when `PersonaToAgentMarkdown` receives nil vs empty slices for `AllowedTools` and `DenyTools`? [Completeness]
- [ ] CHK003 - Are error handling requirements specified for agent file write failures (disk full, permission denied)? [Completeness]
- [ ] CHK004 - Is the expected content of the agent file body fully specified beyond the four-layer ordering (e.g., separator format between layers)? [Completeness]
- [ ] CHK005 - Are rollback or fallback requirements defined if `--agent` flag is not supported by the installed Claude Code version? [Completeness]
- [ ] CHK006 - Is the `permissionMode: dontAsk` value documented as a hard requirement or does the spec address other valid values? [Completeness]
- [ ] CHK007 - Are requirements specified for cleaning up `.claude/wave-agent.md` after step completion or failure? [Completeness]
- [ ] CHK008 - Does the spec define what happens when `cfg.Model` is empty — is the `model:` field omitted from frontmatter? [Completeness]

## Clarity

- [ ] CHK009 - Is the distinction between "sandbox-only settings.json" and "no settings.json" unambiguous for all sandbox configuration combinations? [Clarity]
- [ ] CHK010 - Is "tool lists passed through without modification" clearly defined — does it mean no deduplication, no sorting, no case normalization? [Clarity]
- [ ] CHK011 - Is the agent file path `.claude/wave-agent.md` specified as absolute or relative, and relative to what root? [Clarity]
- [ ] CHK012 - Is the scope boundary between `ClaudeAdapter` changes and `chatworkspace.go` exclusion clearly articulated with no ambiguous overlap? [Clarity]
- [ ] CHK013 - Are the retained CLI flags (`--output-format`, `--verbose`, `--no-session-persistence`) specified as always-present or conditionally-present? [Clarity]
- [ ] CHK014 - Is the phrase "identical results" in US1's independent test measurable and testable, or does it need quantification? [Clarity]

## Consistency

- [ ] CHK015 - Is FR-004's list of removed flags (`--allowedTools`, `--disallowedTools`, `--dangerously-skip-permissions`, `--model`) consistent with buildArgs Phase D changes in the plan? [Consistency]
- [ ] CHK016 - Does the TodoWrite injection site (C-004: in `prepareWorkspace`) align with the test plan tasks T029/T030 that verify it in agent frontmatter? [Consistency]
- [ ] CHK017 - Is the `SandboxOnlySettings` type in the data model consistent with FR-003 and FR-009 in the spec? [Consistency]
- [ ] CHK018 - Does the edge case "Empty tool lists" (omit fields entirely) agree with FR-001's requirement that every agent file contains `tools` and `disallowedTools` fields? [Consistency]
- [ ] CHK019 - Are US2's acceptance scenarios (normalizeAllowedTools deleted) consistent with US1's scenarios (agent frontmatter generated with tools)? [Consistency]
- [ ] CHK020 - Is the plan's Phase E test list fully aligned with the tasks.md T015-T032 task list? [Consistency]

## Coverage

- [ ] CHK021 - Is there an edge case defined for personas with `allowed_tools` but empty `deny` list (and vice versa)? [Coverage]
- [ ] CHK022 - Is the concurrent step scenario (edge case) covered by a specific acceptance test or only stated as a requirement? [Coverage]
- [ ] CHK023 - Is the temperature field intentional-drop documented with an acceptance scenario or test to verify it doesn't appear in agent frontmatter? [Coverage]
- [ ] CHK024 - Is there a test scenario for `wave agent export` producing output matching the new runtime format (FR-010)? [Coverage]
- [ ] CHK025 - Are cross-adapter impacts assessed — does the spec confirm `BrowserAdapter`, `OpenCodeAdapter`, `GitHubAdapter` have no `UseAgentFlag` references? [Coverage]
- [ ] CHK026 - Is there a scenario covering workspace paths with special characters (spaces, unicode) in the `--agent` flag value? [Coverage]
- [ ] CHK027 - Is there a validation scenario for the `SandboxOnlySettings` JSON output to ensure no extra fields leak through? [Coverage]
- [ ] CHK028 - Are the 8 `TestNormalizeAllowedTools` cases enumerated to confirm none test behavior that should survive the migration? [Coverage]
