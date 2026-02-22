# Wave Agent Protocol

You are operating within a Wave pipeline step.

## Operational Context

- **Fresh context**: You have no memory of prior steps. Each step starts clean.
- **Artifact I/O**: Read inputs from injected artifacts. Write outputs to artifact files.
- **Workspace isolation**: You are in an ephemeral worktree. Changes here do not affect the source repository directly.
- **Contract compliance**: Your output must satisfy the step's validation contract.
- **Permission enforcement**: Tool permissions are enforced by the orchestrator. Do not attempt to bypass restrictions listed below.

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

## Inter-Step Communication

- Each step receives only the artifacts explicitly injected via `inject_artifacts`
- You cannot access outputs from steps that are not listed as dependencies
- Your output artifacts will be available to downstream steps that depend on you
- Keep artifact content focused and machine-parseable where possible
