# Plan: Skills rebuild + `.agents/` namespace adoption

**Issue umbrella:** #1113 (refactor(skills): overhaul of existing skills system) + #1118 (adapter-native skill loading)
**Userbase:** 1 (single-user repo, no migration tooling needed)
**Branch target:** new feature branch off main, single bundled PR
**Dated:** 2026-04-18

---

## Goal

Two coupled moves in one PR:

1. Drop the entire current skills feature (tessl integration, lockfile, classifier, custom DirectoryStore, dual-CLI). Rebuild minimal, adapter-agnostic, lazy-loading via each adapter's native skill tool.
2. Rename project config dir `.wave/` → `.agents/` (keep `wave.yaml` at project root). Stake claim on `.agents/{pipelines,personas,contracts,workspaces,output,artifacts,skills}/` orchestrator subfolders.

No migration command. No back-compat shims. Pre-1.0 per memory `versioning policy`.

---

## Research outcomes (verified, do not re-research)

### Adapter skill discovery — all 4 support lazy loading natively

| Adapter | Project paths | User-global paths | Native tool | Loading |
|---|---|---|---|---|
| **claude** | `.claude/skills/<n>/SKILL.md` | `~/.claude/skills/` | `Skill` | metadata at start, body on tool call |
| **opencode** | `.opencode/skills/`, `.claude/skills/`, `.agents/skills/` | `~/.config/opencode/skills/`, `~/.claude/skills/`, `~/.agents/skills/` | `skill` | metadata at start, body on tool call |
| **gemini** | `.gemini/skills/`, `.agents/skills/` | `~/.gemini/skills/`, `~/.agents/skills/` | `activate_skill` | same |
| **codex** | `.agents/skills/` (walks repo root) | `~/.agents/skills/` | `/skills` or `$name` | same |

Sources:
- https://opencode.ai/docs/skills/
- https://geminicli.com/docs/cli/skills/
- https://developers.openai.com/codex/skills

### `.agents/` cross-tool standard

`.agents/skills/` natively read by 3 of 4 adapters (opencode, gemini, codex). Claude reads `.claude/skills/` only — Wave provisions per-adapter.

`.agents/` orchestrator subfolders (pipelines, personas, contracts, workspaces, output, artifacts) — UNCLAIMED. Wave moves first, ships RFC upstream to https://github.com/agentsmd/agents.md.

---

## Architecture

### Detection (read-only walk, project then user-global)

```
project (walk git root down):
  1. .agents/skills/<n>/SKILL.md          ← primary committed team source
  2. .claude/skills/<n>/SKILL.md          ← detect if user committed Claude Code skills
  3. .opencode/skills/, .gemini/skills/   ← opt-in adapter-specific (low priority)

user-global:
  4. ~/.agents/skills/
  5. ~/.claude/skills/
  6. ~/.config/opencode/skills/, ~/.gemini/skills/
```

First match wins per skill name.

### Provisioning (per step, per adapter, into step workspace)

Wave copies one source SKILL.md (+ supporting files) into the path that step's adapter natively reads:

| Adapter | Workspace target | Why |
|---|---|---|
| claude | `<workspace>/.claude/skills/<name>/` | only path claude scans |
| opencode | `<workspace>/.agents/skills/<name>/` | natively scanned |
| gemini | `<workspace>/.agents/skills/<name>/` | natively scanned |
| codex | `<workspace>/.agents/skills/<name>/` | only path codex scans |

Two physical targets (`.claude/skills/` for claude, `.agents/skills/` for the other 3). Possible simplification: always write both (small overhead, zero adapter dispatch). Decide during impl.

All 4 adapters then lazy-load via their native tool. No Wave-side prompt injection needed.

### Safety — workspace-scope assert before destructive write

Before any `RemoveAll(<path>/skills/)` in adapter prepare:

```go
if !strings.HasPrefix(skillsDir, workspacePath+string(os.PathSeparator)) {
    panic(fmt.Sprintf("refusing RemoveAll outside workspace: %s", skillsDir))
}
```

Sentinel `.wave-managed` file written alongside provisioned SKILL.md so future runs distinguish Wave-provisioned from user-committed. Only sentinel-tagged dirs removed on re-provision.

