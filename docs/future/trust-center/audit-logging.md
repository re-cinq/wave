# Audit Logging Specification

This document provides the complete specification for Wave's audit logging system. Enterprise security teams can use this specification to integrate Wave audit logs with SIEM systems, compliance monitoring tools, and security dashboards.

## Overview

Wave produces structured audit logs that capture:

- All tool invocations (reads, writes, executions)
- File system operations
- Permission decisions
- Security events
- Contract validations
- Pipeline execution lifecycle

## Log Format

### NDJSON Structure

Audit logs are written as **Newline-Delimited JSON (NDJSON)**, one JSON object per line:

```
{"timestamp":"2026-02-01T10:00:00.000Z","type":"pipeline_start",...}
{"timestamp":"2026-02-01T10:00:01.123Z","type":"tool_call",...}
{"timestamp":"2026-02-01T10:00:02.456Z","type":"file_write",...}
```

Benefits:

- **Streaming** - Logs can be processed line-by-line
- **Append-only** - No file rewriting required
- **Tooling** - Compatible with `jq`, Splunk, Elasticsearch, and other log processors

### File Naming Convention

```
.wave/traces/<pipeline-id>-<pipeline-name>-<timestamp>.ndjson
```

Example:

```
.wave/traces/a1b2c3d4-speckit-flow-2026-02-01T10:00:00.ndjson
```

## Log Entry Schema

### Base Fields

All log entries contain these common fields:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `timestamp` | ISO 8601 | Yes | UTC timestamp with milliseconds |
| `pipeline_id` | string | Yes | Unique pipeline execution ID |
| `type` | string | Yes | Event type (see Event Types below) |

### Event Types

#### Pipeline Lifecycle Events

**`pipeline_start`**

```json
{
  "timestamp": "2026-02-01T10:00:00.000Z",
  "pipeline_id": "a1b2c3d4",
  "type": "pipeline_start",
  "pipeline_name": "speckit-flow",
  "task": "Implement user authentication",
  "manifest_path": "wave.yaml"
}
```

**`pipeline_complete`**

```json
{
  "timestamp": "2026-02-01T10:30:00.000Z",
  "pipeline_id": "a1b2c3d4",
  "type": "pipeline_complete",
  "status": "success",
  "duration_ms": 1800000,
  "steps_completed": 4,
  "steps_total": 4
}
```

**`pipeline_failed`**

```json
{
  "timestamp": "2026-02-01T10:15:00.000Z",
  "pipeline_id": "a1b2c3d4",
  "type": "pipeline_failed",
  "error": "Contract validation failed",
  "failed_step": "implement",
  "duration_ms": 900000
}
```

#### Step Lifecycle Events

**`step_start`**

```json
{
  "timestamp": "2026-02-01T10:01:00.000Z",
  "pipeline_id": "a1b2c3d4",
  "type": "step_start",
  "step_id": "navigate",
  "persona": "navigator",
  "workspace": "/tmp/wave/a1b2c3d4/navigate"
}
```

**`step_complete`**

```json
{
  "timestamp": "2026-02-01T10:05:00.000Z",
  "pipeline_id": "a1b2c3d4",
  "type": "step_complete",
  "step_id": "navigate",
  "persona": "navigator",
  "status": "success",
  "duration_ms": 240000,
  "artifacts_produced": ["analysis.json"]
}
```

#### Tool Call Events

**`tool_call`**

Logged when `log_all_tool_calls: true`:

```json
{
  "timestamp": "2026-02-01T10:01:15.234Z",
  "pipeline_id": "a1b2c3d4",
  "type": "tool_call",
  "step_id": "implement",
  "persona": "craftsman",
  "tool": "Write",
  "args": {
    "path": "src/models/user.go"
  },
  "result": "success",
  "duration_ms": 12
}
```

#### File Operation Events

Logged when `log_all_file_operations: true`:

**`file_read`**

