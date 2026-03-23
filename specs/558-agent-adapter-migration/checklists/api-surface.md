# API Surface Checklist

**Feature**: #558 — Migrate Adapter to Agent-Based Execution
**Date**: 2026-03-23

This checklist validates that public API changes (types, functions, CLI flags) are
fully specified with clear migration paths.

---

## Type Changes

- [ ] CHK201 - Is the `SandboxOnlySettings` type's JSON serialization format fully specified (field names, omitempty behavior)? [Type Changes]
- [ ] CHK202 - Are all callers of `ClaudeSettings` enumerated in the spec or plan to ensure migration coverage? [Type Changes]
- [ ] CHK203 - Is the `PersonaSpec` struct documented as unchanged, and is there a requirement that no new fields are added? [Type Changes]
- [ ] CHK204 - Is the `AdapterRunConfig` struct change (field removal only) confirmed to not break any external consumers? [Type Changes]

## Function Contracts

- [ ] CHK205 - Is `PersonaToAgentMarkdown`'s contract clearly defined: pure function, no side effects, no normalization? [Function Contracts]
- [ ] CHK206 - Are the input/output types of `PersonaToAgentMarkdown` specified with enough detail to write tests without reading the implementation? [Function Contracts]
- [ ] CHK207 - Is `prepareWorkspace`'s new responsibility (TodoWrite injection) documented as a pre-condition for `PersonaToAgentMarkdown`? [Function Contracts]
- [ ] CHK208 - Is the `buildArgs` return value fully specified — exact flag order, quoting rules, and conditional inclusion criteria? [Function Contracts]

## CLI Interface

- [ ] CHK209 - Is the `wave agent export` command's output format change (no normalization) documented as an intentional behavior change? [CLI Interface]
- [ ] CHK210 - Are `wave agent list` and `wave agent inspect` confirmed as unaffected with specific verification criteria? [CLI Interface]
- [ ] CHK211 - Is the `--agent` flag path format specified (relative to workspace root, no leading `./`)? [CLI Interface]

## Test Plan Alignment

- [ ] CHK212 - Does every FR (FR-001 through FR-012) have at least one corresponding test task in tasks.md? [Test Alignment]
- [ ] CHK213 - Does every success criterion (SC-001 through SC-008) have a corresponding validation task? [Test Alignment]
- [ ] CHK214 - Are all 7 edge cases covered by at least one test task? [Test Alignment]
