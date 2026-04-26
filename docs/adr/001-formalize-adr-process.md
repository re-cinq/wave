# ADR-001: Formalize Architectural Decision Records with Hybrid Manual/Pipeline Approach

## Status
Accepted

## Date
2026-03-07 (proposed) — 2026-04-26 (accepted; pipeline + manual paths verified live)

## Implementation Status

Landed:
- `docs/adr/000-template.md` and the index in `docs/adr/README.md` are in place.
- Manual path: copy template, increment number, open PR.
- Pipeline path: `plan-adr` pipeline ships in `internal/defaults/pipelines/` and produces records in this directory.

ADRs 001–015 follow this process.

## Context

Wave is a multi-agent pipeline orchestrator (v0.32.0+) in rapid prototype phase with a mature codebase spanning 30+ personas, 42+ pipelines, and multiple subsystems (pipeline executor, contract validation, workspace isolation, state management, multi-platform Git integrations). Despite this maturity, architectural decisions are currently scattered across multiple locations:

- `.specify/memory/constitution.md` — 13 constitutional principles governing design
- `CLAUDE.md` — development guidelines and constraints
- `docs/concepts/architecture.md` — system architecture documentation
- Specification templates and inline comments

This scattering makes decisions discoverable but hard to trace. There is no dedicated directory for architectural decision records, no standard format for documenting the rationale behind choices, and no systematic way to query past decisions or understand their status. As the project grows — with frequent releases (v0.15.0 to v0.32.0 in ~2 weeks) and multiple stakeholders across pipeline execution, adapters, contracts, state, security, UI, and multi-platform integration — the lack of a formal decision record process risks losing institutional knowledge and making it harder for new contributors to understand why the system is shaped the way it is.

Wave already has an `adr.yaml` pipeline that uses multi-persona collaboration (navigator, planner, philosopher, craftsman) to generate ADR drafts. However, requiring the pipeline for all ADRs creates a bootstrapping problem: contributors need Wave installed and configured, LLM tokens available, and the pipeline functioning correctly just to document a decision.

## Decision

Adopt a **hybrid approach**: establish `docs/adr/` as the canonical location for architectural decision records with a numbered markdown template for manual creation, while maintaining the `adr.yaml` pipeline as an optional accelerator for complex decisions.

Both paths produce the same format in the same directory:

- **Manual path**: Contributors create ADR files directly using the template (`docs/adr/000-template.md`). No tooling required beyond a text editor and git.
- **Pipeline path**: Run `wave run adr` to generate a pipeline-assisted draft. The pipeline explores the codebase, analyzes options, drafts the record, and opens a PR for human review.

ADR files follow the naming convention `docs/adr/NNN-short-title.md` where NNN is a zero-padded sequential number.

## Options Considered

### Option 1: Pipeline-Automated ADRs in docs/adr/

Use the existing `adr.yaml` pipeline as the sole mechanism for ADR creation. The pipeline explores context via the navigator persona, analyzes options via the planner, drafts the record via the philosopher, and publishes via PR with the craftsman.

**Pros:**
- Dogfooding validates the orchestrator itself
- Multi-persona collaboration produces thorough analysis (navigator explores, planner analyzes, philosopher drafts)
- Contract validation ensures structural consistency via JSON schema
- Automated PR creation integrates into existing review workflow
- Ephemeral workspace isolation prevents corruption of the source repository

**Cons:**
- Bootstrapping problem — requires Wave installed and configured to create any ADR
- Significant LLM token consumption per decision record
- Pipeline failures (adapter errors, contract validation) can block ADR creation entirely
- Generated prose may lack the nuance of human-authored architectural reasoning
- Creates a dependency on LLM availability for a fundamentally documentation task

### Option 2: Manual ADRs with Template

Create `docs/adr/` with a numbered template. Developers write ADR markdown files manually following a standard structure. Standard git workflow for review.

