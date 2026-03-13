# feat(chat): inject pipeline context and artifacts into wave chat sessions

**Issue**: [#276](https://github.com/re-cinq/wave/issues/276)
**Labels**: enhancement, ux
**Author**: nextlevelshit

## Problem

Wave chat sessions currently start with no context about the pipeline that was just run. The chat agent has to manually discover and read artifacts, which is slow and defeats the purpose of an interactive post-pipeline chat.

## Current Behavior

- Chat starts with a blank context — no awareness of pipeline results
- User must wait while the agent explores `.wave/artifacts/` and workspace files
- No pipeline-specific guidance or suggested questions

## Desired Behavior

- **Pre-loaded context**: Chat sessions automatically receive a summary of the pipeline run — which steps completed, what artifacts were produced, key findings (scores, flaws, diffs)
- **Pipeline-specific prompts**: Different pipelines seed the chat with different opening questions and context (e.g., a rewrite pipeline shows a diff abstract; an analysis pipeline shows scores)
- **Lightning-fast first response**: The chat should be ready to answer substantive questions immediately, without needing to read files first

## Acceptance Criteria

- [ ] Chat session receives injected context summarizing the most recent pipeline run (step names, statuses, artifact paths)
- [ ] Key artifact content (or summaries) is pre-loaded into the chat's system prompt or initial context
- [ ] Pipeline manifests can define a `chat_context` section specifying what to inject (artifact summaries, suggested questions, focus areas)
- [ ] Chat opens with pipeline-specific suggested questions (e.g., "Would you like to review the changes?" for a rewrite pipeline)
- [ ] Response latency for the first question is comparable to a pre-loaded chat (no file discovery delay)
- [ ] Ask the user right from the beginning three highly specific questions regarding the pipeline run and estimate what could be the best fitting post mortem questions or tasks

## Technical Notes

- Consider adding a `chat_context` or `post_run` section to the pipeline manifest schema
- Artifact summaries could be generated as a final pipeline step or computed on-the-fly from artifact JSON
- The relay/compaction system may be relevant for keeping context within token limits

## Research Findings (from issue comments)

Research findings with 10 prioritized recommendations covering context engineering, token budget management, Claude Code injection mechanisms, manifest extension, suggested questions UX, artifact summarization, and latency optimization. Recommends two-tier context injection: compact pipeline summary via `--append-system-prompt` for instant context, and artifact file path references for on-demand deep exploration, configured through a new `chat_context` section in the pipeline manifest.
