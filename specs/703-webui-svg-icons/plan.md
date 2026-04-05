# Implementation Plan: SVG Icons for Forges and Adapters

## Objective

Add inline SVG icons for forges (GitHub, GitLab, Bitbucket) and adapters (claude-code, browser, mock) to the webui landing page, replacing plain text badges with icon+text badges for improved visual clarity.

## Approach

Create a dedicated `icons.go` file in `internal/webui/` that provides template functions returning inline SVG strings. Register these functions in the template FuncMap so templates can call `{{adapterIcon .Name}}` and `{{forgeIcon .Name}}` directly. Update the relevant templates to prepend icons to existing text badges. Add CSS for consistent icon sizing and alignment.

## File Mapping

| File | Action | Purpose |
|------|--------|---------|
| `internal/webui/icons.go` | **create** | SVG icon definitions for forges and adapters, plus template functions |
| `internal/webui/icons_test.go` | **create** | Unit tests for icon functions |
| `internal/webui/embed.go` | **modify** | Register `adapterIcon` and `forgeIcon` in template FuncMap |
| `internal/webui/templates/partials/run_row.html` | **modify** | Add adapter icons to adapter badges (line 28) |
| `internal/webui/templates/partials/child_run_row.html` | **modify** | (No adapter column currently — skip unless adding one) |
| `internal/webui/static/style.css` | **modify** | Add `.icon-inline` class for consistent 16x16/20x20 sizing and vertical alignment |
| `internal/webui/templates/runs.html` | **modify** | Resolve pre-existing merge conflict markers (lines 221-334) |

## Architecture Decisions

1. **`icons.go` with template functions** (not template partials): Template functions returning `template.HTML` are simpler than partials for small inline SVGs and allow parameterization (name → SVG string). Consistent with existing `statusIcon()` / `statusClass()` pattern in `embed.go`.

2. **`currentColor` fill**: All SVGs use `fill="currentColor"` so they automatically adapt to light/dark themes, matching the pattern used by nav icons in `layout.html`.

3. **Fallback to text-only**: If a forge/adapter name has no icon mapping, the function returns empty string and the template shows text only — graceful degradation.

4. **Scope limited to landing page**: The runs table (`run_row.html`) is the primary target. The admin page loads adapter info via JavaScript so would need a separate JS-side icon mapping, which is out of scope for this issue.

5. **No forge column in run table currently**: The issue mentions forge icons, but the runs page doesn't display forge names per-row. The forge detection is project-level (one forge per repo). Adding a forge badge to the page header or sidebar brand area is the natural placement.

## Risks

| Risk | Mitigation |
|------|-----------|
| Merge conflict in `runs.html` (lines 221-334) | Resolve as first task — both branches' JS functionality must be preserved |
| SVG size inconsistency across browsers | Use explicit `width`/`height` attributes + CSS class for reliable sizing |
| `template.HTML` XSS risk | SVGs are hardcoded constants, not user input — safe by construction |
| Forge icons unused if no forge column exists | Add forge icon to sidebar brand or page header as project-level indicator |

## Testing Strategy

1. **Unit tests** (`icons_test.go`): Verify all known adapter/forge names return non-empty SVG strings, unknown names return empty string, returned SVGs contain expected attributes (`currentColor`, `viewBox`)
2. **Template compilation**: Existing `TestParseTemplates` (if present) will catch any FuncMap registration issues at compile time
3. **Visual verification**: Manual check in browser for icon rendering, theme switching, and sizing
