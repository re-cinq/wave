# Detection Matrix Quality Checklist

**Feature**: Init Cold-Start Fix, Flavour Auto-Detection
**Date**: 2026-03-16

This checklist validates the quality of requirements for the 25+ language flavour detection matrix.

## Completeness

- [ ] CHK030 - Does every detection rule specify all six output fields (flavour, language, test, lint, build, format, source glob)? [Completeness]
- [ ] CHK031 - Are empty command fields explicitly documented as intentional (e.g., bun has no format_command) vs accidentally omitted? [Completeness]
- [ ] CHK032 - Is the behavior for marker files that exist but are empty or malformed defined? [Completeness]
- [ ] CHK033 - Are all Node ecosystem variants covered (npm, yarn classic, yarn berry, pnpm, bun)? [Completeness]
- [ ] CHK034 - Is the Python ecosystem coverage complete (pyproject.toml, setup.py, setup.cfg, requirements.txt)? [Completeness]

## Clarity

- [ ] CHK035 - Is it clear that markers use ANY-match semantics (one marker suffices) rather than ALL-match? [Clarity]
- [ ] CHK036 - Is the glob pattern matching behavior defined (e.g., does `*.csproj` match in subdirectories or only the root)? [Clarity]
- [ ] CHK037 - Is it clear which command fields accept empty string as "not applicable" vs which require a value? [Clarity]

## Consistency

- [ ] CHK038 - Are command styles consistent across similar ecosystems (e.g., all use `--check` for format verification where available)? [Consistency]
- [ ] CHK039 - Is the Gradle/Kotlin overlap explicitly resolved in the priority ordering (Kotlin rule uses same marker as Gradle)? [Consistency]
- [ ] CHK040 - Are the source glob patterns consistent in format (`*.go` vs `*.{ts,tsx}`) and is the brace syntax documented? [Consistency]

## Coverage

- [ ] CHK041 - Are version-specific tool differences addressed (e.g., yarn classic vs yarn berry have different lock file formats)? [Coverage]
- [ ] CHK042 - Is the Flutter vs pure Dart distinction addressed (both use `pubspec.yaml`)? [Coverage]
- [ ] CHK043 - Are polyglot projects (e.g., Go backend + TypeScript frontend) addressed in the detection strategy? [Coverage]
- [ ] CHK044 - Is the `make` flavour ambiguity addressed — many projects have a Makefile alongside a language-specific build tool? [Coverage]
