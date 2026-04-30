# Phase 1.4: Entry-page branch / → /onboard or /work

Issue: https://github.com/re-cinq/wave/issues/1599
Repository: re-cinq/wave
Labels: enhancement, ready-for-impl, frontend
State: OPEN
Author: nextlevelshit
Branch: 1599-entry-page-branch

Part of Epic #1565 Phase 1.

## Goal

Make `/` smart-redirect: if `.agents/.onboarding-done` sentinel missing → `/onboard` (kick off onboarding). Otherwise → `/work` (now that 2.3 + 2.4 are live).

## Acceptance criteria

- [ ] `internal/webui/handlers_root.go` (or extend routes.go):
  - GET / → 302 to /onboard if sentinel missing
  - GET / → 302 to /work otherwise
- [ ] Test coverage: round-trip both paths via tmp fixtures
- [ ] Removes any old default-landing logic (was /runs)
- [ ] No emojis

## Dependencies

- 1.2 webui driver (PR #1587 MERGED)
- 2.3 /work board (PR #1597 MERGED)
