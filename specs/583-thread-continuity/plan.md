# Implementation Plan: Thread Conversation Continuity

## 1. Objective

Add opt-in conversation continuity via thread groups to Wave steps, so that steps sharing a `thread` attribute can access prior conversation transcripts. This enables fix loops where the fixer sees what the implementer did and what failed, while preserving Wave's fresh-memory-by-default security model.

## 2. Approach

### Core Architecture

Introduce a **ThreadManager** that lives on `PipelineExecution` and manages per-thread-group conversation transcripts. The manager:

1. After each step completes, captures its `ResultContent` and appends it to the thread transcript
2. Before a threaded step executes, formats the prior transcript according to the step's fidelity level
3. Prepends the formatted transcript to the step's prompt in `buildStepPrompt()`

This approach uses **prompt-level context injection** rather than Claude Code session continuation (`--continue`/`--resume`), because:
- Wave uses `--no-session-persistence` — sessions are stateless by design
- Prompt injection works across all adapter types (Claude, mock, process group)
- It preserves the security model: each step still starts fresh, but sees prior context as prompt preamble
- Fidelity control is straightforward: just change the formatting

### Fidelity Strategy

| Level | How it works | Token cost |
|-------|-------------|------------|
| `full` | Verbatim transcript with step attribution headers | High |
| `compact` | Step ID + status + truncated content (first 500 chars per entry) | Medium |
| `summary` | LLM-generated summary via relay `CompactionAdapter` | Low (one extra LLM call) |
| `fresh` | No prior context (explicit opt-out within a thread) | Zero |

Default fidelity when `thread` is set: `full`. Default when no `thread`: `fresh` (unchanged behavior).

### Transcript Size Management

- Cap transcript at configurable `maxTranscriptSize` (default 100,000 chars, ~25k tokens)
- When full transcript exceeds cap, truncate oldest entries first
- `summary` fidelity uses the existing relay `CompactionAdapter` to compress

## 3. File Mapping

### New Files

| Path | Purpose |
|------|---------|
| `internal/pipeline/thread.go` | `ThreadManager` struct, transcript storage, fidelity formatting |
| `internal/pipeline/thread_test.go` | Unit tests for ThreadManager |

### Modified Files

| Path | Change |
|------|--------|
| `internal/pipeline/types.go` | Add `Thread`, `Fidelity` fields to `Step` struct; add fidelity constants |
| `internal/pipeline/executor.go` | Initialize ThreadManager on execution; call `GetTranscript()` in `buildStepPrompt()`; call `AppendTranscript()` after step completion |
| `internal/pipeline/types_test.go` | Tests for fidelity validation if added to Step |
| `internal/pipeline/validation.go` | Validate thread/fidelity values during pipeline validation |
| `internal/pipeline/validation_test.go` | Tests for thread/fidelity validation |

### No Changes Needed

- `internal/adapter/` — adapters are unchanged; context injection happens at the prompt level
- `internal/relay/` — reused as-is via `CompactionAdapter` interface for `summary` fidelity
- `internal/state/` — thread transcripts are in-memory per execution, not persisted to SQLite

## 4. Architecture Decisions

### AD-1: Prompt injection over session continuation

**Decision**: Prepend thread transcript to the step's `-p` prompt argument, not use Claude Code's `--continue`/`--resume`.

**Rationale**:
- Wave already uses `--no-session-persistence` (claude.go:396)
- Prompt injection is adapter-agnostic — works with mock adapter for testing
- Preserves security model: each invocation is independent
- Fidelity levels map cleanly to different formatting functions

### AD-2: In-memory transcripts (not persisted)

**Decision**: Store transcripts in `PipelineExecution.ThreadTranscripts` map, not in SQLite state.

**Rationale**:
- Transcripts are only needed during a single pipeline run
- Pipeline resume creates a new execution anyway (fresh transcripts)
- Keeps implementation simple — no schema migration needed
- Can add persistence later if resume-with-threads is needed

### AD-3: Reuse relay CompactionAdapter for summary fidelity

**Decision**: The `summary` fidelity level calls `relay.CompactionAdapter.RunCompaction()` with the full transcript as chat history.

**Rationale**:
- Existing compaction infrastructure handles LLM summarization
- `CompactionConfig` already supports workspace path, chat history, system prompt, and timeout
- Avoids building a parallel summarization system

### AD-4: ThreadManager as separate type

**Decision**: Create `ThreadManager` struct in `thread.go`, not add methods directly to executor.

**Rationale**:
- Single responsibility: ThreadManager owns transcript lifecycle
- Testable in isolation without full executor setup
- Clear interface boundary for future enhancements (e.g., cross-pipeline threads)

### AD-5: Input sanitization for thread transcripts

**Decision**: Sanitize transcript content before injecting into prompts using the existing `InputSanitizer`.

**Rationale**:
- Thread transcripts contain LLM output from prior steps which could contain injection attempts
- Reuses existing security infrastructure
- Consistent with how Wave handles all prompt content

## 5. Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Transcript bloat exhausts context window | Medium | High | Cap at 100k chars; `compact`/`summary` fidelity as escape hatch |
| Prior step output contains prompt injection | Low | Medium | Sanitize via `InputSanitizer` before injection |
| `summary` fidelity LLM call fails | Low | Medium | Fall back to `compact` on compaction error |
| Threaded steps with different personas see conflicting context | Low | Low | Document that persona permissions are still per-step |
| Thread field on graph-mode steps with max_visits creates growing transcripts | Medium | Medium | Each visit appends — transcript cap prevents unbounded growth |

## 6. Testing Strategy

### Unit Tests (`thread_test.go`)

1. **ThreadManager creation** — verify empty state
2. **AppendTranscript** — single entry, multiple entries, multiple threads
3. **GetTranscript (full)** — verbatim output with step headers
4. **GetTranscript (compact)** — truncated entries with step summaries
5. **GetTranscript (summary)** — mock CompactionAdapter, verify it's called with correct content
6. **GetTranscript (fresh)** — returns empty string even when transcript exists
7. **Transcript cap** — verify oldest entries trimmed when exceeding limit
8. **Empty thread** — GetTranscript returns empty for unknown thread
9. **Sanitization** — verify transcript content is sanitized before formatting

### Validation Tests (`validation_test.go`)

10. **Valid thread values** — alphanumeric strings accepted
11. **Invalid fidelity values** — reject unknown fidelity strings
12. **Fidelity without thread** — warn or reject fidelity set without thread

### Integration Tests (`executor_test.go`)

13. **Two-step thread sharing** — step A produces output, step B (same thread) receives it in prompt
14. **Thread isolation** — step in different thread does NOT see step A's output
15. **No thread = fresh** — unthreaded step has no prior context
16. **Graph loop with thread** — verify transcript grows across loop iterations
