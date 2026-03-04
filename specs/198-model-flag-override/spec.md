# feat(cli): support --model flag to override adapter model per run

**Issue**: [#198](https://github.com/re-cinq/wave/issues/198)
**Labels**: enhancement
**Author**: nextlevelshit
**State**: OPEN

## Summary

Allow overriding the default adapter model via a CLI flag (e.g. `--model`) for a single pipeline run, without modifying `wave.yaml`.

## Original Request

> i want to be able to overwrite the default model for instance with haiku and/or force to use it for one specific run

## Motivation

Different pipeline runs may benefit from different models — e.g. using a faster/cheaper model like Haiku for iteration, or a more capable model like Opus for complex tasks. Currently the model is configured in `wave.yaml` under the adapter section and changing it requires editing the manifest.

## Proposed Behavior

- Add a `--model <model-id>` flag to the `wave run` command (and any other relevant subcommands)
- The flag value overrides `runtime.adapter.model` from `wave.yaml` for that invocation only
- The override applies to all steps in the pipeline unless a step explicitly pins a model
- If the flag is not provided, behavior is unchanged (manifest default is used)

## Examples

```bash
# Use haiku for a quick iteration
wave run my-pipeline --model haiku

# Use opus for a thorough review
wave run my-pipeline --model opus
```

## Acceptance Criteria

- [ ] `--model` flag is accepted by `wave run` and passed through to the adapter
- [ ] Flag overrides the manifest-configured model for all steps in the run
- [ ] CLI --model flag takes precedence over per-persona model pinning
- [ ] Invalid model identifiers produce a clear error message
- [ ] Help text documents the flag and its interaction with manifest config
- [ ] Unit tests cover flag parsing, override logic, and precedence rules
