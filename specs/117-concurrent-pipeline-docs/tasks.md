# Tasks

## Phase 1: New Pipeline

- [X] Task 1.1: Create `.wave/pipelines/dual-analysis.yaml` with independent parallel tracks pattern
  - Two independent tracks (code-quality track: `quality-scan` → `quality-detail`; security track: `security-scan` → `security-detail`)
  - No shared upstream step — tracks start independently
  - Final `merge` step depends on both `quality-detail` and `security-detail`
  - Use `navigator` persona for analysis steps, `summarizer` for merge
  - Readonly mount workspaces, fresh memory, proper artifact injection
  - Well-commented YAML explaining the pattern

## Phase 2: Documentation Updates

- [X] Task 2.1: Update `docs/concepts/pipelines.md` — add "Independent Parallel Tracks" pattern section [P]
  - Add new section after "Convergence (Fan-In)" with YAML example and Mermaid diagram
  - Show how tracks with no shared upstream converge at a merge step
  - Reference the `dual-analysis.yaml` pipeline as a working example

- [X] Task 2.2: Update `docs/concepts/pipelines.md` — add "Verifying Parallel Execution" section [P]
  - Document `.wave/traces/` audit log format (STEP_START/STEP_END with RFC3339Nano timestamps)
  - Document `wave status` per-step timing display
  - Document `wave logs --format json` for machine-parseable event timestamps
  - Show how overlapping timestamps prove concurrent execution

- [X] Task 2.3: Update `docs/guides/pipeline-configuration.md` — add independent parallel tracks to common patterns [P]
  - Add a new subsection under "Common Patterns" (after "Fan-In Pattern")
  - Brief YAML example showing two independent tracks converging
  - Mermaid diagram visualization

- [X] Task 2.4: Update `README.md` — add concurrent execution example to "Pipelines — DAG Workflows" section
  - Add a second YAML snippet after the existing linear example showing fan-out/fan-in
  - Brief text explaining that steps without mutual dependencies run in parallel

## Phase 3: Validation

- [X] Task 3.1: Run `go test ./...` to confirm no regressions
- [X] Task 3.2: Run `wave validate` to confirm the new pipeline is valid YAML

## Phase 4: Polish

- [X] Task 4.1: Review all changes for consistency (naming, formatting, cross-references)
- [X] Task 4.2: Ensure Mermaid diagrams use consistent styling with existing diagrams
