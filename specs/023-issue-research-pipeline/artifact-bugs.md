# Artifact Pipeline Bugs

Discovered during first run of the `issue-research` pipeline.

## Problem Statement

The `issue-research` pipeline run exposed three interacting bugs in Wave's artifact system that cause structured data to be lost between pipeline steps. These bugs likely affect all pipelines that use `output_artifacts` with `inject_artifacts`.

## Observed Failures (issue-research pipeline)

### Step: fetch-issue

| File | Expected | Actual |
|------|----------|--------|
| `artifact.json` | JSON (written by persona) | Correct JSON |
| `output/issue-content.json` | JSON (declared output artifact) | Markdown prose |

The persona correctly wrote JSON to `artifact.json` via the Write tool. However, Wave's `writeOutputArtifacts()` then **overwrote** `output/issue-content.json` with `result.ResultContent` -- Claude's conversational summary text, not the file the persona wrote.

### Step: analyze-topics

| File | Expected | Actual |
|------|----------|--------|
| `artifacts/issue` (injected) | JSON from fetch-issue | Markdown prose (copied from overwritten file) |
| `output/research-topics.json` | JSON research topics | Error message prose |

Two failures compounded:
1. The injected artifact contained prose (because the source was overwritten by Bug 1)
2. The philosopher persona could not write `output/research-topics.json` because it only has `Write(.wave/specs/*)` permission -- no access to `output/`
3. Wave then overwrote `output/research-topics.json` with `ResultContent` again, which was the persona's error message about being blocked

## Root Causes

### Bug 1: `writeOutputArtifacts` clobbers persona-written files

**Location**: `internal/pipeline/executor.go:469-481, 816-826`

```go
// executor.go:469-474
if result.ResultContent != "" {
    artifactContent := []byte(result.ResultContent)
    e.writeOutputArtifacts(execution, step, workspacePath, artifactContent)
}

// executor.go:816-826
func (e *DefaultPipelineExecutor) writeOutputArtifacts(..., stdout []byte) {
    for _, art := range step.OutputArtifacts {
        resolvedPath := execution.Context.ResolveArtifactPath(art)
        artPath := filepath.Join(workspacePath, resolvedPath)
        os.MkdirAll(filepath.Dir(artPath), 0755)
        os.WriteFile(artPath, stdout, 0644)  // <-- overwrites persona's file
        key := step.ID + ":" + art.Name
        execution.ArtifactPaths[key] = artPath
    }
}
```

