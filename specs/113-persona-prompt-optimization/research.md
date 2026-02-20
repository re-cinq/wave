# Research: Persona Prompt Optimization

**Feature Branch**: `113-persona-prompt-optimization`
**Date**: 2026-02-20
**Spec**: `specs/113-persona-prompt-optimization/spec.md`

## Phase 0 — Unknowns & Research

### Unknown 1: Shared Content Across Current Personas

**Question**: What content is duplicated across persona files that should move to a base protocol?

**Finding**: After auditing all 17 persona files, the following patterns are shared (implicitly or explicitly) but NOT currently duplicated verbatim:

- Most personas follow the same structure: `# Title`, `## Responsibilities`, `## Output Format`, `## Constraints`
- Common constraint phrases appear across multiple personas:
  - "NEVER modify source code/files" — navigator, auditor, reviewer, provocateur, supervisor, validator
  - "NEVER commit or push changes" — implementer, reviewer, supervisor, provocateur
  - "NEVER run destructive commands" — auditor, implementer
  - Output format: "When a contract schema is provided, output valid JSON matching the schema" — implementer, reviewer, github-analyst, github-commenter, github-enhancer, provocateur, researcher, supervisor
- **No persona currently references** Wave-specific operational context (fresh memory, artifact injection, workspace isolation, contract compliance). This context is entirely absent from persona prompts today.

**Decision**: The base protocol must introduce Wave operational context that is currently missing from all persona prompts, plus absorb the generic contract output instruction. Individual persona files retain their role-specific behavioral constraints.

**Rationale**: The current persona prompts are role-focused but lack the Wave-specific operational context the spec requires. The base protocol fills this gap while also providing a home for the shared contract compliance instruction.

**Alternatives Rejected**:
- Injecting operational context at the manifest/pipeline level: would require changes to the pipeline executor, not just the adapter — larger blast radius.
- Keeping operational context out of prompts entirely: the issue explicitly calls for it.

---

### Unknown 2: Base Protocol Runtime Injection Mechanism

**Question**: How should the base protocol be loaded and prepended at runtime?

**Finding**: The `prepareWorkspace` method in `internal/adapter/claude.go:213-294` builds CLAUDE.md from:
1. `cfg.SystemPrompt` (if set directly) OR the persona `.md` file from `.wave/personas/<persona>.md`
2. Restriction section from manifest permissions

The injection point is clear: between reading the persona prompt and writing CLAUDE.md, prepend the base protocol content.

**Decision**: Read `base-protocol.md` from `.wave/personas/base-protocol.md` (the same directory as persona files). Prepend its content before the persona-specific prompt, separated by `---`. If the file is missing, return an error (fail-secure). This is the minimal code change: ~10 lines added to `prepareWorkspace`.

**Rationale**: The `.wave/personas/` directory is already used for persona files. The existing `//go:embed personas/*.md` directive in `internal/defaults/embed.go:18` will automatically include `base-protocol.md` without modification.

**Alternatives Rejected**:
- Loading from a separate `base-protocol/` directory: requires new embed directive, new directory convention.
- Passing via `AdapterRunConfig.SystemPrompt`: would require changes to all pipeline step configuration code.

---

### Unknown 3: Impact on `wave init` and `GetPersonas()`

**Question**: Will `base-protocol.md` be treated as a persona by init and other callers?

**Finding**: `GetPersonas()` in `embed.go:31` returns all `*.md` files from the embedded `personas/` directory as a `map[string]string`. The init command (`init.go:69`) iterates this map and writes every entry to `.wave/personas/`. The manifest (`init.go:618-785`) creates persona entries only for specific named personas — it does NOT dynamically create manifest entries from the file list.

The test at `init_test.go:821` asserts `len(allPersonas) == len(entries)`, comparing the count from `GetPersonas()` to the count of files on disk.

**Decision**:
1. `GetPersonas()` will naturally include `base-protocol.md` — this is fine for init (the file gets written to `.wave/personas/`).
2. The init test will pass because both counts increase by 1.
3. The manifest template in init.go does NOT need to be changed — it hardcodes specific persona names and won't create a manifest entry for `base-protocol.md`.
4. `PersonaNames()` will include `base-protocol.md` — callers should be audited but the function is not used to create manifest entries.

**Rationale**: Minimal change — no filtering needed in `GetPersonas()`. The file is just another `.md` file that gets copied. The manifest is the authoritative persona registry, not the file list.

**Alternatives Rejected**:
- Filtering `base-protocol.md` from `GetPersonas()`: adds complexity for no benefit since init already handles it correctly.

---

### Unknown 4: Token Budget Feasibility

**Question**: Can all 17 personas fit within 100–400 tokens after extracting shared content?

**Finding**: Current word counts (estimated tokens at ~0.75 words/token):

