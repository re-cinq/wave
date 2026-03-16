# Resource Limit Enforcement Quality Checklist

**Feature**: Restore and Stabilize `wave meta` Dynamic Pipeline Generation
**Date**: 2026-03-16
**Scope**: US5, FR-008, Edge Case 4

This checklist validates requirements quality for the resource limit enforcement subsystem — the safety boundary preventing runaway meta pipelines.

---

## Limit Definitions

- [ ] CHK201 - Are all four resource limits (depth, steps, tokens, timeout) independently configurable, or can they interact (e.g., does timeout override step limit)? [Clarity]
- [ ] CHK202 - Is "max_total_steps" counted across the entire recursive meta call chain, or per-invocation? [Clarity]
- [ ] CHK203 - Is "max_total_tokens" an input token count, output token count, or combined? [Clarity]
- [ ] CHK204 - Are the manifest configuration paths for each limit specified (e.g., `runtime.meta_pipeline.max_depth`)? [Completeness]
- [ ] CHK205 - Is the behavior defined when a limit is set to zero — does zero mean "unlimited" or "disallowed"? [Completeness]

## Enforcement Semantics

- [ ] CHK206 - Is the enforcement point specified — are limits checked before execution starts, during execution, or both? [Clarity]
- [ ] CHK207 - Is timeout enforcement graceful (cancel context and wait) or hard (kill immediately)? [Clarity]
- [ ] CHK208 - When a step limit is exceeded, is the partially-executed pipeline cleaned up, or are completed steps preserved? [Completeness]
- [ ] CHK209 - Is the "within 1 second" detection requirement (SC-006) testable — how is it measured? [Clarity]
- [ ] CHK210 - Is the depth limit error required to include the full call stack (US5-AS2), and is "call stack" defined (pipeline names, step names, depth counter)? [Clarity]

## Configuration

- [ ] CHK211 - Are defaults documented for the case where `runtime.meta_pipeline` section is entirely absent from the manifest? [Completeness]
- [ ] CHK212 - Can limits be overridden per-invocation via CLI flags, or only via manifest? [Coverage]
- [ ] CHK213 - Is validation defined for limit values — what happens with negative numbers or non-integer values? [Coverage]