---

## CLI surface (minimal)

`wave skills` (plural per clig.dev collection convention):

| Sub | Purpose | LoC |
|---|---|---|
| `list` | scan all detection sources, show name + path + source priority + which pipelines reference + adapter compat | ~100 |
| `check <name>` | validate single skill frontmatter, show resolved path, list pipelines/steps using it | ~60 |
| `add <source>` | thin install — accepts local path / git url / file:// / https://*.tar.gz, copies to `~/.agents/skills/<name>/` (or `.agents/skills/` with `--project` flag) | ~80 |
| `doctor` | diagnose: dirs scanned, duplicates, malformed frontmatter, conflicts, deprecated `.wave/skills/` mentions | ~50 |

No `remove` (rm -rf is fine), no `search`/`sync`/`audit`/`publish`/`verify` (tessl-coupled, gone).

`wave skill` (singular) — DELETED, all routes go through plural.

---

## Pipeline integration

Step-level only. Pipeline-level `requires.skills:` becomes computed `union(steps[].skills)` for preflight (no separate declaration site).

```yaml
steps:
  - id: implement
    persona: craftsman
    skills: [golang, gh-cli]   # provisioned per adapter at step start
```

Preflight: if any declared skill not detectable on host, fail with exact `wave skills add` command suggestion.

---

## `.agents/` rename — file moves

Bulk move, single commit:

```
.wave/pipelines/   → .agents/pipelines/
.wave/personas/    → .agents/personas/
.wave/contracts/   → .agents/contracts/
.wave/skills/      → .agents/skills/      (then rebuild via new pipeline)
.wave/output/      → .agents/output/      (.gitignore update)
.wave/workspaces/  → .agents/workspaces/  (.gitignore update)
.wave/artifacts/   → .agents/artifacts/   (.gitignore update)
```

`wave.yaml` stays at project root (manifest, not subfolder concern).

`~/.config/wave/` stays (user-global Wave config, not project-level `.agents/`).

Code paths to update (rough — verify in impl):
- `internal/workspace/` — workspace base path resolution
- `internal/manifest/` — manifest discovery
- `internal/pipeline/` — pipeline + persona + contract loading
- `internal/defaults/` — bundled defaults
- `cmd/wave/commands/init.go` — scaffolds new project layout
- `internal/onboarding/` — flavour detection writes new layout
- `internal/webui/` — anywhere UI references `.wave/` literally
- `.gitignore` — entries
- `AGENTS.md` / `CLAUDE.md` / docs — every literal mention
- All YAML defaults referencing `.wave/output/`, `.wave/artifacts/` paths
- Test fixtures

---

## Smoke pipelines (`wave-smoke-*` series)

Ship under `internal/defaults/pipelines/`. Each commits a fixture skill `.agents/skills/wave-smoke-test/SKILL.md` with known frontmatter. Each pipeline declares `steps[].skills: [wave-smoke-test]`, prompts the agent to invoke its native skill tool and report the description verbatim. Contract validates output contains the description string from frontmatter.

| Pipeline | Adapter | Native tool exercised |
|---|---|---|
| `wave-smoke-skills-claude.yaml` | claude | `Skill` |
| `wave-smoke-skills-opencode.yaml` | opencode | `skill` |
| `wave-smoke-skills-gemini.yaml` | gemini | `activate_skill` |
| `wave-smoke-skills-codex.yaml` | codex | `/skills` |

CI strategy:
- **PR-gated:** mock-adapter test of provisioning code path (asserts files written to correct workspace paths)
- **Nightly cron, `continue-on-error`:** real claude smoke (ANTHROPIC_API_KEY available); other adapters manual-trigger

---

## Deletion list (explicit)

```
cmd/wave/commands/skills.go              (~1000 LoC)
cmd/wave/commands/skill.go               (~455 LoC, content folded into new skills.go)
internal/skill/publish.go                (~400)
internal/skill/lockfile.go               (~200)
internal/skill/classify.go               (~300)
internal/skill/source_cli.go             (~400)  tessl wrapper
internal/skill/source_*.go               remote prefix parsers — delete except minimal install path
internal/skill/store.go                  (~200)  custom DirectoryStore format
internal/skill/*_test.go                 deleted alongside
.wave/skills.lock                        (artifact, never re-written)
docs/guide/skill-ecosystems.md           (tessl-centric, replaced by minimal guide)
```

