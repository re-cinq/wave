# Feature Specification: Release-Gated Pipeline Embedding

**Feature Branch**: `029-release-gated-embedding`
**Created**: 2026-02-11
**Status**: Draft
**Input**: https://github.com/re-cinq/wave/issues/46

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Release-only pipeline initialization (Priority: P1)

As a Wave user, when I run `wave init` to set up a new project, I only receive production-ready, validated pipelines — not experimental or internal development pipelines. This prevents confusion about which pipelines are stable and supported.

**Why this priority**: This is the core value proposition. Users running `wave init` currently receive all embedded pipelines, including experimental ones that may break or produce unexpected results. Filtering at init time is the primary use case driving this feature.

**Independent Test**: Can be fully tested by running `wave init` in an empty directory and verifying that only pipelines with `metadata.release: true` are extracted, along with their associated contracts and schemas.

**Acceptance Scenarios**:

1. **Given** a clean directory with no `.wave/` folder, **When** the user runs `wave init`, **Then** only pipelines with `metadata.release: true` are written to `.wave/pipelines/`.
2. **Given** a pipeline YAML file with `metadata.release: false`, **When** `wave init` is run, **Then** that pipeline is NOT written to `.wave/pipelines/`.
3. **Given** a pipeline YAML file with no `release` field in metadata, **When** `wave init` is run, **Then** that pipeline is NOT written to `.wave/pipelines/` (default is `false` — explicit opt-in).
4. **Given** a pipeline with `metadata.disabled: true` and `metadata.release: true`, **When** `wave init` is run, **Then** the pipeline IS written to `.wave/pipelines/` (release and disabled are independent concerns; disabled controls runtime execution, release controls distribution).

---

### User Story 2 - Transitive contract/schema exclusion (Priority: P1)

As a Wave user, when non-release pipelines are excluded during `wave init`, the contracts and schemas that are only referenced by those excluded pipelines are also excluded. This keeps the `.wave/contracts/` directory clean and relevant.

**Why this priority**: Shipping orphaned contracts alongside filtered pipelines would confuse users and bloat the initialized project. Transitive exclusion is essential for the feature to deliver a clean experience.

**Independent Test**: Can be tested by running `wave init` and verifying that contracts referenced only by non-release pipelines are absent from `.wave/contracts/`, while contracts referenced by at least one release pipeline are present.

**Acceptance Scenarios**:

1. **Given** a contract schema referenced only by a pipeline with `release: false`, **When** `wave init` is run, **Then** that contract schema is NOT written to `.wave/contracts/`.
2. **Given** a contract schema referenced by both a `release: true` pipeline and a `release: false` pipeline, **When** `wave init` is run, **Then** that contract schema IS written to `.wave/contracts/` (shared contracts are preserved).
3. **Given** a persona referenced by a non-release pipeline, **When** `wave init` is run, **Then** that persona IS still written to `.wave/personas/` (personas are never transitively excluded).
4. **Given** a prompt file referenced only by a non-release pipeline, **When** `wave init` is run, **Then** that prompt file is NOT written to `.wave/prompts/`.

---

### User Story 3 - Pipeline metadata field recognition (Priority: P1)

As a pipeline author, I can control whether my pipeline is included in releases by setting `metadata.release: true` in the pipeline YAML. The field is optional and defaults to `false`.

**Why this priority**: Without the metadata field being parsed and respected, neither filtering nor transitive exclusion can work. This is a foundational requirement.

**Independent Test**: Can be tested by creating pipeline YAML files with various `release` field values and confirming the system correctly reads and interprets them.

**Acceptance Scenarios**:

1. **Given** a pipeline with `metadata.release: true`, **When** the pipeline metadata is parsed, **Then** the release field reads as `true`.
2. **Given** a pipeline with `metadata.release: false`, **When** the pipeline metadata is parsed, **Then** the release field reads as `false`.
3. **Given** a pipeline with no `release` field in metadata, **When** the pipeline metadata is parsed, **Then** the release field defaults to `false`.
4. **Given** a pipeline with `metadata.release: "yes"` (invalid boolean), **When** the pipeline is parsed, **Then** a clear validation error is raised.

