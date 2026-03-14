# Tasks: Ecosystem Adapters for Skill Sources

**Feature**: #383 — Skill source adapters with prefix routing
**Branch**: `383-skill-source-adapters`
**Date**: 2026-03-14
**Spec**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md) | **Data Model**: [data-model.md](data-model.md)

## Phase 1: Setup — Test Infrastructure

- [X] T001 P1 US6 Create mock store and exec helpers for adapter testing in `internal/skill/source_test.go`
  - Implement `memoryStore` struct satisfying `Store` interface with in-memory map storage
  - Track `Write()` calls for assertion (skill names written, call count)
  - Implement `Read()`, `List()`, `Delete()` backed by the map
  - Create helper `makeTestSkillDir(t, dir, name, description)` that writes a valid SKILL.md file into `dir/name/SKILL.md`
  - This is the shared test infrastructure used by all adapter test files

## Phase 2: Foundational — Core Types & Router (US6: Source Prefix Routing)

- [X] T002 P1 US6 Define core types in `internal/skill/source.go`: `SourceAdapter` interface, `InstallResult`, `SourceReference`, `DependencyError`, `CLIDependency`
  - `SourceAdapter` interface: `Install(ctx context.Context, ref string, store Store) (*InstallResult, error)` and `Prefix() string`
  - `InstallResult` struct: `Skills []Skill`, `Warnings []string`
  - `SourceReference` struct: `Prefix string`, `Reference string`, `Raw string`
  - `DependencyError` struct: `Binary string`, `Instructions string` with `Error() string` method
  - `CLIDependency` struct: `Binary string`, `Instructions string`
  - Add timeout constants: `CLITimeout = 2 * time.Minute`, `HTTPTimeout = 2 * time.Minute`, `HTTPHeaderTimeout = 30 * time.Second`

- [X] T003 P1 US6 Implement `SourceRouter` in `internal/skill/source.go`
  - `SourceRouter` struct with `adapters map[string]SourceAdapter`
  - `NewSourceRouter(adapters ...SourceAdapter) *SourceRouter` — registers all provided adapters
  - `Register(adapter SourceAdapter)` — adds adapter keyed by `adapter.Prefix()`
  - `Parse(source string) (SourceAdapter, string, error)` — URL-scheme-first parsing:
    1. Check `strings.HasPrefix(source, "https://")` or `strings.HasPrefix(source, "http://")` → return URL adapter with full URL as reference
    2. Check for `:` in source → split on first `:`, lookup prefix in map
    3. No colon → return error (bare name, not an adapter invocation) per FR-014
    4. Unknown prefix → return error listing all recognized prefixes per FR-015
  - `Install(ctx context.Context, source string, store Store) (*InstallResult, error)` — parse then delegate
  - `Prefixes() []string` — sorted list of registered prefixes

- [X] T004 P1 US6 Router unit tests in `internal/skill/source_test.go`
  - Table-driven `TestSourceRouterParse`: parse `tessl:github/spec-kit`, `github:owner/repo`, `file:./local`, `bmad:install`, `openspec:init`, `speckit:init`, `https://example.com/skill.tar.gz`
  - Verify correct adapter selected and reference extracted for each prefix
  - Test unknown prefix `foobar:something` → error listing recognized prefixes
  - Test bare name `golang` (no colon) → error indicating local store lookup
  - Test empty string → error
  - Test `http://` prefix → same URL adapter as `https://`
  - Test `Install()` delegates correctly (use a stub adapter that records calls)

## Phase 3: US1 — Tessl Adapter (P1)

- [X] T005 P1 US1 Implement shared CLI adapter helpers in `internal/skill/source_cli.go`
  - Define `lookPathFunc` type alias `func(string) (string, error)` for testability (defaults to `exec.LookPath`)
  - Implement `checkDependency(dep CLIDependency, lookPath lookPathFunc) error` — returns `*DependencyError` if binary not found
  - Implement `discoverSkillFiles(dir string) ([]string, error)` — walks directory tree finding all `SKILL.md` files, returns absolute paths
  - Implement `parseAndWriteSkills(ctx context.Context, paths []string, store Store) (*InstallResult, error)` — reads each SKILL.md, calls `Parse()`, writes to store, collects results and warnings

- [X] T006 P1 US1 Implement `TesslAdapter` in `internal/skill/source_cli.go`
  - Struct: `TesslAdapter` with `dep CLIDependency{Binary: "tessl", Instructions: "npm i -g @tessl/cli"}` and `lookPath lookPathFunc`
  - `Prefix() string` returns `"tessl"`
  - `Install(ctx, ref, store)`:
    1. `checkDependency(a.dep, a.lookPath)` → return `*DependencyError` if missing
    2. `os.MkdirTemp("", "wave-skill-tessl-*")` + `defer os.RemoveAll`
    3. `exec.CommandContext(ctx, "tessl", "install", ref)` with `Dir` set to temp dir
    4. Capture stdout+stderr, return error on non-zero exit (include stderr in error message)
    5. `discoverSkillFiles(tmpDir)` → `parseAndWriteSkills(ctx, paths, store)`
  - Constructor: `NewTesslAdapter() *TesslAdapter` (sets `lookPath` to `exec.LookPath`)

