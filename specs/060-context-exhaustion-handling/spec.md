# Handle Claude context window exhaustion gracefully

**Issue**: [#60](https://github.com/re-cinq/wave/issues/60)
**Author**: nextlevelshit
**State**: OPEN
**Complexity**: Medium

## Problem

When Claude Code runs out of context window during a pipeline step, Wave reports it as a generic timeout error:

```
implement failed: adapter execution failed: failed to start claude: context deadline exceeded
```

This is misleading - `context deadline exceeded` is a Go context timeout error, but the actual cause may be that Claude Code exhausted its context window and the session ended, or the step genuinely ran over the time limit.

## Current Behavior

1. **Timeout path** (`internal/adapter/claude.go:55-61`): Default 10-minute timeout via `context.WithTimeout()`. When hit, the process is killed with SIGKILL and `ctx.Err()` (which is `context.DeadlineExceeded`) is returned.

2. **Context window exhaustion**: When Claude Code runs out of context, the process exits normally (exit code 0 or non-zero) but the task may be incomplete. Wave has relay/compaction support (`internal/relay/relay.go`) that monitors token usage, but:
   - It relies on streaming NDJSON events to track `TokensUsed`
   - The relay threshold check only triggers compaction if the *current step* reports high usage
   - If Claude Code auto-compresses and continues, Wave may not detect it
   - If Claude Code hits an absolute limit and exits, Wave treats it as a normal completion or error

3. **No distinction in error messages**: Both scenarios produce similar-looking failures. The user cannot tell from the error whether they need to:
   - Increase the timeout (`--timeout 60`)
   - Break the task into smaller steps
   - Adjust relay compaction thresholds

## Expected Behavior

- Distinguish between time-based timeout and context exhaustion
- Surface Claude Code exit reason (if available in NDJSON output)
- Suggest actionable next steps in the error message
- Consider: if Claude Code exits with a specific signal/code for context exhaustion, detect it

## Research Findings

Key finding from research comment: Claude Code does not provide a dedicated exit code or NDJSON event for context exhaustion. Detection requires parsing the stream-json result event's `subtype` (`success`, `error_max_turns`, `error_during_execution`) and checking error content for `prompt is too long` strings.

### Recommendations from Research

| ID | Priority | Effort | Title |
|---|---|---|---|
| REC-001 | Critical | Medium | Implement three-way error classification in adapter |
| REC-002 | Critical | Small | Switch from SIGKILL to SIGTERM with grace period |
| REC-003 | Critical | Small | Parse buffered output on timeout for diagnostic data |
| REC-004 | High | Medium | Add token usage to pipeline trace for failed steps |
| REC-005 | High | Small | Include remediation suggestions in error messages |
| REC-006 | Medium | Trivial | Reduce relay compaction threshold to 70% |
| REC-007 | Medium | Small | Add context utilization percentage to progress events |
| REC-008 | Medium | Small | Document troubleshooting guidance for context errors |

## Acceptance Criteria

- [ ] Wave distinguishes between timeout and context exhaustion in error messages
- [ ] Error messages suggest specific remediation steps
- [ ] Pipeline traces include token usage data for failed steps
- [ ] Documentation updated with troubleshooting guidance

## Relevant Code

- `internal/adapter/claude.go` - timeout setup (L55-61), error return (L137-143)
- `internal/adapter/adapter.go` - `killProcessGroup` using SIGKILL (L138-141), `AdapterResult` type
- `internal/pipeline/executor.go` - step timeout application (L472-475), relay check (L633)
- `internal/relay/relay.go` - token threshold monitoring, `ShouldCompact`
- `internal/manifest/types.go` - `DefaultTimeoutMin` configuration
- `internal/event/emitter.go` - Event structure with TokensUsed field
