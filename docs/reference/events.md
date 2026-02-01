# Event Format Reference

Muzzle emits structured NDJSON (Newline-Delimited JSON) events to stdout on every state transition during pipeline execution. Events are both human-readable in the terminal and machine-parseable for CI/CD integration.

## Event Schema

Every event contains these fields:

| Field | Type | Always Present | Description |
|-------|------|----------------|-------------|
| `timestamp` | `string` | **yes** | ISO 8601 timestamp with timezone. |
| `pipeline_id` | `string` | **yes** | UUID for this pipeline execution instance. |
| `pipeline_name` | `string` | **yes** | Name from pipeline metadata. |
| `step_id` | `string` | **yes** | Step identifier within the pipeline. |
| `state` | `string` | **yes** | New state: `pending`, `running`, `completed`, `failed`, `retrying`. |
| `duration_ms` | `int` | **yes** | Milliseconds elapsed since step started. `0` for initial `pending`/`running` events. |
| `message` | `string` | **yes** | Human-readable status message. |
| `retry_count` | `int` | when retrying | Current retry attempt number. |
| `error` | `string` | when failed/retrying | Error message describing the failure. |
| `artifacts` | `[]string` | when completed | List of output artifact paths. |
| `worker_id` | `string` | matrix steps | Matrix worker identifier (e.g., `"worker-0"`, `"worker-1"`). |
| `meta` | `object` | no | Additional context-specific metadata. |

## Event Examples

### Pipeline Start

```json
{"timestamp":"2026-02-01T10:00:00.000Z","pipeline_id":"a1b2c3d4","pipeline_name":"speckit-flow","step_id":"navigate","state":"pending","duration_ms":0,"message":"Step queued"}
```

### Step Running

```json
{"timestamp":"2026-02-01T10:00:00.500Z","pipeline_id":"a1b2c3d4","pipeline_name":"speckit-flow","step_id":"navigate","state":"running","duration_ms":0,"message":"Starting navigator persona"}
```

### Step Completed

```json
{"timestamp":"2026-02-01T10:01:30.000Z","pipeline_id":"a1b2c3d4","pipeline_name":"speckit-flow","step_id":"navigate","state":"completed","duration_ms":90000,"message":"Navigation complete","artifacts":["output/analysis.json"]}
```

### Contract Failure / Retry

```json
{"timestamp":"2026-02-01T10:03:00.000Z","pipeline_id":"a1b2c3d4","pipeline_name":"speckit-flow","step_id":"implement","state":"retrying","duration_ms":120000,"message":"Contract validation failed","retry_count":1,"error":"test_suite: 3 tests failed (profile.test.js)"}
```

### Step Failed

```json
{"timestamp":"2026-02-01T10:08:00.000Z","pipeline_id":"a1b2c3d4","pipeline_name":"speckit-flow","step_id":"implement","state":"failed","duration_ms":480000,"message":"Step failed after 3 retries","retry_count":3,"error":"test_suite: max retries exceeded"}
```

### Matrix Worker Events

```json
{"timestamp":"2026-02-01T10:02:00.000Z","pipeline_id":"a1b2c3d4","pipeline_name":"parallel-tasks","step_id":"execute","state":"running","duration_ms":0,"message":"Starting matrix worker","worker_id":"worker-0","meta":{"task":"Create migration scripts"}}
{"timestamp":"2026-02-01T10:02:00.100Z","pipeline_id":"a1b2c3d4","pipeline_name":"parallel-tasks","step_id":"execute","state":"running","duration_ms":0,"message":"Starting matrix worker","worker_id":"worker-1","meta":{"task":"Update service APIs"}}
```

### Relay Trigger

```json
{"timestamp":"2026-02-01T10:05:00.000Z","pipeline_id":"a1b2c3d4","pipeline_name":"speckit-flow","step_id":"implement","state":"running","duration_ms":300000,"message":"Context relay triggered at 82% utilization","meta":{"token_usage_percent":82,"strategy":"summarize_to_checkpoint"}}
```

## Output Modes

### JSON (default)

One JSON object per line. Machine-parseable.

```bash
muzzle run --pipeline flow.yaml --input "task" 2>/dev/null
```

### Text

Human-friendly format with color and formatting.

```bash
MUZZLE_LOG_FORMAT=text muzzle run --pipeline flow.yaml --input "task"
```

Text output example:
```
[10:00:00] ● navigate  pending   Step queued
[10:00:01] ▶ navigate  running   Starting navigator persona
[10:01:30] ✓ navigate  completed  Navigation complete (90s)
[10:01:31] ● specify   pending   Step queued
[10:01:31] ▶ specify   running   Starting philosopher persona
```

## Consuming Events

### Pipe to jq

```bash
muzzle run --pipeline flow.yaml --input "task" | jq 'select(.state == "failed")'
```

### CI Integration

```bash
# Exit code reflects pipeline status
muzzle run --pipeline flow.yaml --input "task" > events.jsonl
EXIT_CODE=$?

# Parse events for reporting
cat events.jsonl | jq -r 'select(.state == "completed") | "\(.step_id): \(.duration_ms)ms"'
```

### Real-time Monitoring

```bash
muzzle run --pipeline flow.yaml --input "task" | while IFS= read -r event; do
  state=$(echo "$event" | jq -r '.state')
  step=$(echo "$event" | jq -r '.step_id')
  echo "Step $step is now $state"
done
```

## Event Ordering Guarantees

1. Each step emits exactly one `pending` event before any `running` event.
2. Each step emits exactly one `running` event before `completed`, `failed`, or `retrying`.
3. `retrying` is always followed by another `running` event (the retry attempt).
4. `completed` and `failed` are terminal — no further events for that step.
5. Events within a step are strictly ordered by timestamp.
6. Events across parallel matrix workers are **interleaved** — no ordering guarantee between workers.
