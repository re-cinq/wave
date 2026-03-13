# ADR-001: CodeSpeak Integration Strategy for Wave Pipelines

## Status

Proposed

## Date

2026-03-13

## Context

Wave is a Go-based multi-agent pipeline orchestrator that wraps LLM CLIs (Claude Code, OpenCode) via subprocess execution. Code generation today is handled entirely by LLM personas — agents like the implementer and craftsman write code through tool calls (Read, Write, Edit, Bash) within isolated workspaces, with outputs validated against step contracts.

[CodeSpeak](https://codespeak.dev) is an emerging spec-to-code compilation tool that takes structured specifications and deterministically compiles them into source code. This represents a fundamentally different code generation model compared to Wave's current LLM-based approach: compilation rather than generation.

The question is whether and how to integrate CodeSpeak into Wave's pipeline system. Several factors drive this decision:

- **Quality variance**: LLM-generated code has inherent non-determinism. A deterministic compiler could improve output consistency for well-specified components.
- **Existing spec infrastructure**: Wave already has a spec-driven pipeline (`speckit-flow`) using the SpeckIt methodology. CodeSpeak would introduce a second spec-driven approach with different semantics — SpeckIt produces specs that LLMs implement, while CodeSpeak compiles specs directly into code.
- **Maturity gap**: CodeSpeak is in alpha preview (v0.1.0). Its CLI interface, spec format, and output behavior are unstable.
- **Dependency model**: Wave targets a single static binary with no runtime dependencies except adapter binaries. CodeSpeak is Python-based, requiring `uv`/`pip` installation, which adds a Python runtime dependency for any pipeline step using it.
- **Prototype phase**: Wave has no backward compatibility constraint during the current prototype phase, allowing rapid experimentation — but compounding two alpha-stage systems increases instability risk.

Relevant codebase context:

- The `AdapterRunner` interface (`internal/adapter/adapter.go`) dispatches to Claude, OpenCode, or a generic `ProcessGroupRunner` based on manifest configuration.
- The preflight system (`internal/preflight/checker.go`) already supports skill auto-installation via `uv tool install`, as demonstrated by the speckit skill in `speckit-flow.yaml`.
- The Nix dev shell (`flake.nix`) includes `uv` for Python tool management.
- The `ProcessGroupRunner` handles arbitrary subprocess execution but lacks streaming events and uses a fragile prompt-as-arguments model.

## Decision

**Integrate CodeSpeak as a pipeline skill dependency using Bash invocation within LLM persona steps**, following the exact pattern established by the SpeckIt skill in `speckit-flow.yaml`.

CodeSpeak will be declared in a pipeline's `requires.skills` block with `check`/`install` commands (`uv tool install codespeak-cli`). LLM personas will invoke the `codespeak` CLI via Bash tool calls as part of their implementation workflow. No changes to Wave core (adapter code, manifest types, or Go packages) are required.

## Options Considered

### Option 1: Dedicated CodeSpeak Adapter

Build a first-class `CodeSpeak` adapter implementing the `AdapterRunner` interface, treating CodeSpeak as a peer execution backend alongside Claude and OpenCode. The adapter would invoke the `codespeak` CLI as a subprocess, translate its output into `StreamEvent` format for progress tracking, and produce artifacts compatible with contract validation.

**Pros:**
- Deepest integration — CodeSpeak becomes a first-class citizen with full streaming event support
- Contract validation works seamlessly since the adapter controls output format and artifact paths
- Persona permissions and sandbox constraints enforced at the adapter level
- Follows the established adapter precedent (OpenCode, GitHub adapters)

**Cons:**
- Highest implementation effort — new Go code in `internal/adapter/`, manifest type changes, test coverage
- CodeSpeak's alpha CLI interface will require frequent adapter updates as their API changes
- Tight coupling to CodeSpeak's execution model — dead code if CodeSpeak pivots or is abandoned
- The `AdapterRunner` interface assumes LLM-like interaction patterns (prompts, tokens, system prompts) that don't map cleanly to a compiler
- Permanent maintenance burden on Wave core for a tool that may not reach stability

**Effort:** Large | **Risk:** High | **Reversibility:** Moderate

### Option 2: Pipeline Skill Dependency (Bash Invocation) — Recommended

Add CodeSpeak as a skill dependency in pipelines that need it, following the `speckit` pattern in `speckit-flow.yaml`. The LLM persona orchestrates CodeSpeak CLI calls via Bash within the existing workspace. No adapter changes needed.

**Pros:**
- Zero changes to Wave core — no adapter code, no manifest type changes, no new Go packages
- Follows the proven `speckit` pattern exactly; implementation is pipeline YAML and prompt engineering
- Preflight auto-installation infrastructure already exists (`uv tool install`)
- LLM persona retains intelligent control over when to use CodeSpeak vs. writing code directly
- Easy to experiment with and easy to remove — no Go code to maintain
- Decoupled from CodeSpeak's API stability — prompts adapt to CLI changes without Go changes

**Cons:**
- No streaming progress events from CodeSpeak execution — appears as opaque Bash command in TUI
- Additional LLM token cost for orchestrating CLI calls (reading output, interpreting errors, retrying)
- Persona must understand CodeSpeak's spec format through prompt engineering alone
- Contract validation only applies to the persona's final output, not intermediate compilation steps

**Effort:** Small | **Risk:** Low | **Reversibility:** Easy

### Option 3: ProcessGroupRunner Wrapper

Use the existing `ProcessGroupRunner` (the generic fallback adapter) to run CodeSpeak as a non-LLM pipeline step. CodeSpeak gets its own step in the pipeline DAG with explicit dependencies, running as a compilation pass between spec-writing and validation steps.

**Pros:**
- Leverages existing subprocess execution infrastructure
- CodeSpeak gets its own pipeline step, making the DAG structure explicit
- Contract validation runs on CodeSpeak output as a distinct step

**Cons:**
- `ProcessGroupRunner` lacks streaming events
- The prompt-as-arguments model (`strings.Fields` split) is fragile for complex CLI invocations
- Requires restructuring pipelines to separate spec-writing from compilation, adding step count and complexity
- Token estimation (`len/4`) is meaningless for compiler output, polluting metrics

**Effort:** Medium | **Risk:** Medium | **Reversibility:** Moderate

### Option 4: Wait and Monitor (No Integration Now)

Defer integration entirely. Continue with LLM-based code generation. Monitor CodeSpeak's development and revisit when it reaches beta or 1.0.

**Pros:**
- Zero cost, zero maintenance burden
- Avoids coupling to an alpha-stage tool with unknown longevity
- No new runtime dependency — Wave stays closer to its single-binary model
- Avoids confusion between two competing spec-driven approaches (SpeckIt vs. CodeSpeak)
- Preserves the option to integrate later with better information

**Cons:**
- Misses potential for improved code generation consistency via deterministic compilation
- The prototype phase is explicitly designed for rapid experimentation — waiting is philosophically at odds with this
- Delayed feedback loop — early integration could influence CodeSpeak's API toward Wave's needs

**Effort:** Trivial | **Risk:** Low | **Reversibility:** Easy

## Consequences

### Positive

- Wave gains spec-to-code compilation capability with near-zero integration cost
- Pipeline authors can experiment with CodeSpeak in isolated pipelines without affecting the rest of the system
- The existing preflight and skill infrastructure is validated with a second real-world skill dependency
- If CodeSpeak proves transformative, concrete CLI usage data informs a future adapter-level integration with known requirements rather than guesswork
- Wave's prototype-phase experimentation philosophy is honored — move fast with reversible decisions

### Negative

- No real-time progress visibility during CodeSpeak execution within persona Bash calls — the TUI shows an opaque command rather than granular compilation progress
- LLM token overhead for orchestrating CodeSpeak CLI calls adds cost per pipeline run
- Prompt engineering reliability is lower than adapter-level enforcement — the persona could misuse or misunderstand the CodeSpeak CLI
- Two spec-driven approaches (SpeckIt methodology + CodeSpeak compilation) may confuse pipeline authors about which to use and when

### Neutral

- Python/`uv` runtime dependency already exists in the dev shell for SpeckIt; CodeSpeak does not introduce a new dependency class
- Pipelines using CodeSpeak will need dedicated prompt sections explaining the CodeSpeak spec format and CLI usage to the persona
- If CodeSpeak's alpha status leads to breakage, the impact is contained to the specific pipelines that declare it as a skill — no Wave core code is affected

## Implementation Notes

- **Pipeline YAML changes**: Add a `requires.skills` entry for CodeSpeak in target pipelines (e.g., a new `codespeak-flow.yaml` or as an optional skill in `speckit-flow.yaml`):
  ```yaml
  requires:
    skills:
      - name: codespeak
        check: "codespeak --version"
        install: "uv tool install codespeak-cli"
  ```
- **Persona prompt additions**: Personas in CodeSpeak-enabled pipelines need prompt sections explaining the spec format, CLI invocation patterns (`codespeak compile <spec-file>`), and error handling for compilation failures.
- **Contract validation**: Use existing `test_suite` or `json_schema` contracts on the persona's final output artifacts. CodeSpeak compilation errors are handled by the persona (retry, fallback to manual implementation, or fail the step).
- **No Go code changes required**: All integration happens at the pipeline definition and persona prompt layers.
- **Evaluation criteria**: After experimentation, consider upgrading to a dedicated adapter (Option 1) if: (a) CodeSpeak reaches stable release, (b) CLI output format is documented and stable, (c) streaming progress visibility becomes a user-reported pain point, and (d) multiple pipelines adopt CodeSpeak as a standard step.
- **Migration path**: The skill dependency approach is a proper subset of any future adapter integration. Pipelines built on this pattern will continue to work even if a dedicated adapter is added later — the adapter just replaces the Bash-level invocation with native execution.

## Assumptions

- **Assumption**: CodeSpeak's CLI follows conventional patterns (`codespeak compile <input> -o <output>`) and returns non-zero exit codes on failure. *If violated*: Persona prompts will need to handle unconventional invocation patterns.
- **Assumption**: `uv tool install codespeak-cli` is the correct installation command and produces a `codespeak` binary on PATH. *If violated*: The skill `check`/`install` commands in pipeline YAML will need updating.
- **Assumption**: CodeSpeak-generated code is plain source files (Go, TypeScript, etc.) that can be validated by Wave's existing contract types. *If violated*: New contract validators may be needed for CodeSpeak-specific output formats.
