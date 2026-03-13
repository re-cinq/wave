# Tasks

## Phase 1: Schema and Types

- [X] Task 1.1: Add `ChatContextConfig` struct to `internal/pipeline/types.go`
  - Fields: `ArtifactSummaries []string`, `SuggestedQuestions []string`, `FocusAreas []string`, `MaxContextTokens int`
  - Add `ChatContext ChatContextConfig` field to `Pipeline` struct with `yaml:"chat_context,omitempty"`
  - Ensure `omitempty` for backward compatibility

- [X] Task 1.2: Add `ArtifactContent` field to `ChatContext` struct in `internal/pipeline/chatcontext.go`
  - New field: `ArtifactContents map[string]string` (artifact name → content/summary)
  - New field: `ChatConfig *ChatContextConfig` (from pipeline definition)

## Phase 2: Artifact Summarization

- [X] Task 2.1: Create `internal/pipeline/artifact_summary.go` with summarization logic [P]
  - `SummarizeArtifact(path string, maxBytes int) (string, error)` — reads and summarizes artifact content
  - JSON artifacts: extract top-level keys and first-level values (truncated)
  - Markdown artifacts: extract headings and first paragraph
  - Plain text: first N lines (respecting maxBytes)
  - Binary detection: check for null bytes, return `"[binary file, N bytes]"`
  - Large file handling: truncate with `"... (truncated, full content at <path>)"`

- [X] Task 2.2: Create `internal/pipeline/artifact_summary_test.go` with tests [P]
  - Test JSON summarization (small, large, nested)
  - Test markdown summarization (headings, paragraphs)
  - Test plain text truncation
  - Test binary file detection
  - Test missing file handling
  - Test empty file handling

## Phase 3: Context Assembly

- [X] Task 3.1: Modify `BuildChatContext()` in `chatcontext.go` to load artifact content
  - If `pipeline.ChatContext` is defined and has `ArtifactSummaries`, load content for each listed artifact
  - Use `SummarizeArtifact()` to generate summaries
  - Populate `ChatContext.ArtifactContents` map
  - Carry `ChatContext.ChatConfig` from pipeline definition
  - Respect token budget from `MaxContextTokens` (default 8000, ~32KB text)

- [X] Task 3.2: Update `chatcontext_test.go` with new test cases
  - Test artifact content loading with mock filesystem
  - Test token budget enforcement
  - Test missing artifact handling (non-fatal)
  - Test `ChatConfig` propagation from pipeline to context

## Phase 4: CLAUDE.md Enhancement

- [X] Task 4.1: Extend `buildChatClaudeMd()` in `chatworkspace.go` to inject artifact content
  - New section: `## Key Artifact Content` — fenced code blocks with artifact summaries
  - New section: `## Suggested Questions` — pipeline-specific opening questions
  - New section: `## Focus Areas` — areas to pay attention to
  - New section: `## Post-Mortem Questions` — auto-generated context-aware questions
  - All new sections are conditional (only if `ChatConfig` is set or auto-generated)

- [X] Task 4.2: Implement auto-generated post-mortem questions in `chatworkspace.go`
  - Generate 3 questions based on run outcome:
    - If failed: "What caused the failure in step X?", "Can the failure be resolved by...", "Should we retry with..."
    - If completed with PR: "Would you like to review the changes in PR X?", "Are there any edge cases...", "Should we add additional tests?"
    - If completed analysis: "What are the key findings?", "Which items need immediate attention?", "What's the recommended next step?"
  - Use artifact names, step statuses, and pipeline category to make questions specific

- [X] Task 4.3: Update `chatworkspace_test.go` with enriched CLAUDE.md tests
  - Test artifact content section appears when content is loaded
  - Test suggested questions section appears
  - Test auto-generated questions are context-specific
  - Test backward compat: no `ChatConfig` = existing behavior unchanged
  - Test token budget truncation

## Phase 5: Pipeline YAML Updates

- [X] Task 5.1: Add `chat_context` to `gh-implement.yaml` pipeline [P]
  - Suggested questions: "Would you like to review the changes?", "Are there failing tests to investigate?", "Should we refine the implementation?"
  - Artifact summaries: reference key output artifacts
  - Focus areas: "code changes", "test results", "PR status"

- [X] Task 5.2: Add `chat_context` to `gh-pr-review.yaml` pipeline [P]
  - Suggested questions: "What issues were found in the review?", "Are there blocking concerns?", "What should be fixed before merging?"
  - Focus areas: "review findings", "code quality", "security concerns"

## Phase 6: Validation and Polish

- [X] Task 6.1: Run `go test ./...` and fix any failures
- [X] Task 6.2: Run `go test -race ./...` to check for data races
- [X] Task 6.3: Run `golangci-lint run ./...` for static analysis
- [X] Task 6.4: Verify existing chat tests still pass (regression)
