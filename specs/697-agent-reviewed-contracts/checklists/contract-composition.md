# Contract Composition Quality Review

**Feature**: #697 | **Dimension**: Plural contracts, ordering, rework interaction

## Completeness

- [ ] CHK101 - Are the semantics of `on_failure: skip` in a plural contracts list fully defined? Does "skip" mean skip remaining contracts and mark the step as passed, or skip remaining contracts and mark based on what passed so far? [Completeness]
- [ ] CHK102 - Does the spec define the aggregate pass/fail semantics for a contracts list? If contract 1 passes but contract 2 uses `on_failure: continue` and fails, does the overall step pass or fail? [Completeness]
- [ ] CHK103 - Does the spec define which contract's `max_retries` governs the re-run count when multiple contracts in the list have `on_failure: rework`? If contract 2 triggers rework (max_retries: 1) and after rework contract 3 triggers rework (max_retries: 3), how are retry counts tracked? [Completeness]
- [ ] CHK104 - Does the spec define whether `on_failure: retry` (non-rework retry) re-runs only the failed contract or all contracts from the beginning? C4 specifies re-run-all for rework, but `retry` is a separate failure mode [Completeness]

## Clarity

- [ ] CHK105 - Is the distinction between "early termination" (contract fails, skip rest) and "continuation" (contract fails, proceed to next) clearly mapped to specific `on_failure` values? The plan pseudocode shows `skip` breaking the loop and `continue` proceeding, but the spec text uses "early termination when a contract fails (unless its on_failure policy allows continuation)" which conflates the two [Clarity]
- [ ] CHK106 - Is it clear that `on_failure: rework` on a non-agent_review contract (e.g., test_suite) would also trigger full re-run of all contracts? Or is rework exclusive to agent_review? [Clarity]

## Consistency

- [ ] CHK107 - Is the behavior of `must_pass: false` consistent with `on_failure` in the plural contracts context? The singular contract uses `must_pass` for soft/hard failure; the plural list uses per-contract `on_failure`. Are both fields respected? Do they conflict? [Consistency]
- [ ] CHK108 - Does the contract validation loop preserve the existing event emission pattern (contract_validated, contract_failed events) for each contract in the list, or are events only emitted once for the aggregate result? [Consistency]

## Coverage

- [ ] CHK109 - Does the spec cover the maximum length of a contracts list? Is there a validation limit or can a step have arbitrarily many contracts? [Coverage]
- [ ] CHK110 - Does the spec cover what the contract validation loop emits when a contract is skipped due to early termination? Is there a "contract_skipped" event or is it silent? [Coverage]
