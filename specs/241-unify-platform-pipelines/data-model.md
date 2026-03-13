# Data Model: Unify Platform-Specific Pipelines

**Feature**: 241-unify-platform-pipelines
**Date**: 2026-03-13

## Entities

### ForgeInfo (Modified)

**Location**: `internal/forge/detect.go`
**Type**: Existing struct — add 2 new fields

```go
type ForgeInfo struct {
    Type           ForgeType `json:"type"`
    Host           string    `json:"host"`
    Owner          string    `json:"owner"`
    Repo           string    `json:"repo"`
    CLITool        string    `json:"cli_tool"`
    PipelinePrefix string    `json:"pipeline_prefix"`
    PRTerm         string    `json:"pr_term"`      // NEW: "Pull Request" or "Merge Request"
    PRCommand      string    `json:"pr_command"`   // NEW: "pr" or "mr"
}
```

**New field values by forge**:

| ForgeType  | PRTerm          | PRCommand |
|------------|-----------------|-----------|
| GitHub     | Pull Request    | pr        |
| GitLab     | Merge Request   | mr        |
| Bitbucket  | Pull Request    | pr        |
| Gitea      | Pull Request    | pr        |
| Unknown    | (empty)         | (empty)   |

**Impact**: `forgeMetadata()` function must be extended to return 4 values instead of 2 (cli, prefix, prTerm, prCommand).

### PipelineContext (Modified)

**Location**: `internal/pipeline/context.go`
**Type**: Existing struct — no new fields, uses `CustomVariables` map

No structural changes. Forge variables are injected via the existing `SetCustomVariable()` method:

```go
// InjectForgeVariables populates forge.* template variables in the context.
func InjectForgeVariables(ctx *PipelineContext, info forge.ForgeInfo) {
    ctx.SetCustomVariable("forge.type", string(info.Type))
    ctx.SetCustomVariable("forge.host", info.Host)
    ctx.SetCustomVariable("forge.owner", info.Owner)
    ctx.SetCustomVariable("forge.repo", info.Repo)
    ctx.SetCustomVariable("forge.cli_tool", info.CLITool)
    ctx.SetCustomVariable("forge.prefix", info.PipelinePrefix)
    ctx.SetCustomVariable("forge.pr_term", info.PRTerm)
    ctx.SetCustomVariable("forge.pr_command", info.PRCommand)
}
```

**Template variables available after injection**:

| Variable             | Example (GitHub)  | Example (GitLab)   |
|---------------------|-------------------|---------------------|
| `forge.type`        | `github`          | `gitlab`            |
| `forge.host`        | `github.com`      | `gitlab.com`        |
| `forge.owner`       | `re-cinq`         | `re-cinq`           |
| `forge.repo`        | `wave`            | `wave`              |
| `forge.cli_tool`    | `gh`              | `glab`              |
| `forge.prefix`      | `gh`              | `gl`                |
| `forge.pr_term`     | `Pull Request`    | `Merge Request`     |
| `forge.pr_command`  | `pr`              | `mr`                |

### Unified Pipeline (New files, replaces existing)

**Location**: `internal/defaults/pipelines/`
**Type**: New YAML files replacing 25 existing files

6 new pipeline files:
- `implement.yaml` — replaces `{gh,gl,bb,gt}-implement.yaml` (4 files)
- `scope.yaml` — replaces `{gh,gl,bb,gt}-scope.yaml` (4 files)
- `research.yaml` — replaces `{gh,gl,bb,gt}-research.yaml` (4 files)
- `rewrite.yaml` — replaces `{gh,gl,bb,gt}-rewrite.yaml` (4 files)
- `refresh.yaml` — replaces `{gh,gl,bb,gt}-refresh.yaml` (4 files)
- `pr-review.yaml` — replaces `gh-pr-review.yaml` (1 file) + new for 3 other forges

Key differences from current pipeline YAML:
- `persona: "{{ forge.prefix }}-commenter"` instead of `persona: github-commenter`
- `source_path: .wave/prompts/implement/create-pr.md` instead of `.wave/prompts/gh-implement/create-pr.md`
- `requires.tools: ["{{ forge.cli_tool }}", "git"]` instead of per-platform tool lists
- `metadata.name: implement` instead of `gh-implement`