---

### User Story 4 - Include all pipelines for contributors (Priority: P2)

As a Wave contributor or developer, I can run `wave init --all` to receive all embedded pipelines regardless of their release status. This lets me test and develop experimental pipelines locally.

**Why this priority**: Contributors need access to non-release pipelines for development and testing. Without this flag, they would need to manually copy files from the source tree.

**Independent Test**: Can be tested by running `wave init --all` and verifying that all embedded pipelines (including those with `release: false` or no release field) are written to `.wave/pipelines/`, along with all contracts and schemas.

**Acceptance Scenarios**:

1. **Given** a clean directory, **When** the user runs `wave init --all`, **Then** all embedded pipelines are written to `.wave/pipelines/` regardless of their `release` field value.
2. **Given** a clean directory, **When** the user runs `wave init --all`, **Then** all embedded contracts and schemas are written to `.wave/contracts/`.
3. **Given** a clean directory, **When** the user runs `wave init` (without `--all`), **Then** only release-flagged pipelines and their contracts are written.

---

### User Story 5 - Build-time audit of release status (Priority: P3)

As a Wave maintainer, I can verify which pipelines are marked for release and which are excluded, so I can audit what ships to users before cutting a release.

**Why this priority**: Operational safety — maintainers need visibility into what the binary will ship. This is a secondary concern that builds on the core metadata parsing.

**Independent Test**: Can be tested by running the embedded defaults API and checking the counts and names of release vs non-release pipelines.

**Acceptance Scenarios**:

1. **Given** the embedded pipeline set, **When** a developer queries the defaults package for release pipelines, **Then** they receive only pipelines with `release: true`.
2. **Given** the embedded pipeline set, **When** a developer queries the defaults package for all pipelines, **Then** they receive the complete set regardless of release status.

---

### Edge Cases

- What happens when a pipeline has `release: true` but references a contract schema file that does not exist in `internal/defaults/contracts/`? The system should emit a warning during init but not fail — the pipeline is still extracted.
- What happens when `wave init --merge` is run on an existing project that already has non-release pipelines from a previous `--all` init? Existing files should be preserved; merge mode should not delete files, only add missing release-filtered ones. The `--merge` flag respects the same release filtering as normal `wave init` (unless `--all` is also passed). It adds missing release pipelines and their transitive dependencies without removing existing files.
- What happens when all pipelines have `release: false`? `wave init` should succeed but produce an empty `.wave/pipelines/` directory (with a warning to the user).
- What happens when a prompt is referenced by both a release and non-release pipeline via `source_path`? The prompt should be included (shared resources are preserved).
- What happens when `wave init` is run with `--all` and `--merge` together? Both flags compose naturally — all pipelines are considered for merge.

### Design Decisions (from issue open questions)

