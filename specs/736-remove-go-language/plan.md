# Implementation Plan: Remove Go-specific language

## Objective

Replace all hardcoded Go-specific commands, patterns, and references in pipeline YAML files, persona Markdown files, contract schemas, and speckit-flow prompts with language-agnostic equivalents using existing `{{ project.* }}` template variables.

## Approach

1. **Pipeline prompts** — Replace hardcoded `go test`, `go build`, `go vet` with `{{ project.test_command }}`, `{{ project.build_command }}`, `{{ project.lint_command }}`. Replace Go-specific pattern descriptions (goroutines, blank imports, init functions, go.mod) with generic language-agnostic equivalents.
2. **Pipeline metadata** — Remove `requires: tools: [go]` from audit pipelines (these are static analysis using Grep/Read, not Go tooling).
3. **Persona files** — The existing test (`TestPersonaFilesNoLanguageReferences`) already enforces no language references. Verify personas pass this test (they should already).
4. **Contract schemas** — Update `shared-findings.schema.json` to remove Go package path assumption.
5. **Speckit-flow prompts** — Replace `go test -race ./...` with `{{ project.test_command }}`.

## File Mapping

### Files to Modify

**Pipeline YAMLs (hardcoded Go commands):**
- `internal/defaults/pipelines/test-gen.yaml` — 4 instances of `go test`
- `internal/defaults/pipelines/audit-dead-code.yaml` — `requires: tools: [go]`, `go build`, `go test`, Go compiler refs
- `internal/defaults/pipelines/audit-dead-code-issue.yaml` — `requires: tools: [go]`, go.mod, blank import refs
- `internal/defaults/pipelines/audit-dead-code-review.yaml` — `requires: tools: [go]`, Go patterns ref
- `internal/defaults/pipelines/audit-dual.yaml` — Go conventions, Go idioms refs
- `internal/defaults/pipelines/audit-junk-code.yaml` — Go patterns ref
- `internal/defaults/pipelines/audit-consolidate.yaml` — "Go projects" example
- `internal/defaults/pipelines/ops-debug.yaml` — goroutine refs, `go test` ref
- `internal/defaults/pipelines/ops-bootstrap.yaml` — go.mod, `go build`, `go test` refs
- `internal/defaults/pipelines/ops-pr-review.yaml` — "Go package" ref
- `internal/defaults/pipelines/ops-pr-fix-review.yaml` — `go vet` ref
- `internal/defaults/pipelines/impl-improve.yaml` — goroutine refs
- `internal/defaults/pipelines/impl-hotfix.yaml` — goroutine ref
- `internal/defaults/pipelines/doc-explain.yaml` — `*_test.go`, goroutine refs
- `internal/defaults/pipelines/doc-onboard.yaml` — go.mod refs
- `internal/defaults/pipelines/ops-supervise.yaml` — `_test.go` ref
- `internal/defaults/pipelines/plan-approve-implement.yaml` — "go build this" (figurative, may keep)
- `internal/defaults/pipelines/plan-research.yaml` — Go testing example

**Speckit-flow prompts:**
- `internal/defaults/prompts/speckit-flow/create-pr.md` — `go test -race`
- `internal/defaults/prompts/speckit-flow/implement.md` — `go test -race`

**Persona files (base-protocol.md):**
- `internal/defaults/personas/base-protocol.md` — Go examples in template variable table

**Contract schemas:**
- `internal/defaults/contracts/shared-findings.schema.json` — "Go package path" description

### Files to NOT Modify
- `internal/defaults/personas_test.go` — Already enforces no language refs (keep as-is)
- `internal/defaults/embed.go` — Infrastructure, no Go-specific content
- `wave.yaml` — Project-specific config (correctly has `go test`)

## Architecture Decisions

1. **Use `{{ project.* }}` variables** for all command references — these are already resolved by the template engine at runtime from the project's `wave.yaml` config.
2. **Generic language patterns** — Replace "goroutines" with "concurrent operations", "blank imports" with "unused imports", "go.mod" with "dependency manifest" etc.
3. **Multi-language examples** — Where examples are needed, list 2-3 languages (e.g., "go.mod, package.json, Cargo.toml") rather than just Go.
4. **Keep `ops-bootstrap.yaml` examples** — This pipeline scaffolds new projects and needs language-specific examples, but should use conditional/multi-language framing.
5. **`plan-approve-implement.yaml`** — The phrase "yes, go build this" is figurative English, not a Go command reference. Keep as-is.

## Risks

1. **Prompt quality regression** — Replacing specific Go patterns with generic language may reduce prompt effectiveness for Go projects. Mitigate by keeping patterns concrete but expressed generically.
2. **Template variable resolution** — If `{{ project.test_command }}` is empty/unset, prompts become nonsensical. The template engine already handles this with fallback defaults from flavour detection.
3. **Test breakage** — The `personas_test.go` already enforces no language references. Pipeline changes need `wave validate --all` to verify schema compliance.

## Testing Strategy

1. Run `go test ./internal/defaults/...` to verify persona tests still pass
2. Run `wave validate --all` to verify all pipeline YAML schemas are valid
3. Grep audit: verify no remaining `go test`, `go build`, `go vet`, `golangci-lint` in `internal/defaults/` (excluding test files and embed.go)
4. Verify `{{ project.* }}` variables are used consistently where commands were hardcoded
