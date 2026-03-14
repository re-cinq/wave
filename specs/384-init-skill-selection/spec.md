# Feature Specification: Wave Init Interactive Skill Selection

**Feature Branch**: `384-init-skill-selection`
**Created**: 2026-03-14
**Status**: Draft
**Input**: User description: "https://github.com/re-cinq/wave/issues/384"

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Ecosystem Selection During Init (Priority: P1)

A user runs `wave init` for the first time. After completing the existing onboarding steps (dependency verification, test configuration, pipeline selection, adapter configuration, model selection), they are presented with an ecosystem selection prompt. The user selects an ecosystem (tessl, BMAD, OpenSpec, Spec-Kit) or chooses to skip skill installation entirely. Choosing "Skip" completes the wizard without installing any skills.

**Why this priority**: This is the entry point for all skill installation during onboarding. Without ecosystem selection, no subsequent skill browsing or installation can occur. It also provides the "Skip" escape hatch for users who don't want skills.

**Independent Test**: Can be fully tested by running `wave init` interactively and verifying the ecosystem prompt appears after model selection. Selecting "Skip" should complete onboarding with no skills in `wave.yaml`.

**Acceptance Scenarios**:

1. **Given** the user runs `wave init` in interactive mode, **When** they complete the model selection step, **Then** a new "Skill Ecosystem" step appears offering: tessl, BMAD, OpenSpec, Spec-Kit, and Skip options.
2. **Given** the ecosystem selection step is shown, **When** the user selects "Skip", **Then** the wizard completes without installing any skills and `wave.yaml` contains no skill entries.
3. **Given** the user runs `wave init --yes` (non-interactive mode), **When** onboarding executes, **Then** the skill selection step is skipped entirely (no skills installed).
4. **Given** the user runs `wave init --reconfigure`, **When** the ecosystem step appears, **Then** any previously installed skills are shown as context but do not constrain the new selection.

---

### User Story 2 - Skill Browsing and Multi-Selection (Priority: P1)

After selecting an ecosystem, the user interaction depends on the ecosystem type:

- **tessl**: The system runs `tessl search ""` (or similar) to fetch available skills from the registry, then presents a `huh.MultiSelect` form with skill names and descriptions. The `huh` library's built-in filtering allows the user to narrow results by typing.
- **BMAD, OpenSpec, Spec-Kit**: These ecosystems use "install-all" adapters — their CLI commands (`npx bmad-method install`, `openspec init`, `specify init`) install all available skills at once. The user is shown a confirmation prompt describing what will be installed, with the option to confirm or skip. Individual skill selection is not supported for these ecosystems because their adapters do not expose a listing API.

**Why this priority**: This is the core interaction for choosing which skills to install. Without browsing and selection, the feature has no practical value beyond ecosystem awareness.

**Independent Test**: Can be tested by selecting the "tessl" ecosystem and verifying a multi-select list of skills appears. The user selects 2-3 skills and proceeds. For BMAD/OpenSpec/Spec-Kit, a confirmation prompt is shown instead of individual multi-select.

**Acceptance Scenarios**:

1. **Given** the user selects "tessl" as ecosystem, **When** the skill list loads, **Then** a multi-select form displays available skills with names and descriptions, populated via `tessl search`.
2. **Given** the user selects "BMAD" as ecosystem, **When** the ecosystem is confirmed, **Then** a confirmation prompt describes the install-all behavior and asks the user to confirm or skip.
3. **Given** the tessl skill list is shown, **When** the user toggles multiple skills and confirms, **Then** all toggled skills are marked for installation.
4. **Given** the user is browsing tessl skills, **When** they type a search term, **Then** the `huh.MultiSelect` built-in filter narrows the displayed options.
5. **Given** a skill list or confirmation is shown, **When** the user selects no skills (tessl) or declines the confirmation (BMAD/OpenSpec/Spec-Kit), **Then** the wizard proceeds without installing any skills (equivalent to skip).

---

### User Story 3 - Skill Installation with Progress Feedback (Priority: P2)

After the user confirms their skill selection, the system installs each selected skill into `.wave/skills/`. Progress feedback is shown for each skill: the skill name being installed and whether it succeeded or failed. On completion, the installed skills are recorded in the generated `wave.yaml` under the `skills:` key.

**Why this priority**: Installation is essential for the feature to have lasting effect, but it depends on the browsing/selection flow being in place first. Progress feedback ensures the user knows what's happening during potentially slow network operations.

**Independent Test**: Can be tested by selecting 2-3 skills and observing that each shows installation progress (name + status), the files appear in `.wave/skills/`, and `wave.yaml` lists them under `skills:`.

**Acceptance Scenarios**:

1. **Given** the user has selected 3 skills for installation, **When** installation begins, **Then** each skill displays its name and a progress indicator (installing/success/failure).
2. **Given** installation completes successfully for all skills, **When** the wizard finishes, **Then** `.wave/skills/` contains a subdirectory with `SKILL.md` for each installed skill.
3. **Given** installation completes, **When** `wave.yaml` is written, **Then** the `skills:` key contains a list of all successfully installed skill names.
4. **Given** one skill fails to install, **When** the remaining skills succeed, **Then** the wizard reports the failure, continues with the successful installs, and only lists successful skills in `wave.yaml`.

