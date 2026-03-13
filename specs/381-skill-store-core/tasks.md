# Tasks: Skill Store Core

**Feature**: #381 — Skill Store Core
**Branch**: `381-skill-store-core`
**Date**: 2026-03-13
**Source**: [spec.md](spec.md), [plan.md](plan.md), [data-model.md](data-model.md)

## Phase 1: Setup

_No setup tasks required. The `internal/skill/` package already exists with `types.go`, `skill.go`, and `skill_test.go`. New files are added alongside existing ones without modification._

## Phase 2: Foundational Types (blocking prerequisites)

- [X] T001 P1 US1 Create `internal/skill/parse.go` with the `Skill` struct (Name, Description, Body, License, Compatibility, Metadata, AllowedTools, SourcePath, ResourcePaths fields per data-model.md), the unexported `frontmatter` struct (maps YAML field names including `allowed-tools` as string), the `ValidateName` function (regex `^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`, max 64 chars, returns `*ParseError`), and the `splitFrontmatter` helper (splits on `---` delimiters, returns yaml block + body + error). Compile with `go build ./internal/skill/`.

- [X] T002 P1 US1 Create `internal/skill/store.go` with error types (`ParseError` with Field/Constraint/Value + Error()/Unwrap(), `DiscoveryError` with Errors []SkillError + Error(), `SkillError` with SkillName/Path/Err + Error()/Unwrap()), the `Store` interface (Read/Write/List/Delete per contracts/store-interface.go), the `SkillSource` struct (Root string, Precedence int), the `DirectoryStore` struct (sources []SkillSource), and `NewDirectoryStore(sources ...SkillSource) *DirectoryStore` that sorts sources by precedence descending. All CRUD methods can be stubs returning `nil`/zero values initially. Compile with `go build ./internal/skill/`.

## Phase 3: Parser — User Story 1 (P1: Parse SKILL.md Files)

- [X] T003 P1 US1 Implement `Parse(data []byte) (Skill, error)` in `internal/skill/parse.go`. Algorithm: call `splitFrontmatter` → unmarshal YAML into `frontmatter` struct → call `validateFrontmatter` (check required name/description, name regex via `ValidateName`, description non-empty + max 1024 chars, compatibility max 500 chars if present) → split `allowed-tools` string on whitespace into `[]string` → map `frontmatter` to `Skill` with body. Return `*ParseError` for validation failures.

- [X] T004 P1 US1 Implement `ParseMetadata(data []byte) (Skill, error)` in `internal/skill/parse.go`. Same as `Parse` but sets `Body` to empty string. Reuse `splitFrontmatter` and `validateFrontmatter` internally — only difference is discarding the body portion.

- [X] T005 P1 US1 Implement `Serialize(skill Skill) ([]byte, error)` in `internal/skill/parse.go`. Validate skill (name + description) before serializing. Marshal the `frontmatter` struct to YAML (reconstruct `allowed-tools` from AllowedTools slice by joining with space), wrap with `---\n` delimiters, append body. Result must round-trip: `Parse(Serialize(s))` produces semantically equivalent Skill.

- [X] T006 P1 US1 Write parser unit tests in `internal/skill/store_test.go`. Table-driven tests covering: (a) `TestParse` — valid SKILL.md with all fields, valid with only required fields, missing name, missing description, no frontmatter delimiters, invalid name (uppercase, dots, path traversal), empty file, malformed YAML, empty body, compatibility > 500 chars, allowed-tools parsing (single, multiple, empty), metadata map. (b) `TestParseMetadata` — same validation as Parse, body field empty. (c) `TestValidateName` — valid names (golang, my-skill, a, a-b-c, 64-char), invalid (uppercase, dots, underscores, path traversal `../etc`, `foo/bar`, 65 chars, empty). (d) `TestSerialize` — round-trip fidelity, validation before serialization. Run with `go test ./internal/skill/...`.

## Phase 4: Store CRUD (User Stories 2-7)

