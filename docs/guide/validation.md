# Validation in Wave: A System That Ships, Not Just Compiles

## The Incident

On March 27, 2026, an autonomous orchestration session implemented epic #589 — eleven features that would transform Wave from a linear pipeline runner into a self-correcting development system. The session ran 35 pipeline executions across 4 hours. It produced 24,000 lines of new Go code across 70+ files. Every test passed. Every build succeeded. Fourteen pull requests were merged.

Then someone opened the product. Nothing had changed.

The pipelines were identical. The TUI showed the same six health checks. The WebUI rendered the same boxes and arrows. A user pulling the new version would experience exactly the same tool they had the day before. The session had built an engine and put it in no car.

This document exists so that never happens again.

## What Validation Is Not

**Validation is not compilation.** `go build ./...` tells you the code is syntactically correct. It says nothing about whether the code matters.

**Validation is not testing.** `go test ./...` tells you the code behaves as the developer intended. It says nothing about whether the developer intended the right thing. A perfectly tested graph walker that no pipeline uses is a perfectly tested irrelevance.

**Validation is not existence.** Checking that `internal/pipeline/graph.go` exists tells you a file was created. It says nothing about whether that file participates in the product a user touches.

**Validation is not coverage.** 90% test coverage on a feature nobody can access is 90% coverage of nothing.

All of these are **build verification** — they confirm the artifact is well-formed. They are necessary. They are not sufficient. They are not validation.

## What Validation Is

Validation answers one question: **did the product change from the user's perspective?**

The user is not a compiler. The user is not a test runner. The user does not read `internal/pipeline/graph.go`. The user runs `wave list pipelines`. The user runs `wave run impl-issue`. The user opens the WebUI. The user types `wave --help`.

If those interactions are identical before and after the work, nothing was delivered. The work is incomplete. It does not matter how many lines of code were written, how many tests pass, how many PRs were merged, or how elegant the architecture is.

Validation is the act of standing where the user stands and checking that the view is different.

## The Misconception

The misconception is natural and pernicious: engineers build systems from the inside out. They start with types, interfaces, packages. They write the foundation, then the middleware, then the handlers. By the time the internal layers are done, the work *feels* complete — the hard part is over. The rest is "just" wiring it up to the surface.

But the surface is the product. Everything underneath is scaffolding. A building with beautiful foundations and no roof is not a building. A feature with beautiful internals and no pipeline adoption is not a feature.

The misconception is reinforced by tooling. CI says green. Tests say pass. The PR review says "clean architecture, comprehensive tests, well-structured code." None of these tools ask: "can a user see this?"

AI agents are especially prone to this failure mode. They optimize for the instruction they were given. "Implement issue #577: graph loops and conditional edge routing" — so they implement graph loops in the executor. The executor now supports loops. The issue's acceptance criteria are technically met. But no pipeline uses loops, so no user benefits. The agent did exactly what was asked and delivered nothing.

## The Structural Fix

The fix is not "remember to check the user experience." Remembering is fragile. People forget. Agents don't read memos from prior sessions. The fix must be structural — embedded in the system itself.

### 1. The Validation Contract

Every implementation pipeline should end with a validation step that checks the user surface, not the code surface:

- Does `wave list pipelines` show changed output?
- Does the modified pipeline behave differently when run?
- Does `internal/defaults/` contain the updated files?
- Do TUI and WebUI render the new capabilities?

If any answer is no, the contract fails and the pipeline loops back.

### 2. The Adoption Checklist

Every issue that adds an executor capability must include adoption scope:

- Which existing pipelines should use this feature?
- What changes in the defaults?
- What changes in the TUI/WebUI?
- What does the user see differently?

If the issue doesn't specify adoption, it's an engine task, not a feature. Label it accordingly. Don't close the parent feature until adoption is done.

## The Anti-Fragile Loop

The incident itself created the fix. The failure to validate led to a validation contract pattern. Every failure that reaches a user creates a structural constraint that prevents recurrence — not a note, not a memory, but a check encoded in the pipeline.

## The Checklist

Before declaring any work complete, answer these questions:

1. **Run the product.** Open it. Use it. Does anything look different?
2. **Check the defaults.** Would a new user installing Wave get this feature?
3. **Check the pipelines.** Does at least one shipped pipeline use this feature?
4. **Check the surfaces.** CLI help, TUI views, WebUI pages — do they reflect the change?
5. **Check from the outside.** If you hadn't written the code, would you know something changed?

If any answer is no, the work is not done. Go back and ship.

## Summary

Validation is not a phase. It is not a step at the end. It is the definition of done. Code that compiles but doesn't ship is inventory, not delivery. Tests that pass but verify unused features are vanity metrics. PRs that merge but don't change the product are noise.

The only question that matters: **did the user's experience change?**

If yes, you shipped. If no, you didn't. Everything else is commentary.

## See Also

- [Contracts Guide](contracts.md) — Practical configuration for validation contracts (Layer 2)
- [Gates Guide](human-gates.md) — Human approval and automated gates for outcome validation (Layer 3)
- [Graph Loops](graph-loops.md) — Fix-loop configuration including safety mechanisms
- [V&V Paradigm Guide](vv-paradigm.md) — Unified three-layer verification and validation model

### Implementation Pointers

The components described in this document map to these code locations:

- **Validation contracts** — `internal/contract/contract.go` (`NewValidator`)
- **Outcome gates** — `internal/pipeline/gate.go` (`Execute`)
