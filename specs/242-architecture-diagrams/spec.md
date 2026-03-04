# docs: create bird's-eye view Mermaid diagrams of Wave architecture for non-technical audiences

**Issue**: [#242](https://github.com/re-cinq/wave/issues/242)
**Labels**: documentation, good first issue
**Author**: nextlevelshit
**State**: OPEN

## Summary

Wave lacks visual architecture diagrams that explain the system to non-technical stakeholders. We need Mermaid diagrams that cover the key concepts at a high level and in detail.

## Background

We are lacking a proper Mermaid diagram to explain the different levels of prompt engineering, context engineering, personas, pipelines, security in the pipeline steps etc.

In best case we have one overview and separate different detailed ones for the different concepts.

## Deliverables

1. **Overview diagram** — A single high-level diagram showing how the major Wave components relate to each other (personas, pipelines, contracts, workspaces, adapters, security boundaries)
2. **Detailed concept diagrams** (one per topic):
   - Pipeline execution lifecycle (steps, dependencies, artifact flow)
   - Persona & prompt engineering layers (base protocol → persona prompt → contract compliance → restrictions)
   - Context engineering (artifact injection, CLAUDE.md assembly, fresh memory per step)
   - Security model (workspace isolation, permission enforcement, credential scrubbing, sandbox layers)

## Requirements

- Use [Mermaid](https://mermaid.js.org/) syntax for all diagrams
- Diagrams should be understandable by non-technical team members (product managers, stakeholders)
- Avoid implementation details — focus on concepts and data flow
- Place diagrams in `docs/architecture/` (or appropriate docs location)
- Each diagram should include a brief prose introduction explaining what it shows

## Acceptance Criteria

- [ ] One high-level overview diagram exists showing all major Wave components
- [ ] At least 3 detailed concept diagrams covering pipelines, personas/prompts, and security
- [ ] All diagrams render correctly in GitHub Markdown
- [ ] Diagrams are reviewed for accuracy by a team member familiar with Wave internals
- [ ] Non-technical stakeholder can understand the overview diagram without additional explanation