**Pros:**
- Zero tooling dependency — works with any text editor and git
- Accessible to all contributors regardless of Wave installation
- Human-authored reasoning captures nuance and institutional knowledge
- Battle-tested pattern (Michael Nygard's ADR format) used by thousands of projects
- No LLM token cost; fast for simple decisions

**Cons:**
- No structural validation — ADRs may drift from template
- No automated context gathering — author must manually explore and document constraints
- Higher friction may lead to fewer ADRs being written
- Misses opportunity to demonstrate Wave's capabilities
- Quality depends entirely on author diligence

### Option 3: Hybrid Manual Creation with Optional Pipeline Assist (Recommended)

Establish `docs/adr/` with a standard template for manual creation, and maintain the `adr.yaml` pipeline as an optional accelerator. Both paths produce the same format. Pipeline output lands as a PR for human review and refinement.

**Pros:**
- No gatekeeping — anyone can create an ADR without Wave installed
- Pipeline available as accelerator for complex decisions requiring deep codebase exploration
- Dogfooding benefit preserved — the pipeline remains available and exercised
- Human review of pipeline-generated ADRs catches AI reasoning gaps
- Graceful degradation — if the pipeline breaks, the manual path always works
- Low ceremony matches the rapid prototype phase

**Cons:**
- Two paths to maintain — template and pipeline must stay in sync on format
- Risk of structural inconsistency between pipeline-generated and manual ADRs
- Slightly more documentation needed to explain both approaches
- Potential confusion about which path to use for a given decision

### Option 4: Structured YAML ADRs with Rendered Markdown

Store ADRs as structured YAML files with machine-readable fields (status, date, options with typed metadata for effort/risk/reversibility). A build step renders them into human-readable markdown.

**Pros:**
- Machine-readable format enables programmatic querying and analysis
- Structured metadata supports automated impact assessment
- Aligns with Wave's philosophy of structured artifacts
- Contract validation via JSON schema is natural

**Cons:**
- YAML is harder to read and write than markdown for narrative reasoning
- Requires a rendering step — adds build complexity
- Over-engineered for the rapid prototype phase
- Steeper learning curve; no established community precedent
- Narrative context and nuanced reasoning fit poorly into structured fields

## Consequences

### Positive
- Architectural decisions become discoverable in a single canonical location (`docs/adr/`)
- New contributors can understand past decisions and their rationale without archaeology across multiple files
- The manual path removes all barriers to entry — a text editor and git are sufficient
- The pipeline path accelerates complex decisions that benefit from automated codebase exploration
- Wave dogfoods its own pipeline system, validating the orchestrator on real workloads
- Human review of all ADRs (whether manual or pipeline-generated) maintains quality

### Negative
- Two creation paths require documentation and may cause initial confusion about which to use
- The template and pipeline contract schema must be kept in sync — format drift is possible if one is updated without the other
- Pipeline-generated ADRs still require human review and refinement, so the time savings are partial rather than complete

### Neutral
- Existing decisions documented in `constitution.md`, `CLAUDE.md`, and architecture docs remain where they are — no migration of historical decisions is required
- The ADR numbering sequence starts fresh at 001; retroactive documentation of past decisions is optional
- The `adr.yaml` pipeline already exists and requires no changes to support this approach
- ADR status lifecycle follows standard conventions: Proposed → Accepted → Deprecated/Superseded

## Implementation Notes

1. **Create the ADR directory and template**: Add `docs/adr/000-template.md` with the standard ADR sections (Status, Date, Context, Decision, Options Considered, Consequences, Implementation Notes). This ADR becomes `docs/adr/001-formalize-adr-process.md`.

2. **Document both creation paths**: Add a brief guide in `docs/adr/README.md` explaining the manual and pipeline paths, when to use each (simple decisions → manual; complex/cross-cutting decisions → pipeline), and the naming convention.

3. **Verify pipeline format alignment**: Confirm that the `adr.yaml` pipeline's philosopher persona output and contract schema produce the same section structure as the manual template. Adjust the pipeline contract if needed.

4. **No changes to existing files**: The constitution, CLAUDE.md, and architecture docs are not modified. They continue to serve their current purposes alongside the new ADR directory.

5. **Guideline for choosing a path**:
   - **Use manual** for straightforward decisions, process changes, or when Wave is not available.
   - **Use the pipeline** for decisions requiring deep codebase exploration, multi-option analysis, or when the decision affects multiple subsystems.
