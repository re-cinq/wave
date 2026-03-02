# Tasks

## Phase 1: Core Forge Detection Package

- [X] Task 1.1: Create `internal/forge/forge.go` — define `ForgeType` string type with constants (`GitHub`, `GitLab`, `Bitbucket`, `Gitea`, `Unknown`), `ForgeDetection` result struct (type, remote URL, hostname, CLI tool name), and `ForgeConfig` for domain mappings
- [X] Task 1.2: Create `internal/forge/detect.go` — implement `Detect(cfg *ForgeConfig)` function that runs `git remote -v`, parses SSH/HTTPS URLs, extracts hostnames, and classifies against built-in defaults + user-configured domains
- [X] Task 1.3: Create `internal/forge/detect_test.go` — table-driven tests for URL parsing (SSH, HTTPS, enterprise, custom ports), hostname classification, custom domain overrides, multi-remote detection, and edge cases (no remotes, malformed URLs) [P]

## Phase 2: Pipeline Filtering

- [X] Task 2.1: Create `internal/forge/filter.go` — implement `FilterPipelines(forgeType ForgeType, pipelineNames []string) []string` that returns only pipelines with the correct forge prefix plus all non-prefixed (universal) pipelines [P]
- [X] Task 2.2: Create `internal/forge/filter_test.go` — table-driven tests for prefix matching, universal pipeline passthrough, unknown forge type behavior, and empty inputs [P]

## Phase 3: Manifest Integration

- [X] Task 3.1: Add `Forge *ForgeConfig` field to `Manifest` struct in `internal/manifest/types.go` — the `ForgeConfig` type includes `Domains map[string]string` for custom hostname-to-forge mappings
- [X] Task 3.2: Update `internal/manifest/parser_test.go` to verify forge config round-trips through YAML parsing correctly

## Phase 4: CLI and TUI Integration

- [X] Task 4.1: Modify `cmd/wave/commands/run.go` — call `forge.Detect()` early in `runRun`, pass detected forge type to the pipeline selector and to preflight as context
- [X] Task 4.2: Modify `internal/tui/run_selector.go` — accept an optional `ForgeType` parameter in `RunPipelineSelector` to pre-filter the pipeline list before display; non-prefixed pipelines always shown
- [X] Task 4.3: Modify `cmd/wave/commands/list.go` — when listing pipelines, optionally annotate them with detected forge compatibility

## Phase 5: Testing and Validation

- [X] Task 5.1: Write integration test that verifies forge detection end-to-end with mock git remotes and pipeline directory
- [X] Task 5.2: Verify `go test -race ./...` passes with all new and existing tests
- [X] Task 5.3: Verify `go vet ./...` reports no issues

## Phase 6: Documentation

- [X] Task 6.1: Add `wave.yaml` forge configuration example to existing documentation or inline comments
- [X] Task 6.2: Update CLAUDE.md if any new packages or patterns need to be documented
