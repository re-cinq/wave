# Implementation Plan: Chat Context Injection (#276)

## 1. Objective

Enable Wave chat sessions to start with pre-loaded pipeline context — artifact content, pipeline-specific suggested questions, and focus areas — so the chat agent can answer substantive questions immediately without file discovery delay.

## 2. Approach

**Two-tier context injection:**

1. **Pipeline-level `chat_context` configuration** in pipeline YAML — allows each pipeline to declare what artifacts to summarize, what questions to suggest, and what focus areas to highlight.
2. **Enhanced CLAUDE.md generation** — the existing `buildChatClaudeMd()` function is extended to inject artifact content/summaries and pipeline-specific opening prompts directly into the chat workspace's CLAUDE.md.

This approach builds on the existing chat infrastructure (`chatcontext.go`, `chatworkspace.go`, `interactive.go`) rather than introducing new injection mechanisms. The CLAUDE.md file is already the primary way context reaches the chat agent — we just make it richer.

### Why CLAUDE.md, not `--system-prompt`?

The existing chat system uses CLAUDE.md as the context carrier (not `--system-prompt`). CLAUDE.md has no practical token limit, supports markdown formatting, and is already the established pattern. Adding a parallel `--system-prompt` path would complicate the architecture for marginal benefit.

## 3. File Mapping

| File | Action | Description |
|------|--------|-------------|
| `internal/pipeline/types.go` | modify | Add `ChatContextConfig` struct and `ChatContext` field to `Pipeline` |
| `internal/pipeline/chatcontext.go` | modify | Add artifact content loading, carry `ChatContextConfig` in `ChatContext` |
| `internal/pipeline/chatworkspace.go` | modify | Inject artifact summaries, suggested questions, focus areas, auto-generated post-mortem questions into CLAUDE.md |
| `internal/pipeline/artifact_summary.go` | create | Artifact summarization logic (JSON key extraction, markdown heading extraction, size-bounded content) |
| `internal/pipeline/chatcontext_test.go` | modify | Test `ChatContextConfig` propagation, artifact content loading |
| `internal/pipeline/chatworkspace_test.go` | modify | Test enriched CLAUDE.md generation with artifact content and suggested questions |
| `internal/pipeline/artifact_summary_test.go` | create | Test artifact summarization (JSON, markdown, large files, binary) |
| `.wave/pipelines/gh-implement.yaml` | modify | Add example `chat_context` section |
| `.wave/pipelines/gh-pr-review.yaml` | modify | Add example `chat_context` section |

## 4. Architecture Decisions

### 4.1 Pipeline-level, not step-level

`chat_context` is a **pipeline-level** field (on `Pipeline`), not step-level. Rationale: chat sessions are about the whole pipeline run, not individual steps. Individual steps already declare `output_artifacts` — the pipeline-level `chat_context` selects *which* of those artifacts to surface in chat.

### 4.2 Artifact content injection via CLAUDE.md

Artifact content is injected directly into CLAUDE.md as fenced code blocks, not as separate files. This ensures the chat agent sees the content immediately in its context window without needing to read files.

For large artifacts (>4KB), only a summary is injected (first N lines, JSON top-level keys, or markdown headings). The full artifact path is always listed so the agent can read the full content on demand.

### 4.3 Token budget

A configurable `max_context_tokens` (default: 8000) limits total injected artifact content. Artifacts are injected in declaration order; once the budget is exhausted, remaining artifacts get path-only references.

### 4.4 Auto-generated post-mortem questions

When no `suggested_questions` are configured, the system generates 3 context-aware questions based on:
- Pipeline type (implementation → "review changes?", analysis → "review findings?")
- Run status (failed → "diagnose the failure?", completed → "review the output?")
- Artifact types present (PR artifact → "review the PR?", test results → "review test results?")

### 4.5 Backwards compatibility

Pipelines without `chat_context` work exactly as before — the existing CLAUDE.md template is unchanged; the new sections are additive.

## 5. Risks

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Token budget exceeded with large artifacts | Medium | Enforce `max_context_tokens` with truncation; always include artifact paths as fallback |
| Binary/non-text artifacts crash summarization | Low | Detect binary content (null bytes) and skip with path-only reference |
| YAML parsing breaks for existing pipelines | Low | `chat_context` field is `omitempty`; missing field = nil pointer = no-op |
| Chat CLAUDE.md becomes too long | Medium | Token budget + summary-only mode for large artifacts |

## 6. Testing Strategy

| Test | Type | Coverage |
|------|------|----------|
| `ChatContextConfig` YAML parsing round-trip | Unit | `types.go` |
| `ChatContext` carries `ChatContextConfig` from pipeline | Unit | `chatcontext.go` |
| Artifact content loading (JSON, markdown, missing file) | Unit | `chatcontext.go` |
| Artifact summarization (JSON top-keys, markdown headings, truncation, binary) | Unit | `artifact_summary.go` |
| Enriched CLAUDE.md with artifact content section | Unit | `chatworkspace.go` |
| Enriched CLAUDE.md with suggested questions | Unit | `chatworkspace.go` |
| Auto-generated post-mortem questions by pipeline type | Unit | `chatworkspace.go` |
| Token budget enforcement | Unit | `chatworkspace.go` |
| Full integration: pipeline YAML → chat workspace → CLAUDE.md content | Integration | `chatworkspace.go` |
| Existing chat behavior unchanged without `chat_context` | Regression | `chatworkspace_test.go` |
