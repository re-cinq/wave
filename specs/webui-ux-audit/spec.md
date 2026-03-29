# WebUI UX Audit — Findings & Spec

## Audit Date: 2026-03-29

## Finding 1: Passive Pages (No Actions)

8 of 14 pages are read-only displays with zero interactive controls:

| Page | What it shows | What's missing |
|------|---------------|----------------|
| `/compose` | Composition pipeline cards | Run button, link to run history |
| `/personas` | Persona cards | "See runs using this persona" link |
| `/contracts` | Contract cards | — (read-only is fine) |
| `/ontology` | Bounded context stats | — (read-only is fine) |
| `/health` | Health checks | "Fix" action for failed checks |
| `/analytics` | Charts and stats | — (read-only is fine) |
| `/retros` | Retrospective list | — (has narrate button already) |
| `/prs` | PR list | "Review with Wave" button |

## Finding 2: Missing Cross-Links

No page links to filtered run history for its entity:
- `/pipelines/impl-issue` doesn't link to `/runs?pipeline=impl-issue`
- `/personas/craftsman` doesn't link to `/runs?persona=craftsman`
- `/compose` pipelines don't link to their run history

## Finding 3: Issues Page Missing "Run Pipeline"

The issues page lists GitHub issues but has no way to launch `impl-issue` directly from an issue row. Users must copy the URL, go to /runs, and manually start.

## Spec: Priority Fixes

### Fix 1: Add "Run" button to /compose pipeline cards
### Fix 2: Add "Run" button to /issues (per-issue "Implement" action)
### Fix 3: Add "See Runs" links to /pipelines, /personas, /compose
### Fix 4: Add "Review" button to /prs list
### Fix 5: Pipeline detail page — add "Run" button + recent runs table
