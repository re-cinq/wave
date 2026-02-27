# feat: inject concurrent agent count into persona system prompt from pipeline definition

**Issue**: [#112](https://github.com/re-cinq/wave/issues/112)
**Labels**: enhancement, personas, pipeline
**Author**: nextlevelshit
**Complexity**: medium

## Summary

When a pipeline step specifies `max_concurrent_agents`, the persona's system prompt should include a clear directive telling Claude the allowed level of concurrency. Currently this configuration value is accepted in the pipeline definition but is not surfaced to the agent at runtime.

## Background

Pipeline steps support a `max_concurrent_agents` field (or equivalent) in the pipeline YAML definition. Claude does not receive any instruction about this limit in its system prompt, so it cannot adapt its behavior accordingly.

## Proposed Change

When `max_concurrent_agents` is set to a value greater than 1 in the pipeline step definition, inject a line into the persona's generated system prompt such as:

```
You may spawn up to <N> concurrent sub-agents or workers for this step.
```

Example pipeline YAML:

```yaml
pipelines:
  - name: my-pipeline
    steps:
      - name: implement
        persona: implementer
        max_concurrent_agents: 3
```

Expected generated prompt addition:

```
You may spawn up to 3 concurrent sub-agents or workers for this step.
```

## Relevant Files

- `internal/pipeline/` — pipeline step execution and persona prompt construction
- `.wave/personas/` — persona system prompt templates
- `internal/manifest/` — pipeline definition loading and validation

## Acceptance Criteria

- [ ] When `max_concurrent_agents > 1` is set on a pipeline step, the persona system prompt includes a concurrency hint
- [ ] When `max_concurrent_agents` is unset or `<= 1`, no concurrency hint is added
- [ ] The configuration key name is documented in the pipeline YAML reference
- [ ] Unit tests cover prompt generation with and without the concurrency field
- [ ] Integration test verifies the hint appears in the adapter invocation

## Research Findings

From the research comment on the issue:

- **Claude Code concurrency limits**: Claude Code caps at 10 concurrent subagents via Task tool; no maxParallelAgents setting exists yet
- **Prompt wording compliance**: Permission language works better than prohibition language
- **CLAUDE.md injection point**: New section between contract compliance and restrictions
- **YAML schema design**: `MaxConcurrentAgents int` on Step struct
- **Task tool vs Agent Teams**: Target Task tool subagents only
- **Prompt drift mitigation**: Fresh-context-per-step mitigates drift
- **Testing strategy**: Follow `TestCLAUDEMDRestrictionSection` pattern
- **Security considerations**: Minimal risk; cap at 10
- **Multi-adapter compatibility**: Carry raw int on `AdapterRunConfig`
- **Documentation requirements**: Document three concurrency levels
