# Tasks: Persona Token Scoping

**Branch**: `213-persona-token-scoping` | **Date**: 2026-03-16
**Spec**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md)

---

## Phase 1: Setup

- [X] T001 [P1] Create `internal/scope/` package directory and `scope.go` with `TokenScope` struct, `Parse()`, and `ValidateScopes()` functions — `internal/scope/scope.go`
- [X] T002 [P1] Add `TokenScopes []string` field to `Persona` struct with `yaml:"token_scopes,omitempty"` tag — `internal/manifest/types.go`

## Phase 2: Foundational — Scope Parsing & Manifest Validation (US1)

- [X] T003 [P1] [US1] Implement `Parse(scopeStr string) (TokenScope, error)` for `<resource>:<permission>` and `<resource>:<permission>@<ENV_VAR>` formats with validation of canonical resources (`issues`, `pulls`, `repos`, `actions`, `packages`) and permissions (`read`, `write`, `admin`) — `internal/scope/scope.go`
- [X] T004 [P1] [US1] Implement `ValidateScopes(scopes []string) []error` that parses all scope strings and returns aggregated errors with descriptive messages for invalid syntax — `internal/scope/scope.go`
- [X] T005 [P1] [US1] Implement `PermissionSatisfies(have, need string) bool` for hierarchical permission comparison (`admin` ⊇ `write` ⊇ `read`) — `internal/scope/scope.go`
- [X] T006 [P1] [US1] Add scope validation call in `validatePersonasListWithFile()` to reject invalid `token_scopes` entries during manifest loading — `internal/manifest/parser.go`
- [X] T007 [P1] [US1] Write table-driven tests for `Parse()`: valid scopes, invalid formats, empty strings, unknown resources (lint warning), missing permission, `@ENV_VAR` suffix — `internal/scope/scope_test.go`
- [X] T008 [P1] [US1] [P] Write table-driven tests for `ValidateScopes()` and `PermissionSatisfies()` — `internal/scope/scope_test.go`
- [X] T009 [P1] [US1] [P] Write tests for manifest validation rejecting invalid `token_scopes` and accepting valid/missing ones (backward compat) — `internal/manifest/parser_test.go`

## Phase 3: Scope Resolution — Platform-Aware Mapping (US3)

- [X] T010 [P2] [US3] Implement `ScopeResolver` struct with `NewResolver(forgeType forge.ForgeType)` constructor — `internal/scope/resolver.go`
- [X] T011 [P2] [US3] Implement GitHub classic PAT scope mapping table (abstract → OAuth scope names) — `internal/scope/resolver.go`
- [X] T012 [P2] [US3] Implement GitLab scope mapping table (abstract → GitLab token scope names) — `internal/scope/resolver.go`
- [X] T013 [P2] [US3] Implement Gitea scope mapping table (abstract → Gitea permission names) — `internal/scope/resolver.go`
- [X] T014 [P2] [US3] Implement `Resolve(scope TokenScope) ([]string, error)` dispatching to platform-specific map; return error for unknown/Bitbucket forge (FR-007) — `internal/scope/resolver.go`
- [X] T015 [P2] [US3] [P] Write table-driven tests for `Resolve()`: each forge type × each resource × each permission level, unknown forge error, Bitbucket warning — `internal/scope/resolver_test.go`

## Phase 4: Token Introspection (US2 prerequisite)

- [X] T016 [P1] [US2] Define `TokenIntrospector` interface and `TokenInfo` struct — `internal/scope/introspect.go`
- [X] T017 [P1] [US2] Implement `GitHubIntrospector` using `gh api user --include` to parse `X-OAuth-Scopes` header — `internal/scope/introspect.go`
- [X] T018 [P2] [US2] Implement `GitLabIntrospector` using `glab api /personal_access_tokens/self` to parse `scopes` array — `internal/scope/introspect.go`
- [X] T019 [P2] [US2] Implement `GiteaIntrospector` using Gitea API `/api/v1/user` with auth header — `internal/scope/introspect.go`
- [X] T020 [P1] [US2] Implement `NewIntrospector(forgeType forge.ForgeType) TokenIntrospector` factory function — `internal/scope/introspect.go`
- [X] T021 [P1] [US2] [P] Write tests for `GitHubIntrospector` with mock command execution: classic PAT headers, fine-grained PAT (no header), command failure — `internal/scope/introspect_test.go`
- [X] T022 [P2] [US2] [P] Write tests for `GitLabIntrospector` and `GiteaIntrospector` with mock command execution — `internal/scope/introspect_test.go`

## Phase 5: Scope Validator & Executor Integration (US2)

- [X] T023 [P1] [US2] Implement `ScopeViolation` and `ValidationResult` structs — `internal/scope/validator.go`
- [X] T024 [P1] [US2] Implement `Validator` struct with `NewValidator(resolver, introspector, forgeInfo, envPassthrough)` constructor — `internal/scope/validator.go`
- [X] T025 [P1] [US2] Implement `ValidatePersonas(personas map[string]manifest.Persona) (*ValidationResult, error)` orchestrating parse → resolve → introspect → compare with aggregated violations (FR-006) and remediation hints (FR-005) — `internal/scope/validator.go`
- [X] T026 [P1] [US2] Add `env_passthrough` check: report config error when required token env vars are not listed — `internal/scope/validator.go`
- [X] T027 [P1] [US2] Integrate scope validation into `executor.Execute()` after existing preflight checks (~line 329); emit `"preflight"` events for results — `internal/pipeline/executor.go`
- [X] T028 [P1] [US2] Write end-to-end validation tests: all scopes satisfied (pass), missing scopes (violations), no `token_scopes` (skip), unknown forge (warning), introspection failure (warning) — `internal/scope/validator_test.go`
- [X] T029 [P1] [US2] [P] Write integration test: multi-persona pipeline with mixed scope requirements, verifying aggregate violation reporting — `internal/scope/validator_test.go`

## Phase 6: Onboarding Integration (US4)

- [X] T030 [P3] [US4] Modify `buildManifest()` in onboarding to include `token_scopes` YAML comments showing recommended scopes per persona role — `internal/onboarding/onboarding.go`
- [X] T031 [P3] [US4] Add forge-specific token creation URL comments (e.g., GitHub fine-grained PAT settings URL) in generated manifest — `internal/onboarding/onboarding.go`
- [X] T032 [P3] [US4] [P] Write tests verifying `wave init` output includes scope recommendation comments for forge-interacting personas — `internal/onboarding/onboarding_test.go`

## Phase 7: Polish & Cross-Cutting

- [X] T033 [P] Run `go test -race ./...` to verify all tests pass with race detector — project root
- [X] T034 [P] Run `golangci-lint run ./...` and fix any findings — project root
- [X] T035 Verify backward compatibility: existing manifests without `token_scopes` load and execute without errors (SC-002) — `internal/manifest/parser_test.go`
- [X] T036 Verify deny lists remain enforced alongside token scoping (SC-006, FR-009) — `internal/pipeline/executor_test.go`
