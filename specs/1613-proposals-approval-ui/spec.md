# Phase 3.4: Approval webui /proposals + CLI

Part of Epic #1565 Phase 3 (evolution loop).

## Goal

Human gate for evolution proposals. Webui /proposals lists `evolution_proposal` rows; CLI `wave proposals list/approve/reject` mirrors.

## Acceptance criteria

### Webui
- [ ] `internal/webui/handlers_proposals.go` — `GET /proposals` (list), `GET /proposals/{id}` (detail with diff render), `POST /proposals/{id}/approve`, `POST /proposals/{id}/reject`
- [ ] Templates under `internal/webui/templates/proposals/` (Tailwind, no JS framework dep)
- [ ] Diff render via existing diff util or chroma highlight; show before/after pipeline yaml + reason + signal_summary
- [ ] CSRF gate on POST routes

### CLI
- [ ] `cmd/wave/commands/proposals.go` — `wave proposals list`, `wave proposals show <id>`, `wave proposals approve <id>`, `wave proposals reject <id> --reason "..."`
- [ ] Approve: calls `state.EvolutionStore.DecideProposal(id, approved, reason)` + creates new `pipeline_version` row with `active=true` (atomically deactivates priors)
- [ ] Reject: `DecideProposal(id, rejected, reason)`

### Acceptance gate
- [ ] Test: synthetic proposal row → list returns it → approve → `pipeline_version` flips active → CLI `wave run <pipeline>` uses new version

## Dependencies

- #1606 EvalSignal hook MERGED
- #1607 pipeline-evolve meta-pipeline MERGED
- #1612 trigger heuristics (3.3) — not blocking; proposals can land via manual `wave run pipeline-evolve`

## Metadata

- Issue: #1613
- URL: https://github.com/re-cinq/wave/issues/1613
- Repository: re-cinq/wave
- Labels: enhancement, frontend
- Author: nextlevelshit
- State: OPEN