- [X] T007 P1 US1 TesslAdapter tests in `internal/skill/source_cli_test.go`
  - Test happy path: mock `lookPath` to succeed, pre-populate temp dir with valid SKILL.md, verify store receives skill
  - Test missing dependency: mock `lookPath` to return error, verify `*DependencyError` with correct binary and instructions
  - Test CLI failure (non-zero exit): verify error includes stderr output
  - Test no SKILL.md found after CLI runs: verify error about no skills discovered
  - Test CLI timeout: use `context.WithTimeout` with very short deadline, verify context deadline exceeded error

## Phase 4: US2 — GitHub Adapter (P1)

- [X] T008 [P] P1 US2 Implement `GitHubAdapter` in `internal/skill/source_github.go`
  - Struct: `GitHubAdapter` with `dep CLIDependency{Binary: "git", Instructions: "install git from https://git-scm.com"}` and `lookPath lookPathFunc`
  - `Prefix() string` returns `"github"`
  - `parseGitHubRef(ref string) (owner, repo, subpath string, err error)` — splits `owner/repo[/path/to/skill]`:
    - Split on `/`, require at least 2 components (owner, repo)
    - Components 3+ form the optional subpath
    - Return error if fewer than 2 components
  - `Install(ctx, ref, store)`:
    1. `checkDependency` for git
    2. Parse reference: `parseGitHubRef(ref)`
    3. `os.MkdirTemp("", "wave-skill-github-*")` + `defer os.RemoveAll`
    4. `git clone --depth 1 https://github.com/<owner>/<repo>.git <tmpDir>/repo`
    5. If subpath: navigate to `<tmpDir>/repo/<subpath>`, verify SKILL.md exists, parse and write
    6. If no subpath: check root for SKILL.md; if found, parse and write single skill
    7. If no root SKILL.md: `discoverSkillFiles(<tmpDir>/repo)` for multi-skill repos
    8. Return `InstallResult` with all discovered skills
  - Constructor: `NewGitHubAdapter() *GitHubAdapter`

- [X] T009 [P] P1 US2 GitHubAdapter tests in `internal/skill/source_github_test.go`
  - Test `parseGitHubRef`: `owner/repo` → owner+repo+empty subpath; `owner/repo/sub/path` → owner+repo+`sub/path`; `single` → error
  - Test happy path with root SKILL.md: create temp dir with SKILL.md at root, mock git clone (test with pre-populated dir), verify store
  - Test happy path with subpath: SKILL.md in subdirectory, verify correct navigation
  - Test multi-skill discovery: no root SKILL.md, multiple subdirectories each with SKILL.md
  - Test missing git dependency: mock `lookPath` error → `*DependencyError`
  - Test invalid reference (single component): verify error message

## Phase 5: US3 — File Adapter (P2)

- [X] T010 [P] P2 US3 Implement `FileAdapter` in `internal/skill/source_file.go`
  - Struct: `FileAdapter` with `projectRoot string`
  - `Prefix() string` returns `"file"`
  - `Install(ctx, ref, store)`:
    1. Resolve reference path: if starts with `/`, use as-is; otherwise resolve relative to `projectRoot`
    2. `filepath.Abs()` and `filepath.Clean()` the resolved path
    3. Path containment check: verify resolved path is within `projectRoot` (use `strings.HasPrefix` after `filepath.EvalSymlinks` on both paths, matching `containedPath` pattern from `store.go`)
    4. Check for symlinks via `os.Lstat` — reject if `Mode()&os.ModeSymlink != 0`
    5. Verify directory exists and contains `SKILL.md`
    6. Read and `Parse()` the SKILL.md
    7. Write to store via `store.Write(skill)`
    8. Return `InstallResult` with the installed skill
  - Constructor: `NewFileAdapter(projectRoot string) *FileAdapter`

- [X] T011 [P] P2 US3 FileAdapter tests in `internal/skill/source_file_test.go`
  - Test relative path (`./my-skills/custom-skill`): create skill dir relative to project root, verify install
  - Test absolute path: create skill dir with absolute path, verify install
  - Test path not found: reference non-existent directory → error with resolved path
  - Test symlink rejection: create symlink to valid skill dir → error with "symlink rejected" or containment message
  - Test path traversal (`../../etc/passwd`): verify containment error
  - Test no SKILL.md in directory: directory exists but no SKILL.md → error
  - Test SKILL.md with invalid content: verify parse error propagated

## Phase 6: US4 — Ecosystem CLI Adapters (P2)

