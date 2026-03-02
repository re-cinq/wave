# Audit traces lack meaningful context — only log tool invocations with minimal metadata

**Issue**: [#189](https://github.com/re-cinq/wave/issues/189)
**Labels**: enhancement, observability
**Author**: nextlevelshit
**State**: OPEN

## Problem

The audit trace output in `.wave/traces/` currently logs only tool invocations with minimal metadata. Each trace line captures a timestamp, event type (`[TOOL]`), pipeline name, step name, tool name, persona, and prompt length — but omits critical execution context needed for debugging and observability.

### Current trace output

```
2026-02-02T18:17:04.933282119+01:00 [TOOL] pipeline=migrate step=impact-analysis tool=adapter.Run args=persona=navigator prompt_len=449
2026-02-02T18:17:04.934962998+01:00 [TOOL] pipeline=migrate step=migration-plan tool=adapter.Run args=persona=philosopher prompt_len=359
2026-02-02T18:17:05.078497628+01:00 [TOOL] pipeline=migrate step=implement tool=adapter.Run args=persona=craftsman prompt_len=251
2026-02-02T18:17:07.238502608+01:00 [TOOL] pipeline=migrate step=implement tool=adapter.Run args=persona=craftsman prompt_len=251
```

These entries provide no insight into what happened during execution — there is no indication of success/failure, execution duration, output size, error messages, or contract validation results.

## Expected behavior

Trace entries should capture rich execution context to support post-mortem debugging and pipeline observability. Each trace line or structured entry should include:

- **Execution duration** — how long the tool/step took
- **Exit code / success status** — whether the invocation succeeded or failed
- **Output summary** — size of output, artifact paths written
- **Error messages** — captured stderr or error details on failure
- **Contract validation result** — pass/fail/skip for the step's contract
- **Token usage** — input/output token counts if available from the adapter
- **Step dependencies** — which artifacts were injected

### Example desired trace output

```
2026-02-02T18:17:04.933Z [STEP_START] pipeline=migrate step=impact-analysis persona=navigator
2026-02-02T18:17:04.933Z [TOOL] pipeline=migrate step=impact-analysis tool=adapter.Run prompt_len=449
2026-02-02T18:17:12.456Z [STEP_END] pipeline=migrate step=impact-analysis status=success duration=7.523s exit_code=0 output_bytes=2048 tokens_in=449 tokens_out=312 contract=pass
```

## Relevant source files

- `internal/audit/` — audit logging and credential scrubbing
- `internal/event/` — progress event emission
- `internal/pipeline/executor.go` — pipeline step execution
- `internal/adapter/claude.go` — adapter subprocess execution

## Acceptance criteria

- [ ] Trace entries include execution duration for each step
- [ ] Trace entries include exit code and success/failure status
- [ ] Trace entries include output size or artifact paths written
- [ ] Trace entries capture error messages on failure (with credential scrubbing preserved)
- [ ] Trace entries include contract validation results
- [ ] Existing trace format is extended (not replaced) to maintain backward parsing compatibility
- [ ] No credentials or sensitive data leak into trace output
