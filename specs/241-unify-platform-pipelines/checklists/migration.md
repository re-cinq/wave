# Migration & Backward Compatibility Checklist

**Feature**: 241-unify-platform-pipelines
**Generated**: 2026-03-13

This checklist validates the quality of requirements around the migration from 25 platform-specific
pipelines to 6 unified pipelines and the backward compatibility guarantees.

## Deprecation Path

- [ ] CHK050 - Is the lifecycle of deprecated names specified — are they permanent aliases or is there a planned removal timeline? [Completeness]
- [ ] CHK051 - Does the spec define whether the deprecation warning is shown once per session, once per invocation, or once ever (with a "don't show again" mechanism)? [Completeness]
- [ ] CHK052 - Is it specified whether deprecated name resolution applies only to `wave run` or also to `wave validate`, `wave list`, and other subcommands that accept pipeline names? [Coverage]
- [ ] CHK053 - Does the spec address the case where a user's automation parses stderr and the new deprecation warning breaks their parsing? [Coverage]
- [ ] CHK054 - Are requirements defined for what `wave list pipelines` shows after unification — only unified names, or both old and new? [Completeness]

## File Deletion Safety

- [ ] CHK055 - Is the deletion of 25 pipeline YAML files and 4 prompt directories gated on the unified replacements being verified as functionally equivalent first? [Completeness]
- [ ] CHK056 - Does the spec address whether deleted prompt directories (`bb-implement/`, `gh-implement/`, etc.) could be referenced by user-created pipelines outside the defaults? [Coverage]
- [ ] CHK057 - Is there a requirement for verifying that no Go source code (beyond embed.go) references the old forge-prefixed pipeline or prompt directory names as string literals? [Coverage]
- [ ] CHK058 - Does the spec define whether the deletion should happen atomically (all 25+4 at once) or incrementally (per family), and what the rollback strategy is if tests fail mid-deletion? [Clarity]

## Supporting System Updates

- [ ] CHK059 - Are requirements for the suggest engine update (T026-T027) specified at the same level of detail as the core executor changes, or are they underspecified? [Completeness]
- [ ] CHK060 - Does the spec address how the doctor's `classifyPipeline()` should categorize unified pipelines — as "universal" or under a new category? [Completeness]
- [ ] CHK061 - Is the interaction between `FilterPipelinesByForge` returning all pipelines and the suggest engine's `FilterByForge` documented — could these produce different results for the same input? [Consistency]
- [ ] CHK062 - Are there requirements for updating the CLAUDE.md pipeline selection table (which lists `gh-implement`, `gl-research`, etc.) after unification? [Coverage]
