# Tasks

## Phase 1: Research & Preparation
- [X] Task 1.1: Read ADR template (`docs/adr/000-template.md`) and existing ADRs (001, 002) for style/tone consistency
- [X] Task 1.2: Verify the full dependency graph of all 25 internal packages (confirm no new imports since analysis)

## Phase 2: Write ADR-003
- [X] Task 2.1: Write the **Status** and **Date** sections (Status: Proposed)
- [X] Task 2.2: Write the **Context** section — document the 25 packages, current layer separation audit, relationship to #298, and why formal boundaries are needed now
- [X] Task 2.3: Write the **Decision** section — define the four-layer model (Presentation, Domain/Orchestration, Infrastructure, Cross-cutting) with explicit package-to-layer mapping and dependency rules
- [X] Task 2.4: Write the **Options Considered** section with 4 options: (1) Convention-only documentation, (2) Go build constraints / sub-packages, (3) CI linting with depguard/go-cleanarch, (4) Go Modules per layer — each with pros/cons
- [X] Task 2.5: Write the **Consequences** section — positive (automated enforcement, agent bounded contexts), negative (linter maintenance, classification disputes), neutral (no code restructuring required)
- [X] Task 2.6: Write the **Implementation Notes** section — concrete steps for CI linting setup, depguard configuration, and migration path
- [X] Task 2.7: Add the **Agent/LLM impact** subsection — how clean layers help personas operate with fresh memory and artifact-based communication

## Phase 3: Update Index
- [X] Task 3.1: Add ADR-003 entry to `docs/adr/README.md` index table

## Phase 4: Validation
- [X] Task 4.1: Verify ADR follows template structure from `000-template.md`
- [X] Task 4.2: Verify all 6 acceptance criteria from issue #364 are met
- [X] Task 4.3: Verify ADR-002 and #298 are properly referenced
- [X] Task 4.4: Verify all 25 packages appear in the layer mapping
