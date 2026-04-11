# Ontology injection: skill files not populated, missing context gaps, no diff-gate contract

**Issue**: re-cinq/wave#771
**URL**: https://github.com/re-cinq/wave/issues/771
**Author**: nextlevelshit
**State**: OPEN
**Labels**: none

---

## Summary

During a pipeline post-mortem on `re-cinq/CFOAgent#207`, we traced a failed implementation (PR that only modified spec files instead of actual code) to several gaps in the ontology injection and contract validation system. This issue documents the findings and proposes fixes.

## Context

Pipeline `impl-issue` ran against a "update Gemini model IDs" issue. The implement step (craftsman persona) checked all 25+ code files, found them "already correct", and only updated old references in spec files. The PR contained zero source code changes. Root cause was a chain of failures across multiple Wave subsystems.

## Findings and Acceptance Criteria

### Finding 1: `wave analyze --deep` should populate SKILL.md files

**Expected:** `--deep` should write extracted invariants back to SKILL.md files.
**Actual (at time of filing):** Only `deep-analysis.json` was written; SKILL.md files stayed as placeholders.

**Note:** Code inspection reveals `runAnalyzeDeep()` in `cmd/wave/commands/analyze.go` already writes enriched SKILL.md files via `generateDeepSkillContent()`. This finding may already be resolved. Verification is required.

**Acceptance criteria:**
- After `wave analyze --deep`, `.wave/skills/wave-ctx-<name>/SKILL.md` files contain extracted invariants, key decisions, domain vocabulary, neighboring contexts, and key files — not stub text.

---

### Finding 2: Undefined contexts silently yield 0 invariants (ONTOLOGY_WARN)

**Acceptance criteria:**
- When a pipeline step's `contexts:` list references a context name that does not exist in `wave.yaml`, the trace log emits `[ONTOLOGY_WARN]` with the undefined context names.
- The warning is visible in audit trace at the `[ONTOLOGY_INJECT]` log site.
- The step is NOT blocked — it continues to run unconstrained as before; only a warning is emitted.

---

### Finding 3: New `source_diff` contract type

**Proposal:**
```yaml
handover:
  contract:
    type: source_diff
    glob: "{{ project.source_glob }}"
    exclude: ["specs/**", ".wave/**"]
    min_files: 1
    must_pass: true
```

**Acceptance criteria:**
- `NewValidator("source_diff")` returns a working validator.
- Validator checks that the current git diff (staged + unstaged) contains at least `min_files` files matching `glob` and not excluded by `exclude`.
- Validation fails (returns error) when no qualifying source files appear in the diff.
- Validation passes when at least `min_files` qualifying files appear.
- Registered in `internal/contract/contract.go` `NewValidator()` switch.

---

### Finding 4: Schema-level guard — `missing_info` implies `clarify` not in `skip_steps`

**Acceptance criteria:**
- `internal/defaults/contracts/issue-assessment.schema.json` enforces: if `assessment.missing_info` is non-empty (length > 0), then `clarify` must NOT appear in `assessment.skip_steps`.
- This is implemented via JSON Schema `if/then` conditional or a custom constraint.

---

### Finding 5: Document context inheritance behavior

**Acceptance criteria:**
- The behavior "a step with no `contexts:` field receives all defined contexts" is documented in `AGENTS.md` or the relevant pipeline/ontology documentation.
- The trace output difference between explicit and implicit context injection is explained.

---

### Finding 6 (bonus): Context inheritance documentation

The plan step in `impl-issue.yaml` has no explicit `contexts:` field, yet receives all defined contexts. This is not necessarily a bug but is surprising and undocumented.

**Acceptance criteria:**
- Add a comment or note to `impl-issue.yaml` or docs explaining the inherit-all behavior.

---

## Related

- CFOAgent issue: re-cinq/CFOAgent#207
- Failed PR: re-cinq/CFOAgent#208 (closed)
