# Requirements Quality Review — Unify Platform-Specific Pipelines

**Feature**: 241-unify-platform-pipelines
**Generated**: 2026-03-13

## Completeness

- [ ] CHK001 - Are all 8 forge template variables (`forge.type`, `forge.host`, `forge.owner`, `forge.repo`, `forge.cli_tool`, `forge.prefix`, `forge.pr_term`, `forge.pr_command`) defined with concrete values for all 4 forge types AND ForgeUnknown? [Completeness]
- [ ] CHK002 - Does the spec define expected system behavior when `InjectForgeVariables` is called with a `ForgeUnknown` type (empty strings for all fields)? Are downstream steps required to handle empty forge variable values? [Completeness]
- [ ] CHK003 - Are acceptance scenarios provided for all 7 user stories covering all 4 forge platforms, or are some platforms only implicitly covered? [Completeness]
- [ ] CHK004 - Is the format and content of the deprecation warning message (US-6) specified, or is it left to implementation? [Completeness]
- [ ] CHK005 - Does FR-008 specify whether `wave list pipelines` should show deprecated aliases alongside unified names, or only unified names? [Completeness]
- [ ] CHK006 - Is the error message format for forge detection failure (FR-011, edge case 1) specified with enough detail to implement consistently? [Completeness]
- [ ] CHK007 - Does the spec define what happens when a user has both a local custom `gh-implement.yaml` AND the unified `implement.yaml` exists — which takes precedence? [Completeness]
- [ ] CHK008 - Are all 6 pipeline families explicitly listed with their step structures in the spec, or only `implement` with others implied to follow the same pattern? [Completeness]
- [ ] CHK009 - Does the spec define how `pr-review` pipeline steps differ across forges (US-5), given that only `gh-pr-review` currently exists and the other 3 are new capabilities? [Completeness]
- [ ] CHK010 - Is there a requirement specifying how the 10 duplication bugs (FR-009) should be validated as resolved, beyond "unification eliminates them by definition"? [Completeness]

## Clarity

- [ ] CHK011 - Is it unambiguous what `forge.cli_tool` resolves to for Bitbucket? FR-007 says "bb (unused placeholder)" — is the intent that `bb` passes preflight (it doesn't exist on disk) or that preflight should skip it? [Clarity]
- [ ] CHK012 - Does the spec clearly distinguish between "inline prompts" (scope/research/rewrite/refresh) and "file-based prompts" (implement) and state requirements for template variable resolution in both contexts? [Clarity]
- [ ] CHK013 - Is the boundary between "prompt content resolution" (existing `ResolvePlaceholders` at line 1762) and "source_path resolution" (new, at line 1695) clearly defined to avoid double-resolution? [Clarity]
- [ ] CHK014 - Is the phrase "system routes to the unified pipeline" (US-6 scenario 1) clear about whether the deprecated name resolves before or after manifest loading? [Clarity]
- [ ] CHK015 - Does FR-007 clearly specify how Bitbucket's `curl`/`jq` tools are added — as static YAML entries in the unified pipeline, or injected by the executor based on forge type? [Clarity]
- [ ] CHK016 - Is the `{{ forge.nonexistent }}` behavior (US-2 scenario 5) consistent with the edge case 3 statement that "unresolved variable causes a command to fail"? Are these requirements in tension? [Clarity]
- [ ] CHK017 - Is it clear which `ResolvePlaceholders` call handles persona field resolution vs prompt content resolution vs tool resolution — or could implementers confuse the three callsites? [Clarity]

## Consistency

- [ ] CHK018 - FR-003 specifies 6 unified pipelines, but SC-001 says "25 to 0 replaced by 6 unified files" — is this count consistent with the deletion list (25 files includes `gh-pr-review.yaml` which is the only PR review variant, not 4)? [Consistency]
- [ ] CHK019 - SC-003 says "24 pipeline×platform combinations" (6×4), but SC-005 says pr-review on 3 platforms is "new capability" — is pr-review tested against 4 platforms (SC-003) or 3 new ones (SC-005)? [Consistency]
- [ ] CHK020 - The plan says `embed.FS` patterns are `pipelines/*.yaml` and `prompts/**/*.md` but the spec doesn't mention embed.go — are there requirements for verifying embed patterns continue to work after directory restructuring? [Consistency]
- [ ] CHK021 - FR-007 says Bitbucket `forge.cli_tool` resolves to `bb` but the data model shows Bitbucket `CLITool` is populated from `forgeMetadata()` — is the `bb` value documented in the data model's forge type table? [Consistency]
- [ ] CHK022 - The plan Phase 2.3 says scope/research/rewrite/refresh use "inline prompts" requiring no prompt directories, but Phase 4 of the tasks only creates prompt files for the `implement` family — is this explicitly stated as intentional in the spec? [Consistency]
- [ ] CHK023 - FR-008 says `FilterPipelinesByForge` "MUST NOT filter out non-prefixed pipelines" but edge case 4 says "local customizations take precedence" — do these interact if a user has a local `implement.yaml` that shadows the embedded one? [Consistency]
- [ ] CHK024 - The tasks list 39 tasks across 9 phases but the plan lists 5 phases — are these aligned on the same logical grouping? [Consistency]

## Coverage

- [ ] CHK025 - Are self-hosted forge instances (GitLab CE/EE on custom domains, Gitea on custom domains) covered by the forge detection requirements, or only SaaS instances? [Coverage]
- [ ] CHK026 - Does the spec address the case where `git remote` returns an SSH URL vs HTTPS URL — are both URL formats handled by existing `DetectFromGitRemotes`? [Coverage]
- [ ] CHK027 - Is there a requirement for what happens when multiple pipelines match a deprecated name (e.g., if both `gh-implement` alias and `implement` unified exist in the manifest simultaneously during migration)? [Coverage]
- [ ] CHK028 - Are requirements defined for running unified pipelines in CI/CD environments where forge detection may not have git remotes configured? [Coverage]
- [ ] CHK029 - Does the spec address the test update requirements (T038) with specific file paths or patterns to search, or is it left to the implementer to discover? [Coverage]
- [ ] CHK030 - Is there a requirement for `wave validate` to check that unified pipeline YAML files with `{{ forge.* }}` variables are syntactically valid even before forge resolution? [Coverage]
- [ ] CHK031 - Does the spec address how `wave run implement --help` or pipeline description changes when the pipeline becomes forge-agnostic? [Coverage]
- [ ] CHK032 - Are concurrent execution scenarios considered — what happens when two unified pipelines run simultaneously on the same repo (both call `DetectFromGitRemotes`)? [Coverage]
