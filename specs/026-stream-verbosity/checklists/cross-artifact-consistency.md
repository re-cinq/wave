# Cross-Artifact Consistency Checklist: Stream Verbosity (026)

**Feature**: Stream Verbosity | **Date**: 2026-02-09
**Purpose**: Validate that spec.md, plan.md, tasks.md, data-model.md, research.md, and contracts agree on critical design decisions.

This checklist targets the naming, signature, and behavioral contradictions found between
the research phase output and the downstream design/contract documents. These must be
resolved before implementation begins to prevent rework.

---

## Naming Contradictions

- [ ] XAC001 - Is the Event field name `Adapter` (plan, data-model, contract, tasks) or `AdapterType` (research)? The JSON tag differs: `"adapter"` vs `"adapter_type"`. Resolution must be applied to all 5 artifacts consistently. [Naming]

- [ ] XAC002 - Does the generic heuristic fallback include 6 fields (plan, data-model, contract, tasks: file_path, url, pattern, command, query, notebook_path) or 8 fields (research: adds path, description)? If 8, the contract and tasks must be updated. If 6, the research should be annotated as superseded. [Naming]

---

## Signature Contradictions

- [ ] XAC003 - Does `extractToolTarget` accept `input json.RawMessage` (data-model, existing code) or `input map[string]json.RawMessage` (contract)? The contract must match the actual function signature to be a valid behavioral specification. [Signature]

---

## Behavioral Contradictions

- [ ] XAC004 - Does the ThrottledProgressEmitter use a self-expiring throttle window (data-model: "when the throttle window expires") or purely lazy forwarding on next event arrival (research: rejects timer-based flushing, plan: no timer/goroutine)? These are architecturally different approaches. The contract does not specify either mechanism. [Behavior]

- [ ] XAC005 - Is `EstimatedTimeMs` tagged with `omitempty` (current code, data-model) or without `omitempty` (progress-event contract: "MUST be present, not omitempty")? The task T018 uses exploratory language ("consider whether to change") despite the contract being mandatory. [Behavior]

- [ ] XAC006 - Does the ProgressEmitter interface live in `internal/event/emitter.go` (data-model) or `internal/display/types.go` (throttled-emitter contract)? The ThrottledProgressEmitter must import from the correct package. [Location]

---

## Resolution Actions

For each contradiction above, the resolution should:
1. Identify which artifact is authoritative (typically: contract > plan > research)
2. Update all non-authoritative artifacts to match
3. Document the resolution in the clarifications section of spec.md if it affects requirements
