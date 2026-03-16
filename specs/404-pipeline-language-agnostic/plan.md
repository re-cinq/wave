# Implementation Plan: Language-Agnostic Pipelines

## Objective

Replace all hardcoded Go-specific commands and skill references in embedded pipeline YAML files with `{{ project.* }}` template variables, enabling Wave pipelines to work with any programming language.

## Approach

The template variable infrastructure already exists via `newContextWithProject()` → `ProjectVars()`. The work is:

1. **Extend the `Project` struct** with three new fields: `Flavour`, `FormatCommand`, `Skill`
2. **Update `ProjectVars()`** to emit the new fields as `project.flavour`, `project.format_command`, `project.skill`
3. **Replace hardcoded values** in ~20 pipeline YAML files with template variable references
4. **Handle the `skills:` field** — replace `golang` with `{{ project.skill }}` where it's a language-specific skill

## File Mapping

### Files to Modify

| File | Action | What Changes |
|------|--------|-------------|
| `internal/manifest/types.go` | modify | Add `Flavour`, `FormatCommand`, `Skill` fields to `Project` struct; update `ProjectVars()` |
| `internal/pipeline/context_test.go` | modify | Add test for `newContextWithProject` with new project fields |
| `internal/defaults/pipelines/impl-feature.yaml` | modify | Replace `golang` skill with `{{ project.skill }}`, replace `go build`/`go test` in prompts |
| `internal/defaults/pipelines/impl-hotfix.yaml` | modify | Replace `golang` skill |
| `internal/defaults/pipelines/impl-improve.yaml` | modify | Replace `golang` skill |
| `internal/defaults/pipelines/impl-issue.yaml` | modify | Replace `golang` skill |
| `internal/defaults/pipelines/impl-refactor.yaml` | modify | Replace `golang` skill |
| `internal/defaults/pipelines/impl-recinq.yaml` | modify | Replace `golang` skill, `go build`/`go test` commands |
| `internal/defaults/pipelines/audit-dead-code.yaml` | modify | Replace `go build`/`go test` commands |
| `internal/defaults/pipelines/audit-dead-code-issue.yaml` | modify | Replace go-specific references |
| `internal/defaults/pipelines/audit-dead-code-review.yaml` | modify | Replace go-specific references |
| `internal/defaults/pipelines/test-gen.yaml` | modify | Replace `golang` skill |
| `internal/defaults/pipelines/ops-pr-review.yaml` | modify | Replace `golang` skill |
| `internal/defaults/pipelines/ops-debug.yaml` | modify | Replace `golang` skill |
| `internal/defaults/pipelines/ops-supervise.yaml` | modify | Replace `go test` commands |
| `internal/defaults/pipelines/wave-test-hardening.yaml` | modify | Replace `golang` skill, `go test` commands |
| `internal/defaults/pipelines/wave-bugfix.yaml` | modify | Replace `golang` skill, `go test` commands |
| `internal/defaults/pipelines/wave-land.yaml` | modify | Replace `golang` skill |
| `internal/defaults/pipelines/wave-audit.yaml` | modify | Replace `golang` skill |
| `internal/defaults/pipelines/wave-review.yaml` | modify | Replace `golang` skill |
| `internal/defaults/pipelines/wave-security-audit.yaml` | modify | Replace `golang` skill |
| `internal/defaults/pipelines/wave-evolve.yaml` | modify | Replace `golang` skill |

## Architecture Decisions

1. **Skills as template variables**: The `skills:` field in pipeline YAML is a list of strings. We'll replace `golang` with `{{ project.skill }}` — the template engine resolves it before skill provisioning. If `project.skill` is empty (no manifest), the placeholder stays unresolved, which the skill loader should handle gracefully (skip unknown skills).

2. **No `InjectProjectVariables()` function**: The issue suggests adding this, but `newContextWithProject()` already exists and does exactly this. No new function needed — just extend the `Project` struct and `ProjectVars()`.

3. **`wave-*` pipelines**: These are Wave self-referencing pipelines. They should ALSO use `{{ project.* }}` variables — Wave's own `wave.yaml` will have `project.language: go`, `project.skill: golang`, etc. This keeps them consistent and proves the template system works end-to-end.

4. **Backward compatibility**: If a project has no `project:` section in `wave.yaml`, template variables remain unresolved (e.g., `{{ project.test_command }}` stays as literal text in the prompt). The doctor/optimize system already detects missing project config and suggests adding it. No runtime breakage.

5. **`requires.tools` field**: Some pipelines have `requires: tools: [go]`. These should NOT be templated — they're runtime dependency checks, not prompt text. Leave them as-is since the actual binary name depends on the toolchain.

## Risks

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Unresolved template variables in prompts | AI sees raw `{{ project.* }}` text | Doctor system already warns about missing project config |
| Skill resolution failure for empty `{{ project.skill }}` | Pipeline fails to provision skills | Skill loader should skip empty/unresolved skill names |
| Breaking existing `wave.yaml` files | Users need to add `project:` section | Doctor optimize already generates these recommendations |

## Testing Strategy

1. **Unit tests**: Add test cases for `ProjectVars()` with new fields (`flavour`, `format_command`, `skill`)
2. **Integration test**: Test `newContextWithProject()` injects all project vars including new ones
3. **Template resolution test**: Verify `{{ project.skill }}` resolves correctly via `ResolvePlaceholders()`
4. **Grep validation**: After changes, verify no pipeline YAML contains hardcoded `go test`, `go vet`, `go build`, or `golang` skill reference