1. **Default value for `release`**: Defaults to `false` (explicit opt-in). This is the safer default — new pipelines must be deliberately marked for release, preventing accidental shipping of incomplete work.
2. **No `wave build` command**: Release filtering is a `wave init` concern, not a compile-time concern. All pipelines remain embedded in the binary (via `go:embed`); filtering happens at extraction time during `wave init`. This avoids build tooling complexity while achieving the same user-facing result.
3. **`wave init --all` for contributors**: The `--all` flag bypasses release filtering, giving contributors access to the full pipeline set for development and testing.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: The pipeline metadata schema MUST include a `release` boolean field that defaults to `false` when absent. This requires adding `Release bool \`yaml:"release,omitempty"\`` to the `PipelineMetadata` struct in `internal/pipeline/types.go`. Go's zero value for `bool` is `false`, which provides the correct default behavior without a pointer type.
- **FR-002**: The `wave init` command MUST filter embedded pipelines by `metadata.release: true` when extracting to `.wave/pipelines/`.
- **FR-003**: The `wave init` command MUST transitively exclude contracts/schemas that are only referenced by non-release pipelines.
- **FR-004**: The `wave init` command MUST transitively exclude prompt files that are only referenced by non-release pipelines.
- **FR-005**: The `wave init` command MUST NOT transitively exclude personas (personas may be shared across pipelines and are always included).
- **FR-006**: The `wave init` command MUST support an `--all` flag that bypasses release filtering and extracts all embedded content.
- **FR-007**: The defaults subsystem MUST provide a way to retrieve only release-flagged pipelines (in addition to the existing get-all capability).
- **FR-008**: Transitive exclusion MUST use a reference-counting approach: a contract/schema/prompt is excluded only if zero release-flagged pipelines reference it.
- **FR-009**: The `release` field MUST be independent of the `disabled` field — a pipeline can be `release: true` and `disabled: true` simultaneously.
- **FR-010**: Existing embedded pipeline tests MUST be updated to validate release filtering behavior.
- **FR-011**: When `wave init` filters out all pipelines (none marked `release: true`), the command MUST succeed with a warning message rather than failing.

### Key Entities

- **Pipeline Metadata**: Extended with a `release` boolean field in the `PipelineMetadata` Go struct (`yaml:"release,omitempty"`). Controls whether a pipeline is distributed to end users via `wave init`. Independent of the `disabled` field which controls runtime execution. The `disabled` field should also be added to the struct at this time for consistency, since it currently exists only in YAML but is not parsed into Go types.
- **Release Pipeline Set**: The subset of embedded pipelines where `metadata.release: true`. Computed at init time by parsing each pipeline's YAML content via the `PipelineMetadata` struct. The defaults package returns raw YAML strings, so filtering requires unmarshalling each pipeline's metadata block.
- **Transitive Dependency Set**: The set of contracts, schemas, and prompts reachable from the release pipeline set. Used to determine which non-pipeline assets to include.
- **Contract Reference**: A link from a pipeline step's `handover.contract.schema_path` field to a contract JSON file. Pipeline YAMLs reference contracts using full relative paths (e.g., `.wave/contracts/plan-exploration.schema.json`), but the embedded FS map keys are bare filenames (e.g., `plan-exploration.schema.json`). Reference matching MUST strip the `.wave/contracts/` prefix from `schema_path` values to match against embedded contract keys.
- **Prompt Reference**: A link from a pipeline step's `exec.source_path` field to an external prompt markdown file. Only `source_path` references with a `.wave/prompts/` prefix qualify for transitive exclusion — inline `source:` blocks contain embedded prompt text and have no external file dependency. Reference matching MUST strip the `.wave/prompts/` prefix from `source_path` values to match against embedded prompt keys (which use relative paths like `speckit-flow/specify.md`).

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: After `wave init`, the `.wave/pipelines/` directory contains only pipelines with `metadata.release: true` — zero non-release pipelines present.
- **SC-002**: After `wave init`, the `.wave/contracts/` directory contains zero contract files that are exclusively referenced by non-release pipelines.
- **SC-003**: After `wave init --all`, the `.wave/pipelines/` directory contains all embedded pipelines (release and non-release) — count matches total embedded pipeline count.
- **SC-004**: All existing tests in `internal/defaults/embed_test.go` continue to pass after the change.
- **SC-005**: At least one new test validates that querying for release-only pipelines returns a strict subset of all pipelines, containing only release-flagged entries.
- **SC-006**: At least one new test validates transitive contract exclusion logic.
- **SC-007**: The `--all` flag for `wave init` is documented in the command's help text.
- **SC-008**: A pipeline with `release: true` and `disabled: true` is included by `wave init` (proving field independence).

## Clarifications _(resolved during spec refinement)_

### CLR-001: Reference path normalization for transitive exclusion

