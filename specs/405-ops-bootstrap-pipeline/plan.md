# Implementation Plan: ops-bootstrap pipeline

## Objective

Create a new `ops-bootstrap` pipeline that scaffolds greenfield projects with language-appropriate project structure, CI config, and initial files after `wave init`.

## Approach

This is a pipeline-definition-only change. No Go code needs modification — the pipeline is a YAML file with prompt-driven steps that use existing personas (navigator, craftsman). A contract schema is needed for the assess step's output artifact.

The pipeline leverages Wave's template variables (`{{ input }}`, `{{ project.build_command }}`, `{{ project.test_command }}`) and existing persona capabilities to:
1. Assess the project's language/intent from `wave.yaml` and existing files
2. Generate appropriate project scaffold via the craftsman persona
3. Commit the result

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `internal/defaults/pipelines/ops-bootstrap.yaml` | create | Pipeline definition with 3 steps |
| `internal/defaults/contracts/bootstrap-assessment.schema.json` | create | JSON Schema for assess step output |
| `.wave/pipelines/ops-bootstrap.yaml` | create | Parity copy for runtime |
| `.wave/contracts/bootstrap-assessment.schema.json` | create | Parity copy for runtime |

## Architecture Decisions

1. **No `project.flavour` field needed**: The issue mentions `project.flavour` but it doesn't exist in the manifest type. The assess step will detect language from `project.language` (which already exists in `manifest.Project`) and from examining project files. No manifest schema change required.

2. **Navigator for assess, craftsman for scaffold+commit**: Navigator is read-only analysis; craftsman has full Bash access for building/testing/committing. This follows existing pipeline patterns (e.g., `impl-feature`).

3. **Workspace type**: Use `worktree` workspace for scaffold and commit steps so changes are isolated and can be pushed. The assess step uses a readonly mount since it only reads.

4. **Build/test verification in scaffold step**: The scaffold step runs `{{ project.build_command }}` and `{{ project.test_command }}` — these template variables are already resolved by Wave's pipeline executor from `wave.yaml`.

5. **Forge-agnostic**: Use `{{ forge.cli_tool }}` and `{{ forge.pr_command }}` templates for any git forge operations, keeping the pipeline portable.

6. **Parity requirement**: Both `internal/defaults/` and `.wave/` directories must contain identical copies of the pipeline and contract files (enforced by parity tests for personas; convention for pipelines/contracts).

## Risks

| Risk | Mitigation |
|------|------------|
| `project.language` not set in wave.yaml | Assess step prompt instructs agent to detect from file extensions, package managers, etc. |
| Build/test commands not configured | Scaffold step generates sensible defaults; wave.yaml may not have them for an empty project |
| Unsupported language | Assess step outputs detected flavour; scaffold prompt handles common languages and can adapt |
| Empty project has no git remote | Commit step checks for remote before pushing; push is conditional |

## Testing Strategy

- **Manual testing**: Run `wave run ops-bootstrap "test project"` on an empty post-init project
- **Parity verification**: Ensure `internal/defaults/` and `.wave/` copies match
- **Schema validation**: Validate the contract schema is valid JSON Schema draft-07
- **Pipeline YAML validation**: Ensure the pipeline loads without errors via `wave list pipelines`