- [X] T012 [P] P2 US4 Implement `BMADAdapter`, `OpenSpecAdapter`, `SpecKitAdapter` in `internal/skill/source_cli.go`
  - **BMADAdapter**: `dep: {Binary: "npx", Instructions: "npm i -g npx (comes with npm)"}`, prefix `"bmad"`, command: `npx bmad-method install --tools claude-code --yes`
  - **OpenSpecAdapter**: `dep: {Binary: "openspec", Instructions: "npm i -g @openspec/cli"}`, prefix `"openspec"`, command: `openspec init`
  - **SpecKitAdapter**: `dep: {Binary: "specify", Instructions: "npm i -g @speckit/cli"}`, prefix `"speckit"`, command: `specify init`
  - Each follows the same pattern as TesslAdapter: check dep → temp dir → run CLI → discover → parse → write
  - Constructors: `NewBMADAdapter()`, `NewOpenSpecAdapter()`, `NewSpecKitAdapter()`

- [X] T013 [P] P2 US4 Ecosystem CLI adapter tests in `internal/skill/source_cli_test.go`
  - Test each adapter's `Prefix()` returns correct string
  - Test missing dependency per adapter: `npx` for BMAD, `openspec` for OpenSpec, `specify` for SpecKit — verify `*DependencyError` with correct binary and install instructions
  - Test happy path per adapter (at minimum one): pre-populate temp dir, verify CLI args are correct

## Phase 7: US5 — URL/Archive Adapter (P3)

- [X] T014 [P] P3 US5 Implement `URLAdapter` HTTP client and archive detection in `internal/skill/source_url.go`
  - Struct: `URLAdapter` with `client *http.Client` configured with `Transport: &http.Transport{ResponseHeaderTimeout: HTTPHeaderTimeout}`
  - `Prefix() string` returns `"https://"`
  - `Install(ctx, ref, store)`:
    1. Validate URL format (must start with `https://` or `http://`)
    2. Create timeout context: `context.WithTimeout(ctx, HTTPTimeout)`
    3. HTTP GET with request context
    4. Detect archive format by URL extension: `.tar.gz` / `.tgz` → tar+gzip, `.zip` → zip, else → error
    5. `os.MkdirTemp("", "wave-skill-url-*")` + `defer os.RemoveAll`
    6. Extract archive to temp dir using appropriate extractor
    7. `discoverSkillFiles(tmpDir)` → `parseAndWriteSkills(ctx, paths, store)`
  - Constructor: `NewURLAdapter() *URLAdapter`

- [X] T015 P3 US5 Implement tar.gz and zip extraction helpers in `internal/skill/source_url.go`
  - `extractTarGz(r io.Reader, destDir string) error` — uses `compress/gzip` + `archive/tar`, validates filenames (no path traversal via `..`), limits extracted file count/size for safety
  - `extractZip(data []byte, destDir string) error` — uses `archive/zip` from bytes (need full body for zip), validates filenames, limits extraction
  - Both reject filenames with `..` components to prevent zip-slip attacks
  - Both create directories as needed via `os.MkdirAll`

- [X] T016 [P] P3 US5 URLAdapter tests in `internal/skill/source_url_test.go`
  - Test tar.gz happy path: use `httptest.NewServer` serving a tar.gz containing a valid SKILL.md, verify store receives skill
  - Test zip happy path: serve a zip archive, verify installation
  - Test non-archive response (HTML content): verify error about unrecognized archive format
  - Test HTTP error (404, 500): verify descriptive error
  - Test unreachable URL: verify network error (use non-routable address or closed server)
  - Test zip-slip attack (archive with `../` paths): verify rejection
  - Helper: `createTestTarGz(t, skillName, description)` → `[]byte`
  - Helper: `createTestZip(t, skillName, description)` → `[]byte`

## Phase 8: Integration & Polish

- [X] T017 P1 Implement `NewDefaultRouter()` in `internal/skill/source.go`
  - Creates a `SourceRouter` pre-registered with all 7 adapters: Tessl, BMAD, OpenSpec, SpecKit, GitHub, File, URL
  - Accepts `projectRoot string` parameter (needed by FileAdapter)
  - Returns `*SourceRouter` ready for use
  - Add test in `source_test.go` verifying all 7 prefixes are registered

- [X] T018 P1 Full test verification pass across `internal/skill/`
  - Run `go test -race ./internal/skill/...` — all tests must pass with zero data races (SC-006)
  - Verify no regressions in existing `store_test.go` and `skill_test.go`
  - Verify all adapters return proper error types (`*DependencyError` for missing deps, descriptive errors for failures)
  - Verify all error messages include recognized prefix list for unknown prefix errors (SC-005)

## Dependency Graph

```
T001 ─┐
      ├─► T002 ─► T003 ─► T004
      │
T002 ─┼─► T005 ─► T006 ─► T007
      │         └─► T012 ─► T013
      ├─► T008 ─► T009
      ├─► T010 ─► T011
      └─► T014 ─► T015 ─► T016
                              │
T004, T007, T009, T011, T013, T016 ─► T017 ─► T018
```

## Parallelization Summary

After Phase 2 (T002–T004), the following can proceed in parallel:
- **US1** (T005–T007): CLI helpers + Tessl adapter
- **US2** (T008–T009): GitHub adapter
- **US3** (T010–T011): File adapter
- **US5** (T014–T016): URL adapter

US4 (T012–T013) can start after T005 completes (depends on shared CLI helpers).

**Total parallel opportunities**: 8 tasks marked [P] can execute concurrently with other tasks in their phase.
