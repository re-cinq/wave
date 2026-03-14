# Requirements Quality Review — Ecosystem Adapters for Skill Sources

**Feature**: #383 | **Date**: 2026-03-14

## Completeness

- [ ] CHK001 - Are all 7 source prefixes (`tessl:`, `bmad:`, `openspec:`, `speckit:`, `github:`, `file:`, `https://`) each backed by a dedicated user story with acceptance scenarios? [Completeness]
- [ ] CHK002 - Does the spec define the exact CLI command and arguments for all 4 CLI-based adapters (tessl, bmad, openspec, speckit)? [Completeness]
- [ ] CHK003 - Are install instructions specified for every soft dependency (`tessl`, `npx`, `openspec`, `specify`, `git`)? [Completeness]
- [ ] CHK004 - Does the spec define what `InstallResult` contains when an adapter finds multiple SKILL.md files (e.g., `tessl:` resolving to multiple skills)? [Completeness]
- [ ] CHK005 - Is the `SourceReference` type fully specified with all fields and their semantics (Prefix, Reference, Raw)? [Completeness]
- [ ] CHK006 - Are timeout values explicitly stated for all operation categories (subprocess, git clone, HTTP download, HTTP headers)? [Completeness]
- [ ] CHK007 - Does the spec define the behavior when a `github:` reference has exactly one path component (e.g., `github:repo` without owner)? [Completeness]
- [ ] CHK008 - Is the relationship between `SourceAdapter` and existing `SkillConfig`/preflight system explicitly documented as coexistence (C-002)? [Completeness]
- [ ] CHK009 - Does the spec define what archive formats are supported for `https://` and explicitly list unsupported formats? [Completeness]
- [ ] CHK010 - Does the spec state what happens when `store.Write()` fails for one skill in a multi-skill install (partial success)? [Completeness]

## Clarity

- [ ] CHK011 - Is the URL-scheme-first parsing strategy for `https://` unambiguous about precedence over generic `prefix:reference` parsing (C-001)? [Clarity]
- [ ] CHK012 - Is it clear that `http://` sources are also supported and route to the same URL adapter as `https://`? [Clarity]
- [ ] CHK013 - Are the `file:` adapter path resolution rules explicit about relative vs absolute path handling? [Clarity]
- [ ] CHK014 - Does the spec clearly distinguish between "no SKILL.md found" (error) vs "multiple SKILL.md files found" (install all) for the `github:` adapter? [Clarity]
- [ ] CHK015 - Is the bare-name behavior (FR-014: no prefix = local store lookup, not adapter invocation) clearly stated with an example? [Clarity]
- [ ] CHK016 - Is the `SourceRouter.Parse()` return type clearly specified — does it return the adapter + reference, or a `SourceReference` struct? [Clarity]
- [ ] CHK017 - Is it clear what "project root" means for the `file:` adapter — is it the git root, the CWD, or a constructor parameter? [Clarity]

## Consistency

- [ ] CHK018 - Does the `SourceAdapter.Install()` signature in the spec match the contract file (`contracts/source-adapter.go`)? [Consistency]
- [ ] CHK019 - Does the `InstallResult.Skills` field type match the existing `skill.Skill` struct in the codebase (`internal/skill/types.go`)? [Consistency]
- [ ] CHK020 - Does the `file:` adapter's path containment approach match the existing `containedPath()` function in `store.go`? [Consistency]
- [ ] CHK021 - Are timeout constants consistent across spec (FR-016, C-004), plan, and data model documents (2min subprocess, 2min network, 30s headers)? [Consistency]
- [ ] CHK022 - Does the `DependencyError` type defined in the data model match the contract file definition (same fields, same semantics)? [Consistency]
- [ ] CHK023 - Are the CLI commands in the spec consistent between user stories and functional requirements (e.g., FR-005 vs US4 for BMAD)? [Consistency]
- [ ] CHK024 - Does the plan's file layout match the data model's entity placement (all in `internal/skill/`, no new packages)? [Consistency]

## Coverage

- [ ] CHK025 - Are error scenarios defined for every adapter (missing dep, invalid reference, network failure, parse failure, empty result)? [Coverage]
- [ ] CHK026 - Does the spec address what happens when a downloaded archive exceeds a size limit or contains an unreasonable number of files? [Coverage]
- [ ] CHK027 - Does the spec address HTTP redirect behavior for the `https://` adapter (3xx responses)? [Coverage]
- [ ] CHK028 - Are concurrent adapter invocations addressed — is thread safety of `SourceRouter` defined? [Coverage]
- [ ] CHK029 - Does the spec cover the case where `git clone --depth 1` fails due to authentication (private repo) for the `github:` adapter? [Coverage]
- [ ] CHK030 - Does the spec define whether adapters log progress or emit events during long-running operations (e.g., cloning a large repo)? [Coverage]
- [ ] CHK031 - Is the zip-slip attack vector (archive entries with `../` path components) addressed in the `https://` adapter requirements? [Coverage]
- [ ] CHK032 - Does the spec define the behavior when the `file:` reference points to a file rather than a directory? [Coverage]
- [ ] CHK033 - Are the success criteria (SC-001 through SC-007) each independently verifiable via automated tests? [Coverage]
- [ ] CHK034 - Does the spec address cleanup behavior when an adapter is cancelled mid-operation (context cancellation)? [Coverage]
