# Work Items

## Phase 1: Setup
- [X] 1.1 Create feature branch `1607-pipeline-evolve-impl` (done in this step)
- [X] 1.2 Confirm `pipeline_eval` and `evolution_proposal` migrations are applied (sanity grep `internal/state/migration_definitions.go`)

## Phase 2: Contracts
- [X] 2.1 Write `.agents/contracts/evolution-findings.schema.json` (output of `analyze`) [P]
- [X] 2.2 Write `.agents/contracts/evolution-proposal.schema.json` (output of `propose`) [P]

## Phase 3: Pipeline YAML
- [X] 3.1 Author `internal/defaults/embedfs/pipelines/pipeline-evolve.yaml` with metadata, input schema, four steps (`gather-eval`, `analyze`, `propose`, `record`), `pipeline_outputs`, `chat_context`
- [X] 3.2 Implement `gather-eval` step: `sqlite3` query against `pipeline_eval`, JSON rollup with eval rows + aggregate stats, fallback when DB missing
- [X] 3.3 Implement `analyze` step: `navigator` persona, prompt that classifies recurring failure classes and computes signal summary, `json_schema` contract
- [X] 3.4 Implement `propose` step: `craftsman` persona, prompt that reads findings + active YAML, writes candidate `<pipeline>.next.yaml` and `prompt.diff`, plus `proposal_summary.json`, `json_schema` contract
- [X] 3.5 Implement `record` step: `sqlite3 INSERT INTO evolution_proposal`, write `record_status.json` with proposal id, handle empty/insufficient-data branch
- [X] 3.6 Mirror file to `.agents/pipelines/pipeline-evolve.yaml` (project-local copy)

## Phase 4: Testing
- [X] 4.1 Write `internal/pipeline/pipeline_evolve_test.go` — seed synthetic eval rows, run record + gather-eval scripts directly, assert proposal row + skip branch [P]
- [X] 4.2 Mirror schemas into `internal/defaults/embedfs/contracts/` (TestSchemaSync gate) [P]
- [X] 4.3 Confirm pipeline loader test (`internal/pipeline/all_pipelines_load_test.go`) discovers new YAML

## Phase 5: Polish
- [X] 5.1 Run `go test -race` against the new tests (full suite green)
- [X] 5.2 `go vet` passes; `golangci-lint` deferred to CI (not installed locally)
- [ ] 5.3 Manual smoke test against a real `pipeline_eval` seed using `wave run pipeline-evolve` (deferred to post-merge)
- [ ] 5.4 Update `docs/guides/` if there is an evolution-loop guide; otherwise skip (no existing guide)
- [ ] 5.5 Open PR linking back to #1607 and Epic #1565 (PR step)
