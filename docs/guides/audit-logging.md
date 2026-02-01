# Audit Logging

Muzzle can log all tool calls and file operations during pipeline execution, producing structured audit trails for compliance, debugging, and monitoring.

## Enabling Audit Logging

```yaml
# In muzzle.yaml
runtime:
  audit:
    log_dir: .muzzle/traces/           # Output directory
    log_all_tool_calls: true           # Log every tool invocation
    log_all_file_operations: true      # Log every file read/write/delete
```

## Audit Log Format

Audit logs are written as NDJSON files, one per pipeline execution:

```
.muzzle/traces/
├── a1b2c3d4-speckit-flow-2026-02-01T10:00:00.ndjson
├── e5f6a7b8-bug-fix-2026-01-30T14:30:00.ndjson
└── ...
```

### Log Entry Schema

```json
{
  "timestamp": "2026-02-01T10:01:15.234Z",
  "pipeline_id": "a1b2c3d4",
  "step_id": "implement",
  "persona": "craftsman",
  "type": "tool_call",
  "tool": "Write",
  "args": {
    "path": "src/models/user.go"
  },
  "result": "success",
  "duration_ms": 12
}
```

### Entry Types

| Type | Description | Requires |
|------|-------------|----------|
| `tool_call` | Any tool invocation (Read, Write, Bash, etc.) | `log_all_tool_calls: true` |
| `file_read` | File read operation | `log_all_file_operations: true` |
| `file_write` | File write or edit | `log_all_file_operations: true` |
| `file_delete` | File deletion | `log_all_file_operations: true` |
| `permission_denied` | Blocked tool call | Always logged |
| `hook_executed` | Pre/post hook execution | Always logged |
| `contract_validation` | Contract check result | Always logged |

## Credential Scrubbing

Audit logs **never** capture credential values. Muzzle automatically redacts environment variables matching these patterns:

- `*_KEY` → `[REDACTED]`
- `*_TOKEN` → `[REDACTED]`
- `*_SECRET` → `[REDACTED]`
- `*_PASSWORD` → `[REDACTED]`
- `*_CREDENTIAL*` → `[REDACTED]`

Example:
```json
{
  "type": "tool_call",
  "tool": "Bash",
  "args": {
    "command": "curl -H 'Authorization: Bearer [REDACTED]' https://api.example.com"
  }
}
```

## Querying Audit Logs

### Find All Writes by a Persona

```bash
cat .muzzle/traces/a1b2c3d4-*.ndjson \
  | jq 'select(.persona == "craftsman" and .type == "file_write")'
```

### Find Permission Denials

```bash
cat .muzzle/traces/a1b2c3d4-*.ndjson \
  | jq 'select(.type == "permission_denied")'
```

### Summarize Tool Usage

```bash
cat .muzzle/traces/a1b2c3d4-*.ndjson \
  | jq -r 'select(.type == "tool_call") | .tool' \
  | sort | uniq -c | sort -rn
```

### Get Timeline of a Step

```bash
cat .muzzle/traces/a1b2c3d4-*.ndjson \
  | jq 'select(.step_id == "implement") | {time: .timestamp, type: .type, tool: .tool}'
```

## Audit Logging Levels

| Configuration | What's Logged | Use Case |
|--------------|---------------|----------|
| Both `false` | Only security events (denials, hooks, contracts) | Production, low overhead |
| `log_all_tool_calls: true` | All tool invocations + security events | Debugging, compliance |
| Both `true` | Everything — full file-level audit trail | Security audits, forensics |

## Performance Impact

Audit logging adds minimal overhead:

- Logs are written asynchronously to disk.
- NDJSON format is append-only (no parsing or rewriting).
- File I/O is buffered.

For long-running pipelines, audit log files can grow large. Monitor disk usage in `.muzzle/traces/`.

## CI/CD Integration

```yaml
# GitHub Actions: upload audit logs as artifacts
- name: Run Muzzle pipeline
  run: muzzle run --pipeline flow.yaml --input "deploy"
  env:
    ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}

- name: Upload audit logs
  uses: actions/upload-artifact@v4
  if: always()
  with:
    name: muzzle-audit-logs
    path: .muzzle/traces/
```

## Further Reading

- [Manifest Schema — AuditConfig](/reference/manifest-schema#auditconfig) — field reference
- [Environment & Credentials](/reference/environment) — credential handling model
- [Events](/reference/events) — pipeline event format (separate from audit logs)