```json
{
  "timestamp": "2026-02-01T10:02:00.000Z",
  "pipeline_id": "a1b2c3d4",
  "type": "file_read",
  "step_id": "navigate",
  "persona": "navigator",
  "path": "src/main.go",
  "size_bytes": 2048
}
```

**`file_write`**

```json
{
  "timestamp": "2026-02-01T10:03:00.000Z",
  "pipeline_id": "a1b2c3d4",
  "type": "file_write",
  "step_id": "implement",
  "persona": "craftsman",
  "path": "src/models/user.go",
  "size_bytes": 1536,
  "operation": "create"
}
```

**`file_delete`**

```json
{
  "timestamp": "2026-02-01T10:04:00.000Z",
  "pipeline_id": "a1b2c3d4",
  "type": "file_delete",
  "step_id": "implement",
  "persona": "craftsman",
  "path": "src/models/user_old.go"
}
```

#### Security Events

**Always logged**, regardless of configuration:

**`permission_denied`**

```json
{
  "timestamp": "2026-02-01T10:02:30.000Z",
  "pipeline_id": "a1b2c3d4",
  "type": "permission_denied",
  "step_id": "navigate",
  "persona": "navigator",
  "tool": "Write",
  "args": {
    "path": "src/config.go"
  },
  "reason": "Tool 'Write' denied for persona 'navigator'"
}
```

**`security_violation`**

```json
{
  "timestamp": "2026-02-01T10:02:45.000Z",
  "pipeline_id": "a1b2c3d4",
  "type": "security_violation",
  "step_id": "implement",
  "violation_type": "path_traversal",
  "severity": "CRITICAL",
  "blocked": true,
  "details": "Path traversal attempt detected"
}
```

**`prompt_injection_detected`**

```json
{
  "timestamp": "2026-02-01T10:02:50.000Z",
  "pipeline_id": "a1b2c3d4",
  "type": "prompt_injection_detected",
  "source": "user_input",
  "severity": "CRITICAL",
  "blocked": true,
  "patterns_matched": ["instruction_override"],
  "action": "rejected"
}
```

#### Contract Events

**`contract_validation`**

```json
{
  "timestamp": "2026-02-01T10:05:00.000Z",
  "pipeline_id": "a1b2c3d4",
  "type": "contract_validation",
  "step_id": "navigate",
  "contract_type": "json_schema",
  "schema": "contracts/analysis.schema.json",
  "result": "pass",
  "validation_duration_ms": 15
}
```

**`contract_violation`**

```json
{
  "timestamp": "2026-02-01T10:05:00.000Z",
  "pipeline_id": "a1b2c3d4",
  "type": "contract_violation",
  "step_id": "navigate",
  "contract_type": "json_schema",
  "schema": "contracts/analysis.schema.json",
  "errors": [
    {
      "path": "$.recommendations",
      "message": "required property missing"
    }
  ]
}
```

#### Hook Events

**`hook_executed`**

```json
{
  "timestamp": "2026-02-01T10:01:30.000Z",
  "pipeline_id": "a1b2c3d4",
  "type": "hook_executed",
  "step_id": "implement",
  "hook_type": "pre",
  "hook_name": "security_check",
  "result": "pass",
  "duration_ms": 500
}
```

## Configuration

### Enabling Audit Logging

```yaml
# wave.yaml
runtime:
  audit:
    log_dir: .wave/traces/
    log_all_tool_calls: true
    log_all_file_operations: true
```

### Logging Levels

| Configuration | Events Logged | Use Case |
|---------------|---------------|----------|
| Default (both `false`) | Security events only | Production, minimal overhead |
| `log_all_tool_calls: true` | All tool calls + security | Debugging, compliance |
| Both `true` | Complete audit trail | Security audits, forensics |

### Log Rotation

Wave does not automatically rotate logs. Implement rotation using standard tools:

```bash
# Example: logrotate configuration
/path/to/.wave/traces/*.ndjson {
    daily
    rotate 30
    compress
    missingok
    notifempty
}
```

## Credential Scrubbing

### Automatic Redaction

Wave automatically redacts credential values before logging:

