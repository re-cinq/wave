# Tasks

## Phase 1: CLI Reference Fixes (`docs/reference/cli.md`)

- [X] Task 1.1: Add `wave serve` documentation section to CLI reference [P]
  - Add a new `## wave serve` section after `## wave clean` and before `## wave migrate`
  - Include: command description, usage examples, output block, options table
  - Document `--port`, `--bind`, `--token`, `--db`, `--manifest` flags
  - Note the `webui` build tag requirement
  - Add `wave serve` to the Quick Reference table at the top of the file

- [X] Task 1.2: Fix `--status` to `--run-status` for `wave list runs` [P]
  - Line 455: Change `wave list runs --status failed` to `wave list runs --run-status failed`

## Phase 2: Environment Reference Fix (`docs/reference/environment.md`)

- [X] Task 2.1: Document `WAVE_SERVE_TOKEN` environment variable [P]
  - Add `WAVE_SERVE_TOKEN` to the Wave Environment Variables table
  - Type: `string`, Default: `""`, Description: Authentication token for `wave serve` dashboard; auto-generated when binding to non-localhost addresses

## Phase 3: Persona Count Fix (`docs/concepts/personas.md`)

- [X] Task 3.1: Update persona count and description [P]
  - Change "four core personas" (line 26) to reflect that Wave includes 14 built-in personas
  - Clarify that the page highlights a representative subset of the full persona library
  - Keep the existing 4-persona table as the "core" set but frame it correctly

## Phase 4: README Pipeline Fixes (`README.md`)

- [X] Task 4.1: Fix stale pipeline names in README [P]
  - Line 324: Change `docs` to `doc-sync` in the Planning & Documentation table
  - Line 325: Remove the `migrate` row from the table (no such pipeline exists)
  - Line 334: Update the "More pipelines" list — remove `docs-to-impl`, `gh-poor-issues`, `umami`, and `issue-research`; keep `hello-world` and `smoke-test`; optionally add real pipelines like `explain`, `onboard`, `improve`, `dead-code`, `security-scan`

## Phase 5: Validation

- [X] Task 5.1: Cross-reference verification
  - Verify persona count matches `ls .wave/personas/ | wc -l` (expect 14)
  - Verify `--run-status` matches `cmd/wave/commands/list.go`
  - Verify all pipeline names in README exist in `.wave/pipelines/`
  - Verify `WAVE_SERVE_TOKEN` usage matches `cmd/wave/commands/serve.go`
  - Run `go build ./...` to confirm no code changes
