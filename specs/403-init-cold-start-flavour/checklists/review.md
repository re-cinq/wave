# Requirements Quality Review Checklist

**Feature**: Init Cold-Start Fix, Flavour Auto-Detection
**Date**: 2026-03-16

## Completeness

- [ ] CHK001 - Are all cold-start failure modes enumerated (no .git, no commits, no remote, no files)? [Completeness]
- [ ] CHK002 - Does the spec define what happens when `git init` itself fails (permissions, disk full, nested .git)? [Completeness]
- [ ] CHK003 - Are all 25+ flavour detection rules fully specified with marker files, commands, and source globs? [Completeness]
- [ ] CHK004 - Is the Node refinement logic (tsconfig override, package.json script parsing) specified with enough detail for implementation? [Completeness]
- [ ] CHK005 - Does the spec cover all existing `wave init` flags (`--merge`, `--force`, `--reconfigure`, `--all`, `--yes`) in combination with the new cold-start logic? [Completeness]
- [ ] CHK006 - Are error messages and user-facing output defined for each failure scenario? [Completeness]
- [ ] CHK007 - Is the behavior for `wave init` in a directory that already has `wave.yaml` AND no `.git` specified? [Completeness]
- [ ] CHK008 - Does the metadata extraction spec define fallback behavior for all supported manifest file formats (not just the happy path)? [Completeness]
- [ ] CHK009 - Is the initial commit scope clearly defined (which files are staged, which are excluded)? [Completeness]

## Clarity

- [ ] CHK010 - Is the priority ordering of detection rules unambiguous (e.g., Kotlin uses `build.gradle.kts` which also matches Gradle — is precedence clear)? [Clarity]
- [ ] CHK011 - Is the distinction between "flavour" and "language" clearly defined with examples of when they differ (e.g., bun flavour vs typescript language)? [Clarity]
- [ ] CHK012 - Are the forge-filtering semantics clear on what "matching" means for pipeline names (prefix matching vs metadata)? [Clarity]
- [ ] CHK013 - Is the `requiredPipelines` safeguard scope defined — just `impl-issue` or extensible to other pipelines? [Clarity]
- [ ] CHK014 - Is the interaction between `--reconfigure` and flavour re-detection clearly specified (overwrite vs merge existing values)? [Clarity]
- [ ] CHK015 - Are the conditions for the first-run suggestion heuristic ("source files exist" vs "empty project") precisely defined? [Clarity]

## Consistency

- [ ] CHK016 - Are the new `project.flavour` and `project.format_command` fields consistent with existing `project.*` field naming conventions in the manifest? [Consistency]
- [ ] CHK017 - Does the detection matrix in data-model.md match the flavour list in FR-013 exactly (no missing or extra entries)? [Consistency]
- [ ] CHK018 - Are the acceptance scenarios in all 7 user stories consistent with the functional requirements (FR-001 through FR-013)? [Consistency]
- [ ] CHK019 - Is the `ensureGitRepo()` placement consistent across both entry points (`runInit` and `runWizardInit`)? [Consistency]
- [ ] CHK020 - Does the plan's Phase ordering match the task dependency graph (no circular or missing dependencies)? [Consistency]
- [ ] CHK021 - Are the Kotlin and Gradle detection rules consistent — both use `build.gradle.kts` but yield different flavours. Is the conflict addressed? [Consistency]

## Coverage

- [ ] CHK022 - Are edge cases for glob-based markers (`*.csproj`, `*.cabal`) covered — what if multiple files match? [Coverage]
- [ ] CHK023 - Is the monorepo case addressed (multiple languages in subdirectories but detection runs at root)? [Coverage]
- [ ] CHK024 - Is the Windows path handling covered for all file detection and git operations? [Coverage]
- [ ] CHK025 - Are concurrent `wave init` invocations in the same directory addressed (race conditions on git init)? [Coverage]
- [ ] CHK026 - Is the non-interactive (`--yes`) mode covered for every new feature (cold-start, flavour, metadata, suggestion)? [Coverage]
- [ ] CHK027 - Are backward compatibility tests specified for loading existing manifests without the new `flavour`/`format_command` fields? [Coverage]
- [ ] CHK028 - Is the enterprise GitHub domain case covered in the forge detection edge cases? [Coverage]
- [ ] CHK029 - Are test requirements specified for the detection priority ordering (verifying bun beats node, deno beats node, etc.)? [Coverage]
