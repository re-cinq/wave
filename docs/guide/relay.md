# Relay Guide

Relay is Wave's context compaction mechanism. When an agent approaches its token limit, relay summarizes the conversation and resumes with fresh context.

## Why Relay?

LLMs have finite context windows. As agents work on complex tasks, conversations grow and quality degrades. Relay solves this by:

1. Detecting when context usage exceeds a threshold
2. Using a summarizer persona to create a checkpoint
3. Resuming with fresh context and the checkpoint

## How It Works

```
Agent working (context growing)
         |
Token threshold reached (80%)
         |
Summarizer creates checkpoint
         |
Fresh agent starts with checkpoint
         |
Work continues
```

## Configuration

### Global Settings

In `wave.yaml`:

```yaml
runtime:
  relay:
    token_threshold_percent: 80
    strategy: summarize_to_checkpoint
```

### Per-Step Settings

In pipeline steps:

```yaml
steps:
  - id: implement
    persona: craftsman
    handover:
      compaction:
        trigger: "token_limit_80%"
        persona: summarizer
```

| Field | Default | Description |
|-------|---------|-------------|
| `token_threshold_percent` | `80` | Context usage trigger (50-95) |
| `trigger` | `token_limit_80%` | Step-level condition |
| `persona` | `summarizer` | Checkpoint persona |

## The Summarizer Persona

A specialized persona for creating checkpoints:

```yaml
personas:
  summarizer:
    adapter: claude
    system_prompt_file: .wave/personas/summarizer.md
    temperature: 0.0
    permissions:
      allowed_tools: ["Read"]
      deny: ["Write(*)", "Bash(*)"]
```

### Summarizer System Prompt

```markdown
# Summarizer Persona

Create structured checkpoints for context relay.

## Output Sections

### Completed Work
- List actions with file paths
- Note decisions made

### Current State
- Modified files
- Test/build status

### Remaining Work
- Tasks not started
- Partial tasks
- Blockers

### Resume Instructions
- Specific next step
- Important context
```

## Checkpoint Structure

```markdown
# Checkpoint: Feature Implementation

## Completed Work
- Created user model at src/models/user.go
- Implemented CreateUser endpoint
- Tests: 12/12 passing

## Current State
- Modified: src/models/user.go, src/api/users.go
- Build: successful

## Remaining Work
- [ ] Implement GetUser endpoint
- [ ] Add integration tests

## Resume Instructions
Next: Implement GetUser. Follow pattern in CreateUser.
```

## Enabling Relay

For long-running steps:

```yaml
- id: implement
  persona: craftsman
  handover:
    compaction:
      trigger: "token_limit_80%"
      persona: summarizer
    contract:
      type: test_suite
      command: "go test ./..."
```

## Relay Limits

Prevent infinite loops:

```yaml
runtime:
  relay:
    token_threshold_percent: 80
  meta_pipeline:
    max_total_tokens: 500000
```

If the summarizer hits its token limit, the pipeline halts.

## Monitoring

Relay events in stdout:
```json
{"step_id":"implement","event":"relay_triggered","context_usage":0.82}
{"step_id":"implement","event":"checkpoint_created"}
{"step_id":"implement","event":"relay_resumed","instance":2}
```

## Best Practices

### When to Enable

Enable for:
- Complex implementation work
- Multiple iterations expected
- Large codebase processing

### Threshold Tuning

| Scenario | Threshold |
|----------|-----------|
| Short tasks | 85-90% |
| Complex tasks | 70-75% |
| Large codebases | 75-80% |

### Checkpoint Quality

Good checkpoints are:
- **Complete** - All relevant information
- **Structured** - Clear sections
- **Actionable** - Explicit next steps
- **Concise** - No repetition

## Troubleshooting

**Relay loops**: Check if task is completable, review checkpoint quality, lower threshold.

**Lost context**: Improve summarizer prompt, ensure "Completed Work" is detailed.

**Summarizer failures**: Check permissions (needs Read), verify temperature is 0.0.

## Related Topics

- [Manifest Schema Reference](/reference/manifest-schema) - Relay configuration
- [Pipeline Schema Reference](/reference/pipeline-schema) - Compaction settings
- [Personas Guide](/guide/personas) - Summarizer persona setup
