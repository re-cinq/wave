# Surface Parity Checklist: 717-run-options-parity

**Generated**: 2026-04-11 | **Scope**: Cross-surface option coverage quality

This checklist validates that the spec adequately defines option availability and behavior for each surface, ensuring the "parity" claim is substantiated by requirements.

## Option-Surface Matrix Completeness

- [x] CHK-SP001 - Does the spec define which tiers are exposed on each surface (not just "Tier 1-3" but which specific options per surface)? [Completeness]
- [x] CHK-SP002 - Is the Input field (Tier 1) specified for surfaces beyond the pipeline detail page (issues/PRs pages may not need free-text input)? [Completeness]
- [x] CHK-SP003 - Are the TUI launcher's new fields enumerated individually in the spec (not just "expand to Tier 1-3")? [Completeness]
- [x] CHK-SP004 - Does the spec clarify whether the TUI exposes Tier 3 (Continuous) options or only Tier 1-2? [Completeness]
- [ ] CHK-SP005 - Is the WebUI issue/PR page's exact option set defined (spec says "at minimum Adapter and Model" — is that the full Tier 1 or a subset)? [Completeness]
- [x] CHK-SP006 - Does the spec state whether the CLI already supports all options (no implementation needed) or whether flag grouping IS the CLI change? [Completeness]

## Behavioral Parity

- [x] CHK-SP007 - Is detach behavior defined for each surface that exposes it (WebUI navigates, TUI returns to list, CLI returns to shell)? [Clarity]
- [x] CHK-SP008 - Is dry-run report rendering defined for each surface (WebUI inline, TUI ?, CLI stdout)? [Clarity]
- [ ] CHK-SP009 - Is the validation error UX defined per-surface (WebUI inline errors, TUI form errors, API JSON error responses)? [Clarity]
- [x] CHK-SP010 - Are the collapsible section names consistent across WebUI pipeline detail and WebUI issue/PR pages? [Consistency]
- [x] CHK-SP011 - Does the TUI user story define the same conditional visibility rules as WebUI (Force hidden without from-step)? [Consistency]
- [x] CHK-SP012 - Is the from-step disabled state defined for both WebUI and TUI (not just TUI as in the current edge case)? [Coverage]

## API Contract Quality

- [x] CHK-SP013 - Do all three API contracts (StartPipelineRequest, StartIssueRequest, StartPRRequest) use consistent field naming conventions? [Consistency]
- [x] CHK-SP014 - Are field types consistent across contracts (e.g., timeout is int everywhere, not string in one contract)? [Consistency]
- [x] CHK-SP015 - Is the "unknown fields silently ignored" behavior defined as a spec requirement or only as an edge case? [Clarity]
- [x] CHK-SP016 - Do the contracts define required vs optional fields explicitly? [Completeness]
- [x] CHK-SP017 - Is the StartPRRequest handler's route (POST /api/prs/start) defined in the plan with a corresponding task? [Coverage]