Estimated **−3500 LoC**.

Add: ~250 LoC new `cmd/wave/commands/skills.go` + ~40 LoC `prepareSkills()` per adapter (4× = 160 LoC) + smoke pipelines + tests.

Net: **−3000 LoC**.

---

## Upstream RFC

After landing internally, file PR/issue at https://github.com/agentsmd/agents.md proposing orchestrator subfolders:

> **RFC: Orchestrator subfolders under `.agents/`**
>
> `.agents/skills/` is established (opencode/gemini/codex). For tools that orchestrate or chain agent runs:
> - `.agents/pipelines/` — declarative multi-step agent workflows
> - `.agents/personas/` — reusable agent role definitions
> - `.agents/contracts/` — output schemas + validation
> - `.agents/workspaces/` — ephemeral per-run scratch (gitignored)
> - `.agents/output/`, `.agents/artifacts/` — pipeline outputs (gitignored)
>
> Reference implementation: https://github.com/re-cinq/wave

---

## Implementation order (single bundled PR)

1. **Branch:** `feat/1113-skills-and-agents-folder` off main
2. **Rename `.wave/` → `.agents/`** (mechanical, big diff, do first to anchor):
   - File moves via `git mv`
   - Replace literals: `grep -rl "\.wave/" --include="*.go" --include="*.md" --include="*.yaml" .` then `sed`
   - Update `.gitignore`
   - `go build ./...` clean
   - `go test ./internal/workspace/... ./internal/manifest/... ./internal/pipeline/...` pass
3. **Delete tessl-coupled skill code** (deletion list above)
4. **Rebuild `cmd/wave/commands/skills.go`** with `list`, `check`, `add`, `doctor` only
5. **Adapter `prepareSkills()`** for each of 4 adapters
6. **Wire step-level `skills:`** in pipeline executor
7. **Workspace-scope safety assert** + sentinel file
8. **Ship 4 smoke pipelines + 1 mock-adapter PR-gated test**
9. **Docs:** `docs/reference/cli.md` rewrite, `docs/guide/skills.md` rewrite (drop ecosystems guide), `AGENTS.md` updated, README skills row collapsed
10. **Verify** real-surface per `feedback_real_verification.md`:
    - Run `wave-smoke-skills-claude.yaml` end-to-end with real API key
    - `curl` API to confirm pipeline detection of new layout
    - Browser-verify webui shows skills page correctly with new paths
11. **Open PR** linking #1113 + #1118 + #1120, body lists every renamed path + every deleted file
12. **After merge:** file upstream RFC at agentsmd/agents.md

---

## Acceptance

- [ ] No reference to `.wave/<subdir>/` anywhere in code or docs (only `.wave/` literal allowed: nowhere)
- [ ] No reference to `tessl` in repo (except changelog/release notes)
- [ ] `wave skills list` finds skills from `.agents/skills/` and `~/.claude/skills/` (both)
- [ ] `wave-smoke-skills-claude.yaml` end-to-end pass with real adapter
- [ ] All 4 adapters provision skills into the path they natively scan
- [ ] No `RemoveAll` outside workspace boundary (panic guard tested)
- [ ] Net LoC delta within 10% of −3000 estimate
- [ ] Upstream RFC filed within 7 days of merge

---

## Out of scope (explicit)

- `wave migrate` — not needed (userbase = 1)
- Back-compat for `.wave/skills/` detection — gone per pre-1.0 policy
- Per-step skill token-budget gates — not applicable (lazy loading on all 4 adapters)
- Cross-adapter skill capability negotiation — out of scope, future
- TUI skills page redesign — separate issue

---

## Resume hint (for post-compact)

Read this file. Cross-reference with feedback memories:
- `feedback_only_trust_admin.md` — never auto-merge
- `feedback_real_verification.md` — verify via real surface
- `feedback_no_emojis.md` — UI work, no emojis

#1113 still open. Filing umbrella issue with this plan as body is step 0.