| Persona | Words | Est. Tokens |
|---------|-------|-------------|
| navigator | 127 | ~169 |
| philosopher | 125 | ~166 |
| implementer | 143 | ~190 |
| planner | 141 | ~188 |
| auditor | 147 | ~196 |
| github-enhancer | 147 | ~196 |
| craftsman | 152 | ~202 |
| summarizer | 152 | ~202 |
| synthesizer | 160 | ~213 |
| reviewer | 165 | ~220 |
| debugger | 171 | ~228 |
| github-analyst | 182 | ~242 |
| validator | 185 | ~246 |
| github-commenter | 223 | ~297 |
| supervisor | 276 | ~368 |
| provocateur | 288 | ~384 |
| researcher | 297 | ~396 |

After extracting shared content (generic contract output instruction, generic "NEVER modify/commit" that belong in base protocol), most personas will lose 20-60 tokens. The longer personas (researcher, provocateur, supervisor) contain role-specific process descriptions and evidence-gathering sections that are NOT duplicated — these are genuinely differentiating content.

Anti-patterns to remove per persona:
- **Researcher**: "Research Process" section (8-step workflow) is generic read-process-output → remove. "Source Evaluation Criteria" and "Handling Conflicting Information" are borderline — some is genuinely differentiating, some is generic.
- **Debugger**: "Debugging Process" section (7-step workflow) is partially generic → compact.
- **Supervisor**: "Evidence Gathering" section is role-specific → keep. "Evaluation Criteria" subsections are role-specific → keep.
- **Provocateur**: "Thinking Style" and "Evidence Gathering" are genuinely differentiating → keep.

**Decision**: All 17 personas can fit within 100–400 tokens. Short personas (navigator, philosopher) will be near the lower bound (~100-130 tokens). Long personas (researcher, provocateur) require aggressive removal of generic process descriptions but will stay within 400 tokens.

**Rationale**: The spec's lower bound of 100 tokens and principle of "every token earns its place" support lean prompts. Personas with genuinely minimal responsibilities should be short.

---

### Unknown 5: Language-Specific Content Audit

**Question**: Which personas contain language-specific references that must be removed?

**Finding**: Grep for language names across all persona files:

- **No persona file** currently contains explicit programming language references (Go, Python, TypeScript, etc.).
- The current persona prompts are already language-agnostic.
- The CLAUDE.md project instructions reference Go, but those are injected separately from a different source (the project's CLAUDE.md, not the persona prompt).

**Decision**: No language-specific content removal needed. Verify this with a grep test as part of SC-003.

**Rationale**: The existing personas were already written in a language-agnostic manner.

---

### Unknown 6: Parity Enforcement Between `internal/defaults/personas/` and `.wave/personas/`

**Question**: How to ensure byte-identical parity between the two locations?

**Finding**: Currently, `wave init` copies from the embedded defaults to `.wave/personas/`. The embedded defaults in `internal/defaults/personas/` are the source of truth. The `.wave/personas/` copies are written at init time.

However, the `.wave/personas/` files in the repository root are NOT the init output — they're checked-in copies maintained manually. Parity between `internal/defaults/personas/` and `.wave/personas/` at the repository level is a manual discipline.

**Decision**:
1. Implementation should update files in both locations simultaneously.
2. A CI test (or `go test` assertion) should diff the two directories and fail on any divergence.
3. This is a testing concern, not a runtime concern — at runtime, only `.wave/personas/` is read.

**Rationale**: The spec requires parity (FR-011). A test is the cheapest and most reliable enforcement mechanism.

---

### Unknown 7: Base Protocol Content Definition

**Question**: What exactly goes in `base-protocol.md`?

**Finding**: Per FR-001 and the Key Entities section, the base protocol must contain:

1. **Fresh memory constraint**: No chat history from prior steps. Each step starts with a clean context.
2. **Artifact communication model**: Read inputs from injected artifacts. Write outputs to artifact files for downstream steps.
3. **Workspace isolation**: Each step runs in an ephemeral worktree. Your workspace is isolated from the source repository.
4. **Contract compliance**: Outputs must satisfy the step's validation contract before the step completes.
5. **Security enforcement**: Permissions are enforced by the orchestrator. Do not attempt to bypass restrictions. Defer to the restriction section below for specifics.

Per FR-009/CLR-005, the base protocol must NOT list specific tool permissions.

**Decision**: Base protocol is ~80-120 tokens of high-signal operational context. It does NOT include role-specific guidance, output format instructions, or permission lists.

**Rationale**: The five areas are Wave-universal. Anything role-specific stays in the persona file.

---

## Summary

| # | Unknown | Decision | Risk |
|---|---------|----------|------|
| 1 | Shared content patterns | Base protocol introduces Wave operational context + shared contract instruction | Low — content is additive, not destructive |
| 2 | Runtime injection | ~10 lines in `prepareWorkspace`, read from `.wave/personas/base-protocol.md` | Low — minimal code change in one function |
| 3 | `wave init` impact | No filtering needed; `GetPersonas()` includes it naturally | Low — init test passes with +1 count |
| 4 | Token budget | All 17 fit in 100–400 range after optimization | Medium — longer personas need careful compaction |
| 5 | Language references | None found in current personas | None |
| 6 | Parity enforcement | Test-based diff assertion | Low — test catches drift |
| 7 | Base protocol content | 5 Wave-universal constraints, ~80-120 tokens | Low — well-defined by spec |
