# feat(webui): achieve actual CLI/TUI feature parity — missing views and controls

**Issue**: [#442](https://github.com/re-cinq/wave/issues/442)
**Labels**: enhancement
**Author**: nextlevelshit
**State**: OPEN

## Description

Issue #299 claimed CLI/TUI feature parity for the WebUI. The actual parity is ~70%.

**Note**: Resume endpoint and API pagination are correctly implemented. The gaps are in missing views.

### Missing Views (TUI has, WebUI doesn't)
- **Issues browser** — GitHub issue list with detail pane and pipeline chooser
- **Pull Requests view** — PR list with state badges and detail
- **Compose/Sequence builder** — Pipeline sequence editor with artifact flow validation
- **Skills view** — Skill list with descriptions and usage
- **Health checks** — Automated infrastructure health with per-check detail
- **Guided flow** — Health → Proposals → Fleet progression

### Missing Controls
- Pipeline composition launch
- Issue-to-pipeline launcher

## Current State (Partially Implemented)

Per comments, the following are already merged:
- Skills view (`handlers_skills.go`, `templates/skills.html`)
- Compose view (`handlers_compose.go`, `templates/compose.html`)
- Pipeline/persona metadata enrichment
- Enhanced start form

## Remaining Gaps

1. **GitHub Issues browser** — requires forge API integration in webui server
2. **Pull Requests view** — requires forge API integration in webui server
3. **Health checks display** — reuse TUI's `HealthDataProvider` logic
4. **Issue-to-pipeline launcher** — start pipeline from issue detail

## Acceptance Criteria

At minimum, the WebUI should support:
1. Viewing issues and PRs
2. Health check display
3. Skills/Personas detail views matching TUI depth
