# Implementation Plan: Run Options Parity Across All Surfaces

**Branch**: `717-run-options-parity` | **Date**: 2026-04-11 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/717-run-options-parity/spec.md`

## Summary

Achieve full run-option parity across CLI, API, WebUI, and TUI by extending
existing request types and config structs to carry all four tiers of the
canonical `RunOptions` model. The CLI is already feature-complete — the work is
extending the other three surfaces to match, wiring missing fields through the
subprocess spawning layer, replacing the WebUI pipeline detail modal with an
inline tiered form, and grouping CLI help output by tier.

## Technical Context

**Language/Version**: Go 1.25+  
**Primary Dependencies**: Cobra (CLI), Bubble Tea / huh (TUI), html/template (WebUI), net/http (API)  
**Storage**: SQLite (pipeline state via `internal/state/`)  
**Testing**: `go test ./...`, table-driven tests  
**Target Platform**: Linux (primary), macOS  
**Project Type**: Single Go binary with embedded WebUI templates  
**Constraints**: No backward compatibility constraints (prototype phase)  
**Scale/Scope**: 5 surfaces (CLI, API, WebUI pipeline detail, WebUI issues/PRs, TUI), ~20 files modified

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-checked after Phase 1 design._

| Principle | Status | Notes |
|-----------|--------|-------|
| P1: Single Binary | ✅ Pass | No new dependencies — extends existing Go types |
| P2: Manifest as SSOT | ✅ Pass | No manifest schema changes; options are runtime overrides |
| P3: Persona-Scoped Execution | ✅ Pass | Run options are pipeline-level, not persona-level |
| P4: Fresh Memory at Step Boundaries | ✅ Pass | No changes to step boundary behavior |
| P5: Navigator-First | ✅ Pass | Not applicable — no pipeline structure changes |
| P6: Contracts at Handovers | ✅ Pass | API contracts defined in `contracts/` directory |
| P7: Relay via Summarizer | ✅ Pass | No relay changes |
| P8: Ephemeral Workspaces | ✅ Pass | No workspace behavior changes |
| P9: Credentials Never Touch Disk | ✅ Pass | No credential handling changes |
| P10: Observable Progress | ✅ Pass | All options flow through existing event emission |
| P11: Bounded Recursion | ✅ Pass | No recursion changes |
| P12: Minimal Step State Machine | ✅ Pass | No state machine changes |
| P13: Test Ownership | ✅ Pass | All changes require passing `go test ./...` |

**No violations.** All changes are additive field extensions to existing types
and UI enhancements. No architectural patterns are modified.

## Project Structure

### Documentation (this feature)

```
specs/717-run-options-parity/
├── plan.md                              # This file
├── spec.md                              # Feature specification
├── research.md                          # Phase 0 research output
├── data-model.md                        # Phase 1 entity model
├── contracts/
│   ├── start-pipeline-request.json      # StartPipelineRequest JSON schema
│   ├── start-issue-request.json         # StartIssueRequest JSON schema
│   └── start-pr-request.json            # StartPRRequest JSON schema
├── checklists/
│   └── requirements.md                  # Requirements checklist
└── tasks.md                             # Phase 2 output (speckit.tasks)
```

### Source Code (repository root)

```
cmd/wave/commands/
└── run.go                    # CLI help grouping (Tier 1-4 sections)

internal/webui/
├── types.go                  # StartPipelineRequest, SubmitRunRequest, StartIssueRequest, StartPRRequest
├── handlers_control.go       # RunOptions struct, spawnDetachedRun(), handleStartPipeline, handleSubmitRun
├── handlers_issues.go        # handleAPIStartFromIssue — extend with overrides
├── handlers_prs.go           # handleAPIStartFromPR [NEW handler]
├── routes.go                 # Register POST /api/prs/start
├── templates/
│   ├── pipeline_detail.html  # Inline tiered run form (replace modal)
│   ├── issue_detail.html     # Add Model/Adapter selectors + Advanced section
│   └── pr_detail.html        # Add Run Pipeline button + dialog with overrides
└── static/
    └── style.css             # Inline form styles

internal/tui/
├── pipeline_messages.go      # LaunchConfig struct — add typed fields
├── pipeline_launcher.go      # Map new LaunchConfig fields to subprocess flags
├── pipeline_detail.go        # Argument form — add adapter, timeout, from-step, etc.
├── run_selector.go           # DefaultFlags — add --detach
└── run_selector_test.go      # Update test expectations

docs/
├── reference/cli.md          # Tier-grouped flag documentation
└── running-pipelines.md      # Cross-surface options guide [NEW or extend existing]

CHANGELOG.md                  # Run options parity entry
```

**Structure Decision**: Single Go project. All changes are within existing
packages (`cmd/wave/commands`, `internal/webui`, `internal/tui`, `docs`).
No new packages needed.

## Complexity Tracking

_No Constitution violations — table empty._

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|--------------------------------------|
| (none)    |            |                                      |