---

### User Story 4 - Graceful Handling of Missing Ecosystem CLI (Priority: P2)

When the user selects an ecosystem whose CLI tool is not installed (e.g., selecting "tessl" when `tessl` is not on PATH, or selecting "BMAD" when `npx` is not available), the system detects this and offers a choice: skip skill installation or display install instructions for the missing CLI.

**Why this priority**: Users may not have ecosystem CLIs pre-installed. Without graceful handling, the onboarding would fail or show cryptic errors. This ensures a smooth experience regardless of the user's environment.

**Independent Test**: Can be tested by selecting an ecosystem without its CLI installed and verifying a helpful message appears offering to skip or showing install instructions.

**Acceptance Scenarios**:

1. **Given** the user selects "tessl" ecosystem, **When** `tessl` CLI is not found on PATH, **Then** the system shows a message explaining the missing dependency and offers: "Skip skill installation" or "Show install instructions".
2. **Given** the user chooses "Show install instructions", **When** instructions are displayed, **Then** the install command (e.g., `npm i -g @tessl/cli`) is shown and the user can retry or skip.
3. **Given** the user chooses "Skip" after a missing CLI warning, **When** the wizard continues, **Then** no skills are installed and the wizard completes normally.

---

### User Story 5 - Reconfigure Preserves Skill Context (Priority: P3)

When a user runs `wave init --reconfigure`, the ecosystem/skill selection step shows their current skill configuration as context. They can keep existing skills, add new ones, or start fresh with a different ecosystem.

**Why this priority**: Reconfiguration is a secondary workflow. Users expect their existing settings to be preserved as defaults when reconfiguring, but this is not the primary onboarding path.

**Independent Test**: Can be tested by running `wave init`, installing skills, then running `wave init --reconfigure` and verifying existing skills are displayed during the ecosystem step.

**Acceptance Scenarios**:

1. **Given** the user has previously installed skills via `wave init`, **When** they run `wave init --reconfigure`, **Then** the ecosystem step shows currently installed skill names as context.
2. **Given** existing skills are shown during reconfigure, **When** the user selects a different ecosystem, **Then** the new ecosystem's skills are shown and previously installed skills are not removed until new installation completes.

---

### Edge Cases

- What happens when the ecosystem registry is unreachable (network failure during skill listing)?
  - The system displays an error message and offers to skip skill installation or retry.
- What happens when the user aborts (Ctrl+C) during skill installation?
  - Partially installed skills are cleaned up. `wave.yaml` only lists fully installed skills.
- What happens when a skill name conflicts with an already-installed skill?
  - The system warns about the conflict and asks the user whether to overwrite or skip.
- What happens when `.wave/skills/` directory doesn't exist yet?
  - The system creates it automatically before installation begins (this is normal for fresh `wave init`).
- What happens when multiple ecosystems are desired?
  - The current scope supports selecting one ecosystem per init run. Users can install additional skills later via `wave skills install`.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: System MUST add a new wizard step after model selection that presents ecosystem choices: tessl, BMAD, OpenSpec, Spec-Kit, and Skip.
- **FR-002**: System MUST display a multi-select form for tessl (populated via `tessl search`) or a confirm/skip prompt for install-all ecosystems (BMAD, OpenSpec, Spec-Kit) after the user chooses an ecosystem.
- **FR-003**: System MUST install selected skills into `.wave/skills/` using the corresponding `SourceAdapter` (tessl, BMAD, OpenSpec, Spec-Kit).
- **FR-004**: System MUST show per-skill progress during installation (skill name plus success/failure status).
- **FR-005**: System MUST record all successfully installed skill names as bare names (e.g., `["golang", "spec-kit"]`, not source-prefixed) in `wave.yaml` under the top-level `skills:` key, matching the `Manifest.Skills []string` field.
- **FR-006**: System MUST skip the skill selection step entirely when running in non-interactive mode (`--yes` flag or non-TTY).
- **FR-007**: System MUST detect when an ecosystem's CLI dependency is missing and offer skip or install instructions.
- **FR-008**: System MUST handle installation failures gracefully — report the failure, continue with remaining skills, and only list successful installs in `wave.yaml`.
- **FR-009**: System MUST support the `--reconfigure` flag by showing existing installed skills as context during the ecosystem step.
- **FR-010**: System MUST use the existing `huh` form library with `WaveTheme()` for consistent styling with the other wizard steps.
- **FR-011**: System MUST implement the `WizardStep` interface for the new ecosystem/skill selection step, maintaining consistency with existing steps.
- **FR-012**: The new step MUST be inserted as Step 6 in `RunWizard()` between model selection (Step 5) and `writeManifest()`. All existing step labels ("Step N of 5") MUST be renumbered to "Step N of 6".
- **FR-013**: `WizardResult` MUST be extended with a `Skills []string` field, and `buildManifest()` MUST be updated to include the `skills:` key when skills are present.

