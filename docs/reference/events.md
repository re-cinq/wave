# Event Format Reference

Wave emits structured NDJSON (Newline-Delimited JSON) events to stdout on every state transition during pipeline execution. Events are both human-readable in the terminal and machine-parseable for CI/CD integration.

## Event Schema

Every event contains these fields:

| Field | Type | Always Present | Description |
|-------|------|----------------|-------------|
| `timestamp` | `string` | **yes** | ISO 8601 timestamp with timezone. |
| `pipeline_id` | `string` | **yes** | UUID for this pipeline execution instance. |
| `step_id` | `string` | no | Step identifier within the pipeline. |
| `state` | `string` | **yes** | New state: `started`, `running`, `completed`, `failed`, `retrying`. |
| `duration_ms` | `int` | no | Milliseconds elapsed since step started. |
| `message` | `string` | no | Human-readable status message. |
| `persona` | `string` | no | Persona executing the step. |
| `artifacts` | `[]string` | when completed | List of output artifact paths. |
| `tokens_used` | `int` | when completed | Token count for the step. |

## Event Examples

### Pipeline Start

```json
{"timestamp":"2026-02-01T10:00:00.500Z","pipeline_id":"a1b2c3d4","step_id":"navigate","state":"started","message":"Starting navigator persona","persona":"navigator"}
```

### Step Completed

```json
{"timestamp":"2026-02-01T10:01:30.000Z","pipeline_id":"a1b2c3d4","step_id":"navigate","state":"completed","duration_ms":90000,"message":"Navigation complete","persona":"navigator","artifacts":["output/analysis.json"],"tokens_used":3200}
```

### Step Failed

```json
{"timestamp":"2026-02-01T10:08:00.000Z","pipeline_id":"a1b2c3d4","step_id":"implement","state":"failed","duration_ms":480000,"message":"Step failed after 3 retries","persona":"craftsman"}
```

## Output Modes

### JSON (default)

One JSON object per line. Machine-parseable.

```bash
wave run flow "task" 2>/dev/null
```

### Text

Human-friendly format with color and formatting.

```bash
WAVE_LOG_FORMAT=text wave run flow "task"
```

Text output example:
```
[10:00:01] → navigate (navigator)
[10:00:01]   navigate: Executing agent
[10:01:30] ✓ navigate completed (89.0s, 3.2k tokens)
[10:01:31] → specify (philosopher)
[10:01:31]   specify: Executing agent
```

## Consuming Events

### Pipe to jq

```bash
wave run flow "task" | jq 'select(.state == "failed")'
```

### CI Integration

```bash
# Exit code reflects pipeline status
wave run flow "task" > events.jsonl
EXIT_CODE=$?

# Parse events for reporting
cat events.jsonl | jq -r 'select(.state == "completed") | "\(.step_id): \(.duration_ms)ms"'
```

### Real-time Monitoring

```bash
wave run flow "task" | while IFS= read -r event; do
  state=$(echo "$event" | jq -r '.state')
  step=$(echo "$event" | jq -r '.step_id')
  echo "Step $step is now $state"
done
```

## Event Ordering Guarantees

1. Each step emits a `started` event before work begins.
2. `completed` and `failed` are terminal — no further events for that step.
3. Events within a step are strictly ordered by timestamp.
