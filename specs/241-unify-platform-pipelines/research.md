# Research: Unify Platform-Specific Pipelines

**Feature**: 241-unify-platform-pipelines
**Date**: 2026-03-13
**Status**: Complete

## Research Questions

### RQ-1: How similar are the 4 platform variants within each pipeline family?

**Decision**: Near-identical structure with CLI/API command substitutions only.

**Findings**:

The `implement` family (4 pipelines × 4 steps = 16 step definitions) differs in exactly 3 dimensions:
1. **Persona name**: `github-commenter` / `gitlab-commenter` / `bitbucket-commenter` / `gitea-commenter`
2. **Prompt source_path**: `.wave/prompts/{gh,gl,bb,gt}-implement/...`
3. **Step ID**: `create-mr` (GitLab) vs `create-pr` (all others)

All step configurations (workspace, artifacts, contracts, retry, handover) are **identical** across all 4 variants. Pipeline YAML structure, dependency chains, artifact names — all the same.

The `implement.md` and `plan.md` prompts are **byte-for-byte identical** across all 4 platforms (only the header line "GitHub issue"/"Bitbucket issue" differs). The `fetch-assess.md` and `create-pr.md` prompts differ substantively — they contain platform-specific CLI commands.

The `scope`, `research`, `rewrite`, `refresh` families follow the same pattern: inline prompts with platform-specific CLI commands, but identical pipeline structure.

**Rationale**: Unification via template variables is the right approach — it eliminates the structural duplication while preserving behavioral differences.

