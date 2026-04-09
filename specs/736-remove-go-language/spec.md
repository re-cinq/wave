# Remove Go-specific language from pipelines, personas, and contracts

**Issue**: [#736](https://github.com/re-cinq/wave/issues/736)
**Author**: nextlevelshit
**State**: OPEN
**Labels**: none

## Problem

Pipeline prompts, persona definitions, and contract schemas still contain Go-specific references (e.g., `go test`, `go build`, `golangci-lint`, Go package conventions, Go error handling patterns). These should be language-agnostic since Wave supports 25+ languages via flavour auto-detection.

## Scope

- **Pipeline prompts**: Replace Go-specific commands/patterns with `{{ project.* }}` template variables or generic language
- **Persona definitions**: Remove references to Go tooling, idioms, or conventions
- **Contract schemas**: Ensure validation criteria don't assume Go project structure

## Context

The prompt enrichment work in PR #734 expanded all prompts to 400+ words but preserved existing Go-specific language. The onboarding system (#403) already introduced `{{ project.test_command }}`, `{{ project.build_command }}`, and `{{ project.lint_command }}` template variables — prompts should use these instead of hardcoded Go commands.

## Acceptance Criteria

- [ ] No hardcoded `go test`, `go build`, `go vet`, `golangci-lint` in pipeline prompts
- [ ] No Go-specific patterns in persona files (e.g., "idiomatic Go", "goroutines")
- [ ] Contract schemas validate structure, not Go-specific content
- [ ] `wave validate --all` passes
- [ ] `go test ./internal/defaults/...` passes
