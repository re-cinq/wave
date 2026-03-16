# feat(pipeline): make all pipelines language-agnostic via template variables

**Issue**: [#404](https://github.com/re-cinq/wave/issues/404)
**Parent**: [#402](https://github.com/re-cinq/wave/issues/402)
**Labels**: enhancement, pipeline
**Author**: nextlevelshit

## Summary

Replace hardcoded Go-specific commands in pipeline YAML files with `{{ project.* }}` template variables, making all pipelines work across any language.

## Problem

10+ embedded pipelines hardcode Go-specific commands:
- `go test ./...` should be `{{ project.test_command }}`
- `go vet ./...` should be `{{ project.lint_command }}`
- `go build ./...` should be `{{ project.build_command }}`
- `golang` skill reference should be `{{ project.skill }}` or removed
- `*.go` glob patterns should be `{{ project.source_glob }}`

## Affected Pipelines

These pipelines in `internal/defaults/pipelines/` need templating:
- `impl-feature.yaml` — hardcodes golang skill
- `impl-hotfix.yaml` — hardcodes golang skill
- `impl-improve.yaml` — hardcodes golang skill
- `impl-issue.yaml` — hardcodes golang skill
- `impl-refactor.yaml` — hardcodes golang skill
- `impl-recinq.yaml` — hardcodes golang skill + `go build`/`go test` in prompts
- `audit-dead-code.yaml` — hardcodes go-specific tool references
- `audit-dead-code-issue.yaml` — same
- `audit-dead-code-review.yaml` — same
- `test-gen.yaml` — hardcodes `go test ./...`
- `ops-pr-review.yaml` — hardcodes golang skill
- `ops-supervise.yaml` — hardcodes `go test ./...` in prompts
- `wave-test-hardening.yaml` — hardcodes golang skill + `go test` commands
- `wave-bugfix.yaml` — hardcodes golang skill + `go test` commands
- `wave-land.yaml` — hardcodes golang skill
- `wave-audit.yaml` — hardcodes golang skill
- `wave-review.yaml` — hardcodes golang skill
- `wave-security-audit.yaml` — hardcodes golang skill
- `wave-evolve.yaml` — hardcodes golang skill
- `ops-debug.yaml` — hardcodes golang skill

## Template Variables

From `wave.yaml` manifest `project:` section:
- `{{ project.language }}` — e.g., "rust", "go", "typescript"
- `{{ project.flavour }}` — e.g., "rust", "bun", "java-maven"
- `{{ project.test_command }}` — e.g., "cargo test"
- `{{ project.lint_command }}` — e.g., "cargo clippy -- -D warnings"
- `{{ project.build_command }}` — e.g., "cargo build"
- `{{ project.format_command }}` — e.g., "cargo fmt -- --check"
- `{{ project.source_glob }}` — e.g., "*.rs"
- `{{ project.skill }}` — e.g., "golang", "rust"

## Acceptance Criteria

- [ ] No pipeline YAML contains hardcoded `go test`, `go vet`, `go build`
- [ ] All language-specific commands use `{{ project.* }}` templates
- [ ] `project.*` variables populated from manifest project section
- [ ] New fields (`flavour`, `format_command`, `skill`) added to `Project` struct
- [ ] `ProjectVars()` updated to include new fields
- [ ] Existing Go projects still work (backwards compatible via manifest defaults)
- [ ] Tests cover template variable injection for new fields
- [ ] `wave-*` pipelines (self-referencing Wave) also use template variables

## Context

`internal/pipeline/context.go` already has `newContextWithProject()` which calls `ProjectVars()` to inject project variables. The infrastructure for `{{ project.* }}` template resolution already exists — this issue just needs to:
1. Add missing fields to the `Project` struct
2. Update `ProjectVars()` to emit them
3. Replace hardcoded values in pipeline YAMLs with template references
