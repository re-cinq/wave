# Lifecycle & Signal Handling Quality Checklist

**Feature**: #201 — Continuous Pipeline Execution
**Domain**: Graceful shutdown, iteration lifecycle, failure policies
**Generated**: 2026-03-16

## Completeness

- [ ] CHK201 - Does the spec define the timeout for "current iteration completes" after SIGINT? SC-002 says "within 30 seconds of the current step finishing" but what if the step itself takes arbitrarily long? [Completeness]
- [ ] CHK202 - Is behavior defined for a second SIGINT while the current iteration is draining? (Force kill? Ignore? Queue?) [Completeness]
- [ ] CHK203 - Does the spec address cleanup of orphaned workspaces if the process exits abnormally (SIGKILL edge case mentions state but not workspace cleanup)? [Completeness]
- [ ] CHK204 - Are the `loop_summary` event fields fully specified — does it include duration, or just counts? [Completeness]
- [ ] CHK205 - Does the failure policy specification address what happens when the source itself fails (not an iteration, but the `Next()` call)? Is that a halt-worthy failure regardless of policy? [Completeness]

## Clarity

- [ ] CHK206 - Is "between iterations" (US2 scenario 2) precisely defined — after iteration cleanup but before `source.Next()`, or after `source.Next()` but before executor start? [Clarity]
- [ ] CHK207 - Does the spec clearly distinguish between "the loop exits" (continuous session ends) and "the process exits" (OS-level exit)? Are there cleanup steps between them? [Clarity]

## Consistency

- [ ] CHK208 - Is the context cancellation pattern for SIGINT/SIGTERM consistent with how `cmd/wave/commands/run.go` currently handles signals for single runs? [Consistency]
- [ ] CHK209 - Does the `on_failure: halt` default match the existing step-level failure behavior precedent? (Clarification C5 says they're independent layers — is this intuitive for users?) [Consistency]
- [ ] CHK210 - Is the `loop_iteration_failed` event structure consistent with existing failure events in the event system? [Consistency]

## Coverage

- [ ] CHK211 - Is there a scenario for `--delay` combined with SIGINT — if the loop is sleeping between iterations and SIGINT arrives, does it wake up immediately? [Coverage]
- [ ] CHK212 - Are requirements defined for what the summary (FR-014) includes when the loop is interrupted via signal? (Partial results should still be summarized) [Coverage]
- [ ] CHK213 - Is there a scenario testing that iteration isolation (FR-004, FR-005) is actually enforced — e.g., one iteration's env vars don't leak to the next? [Coverage]
