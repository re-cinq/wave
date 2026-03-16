# Tasks

## Phase 1: Contract Schema
- [X] Task 1.1: Create `internal/defaults/contracts/bootstrap-assessment.schema.json` — JSON Schema for assess step output (flavour, intent, existing_files, recommendations)
- [X] Task 1.2: Copy contract to `.wave/contracts/bootstrap-assessment.schema.json` for parity

## Phase 2: Pipeline Definition
- [X] Task 2.1: Create `internal/defaults/pipelines/ops-bootstrap.yaml` — 3-step pipeline (assess → scaffold → commit) with proper persona assignments, workspace config, artifact wiring, and contract validation
- [X] Task 2.2: Copy pipeline to `.wave/pipelines/ops-bootstrap.yaml` for parity

## Phase 3: Validation
- [X] Task 3.1: Verify JSON Schema is valid draft-07
- [X] Task 3.2: Verify pipeline YAML parses correctly (well-formed YAML, valid step references, dependency graph)
- [X] Task 3.3: Run `go test ./internal/defaults/...` to check embed/parity tests pass
- [X] Task 3.4: Run `go build ./...` to ensure no compilation issues
