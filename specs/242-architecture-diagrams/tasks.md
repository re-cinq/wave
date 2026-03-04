# Tasks

## Phase 1: Setup

- [X] Task 1.1: Create `docs/architecture/` directory
- [X] Task 1.2: Review existing diagrams in `docs/concepts/architecture.md` to avoid duplication

## Phase 2: Core Diagrams

- [X] Task 2.1: Create `docs/architecture/overview.md` — high-level system overview diagram showing Manifest, Pipeline Executor, Personas, Workspaces, Adapters, Contracts, State Store, Event System, and their relationships [P]
- [X] Task 2.2: Create `docs/architecture/pipeline-lifecycle.md` — sequence diagram of pipeline execution: DAG resolution → workspace creation → artifact injection → persona binding → adapter execution → contract validation → state persistence → event emission [P]
- [X] Task 2.3: Create `docs/architecture/prompt-engineering.md` — layered diagram showing CLAUDE.md assembly: base protocol preamble → persona system prompt → contract compliance section → restriction section (denied/allowed tools, network domains) [P]
- [X] Task 2.4: Create `docs/architecture/context-engineering.md` — data flow diagram showing artifact injection from prior steps, CLAUDE.md assembly, workspace isolation (fresh memory per step), and how each persona sees only its injected context [P]
- [X] Task 2.5: Create `docs/architecture/security-model.md` — nested boundary diagram showing: outer Nix sandbox (read-only FS, hidden $HOME) → adapter sandbox (settings.json, network allowlisting) → prompt restrictions (CLAUDE.md deny/allow) → workspace isolation (ephemeral dirs, mount modes) → credential handling (env-only, scrubbed logs) [P]

## Phase 3: Validation

- [X] Task 3.1: Verify all Mermaid diagrams render correctly (syntax validation)
- [X] Task 3.2: Cross-reference diagram accuracy against source code and CLAUDE.md documentation
- [X] Task 3.3: Run `go test ./...` to confirm no regressions from docs-only changes

## Phase 4: Polish

- [X] Task 4.1: Add prose introductions before each diagram explaining what it shows
- [X] Task 4.2: Ensure consistent terminology across all diagrams (use the same labels for the same concepts)
- [X] Task 4.3: Final review — confirm diagrams are understandable by non-technical audience
