# fix(display): outcome extraction warnings are noisy and confusing on successful runs

**Issue**: [#204](https://github.com/re-cinq/wave/issues/204)
**Labels**: bug, pipeline
**Author**: nextlevelshit

## Summary

When a pipeline completes successfully but an outcome `json_path` points to an empty array, the output shows a technical error that confuses users into thinking something failed:

```
[19:51:55] ⚠ apply-enhancements: [apply-enhancements] outcome: .enhanced_issues[0].url at .wave/artifact.json: array index 0 out of bounds (length 0) at "enhanced_issues"
```

And again in the summary:
```
! [apply-enhancements] outcome: .enhanced_issues[0].url at .wave/artifact.json: array index 0 out of bounds (length 0) at "enhanced_issues"
```

The pipeline ran successfully — the issue scored 91/100 and correctly needed zero enhancements. The empty array is expected behavior, not an error.

## Observed In

`wave run gh-rewrite "re-cinq/wave 201 ..."` — pipeline completed successfully in 48.7s but output shows warning symbols that imply failure.

## Proposed Fix

1. When an outcome `json_path` references an array index on an empty array, show a gentler message like `"No outcome URL — enhanced_issues is empty"` instead of the raw Go error
2. Consider suppressing outcome warnings entirely when the array is legitimately empty (length 0 with index 0 is a common "no results" case)
3. The `⚠` during step execution (`progress.go:626`) and the `!` in the summary (`outcome.go:365`) both show the same warning — consider showing it only once in the summary

## Location

- `internal/pipeline/outcomes.go:58-59` — generates the raw error message
- `internal/pipeline/executor.go:1732-1743` — emits warning event + adds to tracker
- `internal/display/progress.go:626` — renders `⚠` during execution
- `internal/display/outcome.go:362-368` — renders `!` in summary

## Acceptance Criteria

1. When an outcome `json_path` indexes into an empty array (e.g., `.enhanced_issues[0].url` where `enhanced_issues` is `[]`), the system produces a user-friendly message instead of a raw Go error string
2. The real-time `⚠` warning during step execution is suppressed for empty-array cases — these are only shown in the summary
3. The summary `!` line uses the friendly message (e.g., "No items in enhanced_issues") instead of the technical "array index 0 out of bounds (length 0)" error
4. Non-empty-array errors (e.g., index 5 on a length-2 array, missing keys, non-array types) continue to produce the existing warning behavior unchanged
5. All existing tests pass; new tests cover the empty-array detection path
