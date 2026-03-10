# Implementation Plan: Documentation Consistency Fixes

## Objective

Fix 17 remaining documentation inconsistencies identified in issue #264, ensuring all docs accurately reflect the current codebase state (47 pipelines, 30 personas, 10 event states, 22+ event fields).

## Approach

Work through items by file grouping to minimize context switching. Each file is edited individually, cross-referencing the actual source code (wave.yaml, Go source, pipeline YAML files) for accuracy.

**Priority order**: Critical > High > Medium > Low. Within each severity, group by target file.

## File Mapping

### Files to Modify

| File | DOC Items | Action |
|------|-----------|--------|
| `docs/guide/pipelines.md` | DOC-002, DOC-004, DOC-008 | Update pipeline counts, remove non-existent pipelines, fix step counts, add missing pipeline families |
| `docs/reference/contract-types.md` | DOC-005, DOC-017 | Remove undocumented `template` type or mark as unimplemented, document advanced fields |
| `docs/guide/personas.md` | DOC-007, DOC-022 | Update persona count, fix permission tables, note temperature as optional |
| `docs/concepts/personas.md` | DOC-007, DOC-025 | Update persona count, fix permission examples, add cross-reference |
| `docs/reference/events.md` | DOC-010, DOC-011 | Add 13+ missing event fields, add 5 missing event states |
| `docs/reference/cli.md` | DOC-014, DOC-016 | Add --reconfigure, --all to init; add --format to cancel |
| `docs/reference/environment.md` | DOC-018, DOC-019 | Fix NERD_FONT/NO_UNICODE type descriptions, reconcile timeout default |
| `README.md` | DOC-002, DOC-020 | Update pipeline count, fix wave run --pipeline to wave run <pipeline> |
| `docs/quickstart.md` | DOC-024 | Already recommends hello-world (no change needed) |
| `docs/guide/quick-start.md` | DOC-024 | Change recommended first pipeline to hello-world for consistency |
| `docs/guide/contracts.md` | DOC-017 | Add advanced contract fields if missing |

### Files to Potentially Create

| File | DOC Item | Action |
|------|----------|--------|
| `docs/guide/tui.md` | DOC-021 | Add TUI overview documentation |

### Source Files to Cross-Reference (Read Only)

| File | Purpose |
|------|---------|
| `wave.yaml` | Persona definitions, permissions, temperature values |
| `internal/event/emitter.go` | Event struct fields and state constants |
| `cmd/wave/commands/init.go` | --reconfigure, --all flags |
| `cmd/wave/commands/cancel.go` | --format flag |
| `internal/contract/` | Available contract types |
| `.wave/pipelines/*.yaml` | Actual pipeline inventory |
| `internal/display/terminal.go` | NO_UNICODE behavior |
| `internal/deliverable/types.go` | NERD_FONT behavior |

## Architecture Decisions

1. **DOC-004**: Remove references to non-existent pipelines rather than creating placeholder YAML files
2. **DOC-005**: Remove `template` contract type from docs — no implementation exists. Keep `format` which has implementation
3. **DOC-020**: Keep both positional and flag syntax documented but lead with positional (the recommended style)
4. **DOC-022**: Keep temperature values in doc examples but note they are optional — wave.yaml has them commented out intentionally
5. **DOC-024**: Standardize on `hello-world` as the first recommended pipeline across all quickstart guides
6. **DOC-021**: Create a brief TUI guide rather than comprehensive docs — TUI is still evolving

## Risks

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Inaccurate pipeline step counts | Medium | Cross-reference each pipeline YAML file |
| Permission table errors | Medium | Diff each persona's docs vs wave.yaml systematically |
| Event field descriptions wrong | Low | Read Go struct tags and emitter code for accurate descriptions |
| Missing new pipelines by time of merge | Low | Document pipeline count as approximate or reference wave list pipelines |

## Testing Strategy

1. **Manual verification**: Each doc change verified against source of truth (wave.yaml, Go source, pipeline files)
2. **Link checking**: Ensure no broken cross-references introduced
3. **Build validation**: `go test ./...` to ensure no code changes break tests (docs-only)
4. **Consistency check**: Verify counts mentioned in multiple files are synchronized