- [X] T007 [P] P2 US4 Implement `DirectoryStore.Read(name string) (Skill, error)` in `internal/skill/store.go`. Call `ValidateName` first. Iterate sources in precedence order, build path `filepath.Join(source.Root, name, "SKILL.md")`, read file if exists, call `Parse`, set `SourcePath` to the skill directory, call `discoverResources` for ResourcePaths, validate frontmatter `name` matches directory name (FR-004). Return first match. Return `*ParseError{Field:"name", Constraint:"not found"}` if no source has the skill.

- [X] T008 [P] P2 US2,US3,US7 Implement `DirectoryStore.List() ([]Skill, error)` in `internal/skill/store.go`. Iterate sources in precedence order. For each source: skip if root doesn't exist (`os.Stat`), read directory entries, for each subdir check for `SKILL.md`, call `ParseMetadata`, track `seen` map for name dedup (first-name-wins = highest precedence), collect `SkillError` entries for parse failures. Return `(skills, &DiscoveryError{...})` if any parse errors, else `(skills, nil)`.

- [X] T009 [P] P5 US5 Implement `DirectoryStore.Write(skill Skill) error` in `internal/skill/store.go`. Validate name (via `ValidateName`) and description (non-empty). Call `Serialize(skill)`. Write to `filepath.Join(sources[0].Root, skill.Name, "SKILL.md")` — create directory with `os.MkdirAll(dir, 0755)`, write file with `os.WriteFile(path, data, 0644)`. Return `*ParseError` for validation failures.

- [X] T010 [P] P7 US6 Implement `DirectoryStore.Delete(name string) error` in `internal/skill/store.go`. Call `ValidateName` first. Iterate sources in precedence order, build path `filepath.Join(source.Root, name)`, check if directory exists, call `os.RemoveAll(dir)` on first match, return nil. Return `*ParseError{Field:"name", Constraint:"not found"}` if no source contains the skill.

- [X] T011 P2 US4 Implement `discoverResources(skillDir string) []string` helper in `internal/skill/store.go`. Scan for files in `scripts/`, `references/`, `assets/` subdirectories of the skill directory. Return slice of relative paths (relative to skillDir). Skip subdirectories that don't exist. Used by `Read`.

- [X] T012 P2 US2,US3,US4,US5,US6,US7 Write store CRUD and multi-source tests in `internal/skill/store_test.go`. Table-driven tests covering: (a) `TestDirectoryStoreRead` — existing skill returns full Skill, non-existent returns not-found, with resource files populates ResourcePaths, path traversal rejected, name/dir mismatch detected. (b) `TestDirectoryStoreList` — multiple valid skills, mix valid+invalid returns DiscoveryError, empty dir returns empty list, non-existent source skipped silently. (c) `TestDirectoryStoreWrite` — valid skill creates dir+file, overwrite existing, invalid name rejected, empty description rejected, path traversal rejected. (d) `TestDirectoryStoreDelete` — existing skill removed, non-existent returns not-found, path traversal rejected. (e) `TestMultiSourceResolution` — higher precedence shadows lower for Read, skill only in lower source returned, List merges with dedup, Write goes to first source. Run with `go test ./internal/skill/...`.

## Phase 5: Integration & Polish

- [X] T013 P1 SC-001 Write integration test `TestParseExistingSkills` in `internal/skill/store_test.go`. Glob `.claude/skills/*/SKILL.md` from the repository root, parse each file with `Parse`, verify all 13 existing skills parse successfully (fail the test if any do not). Verify each returns non-empty Name and Description.

- [X] T014 P2 SC-002,SC-008 Write `TestSerializeRoundTrip` — parse each of the 13 existing SKILL.md files, serialize, re-parse, verify semantic equivalence (Name, Description, License, Compatibility, Metadata, AllowedTools match). Write `TestListPerformance` — create a temp directory with 50+ skill subdirectories, call `List`, verify completes within 100ms. Add to `internal/skill/store_test.go`.

- [X] T015 P1 SC-005,SC-006 Run `go test -race ./internal/skill/...` and verify all new tests pass. Run `go test -race ./...` to confirm no existing tests were broken (SC-006: legacy provisioning tests pass unchanged). Fix any failures.
