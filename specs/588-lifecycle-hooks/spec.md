# feat: lifecycle hooks for validation, notifications, and guardrails

**Issue**: [re-cinq/wave#588](https://github.com/re-cinq/wave/issues/588)
**Labels**: enhancement
**Author**: nextlevelshit
**State**: OPEN

## Context

Fabro has a comprehensive **hooks system** with 15 lifecycle events — enabling custom validation, notifications, guardrails, and orchestration at every stage of workflow execution. Hook types include shell commands, HTTP webhooks, LLM prompts (single-turn judge), and full agent sessions. Hooks can block or proceed, with regex-based matchers for selective triggering.

Wave has Claude Code hooks (user-configured) but no pipeline-level hook system for pipeline authors to define validation, notification, or guardrail logic.

## Design

### Hook Definition

In `wave.yaml` or per-pipeline config:

```yaml
hooks:
  - name: lint-on-write
    event: step_completed
    type: command
    command: "golangci-lint run ./..."
    matcher: "implement|fix"        # only trigger on these steps
    blocking: true                   # must pass before step is marked complete

  - name: notify-slack
    event: run_completed
    type: http
    url: "${SLACK_WEBHOOK_URL}"
    blocking: false                  # fire and forget

  - name: security-check
    event: step_completed
    type: llm_judge
    model: claude-haiku-4-5
    prompt: "Review the code changes for security vulnerabilities. Return {\"ok\": true} or {\"ok\": false, \"reason\": \"...\"}"
    matcher: "implement"
    blocking: true
```

### Lifecycle Events

| Event | When | Blocking by default |
|-------|------|---------------------|
| `run_start` | Pipeline begins | Yes |
| `run_completed` | Pipeline succeeds | No |
| `run_failed` | Pipeline fails | No |
| `step_start` | Step begins | Yes |
| `step_completed` | Step succeeds | Yes |
| `step_failed` | Step fails | No |
| `step_retrying` | Step about to retry | No |
| `contract_validated` | Contract check done | No |
| `artifact_created` | Artifact written | No |
| `workspace_created` | Workspace ready | No |

### Hook Types

1. **Command** — shell execution, exit code 0 = proceed, non-zero = block
2. **HTTP** — POST event context as JSON to URL, fire-and-forget or wait for response
3. **LLM Judge** — single-turn LLM evaluation, returns `{"ok": true/false}`
4. **Script** — inline multi-line script (more than a one-liner)

### Matcher Patterns

Regex-based filtering on step names:

```yaml
matcher: "^implement$"                 # exact match
matcher: "implement|fix"               # either
matcher: ".*"                          # all steps (default)
```

### Hook Decisions

Command hooks: exit code 0 = proceed, exit code 2 = block (with JSON reason), other = block.

LLM/HTTP hooks: return `{"ok": true}` or `{"ok": false, "reason": "..."}`.

Advanced: `{"action": "skip"}` to skip the step, `{"action": "override", "target": "fix"}` to redirect.

### Fail-Open vs Fail-Closed

```yaml
hooks:
  - name: security-check
    fail_open: false          # default: true for LLM/HTTP, false for commands
```

## What Wave Keeps

- Contract validation (hooks complement, not replace contracts)
- Persona permissions (hooks are pipeline-level, not persona-level)

## What Wave Gains

- **Auto-formatting** — lint/format after every code change
- **Notifications** — Slack/email on pipeline events
- **Guardrails** — security checks, compliance checks at step boundaries
- **Custom validation** — beyond what contracts can express
- **Extensibility** — pipeline authors can inject custom logic without modifying Wave

## Implementation Scope

1. `internal/pipeline/hooks.go` — hook executor with lifecycle integration
2. Command, HTTP, LLM hook type implementations
3. Matcher evaluation (regex on step names)
4. Hook config in manifest schema
5. Blocking/non-blocking execution
6. Integration with executor lifecycle

## Acceptance Criteria

1. Pipeline authors can define hooks in `wave.yaml` under a top-level `hooks:` key
2. Hooks fire at the correct lifecycle events (10 events defined above)
3. Command hooks execute shell commands and interpret exit codes
4. HTTP hooks POST event context as JSON to configured URLs
5. LLM Judge hooks invoke a single-turn LLM call and parse the JSON response
6. Script hooks execute inline multi-line scripts
7. Regex matchers filter hooks to specific step names
8. Blocking hooks prevent step completion on failure; non-blocking hooks fire-and-forget
9. Fail-open/fail-closed semantics respect defaults per hook type
10. Hook execution emits observable events via the event system
11. Existing contract validation and persona permissions are unaffected
12. Hook timeouts prevent long-running hooks from blocking pipelines indefinitely
