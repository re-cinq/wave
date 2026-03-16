# Tasks

## Phase 1: Extend Project Struct and ProjectVars

- [X] Task 1.1: Add `Flavour`, `FormatCommand`, `Skill` fields to `Project` struct in `internal/manifest/types.go`
- [X] Task 1.2: Update `ProjectVars()` to emit `project.flavour`, `project.format_command`, `project.skill`
- [X] Task 1.3: Add unit tests for new `ProjectVars()` fields in `internal/manifest/types_test.go` or appropriate test file

## Phase 2: Template Pipeline Skills

- [X] Task 2.1: Replace `golang` in `skills:` with `{{ project.skill }}` in `impl-feature.yaml` [P]
- [X] Task 2.2: Replace `golang` in `skills:` with `{{ project.skill }}` in `impl-hotfix.yaml` [P]
- [X] Task 2.3: Replace `golang` in `skills:` with `{{ project.skill }}` in `impl-improve.yaml` [P]
- [X] Task 2.4: Replace `golang` in `skills:` with `{{ project.skill }}` in `impl-issue.yaml` [P]
- [X] Task 2.5: Replace `golang` in `skills:` with `{{ project.skill }}` in `impl-refactor.yaml` [P]
- [X] Task 2.6: Replace `golang` in `skills:` with `{{ project.skill }}` in `impl-recinq.yaml` [P]
- [X] Task 2.7: Replace `golang` in `skills:` with `{{ project.skill }}` in `test-gen.yaml` [P]
- [X] Task 2.8: Replace `golang` in `skills:` with `{{ project.skill }}` in `ops-pr-review.yaml` [P]
- [X] Task 2.9: Replace `golang` in `skills:` with `{{ project.skill }}` in `ops-debug.yaml` [P]
- [X] Task 2.10: Replace `golang` in `skills:` with `{{ project.skill }}` in `wave-test-hardening.yaml` [P]
- [X] Task 2.11: Replace `golang` in `skills:` with `{{ project.skill }}` in `wave-bugfix.yaml` [P]
- [X] Task 2.12: Replace `golang` in `skills:` with `{{ project.skill }}` in `wave-land.yaml` [P]
- [X] Task 2.13: Replace `golang` in `skills:` with `{{ project.skill }}` in `wave-audit.yaml` [P]
- [X] Task 2.14: Replace `golang` in `skills:` with `{{ project.skill }}` in `wave-review.yaml` [P]
- [X] Task 2.15: Replace `golang` in `skills:` with `{{ project.skill }}` in `wave-security-audit.yaml` [P]
- [X] Task 2.16: Replace `golang` in `skills:` with `{{ project.skill }}` in `wave-evolve.yaml` [P]

## Phase 3: Template Hardcoded Commands in Prompts

- [X] Task 3.1: Replace `go build ./...` with `{{ project.build_command }}` in `impl-feature.yaml` [P]
- [X] Task 3.2: Replace `go build`/`go test` with template vars in `impl-recinq.yaml` [P]
- [X] Task 3.3: Replace `go test`/`go build` with template vars in `audit-dead-code.yaml` [P]
- [X] Task 3.4: Replace go-specific references in `audit-dead-code-issue.yaml` [P]
- [X] Task 3.5: Replace go-specific references in `audit-dead-code-review.yaml` [P]
- [X] Task 3.6: Replace `go test` with `{{ project.test_command }}` in `ops-supervise.yaml` [P]
- [X] Task 3.7: Replace `go test` commands with template vars in `wave-test-hardening.yaml` [P]
- [X] Task 3.8: Replace `go test` commands with template vars in `wave-bugfix.yaml` [P]

## Phase 4: Testing and Validation

- [X] Task 4.1: Add test for `newContextWithProject()` with all project fields in `internal/pipeline/context_test.go`
- [X] Task 4.2: Add test for `{{ project.skill }}` resolution via `ResolvePlaceholders()`
- [X] Task 4.3: Run `go test ./...` to ensure no regressions
- [X] Task 4.4: Grep validation — confirm no hardcoded `golang` skill or `go test`/`go build`/`go vet` remains in pipeline YAMLs