### Unified Prompt Files (New files, replaces existing)

**Location**: `internal/defaults/prompts/`
**Type**: New directories replacing 4 platform-specific directories per family

New directories:
- `prompts/implement/` — replaces `{gh,gl,bb,gt}-implement/`
- `prompts/scope/` — (inline prompts, no directory needed unless extracted)

For the `implement` family:
- `implement/fetch-assess.md` — uses `{{ forge.cli_tool }}` for CLI commands
- `implement/plan.md` — shared (already identical across platforms)
- `implement/implement.md` — shared (already identical across platforms)
- `implement/create-pr.md` — uses `{{ forge.cli_tool }}`, `{{ forge.pr_term }}`, `{{ forge.pr_command }}`

### DeprecatedNameResolver (New)

**Location**: `internal/pipeline/deprecated.go` (new file)
**Type**: New function

```go
// ResolveDeprecatedName checks if a pipeline name uses a legacy forge-prefixed
// format and returns the unified name with a deprecation flag.
func ResolveDeprecatedName(name string) (resolved string, deprecated bool) {
    prefixes := []string{"gh-", "gl-", "bb-", "gt-"}
    for _, p := range prefixes {
        if strings.HasPrefix(name, p) {
            return name[len(p):], true
        }
    }
    return name, false
}
```

## Relationships

```
ForgeInfo ──(injected into)──> PipelineContext.CustomVariables
                                      │
                                      ▼
                              ResolvePlaceholders()
                                      │
                    ┌─────────────────┼─────────────────┐
                    ▼                 ▼                  ▼
              step.Persona      step.Exec.Source   requires.tools
              (resolved)        (prompt content)    (resolved)
                    │                 │                  │
                    ▼                 ▼                  ▼
            GetPersona()        prompt output     CheckTools()
```

## Files Changed

### New files
- `internal/pipeline/deprecated.go` — `ResolveDeprecatedName()` function
- `internal/defaults/pipelines/implement.yaml` — unified implement pipeline
- `internal/defaults/pipelines/scope.yaml` — unified scope pipeline
- `internal/defaults/pipelines/research.yaml` — unified research pipeline
- `internal/defaults/pipelines/rewrite.yaml` — unified rewrite pipeline
- `internal/defaults/pipelines/refresh.yaml` — unified refresh pipeline
- `internal/defaults/pipelines/pr-review.yaml` — unified pr-review pipeline
- `internal/defaults/prompts/implement/fetch-assess.md` — unified prompt
- `internal/defaults/prompts/implement/plan.md` — unified prompt
- `internal/defaults/prompts/implement/implement.md` — unified prompt
- `internal/defaults/prompts/implement/create-pr.md` — unified prompt

### Modified files
- `internal/forge/detect.go` — add `PRTerm`/`PRCommand` to ForgeInfo, update `forgeMetadata()`
- `internal/forge/detect_test.go` — update tests for new fields
- `internal/pipeline/context.go` — add `InjectForgeVariables()` helper
- `internal/pipeline/context_test.go` — add tests for forge variable injection
- `internal/pipeline/executor.go` — inject forge vars, resolve persona, resolve tools, resolve source_path
- `internal/pipeline/executor_test.go` — test forge integration
- `internal/preflight/preflight.go` — skip empty strings in `CheckTools()`
- `internal/suggest/engine.go` — update to work with unified pipeline names
- `internal/doctor/optimize.go` — update for unified pipeline classification
- `cmd/wave/commands/run.go` — add deprecated name resolution

### Deleted files (25 pipeline YAMLs + prompt dirs)
- `internal/defaults/pipelines/{bb,gh,gl,gt}-implement.yaml`
- `internal/defaults/pipelines/{bb,gh,gl,gt}-scope.yaml`
- `internal/defaults/pipelines/{bb,gh,gl,gt}-research.yaml`
- `internal/defaults/pipelines/{bb,gh,gl,gt}-rewrite.yaml`
- `internal/defaults/pipelines/{bb,gh,gl,gt}-refresh.yaml`
- `internal/defaults/pipelines/gh-pr-review.yaml`
- `internal/defaults/prompts/{bb,gh,gl,gt}-implement/` (all files)
