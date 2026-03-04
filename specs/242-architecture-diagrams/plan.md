# Implementation Plan: Architecture Diagrams

## Objective

Create a set of Mermaid diagrams documenting Wave's architecture for non-technical audiences. The diagrams will live in `docs/architecture/` and cover the system overview plus detailed views of pipelines, prompt engineering, context engineering, and security.

## Approach

Create 5 Markdown files in `docs/architecture/`, each containing prose introductions followed by Mermaid diagrams. The diagrams target product managers and stakeholders — they use plain language, avoid Go types and internal implementation details, and focus on concepts and data flow.

The existing `docs/concepts/architecture.md` has basic Mermaid diagrams, but they are developer-oriented. The new diagrams complement rather than replace it, providing a visual reference layer aimed at non-technical readers.

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `docs/architecture/overview.md` | create | High-level system overview showing all major components |
| `docs/architecture/pipeline-lifecycle.md` | create | Pipeline execution flow: DAG resolution, step loop, artifact flow |
| `docs/architecture/prompt-engineering.md` | create | Persona & prompt layers: base protocol → persona → contract → restrictions |
| `docs/architecture/context-engineering.md` | create | Context assembly: CLAUDE.md layers, artifact injection, fresh memory |
| `docs/architecture/security-model.md` | create | Security: workspace isolation, permissions, credential scrubbing, sandboxing |

## Architecture Decisions

1. **Separate files per topic** rather than one monolithic document — easier to link, reference, and maintain independently
2. **`docs/architecture/` directory** — mirrors the issue requirement and sits alongside existing `docs/concepts/`, `docs/guide/`, etc.
3. **Mermaid diagram types**:
   - Overview: `graph TD` (top-down flowchart) — shows component relationships
   - Pipeline lifecycle: `sequenceDiagram` — shows the temporal execution flow per step
   - Prompt engineering: `graph TD` — shows the layered assembly of CLAUDE.md
   - Context engineering: `graph LR` (left-right flowchart) — shows data flow from artifacts through assembly to persona
   - Security model: `graph TD` with subgraphs — shows the nested security boundaries (outer sandbox → adapter sandbox → workspace isolation)
4. **No code references** — diagrams describe concepts ("Workspace Manager creates isolated directory") not implementations (`workspace.Create()`)
5. **Prose before every diagram** — each diagram is preceded by a brief paragraph explaining what it shows and why it matters

## Risks

| Risk | Mitigation |
|------|------------|
| Diagrams become stale as architecture evolves | Keep diagrams concept-level; changes to code internals won't invalidate them |
| Mermaid rendering differences across viewers | Test with GitHub Markdown preview; avoid exotic Mermaid features |
| Too much detail for non-technical audience | Focus on "what" and "why", not "how"; use plain English labels |
| Missing a key architectural concept | Cross-reference against CLAUDE.md's "How Wave Works" section and `docs/concepts/` |

## Testing Strategy

Since this is a documentation-only change, no automated tests are needed. Validation:

1. **Mermaid syntax**: Each diagram renders correctly in a Mermaid live editor or GitHub preview
2. **Accuracy review**: Cross-reference diagram content against source code and existing docs
3. **Audience check**: Diagrams should be understandable without prior knowledge of Wave internals
4. **Go tests**: Run `go test ./...` to confirm no regressions (docs-only change, should pass)
