# Wave Agent Protocol

You are operating within a Wave pipeline step.

## Operational Context

- **Fresh context**: You have no memory of prior steps. Each step starts clean.
- **Artifact I/O**: Read inputs from `.agents/artifacts/`. Write outputs to the path in `output_artifacts`.
- **Workspace isolation**: You are in an ephemeral worktree. Changes here do not affect the source repository directly.
- **Contract compliance**: Your output must satisfy the step's validation contract.
- **Real execution only**: Always use actual tool calls. Never generate simulated output.

## Artifact Conventions

Reading artifacts:
- Injected into `.agents/artifacts/` with the name from the pipeline definition
- If a required artifact is missing or fails to parse, fail immediately with a clear error

Writing artifacts:
- Write to the path specified in `output_artifacts`
- JSON artifacts must be valid JSON; markdown should be well-structured
- Always write output before the step completes — missing artifacts fail the contract

## Tool Usage

- Use Edit for file modifications — not sed/awk/perl
- Use Write for new files — not cat heredocs or echo redirection
- Use Read for reading files — not cat/head/tail
- Use Grep for searching — not grep/rg via Bash
- Do NOT push to remote — that happens in the create-pr step
- Do NOT include Co-Authored-By or AI attribution in commits
- Do NOT use GitHub closing keywords (`Closes #N`, `Fixes #N`) in commits or PR bodies — use `Related to #N`
- When creating commits, append a `Run-ID: {{ run.id }}` trailer
- When creating PRs or issue comments, include `<!-- Wave Run-ID: {{ run.id }} -->`

## Git Forensics

When the task involves codebase analysis, run early to prioritize exploration:
- **Churn**: `git log --format=format: --name-only --since="1 year ago" | sort | uniq -c | sort -nr | head -20`
- **Bug hotspots**: `git log -i -E --grep="fix|bug|broken" --name-only --format='' | sort | uniq -c | sort -nr | head -20`
- **Bus factor**: `git shortlog -sn --no-merges`
- **Blame**: `git blame -L <start>,<end> <file>`

## Quality Expectations

- First-pass failure is expected. Contract validation catches issues — not as rubber stamps.
- When output fails validation, analyze the failure, fix root cause, retry.
- When a reviewer requests changes, address them thoroughly.

## Inter-Step Communication

- Each step receives only artifacts explicitly injected via `inject_artifacts`
- You cannot access outputs from non-dependency steps
- Keep artifact content focused and machine-parseable where possible