**Alternatives Rejected**:
- **Platform-specific prompt overlays** (rejected — adds complexity without reducing files)
- **Conditional YAML blocks** (rejected — YAML doesn't support conditionals natively; would require custom parser)

### RQ-2: Where should forge detection be invoked in the execution flow?

**Decision**: In `Execute()` method of `DefaultPipelineExecutor`, immediately after `newContextWithProject()` creates the pipeline context (line 276).

**Findings**:

The executor flow is:
1. `Execute()` → DAG validation → preflight → create context (line 276) → create execution → run steps
2. Template resolution happens per-step in `resolvePrompt()` (line 1762) via `ctx.ResolvePlaceholders()`
3. Persona resolution happens per-step in `runStepExecution()` (line 1040): `execution.Manifest.GetPersona(step.Persona)`

Forge variables must be injected **before** preflight (which needs resolved `requires.tools`) and **before** step execution (which needs resolved persona names and prompt paths).

The optimal injection point is right after `newContextWithProject()` returns the context object, before the `PipelineExecution` struct is created.

**Rationale**: This is the earliest point where we have a context but haven't started execution. All downstream placeholder resolution will pick up forge variables automatically.

**Alternatives Rejected**:
- **Per-step injection** (rejected — wasteful, inconsistent, doesn't help preflight)
- **CLI flag for forge override** (deferred — could be added later, not needed for V1)

### RQ-3: How should persona references in YAML be resolved with template variables?

**Decision**: The `step.Persona` field must be resolved through `ResolvePlaceholders()` before being passed to `execution.Manifest.GetPersona()`.

**Findings**:

Currently, `runStepExecution()` at line 1040 does:
```go
persona := execution.Manifest.GetPersona(step.Persona)
```

This does NOT resolve template variables. To support `persona: "{{ forge.prefix }}-commenter"`, we need to add:
```go
resolvedPersona := execution.Context.ResolvePlaceholders(step.Persona)
persona := execution.Manifest.GetPersona(resolvedPersona)
```

This is a 1-line change. The `ResolvePlaceholders` method already handles `CustomVariables` (where `forge.prefix` lives), and the `replaceBoth` helper handles both `{{key}}` and `{{ key }}` syntax.

**Rationale**: Minimal change, leverages existing infrastructure, consistent with how other template variables work.

### RQ-4: How should `source_path` be resolved for unified prompts?

**Decision**: `source_path` should also go through `ResolvePlaceholders()` before being used in `os.ReadFile()`.

**Findings**:

At line 1695-1699, the executor loads prompts:
```go
if step.Exec.SourcePath != "" {
    data, err := os.ReadFile(step.Exec.SourcePath)
```

The `SourcePath` is used raw. For unified pipelines, prompts will live under `.wave/prompts/implement/` (no forge prefix), so `source_path` becomes static. However, if we want forge-specific prompt files in the future (e.g., `source_path: .wave/prompts/implement/{{ forge.prefix }}-fetch-assess.md`), we should resolve it.

For V1, unified prompts use a **single file per step** that contains `{{ forge.* }}` variables in the content. The `source_path` is static (`.wave/prompts/implement/create-pr.md`), and the prompt content is resolved by the existing `ResolvePlaceholders` call at line 1762.

**Rationale**: Resolving `source_path` is cheap and provides forward compatibility. The main prompt content resolution already works via the existing codepath.

### RQ-5: How should `requires.tools` work with template variables?

**Decision**: Resolve `{{ forge.cli_tool }}` in each tool entry before passing to `CheckTools()`, and skip empty strings.

**Findings**:

The preflight check at executor lines 248-269:
```go
if p.Requires != nil {
    checker := preflight.NewChecker(p.Requires.Skills)
    var tools []string
    if len(p.Requires.Tools) > 0 {
        tools = p.Requires.Tools
    }
    ...
    results, err := checker.Run(tools, skillNames)
```

Tools are passed as-is. For `{{ forge.cli_tool }}` to work:
1. Resolve each tool string via `pipelineContext.ResolvePlaceholders(tool)`
2. Filter out empty strings (e.g., when `forge.cli_tool` is empty for unknown forge)
3. Pass resolved tools to `checker.Run()`

This must happen **after** forge variables are injected into the context but **before** the preflight check runs.

For Bitbucket, `forge.cli_tool` resolves to `"bb"` — a placeholder CLI that doesn't exist. The unified pipeline should also hardcode `curl` and `jq` as static entries:
```yaml
requires:
  tools:
    - "{{ forge.cli_tool }}"
    - git
```

The preflight checker should silently skip empty strings from unresolved variables.

**Rationale**: Minimal change to checker, works with existing YAML schema.

### RQ-6: What backward compatibility mechanism is needed?

**Decision**: A `resolveDeprecatedPipelineName()` function that strips forge prefixes and logs a deprecation warning.

**Findings**:

Pipeline lookup happens in `cmd/wave/commands/run.go` when the user specifies a pipeline name. The existing `FilterPipelinesByForge()` in `internal/forge/detect.go` is used by `suggest` and `doctor` but not by the run command.

The simplest approach:
1. Add a `ResolveDeprecatedName(name string) (resolvedName string, deprecated bool)` function
2. Call it in the run command before loading the pipeline
3. If `deprecated` is true, log a warning to stderr
4. The function checks if `name` has a known forge prefix, strips it, and returns the base name

**Rationale**: Simple string transformation, no YAML changes, clear deprecation path.

### RQ-7: Which personas exist per forge and which are shared?

**Decision**: 16 forge-specific personas exist (4 roles × 4 forges). These REMAIN as separate files — they are NOT unified.

**Findings**:

Forge-specific personas discovered:
- `github-analyst`, `gitlab-analyst`, `bitbucket-analyst`, `gitea-analyst`
- `github-commenter`, `gitlab-commenter`, `bitbucket-commenter`, `gitea-commenter`
- `github-enhancer`, `gitlab-enhancer`, `bitbucket-enhancer`, `gitea-enhancer`
- `github-scoper`, `gitlab-scoper`, `bitbucket-scoper`, `gitea-scoper`

Shared personas (no forge prefix, used by all pipelines):
- `implementer`, `craftsman`, `navigator`, `researcher`, `reviewer`, `summarizer`

The forge-specific personas contain different CLI tool permissions (e.g., `Bash(gh *)` for GitHub vs `Bash(curl *)` for Bitbucket). These **must remain separate** — unifying them would require conditional permission logic that doesn't exist.

**Rationale**: Personas are the correct level of forge-specific abstraction. The pipeline references them dynamically via `{{ forge.prefix }}-commenter`, but the persona files themselves remain per-forge.

### RQ-8: What are the 10 known duplication bugs referenced in issue #241?

**Decision**: These are bugs where a fix was applied to one platform's prompt but not the others. Unification eliminates them by definition.

**Findings**:

From the spec (FR-009): "System MUST fix all 10 known duplication bugs documented in issue #241 comments." Since we're replacing all 25 platform-specific pipelines with 6 unified ones, any per-platform divergence is automatically eliminated. The unified prompt files serve as single source of truth.

**Rationale**: The unification itself is the fix — no separate bug-fixing step needed.
