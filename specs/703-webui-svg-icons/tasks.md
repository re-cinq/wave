# Tasks

## Phase 1: Cleanup
- [ ] Task 1.1: Resolve merge conflict in `internal/webui/templates/runs.html` (lines 221-334) — preserve both branches' JavaScript functionality

## Phase 2: Core Implementation
- [ ] Task 2.1: Create `internal/webui/icons.go` with SVG icon definitions [P]
  - Define `adapterIcon(name string) template.HTML` — returns inline SVG for: `claude-code`, `browser`, `mock`, `opencode`, `gemini`, `codex`
  - Define `forgeIcon(name string) template.HTML` — returns inline SVG for: `github`, `gitlab`, `bitbucket`, `gitea`, `forgejo`, `codeberg`
  - All SVGs: 20x20 viewBox, `fill="currentColor"`, `class="icon-inline"`, `aria-hidden="true"`
  - Unknown names return empty string (fallback to text-only)
- [ ] Task 2.2: Register template functions in `internal/webui/embed.go` [P]
  - Add `"adapterIcon": adapterIcon` to FuncMap
  - Add `"forgeIcon": forgeIcon` to FuncMap
- [ ] Task 2.3: Add `.icon-inline` CSS class to `internal/webui/static/style.css` [P]
  - `width: 1em; height: 1em; vertical-align: -0.125em; display: inline-block;`
  - Ensure it works inside `.badge` elements

## Phase 3: Template Integration
- [ ] Task 3.1: Update `internal/webui/templates/partials/run_row.html` — add `{{adapterIcon $a}}` before adapter name in badge
- [ ] Task 3.2: Add forge indicator to the landing page (e.g., sidebar brand area or page header showing detected forge type with icon)

## Phase 4: Testing
- [ ] Task 4.1: Create `internal/webui/icons_test.go` — test all known names return valid SVG, unknown names return empty, SVGs contain `currentColor`
- [ ] Task 4.2: Run `go test ./internal/webui/...` to verify template compilation still works
- [ ] Task 4.3: Run `go vet ./...` and lint checks

## Phase 5: Validation
- [ ] Task 5.1: Build binary and launch webui to visually verify icons render correctly
- [ ] Task 5.2: Toggle light/dark theme to verify `currentColor` adaptation
