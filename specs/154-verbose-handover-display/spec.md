# display: show contract, artifact, and handover references between pipeline steps in verbose mode

**Issue**: [#154](https://github.com/re-cinq/wave/issues/154)
**Feature Branch**: `154-verbose-handover-display`
**Labels**: enhancement, display
**Complexity**: medium
**Status**: Draft

## Summary

When running a pipeline with `-v` (verbose mode), the TUI currently displays steps sequentially with no visibility into the inter-step handover metadata. Contracts, artifacts, and handover summaries that connect one step to the next are invisible to the operator.

## Current Behavior

Running `wave run -v` displays each pipeline step in sequence with no inter-step metadata.

**Non-TTY output:**

```
[step 1: analyst] ✓ completed
[step 2: implementer] ✓ completed
[step 3: reviewer] ✓ completed
```

**TTY output (BubbleTea TUI):**

```
 ╦ ╦╔═╗╦  ╦╔═╗    Pipeline: my-pipeline-20260224-131300-91ea
 ║║║╠═╣╚╗╔╝║╣     Config:   wave.yaml
 ╚╩╝╩ ╩ ╚╝ ╚═╝    Elapsed:  2m 30s

 [████████████░░░░░░░░░░░░░] 33% Step 2/3 (1 ok)

 ✓ analyst (analyst) (45.2s)
 ⠋ implementer (implementer) (30s) • Writing code
   Write → src/main.go
 ○ reviewer (reviewer)

 Press: q=quit
```

There is no indication of what was passed between steps — no artifact paths, no contract validation results, and no handover context.

## Expected Behavior

In verbose mode, each step transition should display the inter-step metadata:

**Non-TTY output:**

```
[step 1: analyst] ✓ completed
  ├─ artifact: .wave/artifacts/analysis (written)
  ├─ contract: gh-issue-analysis.schema.json ✓ valid
  └─ handover → step 2: implementer
[step 2: implementer] ✓ completed
  ├─ artifact: .wave/artifacts/implementation (written)
  ├─ contract: implementation.schema.json ✓ valid
  └─ handover → step 3: reviewer
[step 3: reviewer] ✓ completed
```

**TTY output (BubbleTea TUI):**

```
 ╦ ╦╔═╗╦  ╦╔═╗    Pipeline: my-pipeline-20260224-131300-91ea
 ║║║╠═╣╚╗╔╝║╣     Config:   wave.yaml
 ╚╩╝╩ ╩ ╚╝ ╚═╝    Elapsed:  2m 30s

 [████████████░░░░░░░░░░░░░] 33% Step 2/3 (1 ok)

 ✓ analyst (analyst) (45.2s)
   ├─ artifact: .wave/artifacts/analysis (written)
   ├─ contract: gh-issue-analysis.schema.json ✓ valid
   └─ handover → implementer
 ⠋ implementer (implementer) (30s) • Writing code
   Write → src/main.go
 ○ reviewer (reviewer)

 Press: q=quit
```

Note: The tree format (├─, └─) is already used for deliverables on completed steps (see `bubbletea_model.go` lines 295-303). This enhancement extends that pattern to also show artifacts, contracts, and handover targets.

## Scope

- **Applies to**: `wave run -v` (verbose mode only); default output remains unchanged
- **Metadata to display**: artifact file paths, contract validation status, and handover target step name
- **Affected packages**: `internal/display/`, `internal/event/`

## Acceptance Criteria

- [ ] When running `wave run -v`, each step transition displays the artifact path written by the completed step
- [ ] Contract validation result (pass/fail + schema name) is shown between steps
- [ ] Handover target step name is displayed
- [ ] Default (non-verbose) output is not affected
- [ ] Existing tests pass; new tests cover the verbose display logic

## Requirements

### Functional Requirements

- **FR-001**: In verbose mode, completed steps MUST display output artifact paths in tree format beneath the step line
- **FR-002**: In verbose mode, completed steps MUST display contract validation results (pass/fail + schema name) when a handover contract is configured
- **FR-003**: In verbose mode, completed steps MUST display the handover target step name when a next step exists
- **FR-004**: Default (non-verbose) output MUST NOT be affected by this change
- **FR-005**: The tree format (├─, └─) MUST be used consistently with the existing deliverable display pattern
- **FR-006**: Both TTY (BubbleTea) and non-TTY (BasicProgressDisplay) renderers MUST support the verbose handover metadata

### Key Entities

- **HandoverMetadata**: Per-step metadata containing artifact paths, contract validation status, and handover target
- **Event enrichment**: Events emitted during contract validation and artifact writing already contain the needed data; it must be captured and surfaced in display
