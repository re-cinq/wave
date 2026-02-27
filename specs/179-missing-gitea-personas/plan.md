# Implementation Plan: Missing Forge Personas in createDefaultManifest

## Objective

Add the 10 missing persona entries to `createDefaultManifest()` in `cmd/wave/commands/init.go` so that all bundled pipelines have their persona dependencies satisfied after `wave init --all`.

## Approach

This is a single-file code change. Each missing persona entry follows an existing pattern established by the GitHub forge personas (`github-analyst`, `github-enhancer`, `github-commenter`). The fix is mechanical: add 10 new persona map entries to the `personas` map in `createDefaultManifest()`, each following the template of its GitHub counterpart but adapted for the correct forge CLI and description.

### Persona Permission Patterns

| Role | CLI | Permissions (allowed_tools) | Deny |
|------|-----|----------------------------|------|
| `*-analyst` | `gh`/`glab`/`tea`/`bb` | `Read`, `Write`, `Bash(<cli> *)` | `[]` |
| `*-enhancer` | `gh`/`glab`/`tea`/`bb` | `Read`, `Write`, `Bash(<cli> *)` | `[]` |
| `*-commenter` | `gh`/`glab`/`tea`/`bb` | `Read`, `Bash(<cli> *)` | `[]` |
| `supervisor` | N/A | `Read`, `Glob`, `Grep`, `Bash(git *)`, `Bash(go test *)` | `Write(*)`, `Edit(*)` |

### CLI Mapping

- GitHub: `gh`
- GitLab: `glab`
- Gitea: `tea`
- Bitbucket: `bb`

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `cmd/wave/commands/init.go` | modify | Add 10 persona entries to `createDefaultManifest()` |
| `cmd/wave/commands/init_test.go` | modify | Update test expectations if any hardcode persona counts |

## Architecture Decisions

1. **Follow existing patterns exactly**: Each forge persona mirrors its GitHub counterpart's structure (adapter, description, system_prompt_file, temperature, permissions). This keeps the codebase consistent and predictable.

2. **Supervisor permissions**: The supervisor persona is read-only with git and test execution access, matching its role of inspecting work quality without modifying code. Modeled after the auditor pattern but with broader git access and test execution.

3. **Temperature values**: Follow existing patterns — analysts at 0.1 (precise analysis), enhancers at 0.2 (slight creativity for improvements), commenters at 0.2 (natural language posting).

4. **No structural changes**: This is purely additive — no refactoring of the persona map structure, no changes to how personas are loaded or resolved.

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Existing tests hardcode persona count | Low | Low | Check and update test expectations |
| CLI tool name mismatch | Low | High | Verified from pipeline YAML files (tea, glab, bb) |
| `wave validate` fails after change | Low | Medium | Run validate test to confirm |
| Missing persona .md files | None | N/A | Already verified all 10 .md files exist in embedded defaults |

## Testing Strategy

1. **Existing tests**: Run `go test ./cmd/wave/commands/` to verify no regressions
2. **Validate test**: The `TestInitOutputValidatesWithWaveValidate` test already validates that init output passes wave validate
3. **Persona count test**: `TestInitPersonasNeverExcluded` verifies all personas are present — this test should now pass with the correct count
4. **Full suite**: Run `go test ./...` to confirm no project-wide regressions
