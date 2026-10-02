# Next Session Plan — 2026-05-02

## Current branch: `feat/phase4-docker-vm`

### What's on this branch (3 commits ahead of main)

```
77dee784 fix(docker): golang:1.25, drop tini+gh, reuse node user
523c9199 fix(docker): mkdir /out before tea fetch + map ANTHROPIC_TOKEN env
ed33923d feat(docker): Phase 4 reproducible experiment VM (Dockerfile + entrypoint + compose)
```

Files: `docker/Dockerfile.experiment`, `docker/entrypoint.sh`, `docker/docker-compose.experiment.yml`

Branch is **5 commits behind main** (contract guards + rollback + docs merged while branch was open).
Must rebase before merge.

### Untracked files (safe to ignore, not in .gitignore)

Session-planning docs left over from prior work — no code, no impact:
- `docs/2026-04-28-issue-1452-plan.md`
- `docs/2026-04-28-session-plan.md`
- `docs/scope/2026-04-29-phase1-execution-plan.md`
- `docs/scope/2026-04-30-remaining-work.md`
- `docs/scope/2026-04-30-session-pause.md`

Either commit them as `docs: session planning artifacts` or `git clean -f` them. Low stakes.

`docker/.env.experiment` — gitignored, has real tokens. Keep. Required for smoke run.

---

## What needs doing (in order)

### 1. Rebase branch on main
```bash
git rebase main
```
5 commits, no overlap with docker/ files — should be clean.

### 2. docker compose build (needs unmetered connection)
```bash
set -a && . .env && set +a
docker compose -f docker/docker-compose.experiment.yml \
               --env-file docker/.env.experiment build
```
Pulls `golang:1.25-bookworm` (~700MB) + `node:22-bookworm-slim` (~250MB). First pull only.
Layer cache means rebuilds are fast after.

### 3. Smoke boot — WAVE_SKIP_ONBOARD=1 (zero LLM tokens)
```bash
docker compose -f docker/docker-compose.experiment.yml \
               --env-file docker/.env.experiment up
```
`docker/.env.experiment` already has `WAVE_SKIP_ONBOARD=1`.

Verify:
- Container starts, tea clones `codeberg.org/libretech/wave-testing` into `wave-work` volume
- `wave init` runs, `wave.yaml` + `.agents/` appear in `/work`
- webui boots on `:8080`, root redirects to `/onboard` (no sentinel yet)
- `curl http://localhost:8080/health` → 200

### 4. Full onboard smoke (uses LLM tokens, needs unmetered)
Edit `docker/.env.experiment`, comment out `WAVE_SKIP_ONBOARD=1`, restart:
```bash
docker compose -f docker/docker-compose.experiment.yml \
               --env-file docker/.env.experiment up
```

Verify (per epic gate):
- `docker exec wave-experiment ls /work/.agents/` shows personas, pipelines, prompts
- `docker exec wave-experiment cat /work/.agents/.onboarding-done` exists
- Browser: `http://localhost:8080` → `/work` board (sentinel present, redirect switches)
- `/work` board shows issues from `codeberg.org/libretech/wave-testing`

### 5. Comment results on epic #1565
```bash
gh issue comment 1565 --repo re-cinq/wave --body "Phase 4 4.4 smoke: ..."
```
Update epic checklist boxes 4.1/4.2/4.3/4.4.

### 6. Merge PR #1617
After smoke passes. PR is `MERGEABLE`, not draft.
```bash
gh pr merge 1617 --repo re-cinq/wave --squash --delete-branch
```

---

## Known issues already fixed (on branch)

| Issue | Fix | Commit |
|---|---|---|
| `golang:1.22` < go.mod requirement (1.25.5) | → `golang:1.25-bookworm` | 77dee784 |
| `tini` absent from `node:22-bookworm-slim` | removed; compose `init:true` handles PID1 | 77dee784 |
| `gh` CLI absent from slim + not needed (tea handles Codeberg) | removed | 77dee784 |
| `useradd` absent (no `passwd` pkg in slim) | reuse pre-existing `node` user (UID 1000) | 77dee784 |
| `mkdir /out` missing in tea-fetch stage | pre-create dir before curl | 523c9199 |
| `ANTHROPIC_TOKEN` env not forwarded to claude-code | mapped as `ANTHROPIC_API_KEY` | 523c9199 |

## Open questions before merge

- Does `node:22-bookworm-slim`'s `node` user home default to `/home/node`? Verify `echo $HOME` inside container.
- `wave init` on a fresh clone with no `wave.yaml` — does it work non-interactively? (`--adapter claude` flag in entrypoint is passed but `wave init` may still prompt)
- Volume `wave-work` persists across `docker compose down` — correct by design (idempotent clone). `docker compose down -v` to reset.

---

## Epic state

Phases 0/1/1.5/2/3 + all follow-ups: **MERGED**.
Phase 4: PR open, build fixes on branch, smoke deferred to unmetered.
