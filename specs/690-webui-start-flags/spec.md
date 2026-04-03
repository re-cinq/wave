# feat(webui): start pipeline dialog missing CLI flags (model, from-step, steps, exclude, dry-run)

**Issue**: [re-cinq/wave#690](https://github.com/re-cinq/wave/issues/690)
**Parent**: #687 (item 3)
**Author**: nextlevelshit
**Complexity**: medium

## Problem

The "Start Pipeline" dialog on `/pipelines` and `/runs` only has an Input text field. The `StartPipelineRequest` struct only has `Input string`. The CLI `wave run` supports 18+ flags that are invisible in the web UI.

## Required Flags (High Priority)

- `--model` — model override dropdown (haiku/sonnet/opus)
- `--from-step` — resume from specific step (dropdown of step IDs)
- `--steps` — run only named steps (multi-select)
- `--exclude` / `-x` — skip named steps (multi-select)
- `--dry-run` — preview checkbox

## Nice to Have

- `--timeout` — timeout in minutes
- `--on-failure` — halt or skip radio
- `--auto-approve` — checkbox for gate auto-approval

## Files to Change

- `internal/webui/types.go` — expand `StartPipelineRequest`
- `internal/webui/handlers_control.go` — wire new fields into executor options
- `internal/webui/templates/pipelines.html` — add form fields to the start dialog
- `internal/webui/static/app.js` or inline JS — send new fields in the API call

## Acceptance Criteria

- [ ] Start dialog shows model dropdown, from-step, steps, exclude, dry-run toggle
- [ ] Selected options are sent in the API request and applied to execution
- [ ] Step lists are populated from the pipeline definition (loaded via API)

## Labels

None
