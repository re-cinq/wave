# feat: add possibility to mark pipeline steps as optional

**Issue**: [#118](https://github.com/re-cinq/wave/issues/118)
**Labels**: enhancement, pipeline
**Author**: nextlevelshit
**State**: OPEN

## Summary

Add support for marking individual pipeline steps as optional in `wave.yaml`. When a step is marked optional, the pipeline continues execution even if that step fails or is skipped, rather than halting the entire pipeline.

## Motivation

Currently, all pipeline steps are required — if any step fails, the pipeline stops. However, some workflows include steps that are non-critical (e.g., a notification step, a linting step, or a deployment to a staging environment). Making steps optional allows pipelines to be more resilient and flexible.

## Proposed Configuration

Add an `optional: true` field to individual step definitions in `wave.yaml`:

```yaml
pipelines:
  my-pipeline:
    steps:
      - name: build
        persona: builder
        # required (default)
      - name: notify-slack
        persona: notifier
        optional: true  # pipeline continues even if this step fails
      - name: deploy-staging
        persona: deployer
        optional: true
      - name: deploy-production
        persona: deployer
        # required — pipeline halts on failure
```

## Behavior

- Optional steps that **succeed**: pipeline continues normally
- Optional steps that **fail**: pipeline logs the failure, marks the step as `failed`/`skipped`, and continues to the next step
- Optional steps that are **skipped** (e.g., condition not met): treated the same as a non-blocking failure
- Required steps (default): pipeline halts on failure (existing behavior preserved)

## Acceptance Criteria

- [ ] `optional: true` is a valid field in step configuration
- [ ] When an optional step fails, the pipeline continues to the next step
- [ ] The step status is recorded as `failed` (not blocking) in pipeline state
- [ ] Pipeline summary output distinguishes between required failures and optional step failures
- [ ] Existing pipelines without `optional` field are unaffected (default behavior is required)
- [ ] Unit tests cover optional step failure and continuation behavior

## Use Cases

1. **Notifications**: Send Slack/email notifications after a step — failure shouldn't block the pipeline
2. **Linting**: Run a linter as a soft check without blocking deployment
3. **Staging deployment**: Deploy to staging optionally while keeping production deployment required
4. **Telemetry/reporting**: Emit metrics or reports that should not block the main workflow

## Design Decisions

### Relationship to existing `retry.on_failure`

The codebase already has `retry.on_failure` with values `"fail"`, `"skip"`, and `"continue"`. The `optional` field is a higher-level, user-facing shorthand that maps to this existing mechanism:

- `optional: true` is semantically equivalent to setting `retry.on_failure: "continue"` with `retry.max_attempts: 1`
- However, `optional` is a first-class step-level field, not nested under retry config
- When `optional: true`, the step's failure is recorded but does not halt the pipeline
- `optional` and `retry` can coexist: an optional step can still have retries before giving up

### Dependent steps when optional step fails

When an optional step fails and a downstream step depends on it:
- The dependent step should be **skipped** (not attempted) because its input artifacts are unavailable
- The skip propagates: if step C depends on step B which depends on optional step A, and A fails, both B and C are skipped
- This is consistent with the principle that optional means "pipeline continues" not "pretend it succeeded"

### Pipeline exit code

- Pipeline succeeds (exit 0) if all required steps pass, even if optional steps fail
- Pipeline fails (exit 1) only if a required step fails
- The pipeline summary distinguishes optional failures from required failures