### Key Entities

- **SkillSelectionStep**: A new `WizardStep` implementation (`internal/onboarding/steps.go`) that handles ecosystem choice, skill browsing/confirmation, and installation orchestration. Implements the `WizardStep` interface (`Name() string`, `Run(cfg *WizardConfig) (*StepResult, error)`).
- **EcosystemDef**: A struct defining each ecosystem with fields: display name, `SourceAdapter` prefix (`tessl`, `bmad`, `openspec`, `speckit`), `CLIDependency` (binary name + install instructions), and a flag indicating whether it supports individual skill listing or is install-all.
- **SkillListItem**: A displayable skill entry containing name and description. For tessl, populated from `tessl search` output via `parseTesslSearchOutput()`. Not applicable for install-all ecosystems.
- **InstallProgress**: Per-skill installation status tracking (pending, installing, success, failure) for progress feedback rendering. Leverages the existing `InstallResult.Skills` and `InstallResult.Warnings` from the `SourceAdapter.Install()` return value.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: Users can complete the full `wave init` flow including skill selection in a single interactive session without leaving the terminal.
- **SC-002**: All four supported ecosystems function correctly: tessl lists available skills via `tessl search` for individual selection; BMAD, OpenSpec, and Spec-Kit present confirm/skip for their install-all behavior. All install into `.wave/skills/`.
- **SC-003**: The "Skip" option bypasses skill installation completely — no skills added to `.wave/skills/`, no `skills:` key in `wave.yaml`. (The `.wave/skills/` directory may exist from other operations; Skip simply doesn't add to it.)
- **SC-004**: When an ecosystem CLI is missing, 100% of cases show the missing dependency message rather than an unhandled error.
- **SC-005**: Installation failures for individual skills do not prevent other selected skills from being installed.
- **SC-006**: The `skills:` key in the generated `wave.yaml` accurately reflects all successfully installed skills.
- **SC-007**: The `--reconfigure` flow preserves awareness of previously installed skills.
- **SC-008**: Non-interactive mode (`--yes`) skips skill selection without error.
- **SC-009**: All new functionality has test coverage including unit tests for the wizard step and integration tests for the installation flow.

## Clarifications

The following ambiguities were identified during spec analysis and resolved based on codebase context:

### C1: Skill listing behavior differs by ecosystem

**Ambiguity**: The original spec described "a multi-select form with available skills" for all ecosystems, but the `SourceAdapter` interface only exposes `Install(ctx, ref, store)` — it has no `List()` method. Only tessl has a registry search capability (`tessl search`). The BMAD, OpenSpec, and Spec-Kit adapters are install-all: `npx bmad-method install --tools claude-code --yes`, `openspec init`, and `specify init` each install all available skills at once with no individual listing.

**Resolution**: Two interaction modes — multi-select for tessl (populated via `tessl search`), confirm/skip for install-all ecosystems. This aligns with the actual adapter architecture in `internal/skill/source_cli.go` without requiring new `List()` methods on adapters.

### C2: Wizard step numbering and insertion point

**Ambiguity**: The wizard currently has 5 hardcoded steps labeled "Step N of 5" in `internal/onboarding/steps.go`. The spec says the new step appears "after model selection" but didn't specify how the step count changes or where the manifest write occurs.

**Resolution**: The skill selection step becomes Step 6 (after model selection at Step 5), inserted before `writeManifest()` in `RunWizard()`. All existing step labels are renumbered from "Step N of 5" to "Step N of 6". Added FR-012 and FR-013 to codify this.

### C3: Skills key format in wave.yaml

**Ambiguity**: The spec said "record in `wave.yaml` under the `skills:` key" but didn't specify whether skills are stored as bare names (`["golang"]`) or source-prefixed (`["tessl:golang"]`).

**Resolution**: Bare names. The `Manifest.Skills` field is `[]string` of plain skill names (see `internal/manifest/types.go:23`). `internal/skill/resolve.go` and `DirectoryStore` both operate on bare names. Source prefixes are only used during installation routing via `SourceRouter.Parse()`.

### C4: Search/filter mechanism for tessl

**Ambiguity**: The spec mentioned "search/filter capability" for tessl but didn't specify the implementation mechanism within the `huh` form framework.

**Resolution**: Use `huh.MultiSelect` which has built-in keyboard filtering. Pre-populate the option list by running `tessl search ""` (list all) via the same subprocess pattern used in `cmd/wave/commands/skills.go:runSkillsSearch`. The `parseTesslSearchOutput()` function already exists for parsing tessl search results.

### C5: WizardResult extension for skill data flow

**Ambiguity**: `WizardResult` in `internal/onboarding/onboarding.go` has no skills field. The spec didn't address how skill selections flow from the new step to `writeManifest()` / `buildManifest()`.

**Resolution**: Add `Skills []string` to `WizardResult`. The new step returns installed skill names via `StepResult.Data["skills"]`. `RunWizard()` extracts and sets `result.Skills`. `buildManifest()` conditionally includes `"skills"` key when `result.Skills` is non-empty. Added FR-013 to codify this.
