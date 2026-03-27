# Step Conversation Continuity via Thread IDs for Fix Loops

**Issue**: [re-cinq/wave#583](https://github.com/re-cinq/wave/issues/583)
**Labels**: enhancement
**Author**: nextlevelshit
**Complexity**: complex

## Context

Fabro supports thread IDs where nodes sharing a `thread_id` participate in the same conversation thread. Combined with `fidelity="full"`, this preserves full conversation history across multiple nodes. This is critical for fix loops where the fixer needs to see what the implementer did and what tests failed.

Wave currently enforces strict fresh memory at every step boundary — no chat history inheritance, only explicit artifacts. While this is a security strength, it creates a blind spot for fix loops where conversation continuity dramatically improves fix quality.

## Design Goals — Best of Both Worlds

Add **opt-in** conversation continuity while preserving Wave's fresh-memory-by-default security model.

### Thread Groups

Steps can share a conversation thread via `thread` attribute:

```yaml
steps:
  - name: implement
    persona: craftsman
    thread: impl              # start thread "impl"

  - name: fix
    persona: craftsman
    thread: impl              # continue thread "impl"
    max_visits: 5

  - name: review
    persona: navigator
    # no thread — fresh memory (default)
```

Steps in the same thread group:
1. Share conversation history from the adapter session
2. Each iteration appends to the conversation, not starts fresh
3. The fix step sees exactly what the implement step did and what failed

### Implementation Approach

For Claude Code CLI adapter, thread continuity could work via:
- **Session continuation**: Use `--continue` or `--resume` flags if the adapter supports it
- **Context injection**: Capture the full conversation transcript from step N, inject as system context for step N+1
- **Shared session ID**: If the adapter supports session persistence

Fallback for adapters that don't support continuation: inject previous conversation as a preamble artifact (like Fabro's `fidelity="compact"` mode).

### Fidelity Control

Control how much prior context a step receives from its thread:

```yaml
steps:
  - name: fix
    thread: impl
    fidelity: full            # full conversation history (default for threaded)

  - name: summarize
    thread: impl
    fidelity: compact         # summary of prior steps only
```

Fidelity levels:
- `full` — complete conversation history (default when `thread` is set)
- `compact` — goal + completed steps summary + context vars
- `summary` — LLM-generated summary of prior conversation
- `fresh` — no prior context (default when no `thread`, Wave's current behavior)

### Security Model

- Thread groups are **opt-in** — default remains fresh memory
- Threads are scoped to a single pipeline run — cannot cross pipeline boundaries
- Persona permissions still enforced per-step even within a thread
- Contract validation still runs at step boundaries

## Acceptance Criteria

1. Steps can declare `thread: <group-name>` in pipeline YAML to join a conversation thread
2. Steps in the same thread group receive prior conversation context based on fidelity level
3. Default behavior (no `thread` field) remains fresh memory — no breaking change
4. Thread groups are scoped to a single pipeline run
5. Persona permissions and contract validation remain enforced at step boundaries
6. The `fidelity` field controls context density: `full`, `compact`, `summary`, `fresh`
7. Adapters that don't support native session continuation fall back to transcript injection
8. Thread state is captured in `PipelineExecution` for resume support
9. Comprehensive tests cover thread isolation, fidelity levels, and security boundaries
