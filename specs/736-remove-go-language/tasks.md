# Tasks

## Phase 1: Pipeline Prompts — Hardcoded Commands
- [X] Task 1.1: Replace `go test` commands in `test-gen.yaml` with `{{ project.test_command }}` and `{{ project.contract_test_command }}`
- [X] Task 1.2: Replace `go build`/`go test` in `audit-dead-code.yaml`, remove `requires: tools: [go]`, replace Go compiler refs with generic language [P]
- [X] Task 1.3: Remove `requires: tools: [go]` from `audit-dead-code-issue.yaml` and `audit-dead-code-review.yaml`, replace Go-specific patterns [P]
- [X] Task 1.4: Replace `go test` ref in `ops-debug.yaml` with `{{ project.test_command }}` [P]
- [X] Task 1.5: Replace `go build`/`go test` examples in `ops-bootstrap.yaml` with multi-language framing [P]
- [X] Task 1.6: Replace `go vet` in `ops-pr-fix-review.yaml` with `{{ project.lint_command }}` [P]
- [X] Task 1.7: Replace `go test -race` in speckit-flow prompts (`create-pr.md`, `implement.md`) with `{{ project.test_command }}` [P]

## Phase 2: Pipeline Prompts — Go-Specific Patterns
- [X] Task 2.1: Replace goroutine/concurrency refs in `impl-improve.yaml`, `impl-hotfix.yaml`, `ops-debug.yaml` with generic concurrency language [P]
- [X] Task 2.2: Replace Go convention/idiom refs in `audit-dual.yaml` with generic naming convention language [P]
- [X] Task 2.3: Replace Go pattern refs in `audit-junk-code.yaml` and `audit-dead-code-review.yaml` with generic language [P]
- [X] Task 2.4: Replace `*_test.go` refs in `doc-explain.yaml`, `ops-supervise.yaml`, `test-gen.yaml` with generic test file pattern [P]
- [X] Task 2.5: Replace `go.mod` refs in `doc-onboard.yaml`, `audit-dead-code.yaml`, `audit-dead-code-issue.yaml` with multi-language dependency manifest examples [P]
- [X] Task 2.6: Replace "Go package" ref in `ops-pr-review.yaml` and "Go projects" in `audit-consolidate.yaml` with generic language [P]
- [X] Task 2.7: Replace Go testing example in `plan-research.yaml` with generic example [P]

## Phase 3: Schemas and Personas
- [X] Task 3.1: Update `shared-findings.schema.json` — change "Go package path" to language-agnostic description
- [X] Task 3.2: Update `base-protocol.md` — replace Go-specific examples in template variable table with multi-language examples [P]

## Phase 4: Validation
- [X] Task 4.1: Run `go test ./internal/defaults/...` to verify persona tests pass
- [X] Task 4.2: Grep audit for remaining Go-specific references in `internal/defaults/`
- [X] Task 4.3: Run `wave validate --all` if available, otherwise manual YAML schema check
