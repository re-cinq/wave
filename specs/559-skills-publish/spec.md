# Feature Specification: Publish Wave Skills as Standalone SKILL.md Artifacts

**Feature Branch**: `559-skills-publish`
**Created**: 2026-03-24
**Status**: Draft
**Input**: https://github.com/re-cinq/wave/issues/559

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Skills Audit & Classification (Priority: P1)

As a Wave maintainer, I want to audit all 14 built-in skills and classify each as `standalone`, `wave-specific`, or `both`, so that I know which skills are suitable for public distribution.

**Why this priority**: Without an audit, we cannot determine which skills to publish. This is the foundational step that gates all publishing work. It also ensures we don't accidentally publish Wave-internal skills that have no value outside the Wave ecosystem.

**Independent Test**: Can be fully tested by running the audit command against the built-in skills directory and verifying each skill receives a classification tag.

**Acceptance Scenarios**:

1. **Given** a Wave project with 14 built-in skills in `.claude/skills/`, **When** the user runs `wave skills audit`, **Then** each skill is classified as `standalone`, `wave-specific`, or `both` and the results are displayed in a table.
2. **Given** a skill with no `description` or `name` frontmatter field, **When** the audit runs, **Then** the skill is flagged with missing metadata and a compliance warning is shown.
3. **Given** the audit has completed, **When** the user runs `wave skills audit --format json`, **Then** the output is a JSON array with each skill's name, classification, and any compliance warnings.

---

### User Story 2 - Publish a Single Skill (Priority: P1)

As a skill author, I want to publish a single skill to a public registry via `wave skills publish <name>`, so that the skill becomes installable by anyone using Claude Code or compatible agents without requiring the Wave binary.

**Why this priority**: This is the core feature requested in the issue. Publishing individual skills is the minimum viable capability that enables external distribution.

**Independent Test**: Can be fully tested by publishing a skill to a registry, then installing it fresh in a clean environment and verifying it loads correctly.

**Acceptance Scenarios**:

1. **Given** a valid standalone skill "golang" exists in `.claude/skills/golang/SKILL.md`, **When** the user runs `wave skills publish golang`, **Then** the skill is published to the default registry and a success message with the published URL is displayed.
2. **Given** a skill that fails agentskills.io spec validation (missing required fields), **When** the user runs `wave skills publish <name>`, **Then** the command fails with a validation error listing the missing fields and does NOT publish.
3. **Given** a skill classified as `wave-specific`, **When** the user runs `wave skills publish <name>`, **Then** a warning is displayed that the skill may have limited standalone value, and the user is prompted to confirm or pass `--force` to bypass.
4. **Given** the publish succeeds, **When** the lockfile is checked, **Then** an entry is recorded with the skill name, version, content digest, registry URL, and publish timestamp.

---

### User Story 3 - Batch Publish All Standalone Skills (Priority: P2)

As a Wave maintainer, I want to publish all standalone-eligible skills at once via `wave skills publish --all`, so that I can efficiently distribute the entire skill catalog without running individual commands.

**Why this priority**: Batch publishing is a workflow efficiency improvement. The core single-publish capability (P1) must work first, but batch mode is important for release automation.

**Independent Test**: Can be tested by running `--all` and verifying each standalone skill is published, wave-specific skills are skipped, and the lockfile reflects all published entries.

**Acceptance Scenarios**:

1. **Given** 14 skills where 8 are classified as `standalone` or `both`, **When** the user runs `wave skills publish --all`, **Then** only the 8 eligible skills are published and the 6 `wave-specific` skills are skipped with a summary.
2. **Given** one skill in the batch fails validation, **When** batch publish runs, **Then** the failing skill is skipped, remaining skills continue publishing, and a summary shows successes and failures.
3. **Given** a skill was already published with the same content digest, **When** batch publish runs, **Then** the skill is skipped as "up-to-date" (no redundant publish).

---

### User Story 4 - Content Integrity & Lockfile (Priority: P2)

As a security-conscious user, I want every published skill to be content-addressed with a cryptographic hash recorded in a lockfile, so that I can detect upstream tampering when installing skills.

**Why this priority**: The issue explicitly calls out 13-36% vulnerability rates in public registries. Content integrity is essential for trust but depends on the publish flow (P1) being functional first.

**Independent Test**: Can be tested by publishing a skill, modifying its content on the registry side, then verifying that install or verify detects the mismatch.

**Acceptance Scenarios**:

1. **Given** a skill is published, **When** the lockfile is written, **Then** it contains the SHA-256 digest of the SKILL.md content (including frontmatter and body).
2. **Given** a lockfile exists with a recorded digest, **When** the user runs `wave skills verify`, **Then** local skills are re-hashed and compared against lockfile entries, with mismatches reported.
3. **Given** a skill has resource files (scripts/, references/), **When** the content digest is computed, **Then** all resource files are included in the hash computation (Merkle-style or concatenated hash).

---

### User Story 5 - SKILL.md Spec Compliance Validation (Priority: P3)

As a skill author, I want the publish command to validate my SKILL.md against the agentskills.io specification before publishing, so that my skill is guaranteed to work across all compatible agent platforms.

**Why this priority**: Validation prevents broken skills from reaching registries. It's important but builds on top of the core publish flow.

**Independent Test**: Can be tested by creating a SKILL.md with intentionally missing fields and verifying the validation catches each issue.

**Acceptance Scenarios**:

1. **Given** a SKILL.md with all required agentskills.io fields present, **When** validation runs, **Then** it passes with no warnings.
2. **Given** a SKILL.md missing `description` or `name`, **When** validation runs, **Then** it fails with specific field-level error messages.
3. **Given** a SKILL.md with optional fields missing (e.g., `license`, `compatibility`), **When** validation runs, **Then** warnings are emitted but the skill is still publishable.