**What happens**: After a step completes, Wave unconditionally writes `result.ResultContent` (Claude's conversational result text) to every declared `output_artifacts` path. If the persona already wrote the correct file at that path, it gets clobbered.

**Why `ResultContent` is prose**: The Claude adapter extracts `ResultContent` from the `"result"` field in Claude's JSON-lines output (`claude.go:246-289`). This is Claude's final conversational message -- a summary of what it did -- not the contents of files it wrote.

### Bug 2: Philosopher persona lacks Write permission for output paths

**Location**: `wave.yaml:79-88` (persona definition), `internal/adapter/claude.go:171-187` (settings generation)

The philosopher persona is defined with:
```yaml
philosopher:
  permissions:
    allowed_tools:
      - Read
      - Write(.wave/specs/*)
    deny:
      - Bash(*)
```

This gets written to `.claude/settings.json` as:
```json
{
  "allowed_tools": ["Read", "Write(.wave/specs/*)"]
}
```

The pipeline step tells the persona to `Write the result to output/research-topics.json`, but the persona can only write to `.wave/specs/*`. There is no mechanism for the pipeline step to grant additional permissions beyond the persona's base set.

### Bug 3: Persona/pipeline role mismatch

The `analyze-topics` step uses the `philosopher` persona, whose CLAUDE.md says:

> Write specifications in **markdown** with clear sections

But the pipeline expects JSON output (`type: json` on the artifact, `json_schema` contract). The philosopher persona is designed for markdown specification writing, not structured JSON extraction.

## Proposed Fixes

### Fix 1: Read persona-written files instead of using ResultContent

**Change**: `writeOutputArtifacts` should check if the file at the declared path already exists (written by the persona) and only fall back to `ResultContent` if it doesn't.

```go
func (e *DefaultPipelineExecutor) writeOutputArtifacts(
    execution *PipelineExecution, step *Step, workspacePath string, resultContent []byte,
) {
    for _, art := range step.OutputArtifacts {
        resolvedPath := execution.Context.ResolveArtifactPath(art)
        artPath := filepath.Join(workspacePath, resolvedPath)

        // If the persona already wrote the file, trust it -- don't overwrite
        if _, err := os.Stat(artPath); err == nil {
            key := step.ID + ":" + art.Name
            execution.ArtifactPaths[key] = artPath
            continue
        }

        // File doesn't exist -- fall back to ResultContent
        os.MkdirAll(filepath.Dir(artPath), 0755)
        os.WriteFile(artPath, resultContent, 0644)
        key := step.ID + ":" + art.Name
        execution.ArtifactPaths[key] = artPath
    }
}
```

**Rationale**: The pipeline YAML explicitly declares the artifact path (`path: output/issue-content.json`). The persona is instructed to write to that path. If the persona did its job, the file is already there with the correct content. Wave should not replace it with conversational text.

**Edge case**: If the persona writes invalid content, the `handover.contract` validation will catch it and trigger a retry. This is the designed safety net.

### Fix 2: Pipeline-level permission grants

**Option A (minimal)**: Add `Write(output/*)` and `Write(artifact.json)` as implicit grants for any step that declares `output_artifacts`.

In `internal/adapter/claude.go:prepareWorkspace`:

```go
allowedTools := cfg.AllowedTools
// Auto-grant Write to output paths declared in output_artifacts
for _, art := range cfg.OutputArtifacts {
    dir := filepath.Dir(art.Path)
    grant := fmt.Sprintf("Write(%s/*)", dir)
    if !contains(allowedTools, grant) {
        allowedTools = append(allowedTools, grant)
    }
}
// Also grant Write(artifact.json) for backward compat
if len(cfg.OutputArtifacts) > 0 {
    allowedTools = append(allowedTools, "Write(artifact.json)")
}
```

**Option B (explicit)**: Allow pipeline steps to declare additional permissions that merge with the persona's base permissions:

```yaml
- id: analyze-topics
  persona: philosopher
  permissions_override:
    additional_allowed:
      - Write(output/*)
```

**Recommendation**: Option A. Personas already opt in to producing artifacts by being assigned to a step with `output_artifacts`. The permission to write to those paths is implied by the contract.

### Fix 3: Use appropriate persona for analyze-topics

**Change the pipeline** to use the `researcher` persona (or a new `analyst` persona) instead of `philosopher` for the `analyze-topics` step.

The `researcher` persona already has:
```yaml
researcher:
  permissions:
    allowed_tools:
      - Read
      - Glob
      - Grep
      - WebSearch
      - WebFetch
      - Write(artifact.json)
      - Write(output/*)
```

This is a better fit because:
- Has `Write(output/*)` permission
- Its role is information extraction and synthesis
- The step doesn't need to write specs or architecture docs

**Alternative**: If the philosopher persona must be used, update the pipeline prompt to match its strengths (write a markdown analysis) and change the artifact `type` to `markdown`.

## Affected Files

### Must change
- `internal/pipeline/executor.go` -- Fix `writeOutputArtifacts` (Bug 1)
- `internal/adapter/claude.go` -- Auto-grant Write for output artifact paths (Bug 2)
- `.wave/pipelines/issue-research.yaml` -- Use correct persona for analyze-topics (Bug 3)

### Should verify
- All other pipelines in `.wave/pipelines/` -- Same overwrite bug may have gone unnoticed if personas happen to not write to the declared output path (Wave writes ResultContent, which might coincidentally be valid)
- `internal/pipeline/executor.go:injectArtifacts` -- The fallback to `execution.Results[ref.Step]["stdout"]` has the same problem (raw stdout contains Claude's JSON wrapper, not useful content)

## Impact Assessment

**Bug 1 is systemic**: Every pipeline that uses `output_artifacts` is affected. The only reason it hasn't been more visible is:
- Some personas don't write to the declared path (they write to `artifact.json` per CLAUDE.md instructions), so `ResultContent` is the only content available
- Some pipelines use markdown artifacts where prose `ResultContent` happens to be acceptable

**Bug 2 is pipeline-specific**: Only affects steps where the persona's base permissions don't cover the output path. Currently affects: `philosopher`, `planner`, `summarizer`, `auditor`, `navigator`, `debugger` -- any read-only or restricted persona used in a step with `output_artifacts`.

**Bug 3 is pipeline-specific**: Only affects `issue-research.yaml`'s `analyze-topics` step.

## Testing Plan

1. **Unit test for Fix 1**: Step writes file to output path -> `writeOutputArtifacts` should not overwrite it
2. **Unit test for Fix 1 fallback**: Step does NOT write file -> `writeOutputArtifacts` should use ResultContent
3. **Unit test for Fix 2**: Step with `output_artifacts` -> persona settings.json should include Write grants for those paths
4. **Integration test**: Run issue-research pipeline end-to-end, verify `artifacts/issue` in analyze-topics contains valid JSON matching the schema

## Sequencing

1. Fix 1 first (executor.go) -- highest impact, fixes the data loss
2. Fix 2 second (claude.go) -- unblocks restricted personas
3. Fix 3 last (pipeline YAML) -- pipeline-specific, least risk

## Notes

- The `artifact.json` vs `output/issue-content.json` split is a symptom of competing conventions: the `github-analyst` persona's CLAUDE.md says "Write output to artifact.json", while the pipeline says `path: output/issue-content.json`. Fix 1 resolves this by trusting whatever the persona actually wrote to the declared path.
- The `injectArtifacts` stdout fallback (`executor.go:798-809`) should be reviewed separately. It writes raw Claude stdout which contains JSON-lines wrapper markup, not useful artifact content.