**Ambiguity**: Pipeline YAMLs reference contracts as `.wave/contracts/foo.schema.json` and prompts as `.wave/prompts/speckit-flow/specify.md`, but the embedded FS map keys are bare filenames (`foo.schema.json`) for contracts and relative paths (`speckit-flow/specify.md`) for prompts. The spec didn't specify how to match these.

**Resolution**: The filtering logic MUST normalize reference paths by stripping known prefixes before matching against embedded map keys:
- For contracts: strip `.wave/contracts/` from `schema_path` values to get the embedded key (e.g., `.wave/contracts/plan-exploration.schema.json` → `plan-exploration.schema.json`).
- For prompts: strip `.wave/prompts/` from `source_path` values to get the embedded key (e.g., `.wave/prompts/speckit-flow/specify.md` → `speckit-flow/specify.md`).

**Rationale**: This follows the existing pattern where `readDir()` and `readDirNested()` in `embed.go` strip directory prefixes when building map keys.

### CLR-002: Go struct extension strategy for the `release` field

**Ambiguity**: The `PipelineMetadata` struct has only `Name` and `Description`. The `disabled` field exists in pipeline YAMLs but was never added to the Go struct. Should `release` follow the same unstructured pattern or extend the struct?

**Resolution**: Extend the `PipelineMetadata` struct with both `Release bool \`yaml:"release,omitempty"\`` and `Disabled bool \`yaml:"disabled,omitempty"\`` fields. Using Go's bool zero value (`false`) provides the correct default-to-false behavior without requiring pointer types. Adding both fields at once prevents further tech debt and enables the pipeline executor to eventually use `Disabled` for runtime gating.

**Rationale**: The spec explicitly requires the `release` field to be parsed and queryable via the defaults subsystem (FR-007). Raw YAML string scanning would be fragile. Extending the struct is the idiomatic Go approach and aligns with the existing `yaml` struct tag pattern.

### CLR-003: Inline prompts vs external prompt files for transitive exclusion

**Ambiguity**: Only `speckit-flow.yaml` uses `source_path` for prompt references. Other pipelines (e.g., `plan.yaml`) use inline `source:` blocks. The spec didn't clarify whether inline prompts affect transitive exclusion.

**Resolution**: Only `exec.source_path` references to files under `.wave/prompts/` qualify for transitive exclusion. Inline `source:` blocks contain embedded prompt text with no external file dependency and are excluded from reference counting. This means pipelines using only inline prompts have no prompt dependency footprint.

**Rationale**: Transitive exclusion is about avoiding orphaned *files*. Inline content has no file to orphan.

### CLR-004: `--merge` respects release filtering

**Ambiguity**: The edge case mentioned merge should "not delete files, only add missing ones" but didn't specify whether the added files should be release-filtered.

**Resolution**: `--merge` applies the same release filtering as normal `wave init`. When merging, only missing release-flagged pipelines and their transitive dependencies are added. Existing files (including non-release pipelines from a previous `--all` init) are preserved. When `--all` and `--merge` are combined, all missing pipelines are added regardless of release status.

**Rationale**: Merge is additive but should respect the same filtering contract as fresh init. A user who runs `wave init` followed by `wave init --merge` should get consistent results. The `--all` flag explicitly overrides filtering in both modes.

### CLR-005: `printInitSuccess` display should reflect filtered counts

**Ambiguity**: The current `printInitSuccess` function displays counts from `defaults.GetPipelines()` (all pipelines), `defaults.GetContracts()` (all contracts), etc. After release filtering, the displayed counts should reflect what was actually written, not the total embedded count.

**Resolution**: After implementing release filtering, the init success output MUST display counts of *extracted* assets (post-filtering), not total embedded assets. The pipeline names listed in the output should also reflect only the extracted set. This ensures the user sees an accurate summary of what was initialized.

**Rationale**: Displaying "18 pipelines" when only 3 were extracted would confuse users and contradict the feature's purpose of providing a clean, curated experience.