---

### Edge Cases

- What happens when the user has no network connectivity during publish? The command fails with a clear network error, does not corrupt the lockfile, and suggests retrying.
- What happens when the target registry is unreachable or returns an error? The command fails with the HTTP status/error, does not update the lockfile, and provides the registry URL for debugging.
- What happens when a skill name conflicts with an existing published skill from another author? The publish command reports the conflict and suggests using a namespaced name (e.g., `re-cinq/golang`) or contacting the registry.
- What happens when the SKILL.md body exceeds the registry's size limit? A pre-publish size check warns the user before attempting upload.
- What happens when multiple users concurrently publish the same skill? The registry's server-side conflict resolution applies; the CLI reports the conflict.
- What happens when `.claude/skills/` and `.wave/skills/` both contain a skill with the same name? The publish command uses the highest-precedence source (project `.wave/skills/` > user `~/.claude/skills/`) and warns about the shadow.
- What happens when the user attempts to publish a skill that has no SKILL.md file (only resource files)? The command fails with a clear error: "No SKILL.md found in skill directory."
- What happens when the lockfile is corrupted or contains invalid JSON? The command detects the parse error, refuses to proceed, and suggests re-running with `--force` to regenerate the lockfile.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: System MUST provide a `wave skills audit` subcommand that classifies each installed skill as `standalone`, `wave-specific`, or `both`.
- **FR-002**: System MUST provide a `wave skills publish <name>` subcommand that publishes a single skill to a configured registry.
- **FR-003**: System MUST validate each skill against the agentskills.io specification before publishing, rejecting skills that fail required-field validation.
- **FR-004**: System MUST compute a SHA-256 content digest for every published skill, covering the SKILL.md content and all resource files.
- **FR-005**: System MUST maintain a lockfile (`wave-skills.lock`) recording each published skill's name, version, content digest, registry URL, and publish timestamp.
- **FR-006**: System MUST provide a `--all` flag on the publish command to batch-publish all standalone-eligible skills.
- **FR-007**: System MUST skip already-published skills when the local content digest matches the lockfile entry (idempotent publish).
- **FR-008**: System MUST warn (but not block by default) when attempting to publish a skill classified as `wave-specific`. The `--force` flag overrides the warning.
- **FR-009**: System MUST provide a `wave skills verify` subcommand that checks local skill content against lockfile digests and reports mismatches.
- **FR-010**: System MUST support `--format json|table` output for the audit, publish, and verify commands, consistent with existing `wave skills` commands.
- **FR-011**: System MUST support `--registry <name>` flag to target a specific registry when multiple registries are configured.
- **FR-012**: System MUST NOT modify the lockfile if any publish operation fails (atomic lockfile updates).
- **FR-013**: System MUST validate skill name format (`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`) before attempting publish.
- **FR-014**: System MUST report structured errors with error codes (`skill_publish_failed`, `skill_validation_failed`, `skill_already_exists`) consistent with existing CLI error patterns.
- **FR-015**: System MUST support a `--dry-run` flag that validates and computes digests without actually publishing [NEEDS CLARIFICATION: is dry-run essential for MVP or a future enhancement?].
- **FR-016**: System MUST organize standalone skills in a distributable directory structure (e.g., `skills/` at repo root) so that they can be consumed independently of the Wave binary or published to a separate skills repository.
- **FR-017**: System MUST support an end-to-end roundtrip: publish a skill, install it fresh in a clean environment, and verify it loads correctly in Claude Code.

### Key Entities

- **SkillClassification**: Represents the audit result for a skill — name, classification tag (`standalone` | `wave-specific` | `both`), compliance status, and any validation warnings.
- **PublishRecord**: Represents a single published skill entry — skill name, version, content digest (SHA-256), registry name, registry URL, and publish timestamp.
- **Lockfile** (`wave-skills.lock`): A JSON file at the project root containing an array of PublishRecords, updated atomically on successful publishes.
- **PublishResult**: The outcome of a single publish operation — skill name, success/failure, published URL, digest, and any warnings.
- **RegistryConfig**: Configuration for a target registry — name, base URL, authentication method (token, API key), and any registry-specific settings.
- **ValidationReport**: The result of agentskills.io spec validation — list of errors (blocking) and warnings (non-blocking) per field.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: All 14 built-in skills are audited and classified, with each classification justified by examining the skill's content for Wave-specific references.
- **SC-002**: `wave skills publish <name>` successfully publishes a standalone skill end-to-end: validation passes, skill is uploaded, lockfile is updated, and published URL is returned.
- **SC-003**: `wave skills publish --all` publishes all standalone-eligible skills in a single invocation, skipping wave-specific skills, with a summary report.
- **SC-004**: Every published skill has a SHA-256 content digest recorded in `wave-skills.lock`, and `wave skills verify` detects content changes with zero false positives.
- **SC-005**: A skill that fails agentskills.io spec validation is rejected before upload, with specific field-level error messages guiding the author.
- **SC-006**: An already-published skill with unchanged content is detected as up-to-date and skipped (no redundant registry writes).
- **SC-007**: The publish command completes for a single skill in under 10 seconds on a standard network connection (excluding registry latency).
- **SC-008**: All new CLI subcommands (audit, publish, verify) follow existing Wave CLI patterns: `--format json|table`, structured error codes, and consistent help text.
- **SC-009**: A full end-to-end roundtrip succeeds: publish a skill to a registry, install it fresh in a clean environment without the Wave binary, and verify it loads correctly in Claude Code.
- **SC-010**: Benchmark data collected comparing install + load time for published registry skills vs. local filesystem provisioning, demonstrating registry overhead is bounded.
