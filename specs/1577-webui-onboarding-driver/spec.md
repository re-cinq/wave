# 1.2: Webui onboarding driver (SSE stream + form-answer POST)

**Issue:** [re-cinq/wave#1577](https://github.com/re-cinq/wave/issues/1577)
**State:** OPEN
**Author:** nextlevelshit
**Labels:** (none)
**Branch:** `1577-webui-onboarding-driver`

## Body

Part of Epic #1565. Phase 1, depends on 1.1 (#1576), PRE-1 (#1566 ✓).

Webui handlers + templates that drive the onboarding session through a chat-style UI. Browser users hit `/onboard` and step through detection/confirmation/manifest-write via SSE.

**Files:**
- New: `internal/webui/handlers_onboard.go`
- New: `internal/webui/templates/onboard/*.html`

**Acceptance:**
- [ ] `/onboard` renders chat shell (Tailwind via 1.5c if landed, else minimal CSS)
- [ ] SSE streams claude-code stdout as conversation
- [ ] Form-answer POST routes through onboarder agent
- [ ] Session state persists across reloads via PRE-2 `Service.Resume`
- [ ] Real browser smoke on throwaway repo

**Pipeline:** `impl-issue` (`--adapter claude --model cheapest`)

## Dependency Status

- **PRE-1 #1566** ✓ merged
- **PRE-2** baseline `Service` w/ `Resume` ✓ available at `internal/onboarding/service.go:42`
- **1.1 #1576** ✓ merged (commit `5065e287` — onboarder persona + `onboard-project` meta-pipeline)
- **1.5c Tailwind** — phase A only (preview/* per #1585), not site-wide → use minimal custom CSS (extend `static/style.css`)

## Acceptance Criteria (extracted)

1. `GET /onboard` returns HTML chat shell.
2. SSE endpoint streams onboarder-agent stdout as conversation events.
3. Form POST endpoint accepts user answers; routed into the onboarder Service `UI.PromptString`/`PromptChoice` resume points.
4. Reload mid-session calls `Service.Resume(sessionID)` and re-renders prior conversation.
5. Manual smoke: fresh repo → `/onboard` → answer prompts → manifest written.

## Out of Scope

- Tailwind site-wide redesign (1.5c proper)
- Async/durable session persistence (baseline is in-memory; future phase)
- Multi-user concurrency / auth gating beyond existing middleware
