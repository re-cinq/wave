# feat: skill dependency management, git worktree workspaces, and preflight validation

**Issue**: [re-cinq/wave#76](https://github.com/re-cinq/wave/issues/76)
**Author**: nextlevelshit
**Labels**: ready-for-impl
**State**: OPEN

## Problem

Pipeline steps depend on external skills (speckit, BMAD, OpenSpec) and tools (`gh`, `git`, `node`) but Wave has no mechanism to:

1. **Install skills** — The `.claude/commands/` and `.specify/` files in the repo were manually created. There's no declarative, reproducible installation path.
2. **Provision workspaces** — Workspaces are empty directories (workspace `root: ./` with no `mount:` creates just `mkdir`). Skills, commands, scripts, and templates aren't available inside them. Claude Code discovers slash commands only by accident (walking up the directory tree to the project root).
3. **Validate dependencies** — If a pipeline requires `speckit` or `gh`, Wave doesn't check until the step fails mid-execution — after burning tokens and time.
4. **Isolate git operations** — PR #64 fixed a symptom where concurrent pipeline runs collided because `create-new-feature.sh` ran `git checkout -b` on the main repo. The root cause is that workspaces aren't real git checkouts, so all git operations reach back to the shared project root.

The `skill_mounts` config exists in the manifest schema but is dead code — parsed, validated, never wired into the executor or adapter.

## External Skill Sources

These are external open-source projects, not Wave-owned code:

| Skill | Source | Install | Init |
|-------|--------|---------|------|
| **Spec-Kit** | [github/spec-kit](https://github.com/github/spec-kit) | `uv tool install specify-cli --from git+https://github.com/github/spec-kit.git` | `specify init <project>` |
| **BMAD** | [bmad-code-org/BMAD-METHOD](https://github.com/bmad-code-org/BMAD-METHOD) | `npx bmad-method install --tools claude-code --yes` | included in install |
| **OpenSpec** | [Fission-AI/OpenSpec](https://github.com/Fission-AI/OpenSpec) | `npm install -g @fission-ai/openspec@latest` | `openspec init` |

## Proposed Design

### 1. Skill declarations in wave.yaml (optional)

```yaml
skills:
  speckit:
    install: "uv tool install specify-cli --from git+https://github.com/github/spec-kit.git"
    init: "specify init"
    check: "specify --version"
  bmad:
    install: "npx bmad-method install --tools claude-code --yes"
    check: "ls .claude/commands/bmad.*.md"
  openspec:
    install: "npm install -g @fission-ai/openspec@latest"
    init: "openspec init"
    check: "openspec --version"
```

### 2. Pipeline `requires:` block

```yaml
kind: WavePipeline
metadata:
  name: speckit-flow
requires:
  skills: [speckit]
  tools: [git]
```

### 3. Preflight validation with auto-install

Before any step runs, check that required tools exist on `$PATH` and skills are installed. Auto-install skills if `install:` is declared.

### 4. Git worktree workspaces

```yaml
workspace:
  type: worktree
  branch: "{{ branch }}"
```

Wave creates/removes worktrees as part of step lifecycle. Each worktree is independent, enabling concurrent pipeline runs.

### 5. Skill provisioning into workspaces

During workspace setup, `prepareWorkspace()` copies skill commands into the workspace `.claude/commands/` directory.

### 6. Prompt simplification

Pipeline prompts shrink from 80 lines of plumbing to pure intent:
```markdown
Run `/speckit.specify` with: {{ input }}
```

## Acceptance Criteria

1. `skills:` section parsed and validated in `wave.yaml` manifest
2. `requires:` block parsed and validated in pipeline YAML
3. Preflight validation runs before any pipeline step, checking tools and skills
4. Auto-install triggers when skills are missing and `install:` is declared
5. `workspace.type: worktree` creates a git worktree for the step
6. Worktree cleanup happens after step completion
7. Skill commands are provisioned into workspace `.claude/commands/`
8. Concurrent pipeline runs do not collide on git operations
9. Existing pipelines without `requires:` or `workspace.type` continue to work unchanged
10. All changes have unit tests with race condition testing

## Related Issues

- Supersedes #48 (skill/command support in workspaces)
- Addresses dependency validation from #74 (fresh VM testing)
- Solves #29 concurrency concern (git worktree isolation)
- Makes #27 (`--preserve-workspace`) more meaningful
- Replaces PR #64 approach (prompt-level workarounds → infrastructure)
