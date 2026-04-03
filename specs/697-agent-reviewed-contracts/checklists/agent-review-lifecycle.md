# Agent Review Lifecycle Quality Review

**Feature**: #697 | **Dimension**: Reviewer spawning, context assembly, feedback extraction

## Completeness

- [ ] CHK201 - Does the spec define the reviewer agent's tool permissions beyond "read-only"? Can the reviewer run Bash commands (e.g., `go vet`, `grep`) in the workspace? C2 says deny Edit/Write "unless explicitly allowed" but doesn't specify Bash [Completeness]
- [ ] CHK202 - Does the spec define how the `criteria_path` content is validated beyond file existence? Could a criteria file be empty, binary, or excessively large? Are there format requirements (must be markdown, must have sections)? [Completeness]
- [ ] CHK203 - Does the spec define the prompt structure order for the reviewer? C5 says "criteria + context + schema" but the exact prompt template (section headers, separators, instructions for producing JSON) is not specified [Completeness]
- [ ] CHK204 - Does the spec define behavior when the reviewer agent uses tools (reads files, runs commands) but produces no stdout JSON? The agent may write findings to a file instead of stdout [Completeness]
- [ ] CHK205 - Does the spec define the `extractJSON()` behavior when multiple JSON objects appear in stdout? The llm_judge pattern extracts one — does agent_review expect exactly one ReviewFeedback block? [Completeness]

## Clarity

- [ ] CHK206 - Is it clear whether `git_diff` runs in the workspace directory or the project root? For mount-based workspaces, the workspace directory and git root may differ [Clarity]
- [ ] CHK207 - Is "uncommitted diff" precisely defined? `git diff HEAD` shows diff between HEAD and working tree (including staged). `git diff` shows only unstaged. The spec says "uncommitted" which is ambiguous — does it mean unstaged only or all changes since last commit? [Clarity]
- [ ] CHK208 - Is the confidence score's role in pass/fail determination clearly defined? FR-008 defines the field but acceptance scenarios only check verdict. Is confidence purely informational, or does it gate pass/fail when a threshold is configured? [Clarity]

## Coverage

- [ ] CHK209 - Does the spec cover reviewer agent cleanup? If the adapter spawns a subprocess that hangs, is there a timeout-based kill mechanism specific to review, or does it rely on the adapter's general timeout? [Coverage]
- [ ] CHK210 - Does the spec cover the token accounting for context injection? The token budget applies to the review, but large context (50KB diff + artifacts) consumes tokens before the reviewer reasons. Is context size counted against the budget? [Coverage]
- [ ] CHK211 - Does the spec cover what `git diff HEAD` returns in a workspace without a git repository? Mount-based workspaces may not have `.git` — the memory notes `git init -q` is applied, but criteria path and diff behavior in this edge case is unspecified [Coverage]
- [ ] CHK212 - Does the spec define artifact context source resolution when the artifact name is ambiguous (same artifact name produced by multiple prior steps)? [Coverage]
