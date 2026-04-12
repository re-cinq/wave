# Implementation Plan: Integrate task classifier into wave do

## 1. Objective

Wire `classify.Classify()` and `classify.SelectPipeline()` into the `wave do` command so it automatically routes tasks to the best pipeline instead of always generating a two-step ad-hoc pipeline. Add `--no-classify` flag to bypass and enhance `--dry-run` to show classification details.

## 2. Approach

Modify the `runDo` function in `do.go` to:
1. After manifest loading, call `classify.Classify(input, "")` to produce a `TaskProfile`
2. Call `classify.SelectPipeline(profile)` to get a `PipelineConfig`
3. Look up the selected pipeline name in the manifest's pipelines
4. If found, execute it via the existing pipeline executor; if not found, fall back to the ad-hoc pipeline
5. When `--no-classify` is set, skip steps 1-4 entirely and use the original ad-hoc behavior
6. When `--dry-run` is set and classification is active, print classification details (domain, complexity, blast radius, selected pipeline, reason)

The `--persona` flag applies only to the ad-hoc fallback path (where it overrides the execute persona). When a classified pipeline is selected, the pipeline's own persona configuration takes precedence.

## 3. File Mapping

| File | Action | Description |
|------|--------|-------------|
| `cmd/wave/commands/do.go` | modify | Add `NoClassify` to `DoOptions`, import `classify` package, add classification logic before pipeline generation, enhance dry-run output |
| `cmd/wave/commands/do_test.go` | modify | Add tests for classification integration, `--no-classify` flag, enhanced dry-run output, fallback behavior |

## 4. Architecture Decisions

- **Fallback-first**: Classification is additive. The existing ad-hoc pipeline remains the fallback, triggered when: (a) `--no-classify` is set, (b) the classified pipeline doesn't exist in the manifest, or (c) `--persona` is explicitly set (user wants ad-hoc with specific persona)
- **No new packages**: All needed code already exists in `internal/classify`
- **Manifest pipeline lookup**: Check `m.Pipelines` map for the classified pipeline name. This requires loading pipeline definitions from the manifest, which is already available
- **Issue body**: `classify.Classify()` accepts an `issueBody` parameter. For `wave do`, pass empty string since we only have a task description, not an issue body

## 5. Risks

| Risk | Mitigation |
|------|------------|
| Classification selects a pipeline not defined in manifest | Graceful fallback to ad-hoc pipeline with a stderr warning |
| `--persona` flag conflicts with classified pipeline | When `--persona` is explicitly set, skip classification (user is choosing ad-hoc mode) |
| Existing dry-run tests break due to output format change | Update existing assertions and add new ones; dry-run output is additive |
| Pipeline execution path differs from ad-hoc path | Both paths already converge on `pipeline.NewDefaultPipelineExecutor` — just different pipeline inputs |

## 6. Testing Strategy

- **Unit tests**: Test `--no-classify` bypasses classification, test dry-run shows classification output, test fallback when pipeline not in manifest
- **Flag tests**: Verify `--no-classify` flag exists with correct default
- **Integration with existing tests**: Ensure all existing `do_test.go` tests pass unchanged (backward compat)
- **Edge cases**: Empty input classification, persona override skipping classification
