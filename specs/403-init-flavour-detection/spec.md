# feat(init): cold-start fix, flavour auto-detection, and smart init

**Issue**: [#403](https://github.com/re-cinq/wave/issues/403)
**Parent**: [#402](https://github.com/re-cinq/wave/issues/402)
**Labels**: enhancement
**Author**: nextlevelshit

## Summary

Overhaul `wave init` to support cold-start (greenfield) projects and auto-detect project language/flavour for customized onboarding.

## Phase 1: Cold-Start Fix

### Problem
- `wave init` does NOT run `git init` — worktree manager crashes
- Worktrees need at least 1 commit — second crash
- Forge detection needs remote — third crash

### Solution
In `cmd/wave/commands/init.go`:
1. Check if `.git` exists, if not: `git init`
2. After writing Wave files: create initial commit
3. Detect remote from existing `.git/config` OR prompt in wizard
4. Ensure ALL release pipelines are included (bug: `impl-issue` was missing in aide)

## Phase 2: Flavour System

### New manifest field
```yaml
project:
  flavour: rust
  language: rust
  test_command: cargo test
  lint_command: cargo clippy -- -D warnings
  build_command: cargo build
  format_command: cargo fmt -- --check
  source_glob: "*.rs"
```

### Auto-Detection Matrix (comprehensive)

| Marker File(s) | Flavour | Test | Lint | Build | Format |
|----------------|---------|------|------|-------|--------|
| `go.mod` | go | `go test ./...` | `go vet ./...` | `go build ./...` | `gofmt -l .` |
| `Cargo.toml` | rust | `cargo test` | `cargo clippy -- -D warnings` | `cargo build` | `cargo fmt -- --check` |
| `package.json` + npm | node | `npm test` | `npm run lint` | `npm run build` | `npm run format` |
| `package.json` + `yarn.lock` | node-yarn | `yarn test` | `yarn lint` | `yarn build` | `yarn format` |
| `package.json` + `pnpm-lock.yaml` | node-pnpm | `pnpm test` | `pnpm lint` | `pnpm build` | `pnpm format` |
| `package.json` + `bun.lock` / `bun.lockb` | bun | `bun test` | `bun lint` | `bun run build` | `bun format` |
| `deno.json` / `deno.jsonc` | deno | `deno test` | `deno lint` | `deno compile` | `deno fmt --check` |
| `pyproject.toml` | python | `pytest` | `ruff check .` | — | `ruff format --check .` |
| `setup.py` / `requirements.txt` | python-legacy | `python -m pytest` | `flake8` | `python setup.py build` | `black --check .` |
| `*.csproj` / `*.sln` | csharp | `dotnet test` | `dotnet format --verify-no-changes` | `dotnet build` | `dotnet format` |
| `pom.xml` | java-maven | `mvn test` | — | `mvn package` | — |
| `build.gradle` / `build.gradle.kts` | java-gradle | `gradle test` | — | `gradle build` | — |
| `build.gradle.kts` + `*.kt` | kotlin | `gradle test` | `gradle detekt` | `gradle build` | `gradle ktlintCheck` |
| `mix.exs` | elixir | `mix test` | `mix credo` | `mix compile` | `mix format --check-formatted` |
| `pubspec.yaml` | dart | `dart test` | `dart analyze` | `dart compile exe` | `dart format --set-exit-if-changed .` |
| `pubspec.yaml` + flutter | flutter | `flutter test` | `dart analyze` | `flutter build` | `dart format --set-exit-if-changed .` |
| `CMakeLists.txt` | cpp-cmake | `ctest` | — | `cmake --build .` | `clang-format -i` |
| `Makefile` (no other markers) | make | `make test` | `make lint` | `make build` | `make format` |
| `composer.json` | php | `vendor/bin/phpunit` | `vendor/bin/phpstan analyse` | — | `vendor/bin/php-cs-fixer fix --dry-run` |
| `Gemfile` | ruby | `bundle exec rspec` | `bundle exec rubocop` | — | `bundle exec rubocop -a` |
| `Package.swift` | swift | `swift test` | `swiftlint` | `swift build` | `swift format` |
| `build.zig` / `zig.zon` | zig | `zig build test` | — | `zig build` | `zig fmt` |
| `sbt` / `build.sbt` | scala | `sbt test` | — | `sbt compile` | `sbt scalafmtCheck` |
| `cabal.project` / `*.cabal` | haskell | `cabal test` | `hlint .` | `cabal build` | `ormolu --check` |
| `stack.yaml` | haskell-stack | `stack test` | `hlint .` | `stack build` | `ormolu --check` |
| `tsconfig.json` (no package.json) | typescript-standalone | `tsc --noEmit` | `eslint .` | `tsc` | `prettier --check .` |

### Detection Priority
Check markers top-to-bottom, first match wins. More specific markers (e.g., `bun.lockb`) checked before generic ones (`package.json`).

## Phase 4: Smart Init

1. **Pipeline selection based on flavour**: Only include pipelines relevant to detected language
2. **Forge-specific personas only**: GitHub -> only `github-*` personas, not gitlab/bitbucket/gitea
3. **Project metadata extraction**: Read name/description from `Cargo.toml`, `go.mod`, `package.json` etc.
4. **First-run suggestion**: Empty project -> suggest `ops-bootstrap`; Has code -> suggest `audit-dx`

## Files to Modify
- `cmd/wave/commands/init.go` — cold-start logic, flavour detection, smart selection
- `internal/onboarding/steps.go` — expanded language detection
- `internal/manifest/types.go` — `Flavour`, `FormatCommand` fields
- `internal/defaults/embed.go` — forge-filtered persona loading
- `internal/onboarding/onboarding.go` — wizard flow updates

## Acceptance Criteria
- [ ] `wave init` in empty dir: auto git init + initial commit
- [ ] Auto-detects all 25+ languages from matrix
- [ ] `wave.yaml` includes `project.flavour` and all command fields
- [ ] Only relevant forge personas included
- [ ] Project name/description extracted from language manifest
- [ ] `impl-issue` always included in pipeline set
