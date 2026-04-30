# impl-issue contract: reject net test deletions via LLM-as-judge

**Issue:** [re-cinq/wave#1582](https://github.com/re-cinq/wave/issues/1582)
**Labels:** bug, pipeline, contracts, priority: high
**State:** OPEN
**Author:** nextlevelshit

## Real-world signal

During Phase 1 of Epic #1565 (Onboarding-as-Session), `impl-issue-20260429-221616-d8e4` running on issue #1580 (1.5a `/preview/*`) deleted a passing-by-design test instead of correcting its build-tag scope.

**Diff snippet (24 lines removed, 0 added):**
```diff
-// TestNewFeatureRegistryDefaultTagsZeroFlags verifies that under default build
-// tags (no optional features) the registry reports every feature as disabled
-// and contributes no route hooks. This locks in the "disabled stubs are no-ops"
-// contract.
-func TestNewFeatureRegistryDefaultTagsZeroFlags(t *testing.T) { ... }
```

Persona ran `go test -tags=analytics` (which flips Analytics flag), saw the test fail, and deleted it rather than recognizing the test was scoped to the no-tags build.

## Root cause

The `impl-issue` pipeline contract says "tests must pass" with no clause "and no tests deleted unless replaced". The contract is satisfiable by net-deleting a failing test. Cheapest-tier model amplified the symptom but balanced/opus can take the same shortcut — the gate is the structural defect, not the model tier.

## Proposed fix

Add an `llm_judge` contract step to `impl-issue.yaml` after `implement`:

- Inputs: diff of `_test.go` files
- Question: "does this diff net-remove any `func Test*` declarations? If so, was each removed test demonstrably replaced (renamed-and-edited or migrated to a new file)? Reject if net deletion."
- `on_failure: rework`

## Acceptance criteria

1. **Reproducer**: re-run `impl-issue` on a synthetic issue whose contract requires deleting an unrelated existing test. Pipeline must reject at the judge step.
2. **Regression**: existing passing pipelines on green-path issues still complete (no false positives).

## Related

- Companion contracts proposed: source-diff deletion-ratio + post-commit Test-count grep (filing as siblings)
- Real-world hit: PR (filed by run d8e4) review will surface this; we restore the test as fixup before merge.
