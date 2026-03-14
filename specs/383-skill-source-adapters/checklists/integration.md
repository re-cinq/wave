# Integration Requirements Quality — Ecosystem Adapters for Skill Sources

**Feature**: #383 | **Date**: 2026-03-14

## Codebase Integration

- [ ] CHK201 - Does the spec define how the `SourceRouter` is constructed and wired into the application (who creates it, where it lives)? [Completeness]
- [ ] CHK202 - Is the `NewDefaultRouter(projectRoot)` factory function specified for creating a fully-wired router with all 7 adapters? [Completeness]
- [ ] CHK203 - Does the spec clarify whether existing `skill.Provisioner` needs modification to use the source adapter system, or whether integration is deferred? [Clarity]
- [ ] CHK204 - Are all new source files placed in `internal/skill/` consistent with the existing package structure (no new subpackages)? [Consistency]
- [ ] CHK205 - Does the spec confirm that existing `Store.Write()` validation (name validation, description check, serialization) applies to adapter-installed skills? [Consistency]

## Testability

- [ ] CHK206 - Is the `lookPathFunc` abstraction for mocking `exec.LookPath` defined as a requirement, not just an implementation detail? [Completeness]
- [ ] CHK207 - Does the spec require dependency injection patterns (store, lookPath, HTTP client) that enable unit testing without real external tools? [Coverage]
- [ ] CHK208 - Is the test strategy documented — table-driven tests, mocked deps, `go test -race` requirement? [Completeness]
- [ ] CHK209 - Are test data helpers (e.g., `makeTestSkillDir`, `createTestTarGz`) specified for consistent test setup? [Completeness]
- [ ] CHK210 - Does SC-006 (`go test -race ./internal/skill/...`) cover all new source files? [Coverage]

## Extensibility

- [ ] CHK211 - Is SC-007 (new adapter without modifying existing code) verifiable — can a third-party adapter be added by implementing the interface and calling `Register()`? [Coverage]
- [ ] CHK212 - Does the `SourceRouter.Register()` method handle duplicate prefix registration (overwrite silently, or error)? [Completeness]
- [ ] CHK213 - Is the adapter self-registration pattern (adapter declares its own prefix via `Prefix()`) clearly documented as the extensibility mechanism? [Clarity]

## Error Contract

- [ ] CHK214 - Is the error type hierarchy clear — `*DependencyError` for missing tools, `*ParseError` for invalid SKILL.md, generic `error` for other failures? [Clarity]
- [ ] CHK215 - Does the spec define whether adapter errors are wrapped with context (e.g., which adapter failed, what source string was used)? [Completeness]
- [ ] CHK216 - Are all error paths in the acceptance scenarios mapped to specific error types or messages? [Coverage]
