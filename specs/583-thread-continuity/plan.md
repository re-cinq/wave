# Implementation Plan: Thread Continuity

## Objective

Add opt-in conversation continuity to Wave's pipeline executor via `thread` and `fidelity` step attributes, enabling steps in the same thread group to share conversation context while preserving fresh-memory-by-default security.

## Approach

### Strategy: Transcript-Based Context Injection

After evaluating the codebase, the most robust approach is **transcript-based context injection** rather than Claude Code `--resume` sessions. Reasons:

1. **`--resume` is interactive-mode only** — Wave's pipeline adapter (`ClaudeAdapter.buildArgs`) explicitly uses `--no-session-persistence` and `-p` (non-interactive) mode. Session IDs are only captured in `interactive.go`, not the pipeline adapter.
2. **Adapter-agnostic** — transcript injection works with any adapter (Claude, Gemini, mock), not just Claude Code.
3. **Fidelity control** — transcript text can be sliced/summarized per fidelity level, whereas `--resume` is all-or-nothing.
4. **Resume-safe** — transcripts stored in `PipelineExecution` survive state serialization; session IDs may not.

### Architecture

```
Step execution flow (threaded step):

1. Executor checks step.Thread != ""
2. ThreadManager.GetTranscript(threadGroup, fidelity) returns prior context
3. Transcript is prepended to the step's prompt as a structured preamble
4. After execution, ThreadManager.AppendTranscript(threadGroup, stepID, stdout)
5. Transcript stored in PipelineExecution for resume support
```

### Key Components

1. **ThreadManager** — new component in `internal/pipeline/` that tracks conversation transcripts per thread group within a pipeline execution
2. **Step schema extension** — `thread` and `fidelity` fields on `Step` struct
3. **Executor integration** — transcript injection in `runStepExecution`, capture in result processing
4. **Fidelity preamble generator** — transforms raw transcripts into fidelity-appropriate context
5. **Validation** — DAG validation ensures thread groups are well-formed

## File Mapping

### New Files

| File | Purpose |
|------|---------|
| `internal/pipeline/thread.go` | `ThreadManager` — transcript storage, retrieval, fidelity-based formatting |
| `internal/pipeline/thread_test.go` | Unit tests for ThreadManager |

### Modified Files

| File | Change |
|------|--------|
| `internal/pipeline/types.go` | Add `Thread` and `Fidelity` fields to `Step` struct |
| `internal/pipeline/executor.go` | Integrate ThreadManager: inject transcript before adapter call, capture stdout after |
| `internal/pipeline/validation.go` | Validate thread group consistency (same workspace, persona compatibility) |
| `internal/pipeline/dryrun_test.go` | Add thread field to dry-run validation tests |
| `internal/pipeline/template_test.go` | Add thread field to template detection tests |

### Files NOT Modified

| File | Reason |
|------|--------|
| `internal/adapter/adapter.go` | No changes to `AdapterRunConfig` — transcript is prepended to the prompt string, not a new config field |
| `internal/adapter/claude.go` | No changes to CLI args — transcript flows through the existing prompt path |
| `internal/relay/relay.go` | Relay compaction is orthogonal; `summary` fidelity uses the same `CompactionAdapter` interface but is invoked from ThreadManager, not relay |

## Architecture Decisions

### AD-1: Transcript in prompt vs. system prompt

**Decision**: Prepend transcript to the **user prompt** (`-p` argument), not the system prompt.

**Rationale**: The system prompt is the persona's agent `.md` file built from 5 layers (base protocol, ontology, persona, contract, restrictions). Injecting conversation history there would violate the clean persona boundary. The user prompt is already where task context flows (input, artifacts, contract schema). A `## Prior Conversation Context` section at the top of the prompt is consistent with this pattern.

### AD-2: Transcript storage location

**Decision**: Store transcripts in a new `ThreadTranscripts map[string][]ThreadEntry` field on `PipelineExecution`.

**Rationale**: `PipelineExecution` already holds all per-run state (Results, ArtifactPaths, WorkspacePaths, AttemptContexts). Thread transcripts are per-run state that must survive concurrent step access (already protected by `execution.mu`). This avoids a new persistence layer.

### AD-3: Fidelity "summary" uses relay CompactionAdapter

**Decision**: The `summary` fidelity level reuses the existing `relay.CompactionAdapter` interface to LLM-summarize transcripts.

**Rationale**: The relay package already has a battle-tested `RunCompaction()` path that spawns a summarizer with controlled tools (read-only) and low temperature (0.3). Building a parallel summarization path would be redundant.

### AD-4: Thread validation is a warning, not a hard error

**Decision**: Thread groups with mixed personas emit a **warning** but are allowed. Thread groups where steps have no shared workspace emit a **warning**.

**Rationale**: The issue explicitly states persona permissions are enforced per-step even within a thread. Mixed personas in a thread are unusual but valid (e.g., implement → review where the reviewer sees what was done). Blocking this would be overly restrictive.

### AD-5: Max transcript size with automatic truncation

**Decision**: Transcripts are capped at a configurable size (default 100K chars). When exceeded, oldest entries are truncated first, preserving the most recent conversation context.

**Rationale**: Without bounds, a long fix loop (5+ iterations) could exhaust the context window. The cap ensures the transcript preamble doesn't crowd out the step's own prompt and contract schema.

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Transcript too large for context window | Medium | High | Max transcript size cap with oldest-first truncation; `compact` fidelity as fallback |
| Thread + retry interaction | Low | Medium | Retry attempts within a threaded step append to the same thread — prior attempts provide context |
| Resume with thread state | Medium | Medium | ThreadTranscripts serialized with PipelineExecution state; verified in tests |
| Concurrent steps in same thread | Low | High | Validate: steps in same thread cannot be concurrent (must have dependency chain). Emit error during DAG validation |
| Performance: large transcript prepend | Low | Low | Transcript is just string concatenation in prompt; no extra adapter calls except for `summary` fidelity |

## Testing Strategy

### Unit Tests (`internal/pipeline/thread_test.go`)

1. **ThreadManager basic operations**: append, retrieve, clear
2. **Fidelity formatting**: full returns raw transcript, compact returns structured summary, fresh returns empty
3. **Transcript size cap**: verify truncation when exceeding max size
4. **Thread isolation**: verify different thread groups don't share transcripts
5. **Empty thread**: verify no-op when thread group has no prior entries

### Integration in Existing Tests

6. **Validation tests** (`validation.go`): thread groups with concurrent steps are rejected
7. **Dry-run tests** (`dryrun_test.go`): threaded steps show transcript injection in dry-run output
8. **Executor tests**: end-to-end threaded step execution with mock adapter, verifying transcript appears in prompt

### Manual Validation

9. Create a test pipeline with `thread: impl` on implement + fix steps
10. Verify fix step receives implement step's conversation in its prompt
11. Verify non-threaded steps get no prior context (fresh memory preserved)
