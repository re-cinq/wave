# Implementation Plan — WebUI UX Polish (#455)

## 1. Objective

Bring the Wave web dashboard UX to GitHub Actions-level quality by fixing rough edges in the run list, run detail, step cards, log viewer, and all secondary pages, while increasing test coverage.

## 2. Approach

This is a **CSS/template/JS-heavy frontend polish** with targeted Go handler improvements for test coverage. No new API endpoints are needed — the existing 18 endpoints are functionally complete. The work decomposes into:

1. **Run list view polish** — improved status indicators, progress column, relative timestamps, empty/loading states
2. **Run detail view polish** — GitHub Actions-style step timeline, duration formatting, progress bar per-step, better DAG sidebar
3. **Log viewer UX** — auto-scroll improvements, collapsible section defaults, search UX, line-wrap toggle
4. **Global consistency** — unified empty states, loading spinners, error pages, responsive fixes, nav active states
5. **Test coverage** — handler tests for untested endpoints (compose, contracts, pipelines, personas, skills), SSE edge cases

## 3. File Mapping

### Templates (modify)
| File | Changes |
|------|---------|
| `internal/webui/templates/layout.html` | Add loading indicator wrapper, improve nav responsiveness |
| `internal/webui/templates/runs.html` | Add progress column, improve empty state, enhance filter UI |
| `internal/webui/templates/run_detail.html` | GitHub Actions-style step timeline, summary section with stats |
| `internal/webui/templates/partials/run_row.html` | Add progress indicator, relative time display, input preview |
| `internal/webui/templates/partials/step_card.html` | Timeline-style layout, better duration/token display |
| `internal/webui/templates/notfound.html` | Proper styled 404 page |
| `internal/webui/templates/pipelines.html` | Consistent card layout |
| `internal/webui/templates/personas.html` | Consistent card layout |
| `internal/webui/templates/contracts.html` | Consistent card layout |
| `internal/webui/templates/skills.html` | Consistent card layout |
| `internal/webui/templates/issues.html` | Consistent table/card layout |
| `internal/webui/templates/prs.html` | Consistent table/card layout |
| `internal/webui/templates/health.html` | Consistent card layout |
| `internal/webui/templates/compose.html` | Consistent card layout |

### Static Assets (modify)
| File | Changes |
|------|---------|
| `internal/webui/static/style.css` | Step timeline styles, progress indicators, loading states, empty states, responsive fixes |
| `internal/webui/static/app.js` | Enhanced relative time, loading state management, improved sort UX |
| `internal/webui/static/log-viewer.js` | Line-wrap toggle, improved auto-scroll, section expand/collapse defaults |
| `internal/webui/static/sse.js` | Better step card creation, progress indicators, token count updates |

### Go Handlers (modify)
| File | Changes |
|------|---------|
| `internal/webui/handlers_runs.go` | Add progress calculation, input preview to RunSummary |
| `internal/webui/types.go` | Add progress field to RunSummary if needed |

### Tests (modify/create)
| File | Changes |
|------|---------|
| `internal/webui/handlers_compose_test.go` | New — test compose handler |
| `internal/webui/handlers_contracts_test.go` | New — test contracts handler |
| `internal/webui/handlers_pipelines_test.go` | New — test pipelines handler |
| `internal/webui/handlers_personas_test.go` | New — test personas handler |
| `internal/webui/handlers_skills_test.go` | New — test skills handler |
| `internal/webui/handlers_runs_test.go` | Add tests for progress calculation, edge cases |
| `internal/webui/handlers_sse_test.go` | Add SSE reconnection edge cases |

## 4. Architecture Decisions

- **No JavaScript framework** — stay with vanilla JS + Go templates. The current approach is well-suited for this dashboard and adding React/Vue would be over-engineering.
- **CSS-only improvements where possible** — prefer CSS transitions and modern CSS features over JS DOM manipulation for visual polish.
- **GitHub Actions UX as reference** — use the step timeline layout pattern (vertical timeline with collapsible sections, status icons on a vertical line connecting steps).
- **Progressive enhancement** — all pages must work without JS (server-rendered), with JS adding real-time updates.

## 5. Risks

| Risk | Mitigation |
|------|-----------|
| Breaking existing SSE streaming | Keep SSE protocol unchanged; only modify client-side rendering |
| Template rendering regressions | Existing handler tests catch template execution errors |
| Large CSS changes causing visual regressions | Changes are additive (new classes) not destructive (modifying existing classes) |
| Test coverage changes breaking CI | Run `go test ./...` after each test file addition |

## 6. Testing Strategy

- **Unit tests**: Handler tests for all untested page handlers (compose, contracts, pipelines, personas, skills)
- **Integration tests**: SSE broker reconnection scenarios, artifact viewer edge cases
- **Manual verification**: Visual comparison against GitHub Actions screenshots from issue
- **Regression**: `go test -race ./...` must pass clean