| Pattern | Example | Logged As |
|---------|---------|-----------|
| `*_KEY` | `sk-ant-api03-...` | `[REDACTED]` |
| `*_TOKEN` | `ghp_xxxx...` | `[REDACTED]` |
| `*_SECRET` | `aws_secret...` | `[REDACTED]` |
| `*_PASSWORD` | `db_password123` | `[REDACTED]` |

### Example: Scrubbed Command

**Before scrubbing:**
```bash
curl -H "Authorization: Bearer sk-ant-api03-xxxx" https://api.example.com
```

**After scrubbing (logged):**
```json
{
  "type": "tool_call",
  "tool": "Bash",
  "args": {
    "command": "curl -H 'Authorization: Bearer [REDACTED]' https://api.example.com"
  }
}
```

## Integration Guide

### SIEM Integration

#### Splunk

```bash
# Forward audit logs to Splunk HTTP Event Collector
tail -f .wave/traces/*.ndjson | \
  curl -X POST \
    -H "Authorization: Splunk $SPLUNK_TOKEN" \
    -d @- \
    https://splunk.example.com:8088/services/collector/raw
```

#### Elasticsearch

```bash
# Index audit logs to Elasticsearch
cat .wave/traces/*.ndjson | \
  while read line; do
    echo '{"index":{"_index":"wave-audit"}}'
    echo "$line"
  done | curl -X POST \
    -H "Content-Type: application/x-ndjson" \
    https://elasticsearch.example.com/_bulk
```

### Querying with jq

**Find all permission denials:**

```bash
cat .wave/traces/*.ndjson | jq 'select(.type == "permission_denied")'
```

**List files written by persona:**

```bash
cat .wave/traces/*.ndjson | \
  jq 'select(.type == "file_write" and .persona == "craftsman") | .path'
```

**Get security events by severity:**

```bash
cat .wave/traces/*.ndjson | \
  jq 'select(.type == "security_violation" and .severity == "CRITICAL")'
```

**Timeline of step execution:**

```bash
cat .wave/traces/*.ndjson | \
  jq 'select(.type | startswith("step_")) | {time: .timestamp, step: .step_id, type: .type}'
```

### CI/CD Integration

#### GitHub Actions

```yaml
- name: Run Wave pipeline
  run: wave run .wave/pipelines/ci-flow.yaml "CI run"

- name: Upload audit logs
  uses: actions/upload-artifact@v4
  if: always()
  with:
    name: wave-audit-logs
    path: .wave/traces/
    retention-days: 90
```

#### GitLab CI

```yaml
wave-pipeline:
  script:
    - wave run .wave/pipelines/ci-flow.yaml "CI run"
  artifacts:
    paths:
      - .wave/traces/
    expire_in: 90 days
    when: always
```

## Schema Download

Download the complete JSON schema for audit log validation:

[Download Audit Log Schema (JSON)](/trust-center/downloads/audit-log-schema.json)

The schema can be used for:

- Log format validation
- SIEM field mapping
- Integration testing
- Documentation generation

## Performance Considerations

### Overhead

| Logging Level | Typical Overhead |
|---------------|------------------|
| Security events only | < 1% |
| All tool calls | 2-5% |
| Full audit trail | 5-10% |

### Disk Usage

Estimate log size:

- **Security events only**: ~1 KB per pipeline
- **All tool calls**: ~10-50 KB per pipeline
- **Full audit trail**: ~50-200 KB per pipeline

Monitor disk usage:

```bash
du -sh .wave/traces/
```

### Best Practices

1. **Production**: Use security-events-only logging
2. **Development**: Enable full audit trail for debugging
3. **Compliance**: Archive logs to long-term storage
4. **Monitoring**: Set up alerts for security events

## Further Reading

- [Security Model](./security-model) - Complete security architecture
- [Compliance Roadmap](./compliance) - Certification status
- [Enterprise Patterns](/guides/enterprise) - Enterprise deployment guide
- [Audit Logging Guide](/guides/audit-logging) - Practical configuration guide

---

*Last updated: February 2026*
