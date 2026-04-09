# Thread Continuity

Threads enable conversation continuity across pipeline steps. Steps sharing the same `thread` value receive transcripts from prior steps in the thread, enabling multi-step reasoning chains where each step builds on previous context.

## Basic Thread

```yaml
steps:
  - id: research
    persona: navigator
    thread: analysis
    exec:
      type: prompt
      source: "Research the problem space"

  - id: synthesize
    persona: navigator
    thread: analysis
    dependencies: [research]
    exec:
      type: prompt
      source: "Synthesize findings into recommendations"

  - id: implement
    persona: craftsman
    thread: impl
    dependencies: [synthesize]
    exec:
      type: prompt
      source: "Implement the recommendations"
```

The `research` and `synthesize` steps share the `analysis` thread, so `synthesize` receives the full conversation history from `research`. The `implement` step starts a new `impl` thread with fresh context.

## Fidelity Levels

Control how much prior context each step receives with the `fidelity` field:

| Level | Description | Use When |
|-------|-------------|----------|
| `full` | Complete conversation history | Deep reasoning chains, debugging |
| `compact` | Step ID + status + truncated summary | Long pipelines, token budget concerns |
| `summary` | LLM-generated summary via compaction | Very long conversations, cross-domain handoffs |
| `fresh` | No prior context | Independent work, security isolation |

Default: `full` when `thread` is set, `fresh` when no thread.

### Fidelity Example

```yaml
steps:
  - id: deep-analysis
    persona: navigator
    thread: review
    fidelity: full
    exec:
      type: prompt
      source: "Perform deep analysis"

  - id: quick-check
    persona: auditor
    thread: review
    fidelity: compact
    dependencies: [deep-analysis]
    exec:
      type: prompt
      source: "Verify the analysis"

  - id: summarize
    persona: navigator
    thread: review
    fidelity: summary
    dependencies: [quick-check]
    exec:
      type: prompt
      source: "Write the final summary"
```

## Multiple Threads

A pipeline can use multiple independent threads. Each thread group maintains its own conversation context.

```yaml
steps:
  - id: plan-frontend
    persona: navigator
    thread: frontend

  - id: plan-backend
    persona: navigator
    thread: backend

  - id: impl-frontend
    persona: craftsman
    thread: frontend
    dependencies: [plan-frontend]

  - id: impl-backend
    persona: craftsman
    thread: backend
    dependencies: [plan-backend]

  - id: integration
    persona: craftsman
    thread: integration
    dependencies: [impl-frontend, impl-backend]
```

The `frontend` and `backend` threads run independently. The `integration` step starts a new thread since it needs to combine results from both.

## When to Use Threads

**Use threads when:**
- Steps build on each other's reasoning (research, then synthesize, then implement)
- A persona needs to remember earlier decisions in the same pipeline
- Graph loops need context from previous iterations

**Avoid threads when:**
- Steps are independent and don't need prior context
- Token budget is tight (use artifacts instead)
- Different personas handle different concerns (use artifact injection)

## See Also

- [Pipeline Schema: Threads](/reference/pipeline-schema#threads) - Field reference
- [Relay Compaction](/guides/relay-compaction) - Managing context size in long threads
- [Graph Loops](/guide/graph-loops) - Using threads with feedback loops
