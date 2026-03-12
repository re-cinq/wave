# Pipeline Design Quality Review — Closed-Issue/PR Audit Pipeline (#305)

**Feature**: `wave-audit` pipeline | **Date**: 2026-03-11

This checklist validates the quality of the pipeline architecture and step decomposition requirements.

## Step Decomposition

- [ ] CHK201 - Is the separation of concerns between `audit-items` and `compose-triage` justified — are the requirements clear about why aggregation cannot happen within the audit step itself? [Clarity]
- [ ] CHK202 - Does the plan specify the exact artifact names and paths for inter-step communication (e.g., `inventory.json` → where exactly is it written and how is it injected)? [Completeness]
- [ ] CHK203 - Is the DAG dependency chain fully specified — are there missing edges (e.g., should `publish` also depend on `collect-inventory` for repository metadata)? [Completeness]
- [ ] CHK204 - Does the spec define whether the `publish` step is opt-in (requiring a flag to enable) or opt-out (running by default with a flag to skip)? [Clarity]

## Persona-Step Alignment

- [ ] CHK205 - Are the tool permissions required by each step documented against the actual persona definitions — does `navigator` have all tools needed for `audit-items` (specifically `Bash(git log*)`)? [Consistency]
- [ ] CHK206 - Does the P5 deviation (github-analyst as first step) have a clear constitutional exception documented, or is it a silent violation? [Completeness]
- [ ] CHK207 - Are the `craftsman` persona's permissions appropriate for the `publish` step — does it have `gh issue create` but not overly broad access? [Coverage]
- [ ] CHK208 - Does the spec define what happens if a persona's tool permissions are insufficient at runtime — error handling for permission denials? [Coverage]

## Prompt Engineering Requirements

- [ ] CHK209 - Are the prompt requirements for scope parsing (C3) specific enough to produce consistent behavior — what if the persona misparses "last 30 days" as a label filter? [Clarity]
- [ ] CHK210 - Does the spec define the expected prompt structure for handling inventory items that span the adapter's context window — is there a batching or prioritization strategy? [Completeness]
- [ ] CHK211 - Are the verification instructions in the `audit-items` prompt specific enough for consistent classification — would two different navigators classify the same item identically? [Clarity]
- [ ] CHK212 - Does the spec require prompt testing or validation beyond `go test ./...` — how is prompt quality assured for the four step prompts? [Coverage]

## Resilience and Operability

- [ ] CHK213 - Are retry semantics specified for each step — does the plan define `on_failure` and `max_retries` per step, and are these consistent with existing pipeline patterns? [Completeness]
- [ ] CHK214 - Does the spec define observability requirements — what progress events should be emitted during long-running steps like `audit-items`? [Completeness]
- [ ] CHK215 - Is the expected runtime performance characterized — what is the estimated token/time cost per inventory item during the audit step? [Completeness]
- [ ] CHK216 - Does the spec address the workspace isolation model — are all steps using the same worktree, and is this documented as a deliberate choice? [Clarity]
