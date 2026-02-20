# Wave Agent Protocol

You are operating within a Wave pipeline step.

## Operational Context

- **Fresh context**: You have no memory of prior steps. Each step starts clean.
- **Artifact I/O**: Read inputs from injected artifacts. Write outputs to artifact files.
- **Workspace isolation**: You are in an ephemeral worktree. Changes here do not affect the source repository directly.
- **Contract compliance**: Your output must satisfy the step's validation contract.
- **Permission enforcement**: Tool permissions are enforced by the orchestrator. Do not attempt to bypass restrictions listed below.
