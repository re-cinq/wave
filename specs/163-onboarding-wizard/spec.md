# Feature Specification: Interactive Onboarding Wizard for First-Time Setup

**Feature Branch**: `163-onboarding-wizard`
**Created**: 2026-02-28
**Status**: Draft
**Issue**: [#163](https://github.com/re-cinq/wave/issues/163)
**Labels**: enhancement, ux, priority: high

## Problem

The current `wave init` command relies on heuristics and does not interactively confirm configuration with the user. This leads to a poor first-run experience where pipelines may fail due to missing dependencies, unconfigured adapters, or incorrect test commands.

## Proposed Solution

Implement an interactive onboarding wizard that guides users through complete first-time setup. The wizard runs before any pipeline execution and ensures the environment is fully configured.

### Onboarding Steps

1. **Dependency verification** — detect and install required dependencies (e.g., `gh`, adapter binaries, skills)
2. **Test command configuration** — show heuristically-guessed test commands and allow the user to confirm or override them
3. **Pipeline selection** — display available pipelines, let the user select which to enable, and allow marking non-release pipelines as experimental
4. **Adapter configuration** — select the LLM adapter (Claude, OpenCode, etc.) and verify authentication tokens or subscription status
5. **Model selection** — set the default model for pipeline execution

### Pipeline Gating

Prevent pipeline execution until onboarding is complete. If a user attempts to run a pipeline without completing onboarding, prompt them to finish setup first.

## User Scenarios & Testing

### User Story 1 - First-Time Setup (Priority: P1)

A new user runs `wave init` for the first time and is guided through an interactive wizard that configures all essential settings, producing a valid `wave.yaml` that passes `wave validate`.

**Why this priority**: Without the wizard, users must manually edit YAML and guess configuration values, leading to broken setups and frustration.

**Independent Test**: Can be tested by running `wave init` in a fresh directory and verifying that the wizard walks through all steps, producing a valid manifest.

**Acceptance Scenarios**:

1. **Given** a directory with no `wave.yaml`, **When** the user runs `wave init`, **Then** the interactive wizard launches with 5 steps: dependency verification, test command configuration, pipeline selection, adapter configuration, and model selection.
2. **Given** heuristic defaults are detected (e.g., `go test ./...` in a Go project), **When** the wizard presents test commands, **Then** the user can confirm or override each command.
3. **Given** multiple adapters are available, **When** the wizard reaches adapter configuration, **Then** the user can select an adapter and verify authentication.
4. **Given** the wizard completes all steps, **When** the user finishes, **Then** a valid `wave.yaml` is written and `wave validate` passes.

---

### User Story 2 - Pipeline Execution Gating (Priority: P1)

When a user attempts `wave run` before completing onboarding, the system blocks execution and prompts them to finish setup.

**Why this priority**: Running pipelines with incomplete configuration wastes tokens and produces confusing failures.

**Independent Test**: Can be tested by running `wave run <pipeline>` without a completed onboarding state and verifying the error message directs the user to `wave init`.

**Acceptance Scenarios**:

1. **Given** onboarding has not been completed, **When** the user runs `wave run`, **Then** execution is blocked with a message: "Run 'wave init' to complete setup before running pipelines".
2. **Given** onboarding has been completed, **When** the user runs `wave run`, **Then** execution proceeds normally.

---

### User Story 3 - Reconfigure Existing Setup (Priority: P2)

An existing user can re-run the wizard to update their configuration using `wave init --reconfigure`.

**Why this priority**: Users need to update adapter settings, add new pipelines, or change model selection without starting from scratch.

**Independent Test**: Can be tested by running `wave init --reconfigure` on an existing project and verifying each step shows current values as defaults.

**Acceptance Scenarios**:

1. **Given** a project with completed onboarding, **When** the user runs `wave init --reconfigure`, **Then** the wizard launches with current values pre-filled as defaults.
2. **Given** the user modifies only the adapter selection during reconfigure, **When** the wizard completes, **Then** only the adapter setting is updated in `wave.yaml`.

---

### User Story 4 - Non-Interactive Mode (Priority: P2)

Users in CI/CD or scripting contexts can complete setup non-interactively using `wave init --yes` with sensible defaults.

**Why this priority**: CI pipelines cannot interact with the wizard; they need a way to accept defaults.

**Independent Test**: Can be tested by running `wave init --yes` and verifying all defaults are applied without prompting.

**Acceptance Scenarios**:

1. **Given** no TTY is available, **When** the user runs `wave init`, **Then** defaults are applied automatically (equivalent to `--yes`).
2. **Given** `--yes` flag is provided, **When** the user runs `wave init --yes`, **Then** all heuristic defaults are accepted without prompting.

---

### User Story 5 - Pipeline Category Selection (Priority: P3)

Users can select pipelines by category (stable, experimental, contrib) rather than individually.

**Why this priority**: With many pipelines, individual selection becomes tedious. Categories derived from pipeline metadata offer a faster opt-in mechanism.

**Independent Test**: Can be tested by verifying pipeline categories are derived from `metadata.category` and users can toggle entire groups.

**Acceptance Scenarios**:

1. **Given** pipelines with `metadata.category: stable|experimental|contrib`, **When** the wizard presents pipeline selection, **Then** pipelines are grouped by category.
2. **Given** a category is toggled, **When** the user selects "experimental", **Then** all pipelines in the experimental category are enabled.

---

### Edge Cases

- What happens when no adapter binary is found on PATH? Report missing dependency with install instructions.
- What happens when adapter authentication fails? Allow the user to retry or skip (with a warning that pipelines will fail).
- What happens when `wave init` is run in a non-interactive environment without `--yes`? Apply defaults and warn on stderr.
- What happens when the project type cannot be detected? Skip test command step and allow manual entry.
- What happens when the onboarding state file is corrupted or deleted? Treat as incomplete onboarding, re-prompt on next `wave run`.

## Requirements

### Functional Requirements

- **FR-001**: System MUST present an interactive multi-step wizard when `wave init` is run in an interactive terminal.
- **FR-002**: System MUST detect project type and pre-fill test/lint/build commands with heuristic defaults.
- **FR-003**: System MUST allow users to confirm or override each heuristic default.
- **FR-004**: System MUST check for required dependencies (adapter binaries, `gh` CLI, skills) and report missing ones with install instructions.
- **FR-005**: System MUST verify adapter authentication during the adapter configuration step.
- **FR-006**: System MUST persist onboarding completion state so it only runs once.
- **FR-007**: System MUST block pipeline execution (`wave run`, `wave do`) until onboarding is complete.
- **FR-008**: System MUST support `wave init --reconfigure` to re-run the wizard with current values as defaults.
- **FR-009**: System MUST support `wave init --yes` for non-interactive setup with defaults.
- **FR-010**: System MUST support pipeline category metadata (`metadata.category`) for grouped selection.
- **FR-011**: Pipeline metadata MUST support a `release: true/false` flag for experimental classification (already partially exists).

### Key Entities

- **OnboardingState**: Tracks whether onboarding is complete; persisted in `.wave/state.db` or a file marker.
- **WizardStep**: Represents one step in the wizard (dependency check, test config, pipeline selection, adapter config, model selection).
- **PipelineCategory**: Grouping of pipelines by `metadata.category` (stable, experimental, contrib).

## Success Criteria

### Measurable Outcomes

- **SC-001**: Users can complete first-time setup by running `wave init` and answering interactive prompts, producing a valid `wave.yaml`.
- **SC-002**: All 7 acceptance criteria from the issue are satisfied.
- **SC-003**: Pipeline execution is blocked until onboarding completes, with a clear error message.
- **SC-004**: `wave init --reconfigure` re-runs the wizard preserving current settings.
- **SC-005**: `wave init --yes` completes setup non-interactively.
- **SC-006**: All existing tests continue to pass (`go test ./...`).

## Prerequisites

- Preflight/skill dependency system may need rework to support interactive install prompts (see `internal/preflight/`).
- Pipeline metadata needs a `release: true/false` flag for experimental classification (already partially implemented in `PipelineMetadata.Release`).
