# Implementation Plan — Issue #117

## Objective

Add a second example pipeline demonstrating the **independent parallel tracks** concurrency pattern, update the README to showcase concurrent DAG execution, and document how users can verify parallel step execution via audit logs and timing data.

## Approach

This is primarily an **examples and documentation** task — the DAG executor already supports concurrent execution via `errgroup`. The work consists of three deliverables:

1. **New pipeline YAML** — A `dual-analysis` pipeline that runs two fully independent analysis tracks (e.g., code-quality and security) in parallel from the start, each with their own multi-step sequence, converging at a final merge step. This is distinct from fan-out because there is no shared upstream step.

2. **README update** — Replace or augment the linear-only pipeline example in the "Pipelines — DAG Workflows" section with a concurrent example showing the fan-out/fan-in pattern.

3. **Verification documentation** — Add a section to `docs/concepts/pipelines.md` explaining how to confirm parallel execution using:
   - `.wave/traces/` audit logs (STEP_START/STEP_END timestamps)
   - `wave status` per-step timing display
   - `wave logs --format json` for machine-parseable timestamps

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `.wave/pipelines/dual-analysis.yaml` | **create** | New pipeline demonstrating independent parallel tracks |
| `README.md` | **modify** | Update "Pipelines — DAG Workflows" section with concurrent example |
| `docs/concepts/pipelines.md` | **modify** | Add "Independent Parallel Tracks" pattern section and verification documentation |
| `docs/guides/pipeline-configuration.md` | **modify** | Add independent parallel tracks example to "Common Patterns" |

## Architecture Decisions

1. **Pipeline name: `dual-analysis`** — Descriptive, indicates two parallel analysis tracks. Avoids generic names like `concurrent-example` that don't suggest a real use case.

2. **Realistic use case** — The pipeline performs code-quality analysis on one track and security analysis on the other, then merges findings. This mirrors what teams actually do and serves as a useful template.

3. **Personas**: Use `navigator` for analysis steps (readonly) and `summarizer` for the merge step. These are existing personas with appropriate permissions — no new personas needed.

4. **Workspace isolation**: Each track gets readonly mount access to the project. The merge step injects artifacts from both tracks.

5. **Verification docs go in `docs/concepts/pipelines.md`** — This is where the concurrency patterns are already documented. Adding a verification section here keeps related content together.

## Risks

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Pipeline YAML syntax error | Low | Validate with `wave validate` before committing |
| Existing pipeline tests break | Very Low | This adds new files and modifies docs only — run `go test ./...` to confirm |
| README formatting drift | Low | Read existing README structure carefully, maintain consistent style |

## Testing Strategy

1. **`go test ./...`** — Ensure no existing tests break (the changes are additive: new pipeline + doc edits)
2. **`wave validate`** — Validate the new pipeline YAML against the manifest schema
3. **Manual verification** — Run `wave run dual-analysis "internal/pipeline"` to confirm parallel execution and artifact handoff
4. **Doc review** — Verify Mermaid diagram renders correctly in the documentation site context
