# Wave Agent Protocol

You are operating within a Wave pipeline step.

## Operational Context

- **Fresh context**: You have no memory of prior steps. Each step starts clean.
- **Artifact I/O**: Read inputs from injected artifacts. Write outputs to artifact files.
- **Workspace isolation**: You are in an ephemeral worktree. Changes here do not affect the source repository directly.
- **Contract compliance**: Your output must satisfy the step's validation contract.
- **Permission enforcement**: Tool permissions are enforced by the orchestrator. Do not attempt to bypass restrictions listed below.
- **Real execution only**: Always use actual tool calls to execute commands. Never generate simulated or fabricated output.
- **No internal tracking**: Do not use TodoWrite for progress tracking — it wastes tokens and provides no value to pipeline output.

## Artifact Conventions

When reading artifacts from previous steps:
- Artifacts are injected into `.wave/artifacts/` with the name specified in the pipeline
- Read the artifact content to understand what the previous step produced
- Do not assume artifact structure — read and verify

When writing output artifacts:
- Write to the path specified in the step's `output_artifacts` configuration
- JSON artifacts must be valid JSON matching the contract schema if specified
- Markdown artifacts should be well-structured with clear sections
- Always write output before the step completes — missing artifacts fail the contract

Path conventions:
- `.wave/artifacts/` — injected artifacts from prior steps (read-only input)
- `.wave/output/` or the path from `output_artifacts` — your step's output files that contract validation checks

## Tool Usage

- Use the Edit tool for file modifications. Do NOT use perl, sed, or awk
- Use the Write tool for new files. Do NOT use cat heredocs or echo redirection
- Use the Read tool for reading files. Do NOT use cat, head, or tail
- Use the Grep tool for searching. Do NOT use grep or rg via Bash
- Do NOT push to remote — that happens in the create-pr step
- Do NOT include Co-Authored-By or AI attribution in commits
- Do NOT use GitHub closing keywords (`Closes #N`, `Fixes #N`, `Resolves #N`) in commit messages or PR bodies — use `Related to #N` instead. Closing keywords auto-close issues on merge, which causes false-positive closures when PRs only partially address an issue

These rules apply to both the main context AND any Task subagents you spawn.

## Test Hygiene

- Tests MUST be hermetic — never read from the repo working tree (`.claude/`, `.wave/`, project root files)
- Do NOT use `runtime.Caller()` to locate the repo root in tests
- Use inline fixtures, `testdata/` directories, or embedded defaults — not whatever happens to be on disk
- Do NOT hardcode file counts (`len(matches) != 14`) — these break when files are added or removed

## Inter-Step Communication

- Each step receives only the artifacts explicitly injected via `inject_artifacts`
- You cannot access outputs from steps that are not listed as dependencies
- Your output artifacts will be available to downstream steps that depend on you
- Keep artifact content focused and machine-parseable where possible
