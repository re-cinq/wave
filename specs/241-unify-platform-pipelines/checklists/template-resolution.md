# Template Resolution Quality Checklist — Forge Variables

**Feature**: 241-unify-platform-pipelines
**Generated**: 2026-03-13

This checklist validates the quality of requirements around the forge template variable system,
which is the core technical mechanism enabling unification.

## Variable Lifecycle

- [ ] CHK033 - Is the injection point for forge variables (after `newContextWithProject`, before preflight) specified with enough precision that an implementer knows the exact code location? [Completeness]
- [ ] CHK034 - Are requirements defined for the order of resolution — do forge variables resolve before or after existing built-in variables (e.g., `{{ pipeline.name }}`, `{{ step.name }}`)? Does order matter? [Clarity]
- [ ] CHK035 - Is the spec clear on whether `ResolvePlaceholders` is called once per field type (persona, source_path, tool) or whether a single pass resolves all fields? [Clarity]
- [ ] CHK036 - Does the spec define whether forge variables should be visible in debug/trace output (e.g., `--debug` flag) for troubleshooting? [Completeness]
- [ ] CHK037 - Is there a requirement for validating that all injected `forge.*` variables are non-empty when forge is successfully detected (i.e., no silent partial injection)? [Coverage]

## Resolution Contexts

- [ ] CHK038 - Are template variable resolution requirements specified for ALL three contexts where variables appear: (1) YAML pipeline fields, (2) prompt file content, (3) `requires.tools` arrays? [Completeness]
- [ ] CHK039 - Is it specified whether `{{ forge.* }}` variables are resolved in pipeline-level metadata fields (name, description) or only in step-level fields? [Clarity]
- [ ] CHK040 - Does the spec address template variable resolution in `inject_artifacts` or `output_artifacts` path fields, or are these excluded? [Coverage]
- [ ] CHK041 - Is the spec clear on whether inline prompts (embedded in YAML `exec.prompt` field) go through the same `ResolvePlaceholders` path as file-based prompts loaded via `source_path`? [Clarity]

## Error Handling

- [ ] CHK042 - Is the expected behavior for partially-resolved templates clearly specified? (e.g., `{{ forge.prefix }}-commenter` where prefix is empty yields `-commenter` — is this caught as an error?) [Completeness]
- [ ] CHK043 - Does the spec define error behavior when a resolved persona name (e.g., `github-commenter`) doesn't exist in the manifest? Is this a step failure, pipeline failure, or something else? [Completeness]
- [ ] CHK044 - Is there a requirement distinguishing between "unresolvable variable" (typo like `forge.nonexistent`) and "empty variable" (valid `forge.cli_tool` that resolves to empty string for unknown forge)? [Clarity]
- [ ] CHK045 - Does the spec define whether failed forge detection is a hard error (pipeline aborts) or a soft error (pipeline continues with empty forge variables)? [Completeness]

## Platform Parity

- [ ] CHK046 - Are the forge variable values for Bitbucket's `cli_tool` ("bb") documented as intentionally non-functional, and is the rationale captured for why Bitbucket uses curl/jq directly rather than a CLI? [Clarity]
- [ ] CHK047 - Is the difference between Gitea's CLI tool (`tea`) vs the other forge CLIs addressed — does `tea` have feature parity with `gh`/`glab` for the operations used in pipelines? [Coverage]
- [ ] CHK048 - Are requirements defined for forge-specific prompt sections that cannot be handled by simple variable substitution (e.g., Bitbucket's `$BB_TOKEN` auth headers vs GitHub's `gh auth` flow)? [Completeness]
- [ ] CHK049 - Does the spec address whether the `{{ forge.pr_term }}` variable handles pluralization (e.g., "Pull Requests" vs "Merge Requests" in contexts that reference multiple)? [Coverage]
